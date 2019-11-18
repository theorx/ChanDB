package ChanDB

import (
	"bufio"
	"errors"
	"github.com/theorx/ChanDB/internal/Version"
	"github.com/theorx/ChanDB/pkg/Signal"
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
	subRoutineSpawnLock      *sync.Mutex
	header                   *Header
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

func createDatabase(dbFile string, syncIntervalMilliseconds int, logFunction LogFunction) (*database, error) {
	if logFunction == nil {
		return nil, errors.New("invalid log function given for createDatabase() function")
	}

	instance := &database{
		signal:    Signal.CreateSignal(),
		writeLock: &sync.Mutex{},
		readLock:  &sync.Mutex{},
		log: func(v ...interface{}) {
			params := make([]interface{}, 0)
			params = append(params, dbFile+" => ")
			for _, param := range v {
				params = append(params, param)
			}
			logFunction(params...)
		},
		subRoutineSpawnLock:      &sync.Mutex{},
		readStream:               make(chan string, 0),
		storageFile:              dbFile,
		syncIntervalMilliseconds: syncIntervalMilliseconds,
		header: &Header{
			Version: Version.Version,
		},
	}

	return instance, instance.loadDatabase()
}

func (d *database) resetScanner() {
	d.readScanner = bufio.NewScanner(d.fileHandle)
	atomic.StoreInt64(&d.tokenPosition, HeaderBytes)
}

func (d *database) loadDatabase() error {
	d.log("initializing database in the loadDatabase function")

	fh, err := os.OpenFile(d.storageFile, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}

	d.fileHandle = fh
	d.scannerEOF = false

	err = d.setDatabaseSize()
	if err != nil {
		return err
	}

	err = d.header.Read(d.fileHandle)

	if err != nil {
		log.Println("CCC Reading records from disk..")
		err = d.updateStoredRecords()
		if err != nil {
			return err
		}
	}

	d.header.Records = d.recordsStored
	err = d.header.Write(d.fileHandle)
	if err != nil {
		return err
	}

	d.spawnSyncRoutine()

	return nil
}

func (d *database) updateStoredRecords() error {
	err := d.countRecords()
	if err != nil {
		return err
	}
	//reset the scanner to position 0
	d.resetScanner()
	return nil
}

func (d *database) setDatabaseSize() error {
	//write the database size (number of bytes within the file) to the internal variable
	statInfo, err := d.fileHandle.Stat()
	if err != nil {
		return err
	}
	d.dbSize = statInfo.Size()
	return nil
}

func (d *database) spawnSyncRoutine() {
	d.subRoutineSpawnLock.Lock()
	if d.syncQuitSignal != nil {
		d.syncQuitSignal <- true
	} else {
		d.syncQuitSignal = make(chan bool, 0)
	}
	go d.syncRoutine()
	d.subRoutineSpawnLock.Unlock()
}

func (d *database) syncRoutine() {
	d.log("Starting sync routine")
	for {
		select {
		case <-d.syncQuitSignal:
			d.log("Quitting sync routine")
			return
		default:
			err := d.fileHandle.Sync()
			if err != nil {
				d.log("Sync error:", err)
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

		if seekPosition < HeaderBytes {
			seekPosition = HeaderBytes
		}
		atomic.StoreInt64(&d.tokenPosition, seekPosition)

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

	atomic.AddInt64(&d.tokenPosition, int64(len(row)+1))

	if len(row) == 0 {
		return "", nil
	}

	return row[1:], nil
}

func (d *database) readStreamRoutine() {
	d.log("Starting readStreamRoutine() ")
	for {
		select {
		case <-d.readStreamQuitSignal:
			d.log("Quitting readStreamRoutine() ")
			//d.readStreamQuitSignal = nil
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
	d.subRoutineSpawnLock.Lock()
	if d.readStreamQuitSignal == nil {
		d.readStreamQuitSignal = make(chan bool, 1)
		go d.readStreamRoutine()
	}
	d.subRoutineSpawnLock.Unlock()

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
	d.header.Records = 0
	//update the header after truncate

	return d.header.Write(d.fileHandle)
}

/**
Close() will have to be called 100% of the times when database is being shut down
Closing database ensures that all of the data stored in the database is actually stored on the disk
and all of the routines using any of the data that is temporarily read from the database will store
all of it back before database file handles are closed
*/
func (d *database) close() error {
	d.readLock.Lock()
	defer d.readLock.Unlock()
	//write the records back from the readStream when shutting down
	d.shutDownReadStream()

	if d.fileHandle == nil {
		return nil
	}
	defer d.fileHandle.Close()

	//update the number of records stored in the header
	d.header.Records = d.recordsStored
	err := d.header.Write(d.fileHandle)

	if err != nil {
		return err
	}

	d.syncQuitSignal <- true
	d.syncQuitSignal = nil

	return d.fileHandle.Sync()
}

func (d *database) shutDownReadStream() {
	d.subRoutineSpawnLock.Lock()
	if len(d.readStreamQuitSignal) == 0 && d.readStreamQuitSignal != nil {
		d.readStreamQuitSignal <- true
	}

writeBackLoop:
	for {
		select {
		case row, _ := <-d.readStream:
			err := d.write(row)
			d.log("Got a record from shutDOwnReadStream() - writing it back", row)
			if err != nil {
				d.log("database.close() failed to write back a records from readStream", row, err)
			}
		default:
			break writeBackLoop
		}
	}

	d.subRoutineSpawnLock.Unlock()
}

func (d *database) length() int64 {
	return atomic.LoadInt64(&d.recordsStored)
}

func (d *database) incrementRecordsStored() {
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
	_, err := d.fileHandle.Seek(HeaderBytes, io.SeekStart)

	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(d.fileHandle)
	records := int64(0)

	for scanner.Scan() {
		row := scanner.Text()

		if len(row) > 1 && row[:1] == " " {
			records++
		}
	}
	_, err = d.fileHandle.Seek(HeaderBytes, io.SeekStart)
	atomic.StoreInt64(&d.recordsStored, records)

	return err
}
