package ChanDB

import (
	"bufio"
	"errors"
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
			//log.Println("Syncing..")
			//err := d.fileHandle.Sync()
			//if err != nil {
			//	log.Println("Sync error on file", d.storageFile, ":", err)
			//}
			time.Sleep(time.Millisecond * time.Duration(d.syncIntervalMilliseconds))
		}
	}
}

func (d *database) seekNextRecord() {
	position := d.tokenPosition
	scanner := d.readScanner

	for scanner.Scan() {
		row := scanner.Text()

		//Database out of bounds, no active records left
		if len(row) == 0 && d.dbSize <= d.tokenPosition+1 {
			//	log.Println("Reached the end of the file? record is empty, current position", position)
			d.tokenPosition = d.dbSize
			return
		}

		//handle empty records
		if len(row) == 0 && d.dbSize > d.tokenPosition+1 {
			position += 1
			continue
		}

		if row[:1] == " " {
			d.tokenPosition = position
			return
		}
		position += int64(len(row) + 1)
	}

	d.tokenPosition = position
}

func (d *database) read(discardRecord bool) (string, error) {
	d.transactionLock.Lock()
	defer d.transactionLock.Unlock()

	row := d.readScanner.Text()

	if d.tokenPosition == 0 && len(row) == 0 || d.tokenPosition >= d.dbSize {
		d.seekNextRecord()
		row = d.readScanner.Text()

		if len(d.readScanner.Text()) == 0 {
			return "", errors.New("nothing was found")
		}
	}

	if discardRecord == true {

		_, err := d.fileHandle.WriteAt([]byte("-"), d.tokenPosition)
		if err != nil {
			return "", err
		}
	}

	d.tokenPosition += int64(len(row) + 1)

	//only defer when in non-error state
	defer d.seekNextRecord()

	if len(row) == 0 {
		return "", nil
	}

	return row[1:], nil
}

func (d *database) write(payload string) error {
	d.transactionLock.Lock()
	defer d.transactionLock.Unlock()

	num, err := d.fileHandle.WriteAt([]byte(" "+payload+"\n"), d.dbSize)

	if err != nil {
		log.Println("Error occurred when writing bytes with writeAt", err)
		return err
	}

	d.dbSize += int64(num)
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
