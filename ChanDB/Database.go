package ChanDB

import (
	"bufio"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

type database struct {
	storageFile              string
	fileHandle               *os.File
	transactionLock          *sync.Mutex
	dbSize                   int64
	tokenPosition            int64
	readScanner              *bufio.Scanner
	syncQuitSignal           chan bool
	syncIntervalMilliseconds int
	scannerEOF               bool
}

func (d *database) resetScanner() {
	d.readScanner = bufio.NewScanner(d.fileHandle)
	d.tokenPosition = 0
}

func (d *database) loadDatabase() error {
	log.Println("loadDatabase()", d.storageFile)

	if d.transactionLock == nil {
		d.transactionLock = &sync.Mutex{}
	}

	fh, err := os.OpenFile(d.storageFile, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}

	d.fileHandle = fh

	//
	d.scannerEOF = false

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
	log.Println("Starting sync routine for", d.storageFile)
	for {
		select {
		case <-d.syncQuitSignal:
			log.Println("Quitting sync routine for", d.storageFile)
			return
		default:
			err := d.fileHandle.Sync()
			if err != nil {
				log.Println("Sync error on file", d.storageFile, ":", err)
			}
			time.Sleep(time.Millisecond * time.Duration(d.syncIntervalMilliseconds))
		}
	}
}

func (d *database) seekNextRecord() bool {

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
			log.Println("Failed to seek to new position after scannerEOF", seekPosition, err)
		}

		d.resetScanner()
	}

	position := d.tokenPosition
	scanner := d.readScanner

	for scanner.Scan() {
		row := scanner.Text()

		if len(row) == 0 && d.dbSize > d.tokenPosition+1 {
			position += 1
			continue
		}

		if row[:1] == " " {
			d.tokenPosition = position
			return true
		}
		position += int64(len(row) + 1)
	}

	d.tokenPosition = position
	return false
}

func (d *database) read(discardRecord bool) (string, error) {
	status := d.seekNextRecord()

	if status == false {
		d.scannerEOF = true
		return "", io.EOF
	}

	row := d.readScanner.Text()

	if discardRecord == true {

		//d.transactionLock.Lock()
		_, err := d.fileHandle.WriteAt([]byte("-"), d.tokenPosition)
		//d.transactionLock.Unlock()

		if err != nil {
			return "", err
		}
	}

	d.tokenPosition += int64(len(row) + 1)

	if len(row) == 0 {
		return "", nil
	}

	return row[1:], nil
}

func (d *database) write(payload string) error {
	d.transactionLock.Lock()

	num, err := d.fileHandle.WriteAt([]byte(" "+payload+"\n"), d.dbSize)

	if err != nil {
		d.transactionLock.Unlock()
		log.Println("Error occurred when writing bytes with writeAt", err)
		return err
	}

	d.dbSize += int64(num)
	d.transactionLock.Unlock()
	return nil
}

func (d *database) truncate() error {

	err := d.fileHandle.Truncate(0)
	if err != nil {
		return err
	}

	d.resetScanner()
	d.dbSize = 0

	return nil
}

func (d *database) close() error {
	if d.fileHandle == nil {
		return nil
	}
	defer d.fileHandle.Close()
	d.syncQuitSignal <- true
	d.syncQuitSignal = nil

	return d.fileHandle.Sync()
}
