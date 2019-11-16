package ChanDB

import (
	"bufio"
	"github.com/theorx/goDB/Signal"
	"io"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type database struct {
	readScanner              *bufio.Scanner
	signal                   *Signal.Signal
	fileHandle               *os.File
	writeLock                *sync.Mutex
	readLock                 *sync.Mutex
	log                      LogFunction
	readStream               chan string
	storageFile              string
	dbSize                   int64
	tokenPosition            int64
	recordsStored            int64
	syncIntervalMilliseconds int
	scannerEOF               bool
	syncQuitSignal           chan bool
	readStreamQuitSignal     chan bool
}

func (d *database) resetScanner() {
	d.readScanner = bufio.NewScanner(d.fileHandle)
	d.tokenPosition = 0
}

func (d *database) loadDatabase() error {
	d.log("loadDatabase()", d.storageFile)

	if d.writeLock == nil {
		d.writeLock = &sync.Mutex{}
	}

	if d.readLock == nil {
		d.readLock = &sync.Mutex{}
	}

	//create read stream
	if d.readStream == nil {
		d.readStream = make(chan string, 1)
	}

	//signal for streamReads()
	if d.signal == nil {
		d.signal = Signal.CreateSignal()
	}

	fh, err := os.OpenFile(d.storageFile, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}

	d.fileHandle = fh
	d.scannerEOF = false

	//count the database records
	err = d.countRecords()

	if err != nil {
		d.log("Failed counting rows in countRecords() while initializing the database", err)
		return err
	}
	//reset the scanner to position 0
	d.resetScanner()

	//write the database size (number of bytes within the file) to the internal variable
	statInfo, err := d.fileHandle.Stat()
	if err != nil {
		return err
	}
	d.dbSize = statInfo.Size()
	//handle sync

	if d.syncQuitSignal != nil {
		d.syncQuitSignal <- true
	} else {
		d.syncQuitSignal = make(chan bool, 0)
	}

	go d.syncRoutine()

	return nil
}

func (d *database) syncRoutine() {
	d.log("Starting sync routine for", d.storageFile)
	for {
		select {
		case <-d.syncQuitSignal:
			d.log("Quitting sync routine for", d.storageFile)
			return
		default:
			err := d.fileHandle.Sync()
			if err != nil {
				d.log("Sync error on file", d.storageFile, ":", err)
			}
			time.Sleep(time.Millisecond * time.Duration(d.syncIntervalMilliseconds))
		}
	}
}

func (d *database) seekNextRecord() (string, bool) {

	if d.scannerEOF {
		time.Sleep(time.Millisecond * 25)
		d.scannerEOF = false
		//here we will reset the position and create a new scanner

		seekPosition := d.tokenPosition - 100

		if seekPosition < 0 {
			seekPosition = 0
		}
		d.tokenPosition = seekPosition

		_, err := d.fileHandle.Seek(seekPosition, io.SeekStart)

		if err != nil {
			d.log("Failed to seek to new position after scannerEOF", seekPosition, err)
		}

		d.resetScanner()
	}

	position := atomic.LoadInt64(&d.tokenPosition)
	scanner := d.readScanner

	d.readLock.Lock()
	for scanner.Scan() {
		row := scanner.Text()

		if len(row) == 0 && d.dbSize > d.tokenPosition+1 {
			position += 1
			continue
		}

		if row[:1] == " " {
			d.tokenPosition = position
			d.readLock.Unlock()
			return row, true
		}
		position += int64(len(row) + 1)
	}
	d.readLock.Unlock()

	atomic.StoreInt64(&d.tokenPosition, position)

	return "", false
}

func (d *database) read(discardRecord bool) (string, error) {
	row, status := d.seekNextRecord()

	if status == false {
		d.scannerEOF = true
		return "", io.EOF
	}

	if discardRecord == true {
		_, err := d.fileHandle.WriteAt([]byte("-"), d.tokenPosition)
		d.decrementRecordsStored()
		if err != nil {
			return "", err
		}
	}

	//d.readLock.Lock()
	atomic.AddInt64(&d.tokenPosition, int64(len(row)+1))
	//d.tokenPosition += int64(len(row) + 1)
	//d.readLock.Unlock()

	if len(row) == 0 {
		return "", nil
	}

	return row[1:], nil
}

func (d *database) readStreamRoutine() {
	d.log("Starting readStreamRoutine() for", d.storageFile)
	for {
		select {
		case <-d.readStreamQuitSignal:
			d.log("Quitting readStreamRoutine() for", d.storageFile)
			return
		default:

			msg, err := d.read(true)
			if err == io.EOF {
				//wait for the signal to continue
				<-d.signal.Channel()
				continue
			}

			d.readStream <- msg
		}
	}
}

func (d *database) streamReads() <-chan string {

	if d.readStreamQuitSignal == nil {
		d.readStreamQuitSignal = make(chan bool, 0)
		go d.readStreamRoutine()
	}

	//block until data is available
	return d.readStream
}

func (d *database) write(payload string) error {
	d.writeLock.Lock()

	num, err := d.fileHandle.WriteAt([]byte(" "+payload+"\n"), d.dbSize)

	if err != nil {
		d.writeLock.Unlock()
		d.log("Error occurred when writing bytes with writeAt", err)
		return err
	}

	d.dbSize += int64(num)
	d.writeLock.Unlock()
	d.incrementRecordsStored()
	return nil
}

func (d *database) truncate() error {

	err := d.fileHandle.Truncate(0)
	if err != nil {
		return err
	}

	d.resetScanner()
	d.dbSize = 0
	d.recordsStored = 0

	return nil
}

/**
Close() will have to be called 100% of the times when database is being shut down
Closing database ensures that all of the data stored in the database is actually stored on the disk
and all of the routines using any of the data that is temporarily read from the database will store
all of it back before database file handles are closed
*/
func (d *database) close() error {
	d.readLock.Lock()
	//write the records back from the readStream when shutting down
writeBackLoop:
	for {
		select {
		case row, _ := <-d.readStream:
			log.Println("Writing back..", row)
			err := d.write(row)

			if err != nil {
				d.log("database.close() failed to write back a records from readStream", row, err)
			}
		default:
			break writeBackLoop
		}
	}

	if d.fileHandle == nil {
		return nil
	}
	defer d.fileHandle.Close()

	d.syncQuitSignal <- true
	d.syncQuitSignal = nil

	log.Println("Waiting for readStream to shut down")
	//	d.readStreamQuitSignal <- true
	//d.readStreamQuitSignal = nil

	return d.fileHandle.Sync()
}

func (d *database) length() int64 {
	return atomic.LoadInt64(&d.recordsStored)
}

func (d *database) incrementRecordsStored() {
	//signal new insert
	d.signal.Signal()

	atomic.AddInt64(&d.recordsStored, 1)
}

func (d *database) decrementRecordsStored() {
	atomic.AddInt64(&d.recordsStored, -1)
}

func (d *database) setRecordsStored(count int64) {
	atomic.StoreInt64(&d.recordsStored, count)
}

func (d *database) countRecords() error {
	//todo implement header
	scanner := bufio.NewScanner(d.fileHandle)
	records := int64(0)

	for scanner.Scan() {
		row := scanner.Text()

		if row[:1] == " " {
			records++
		}
	}
	_, err := d.fileHandle.Seek(0, io.SeekStart)
	atomic.StoreInt64(&d.recordsStored, records)

	return err
}
