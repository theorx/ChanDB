package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"log"
	"os"
	"sync"
	"time"
)

func main() {
	log.SetFlags(log.Lshortfile)

	db, err := CreateDB("db.txt")

	if err != nil {
		log.Println(err)
		return
	}

	record := GetRecord()
	start := time.Now().UnixNano()

	for i := 0; i < 50000000; i++ {
		db.Write(record)
	}
	log.Println("[Writes] elapsed time", time.Duration(time.Now().UnixNano()-start))

	start = time.Now().UnixNano()
	for i := 0; i < 50000001; i++ {
		_, err := db.Read()

		if err != nil {
			log.Println(err, i)
		}

	}

	log.Println("[Reads] Elapsed time", time.Duration(time.Now().UnixNano()-start))
}

type ChanDB struct {
	storageFile     string
	fileHandle      *os.File
	transactionLock *sync.Mutex
	dbSize          int64
	tokenPosition   int64
	readScanner     *bufio.Scanner
}

/**
 */
func CreateDB(file string) (*ChanDB, error) {
	//open database file
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}

	db := &ChanDB{
		storageFile:     file,
		fileHandle:      f,
		transactionLock: &sync.Mutex{},
		tokenPosition:   0,
		dbSize:          0,
		readScanner:     bufio.NewScanner(f), //will eventually move this initialization to the database initial start / restart logic
	}

	return db, db.loadDatabase()
}

func (c *ChanDB) loadDatabase() error {
	//write the database size (number of bytes within the file) to the internal variable

	statInfo, err := c.fileHandle.Stat()
	if err != nil {
		return err
	}

	c.dbSize = statInfo.Size()

	return nil
}

func (c *ChanDB) seekNextRecord() {
	//seek for the next "newline and space" -> "\n "

	position := c.tokenPosition
	scanner := c.readScanner

	for scanner.Scan() {
		row := scanner.Text()

		//Database out of bounds, no active records left
		if len(row) == 0 && c.dbSize <= c.tokenPosition+1 {
			log.Println("Reached the end of the file? record is empty, current position", position)
			c.tokenPosition = c.dbSize
			return
		}

		//handle empty records
		if len(row) == 0 && c.dbSize > c.tokenPosition+1 {
			position += 1
			continue
		}

		if row[:1] == " " {
			//log.Println("First 5 chars of the row", row[:5])
			c.tokenPosition = position
			//		log.Println(c.tokenPosition, position, "<<<<-")
			return
		}
		//	log.Println(c.tokenPosition, position, "<<<<-")
		position += int64(len(row) + 1)
	}

	c.tokenPosition = position
}

func (c *ChanDB) Read() (string, error) {
	c.transactionLock.Lock()
	defer c.transactionLock.Unlock()

	row := c.readScanner.Text()

	if c.tokenPosition == 0 && len(row) == 0 {
		c.seekNextRecord()
		row = c.readScanner.Text()

		if len(c.readScanner.Text()) == 0 {
			return "", errors.New("Nothing was found")
		}
	}

	_, err := c.fileHandle.WriteAt([]byte("-"), c.tokenPosition)
	if err != nil {
		return "", err
	}
	c.tokenPosition += int64(len(row) + 1)

	//only defer when in non-error state
	defer c.seekNextRecord()

	return row, nil
}

func (c *ChanDB) Write(payload string) error {
	c.transactionLock.Lock()
	defer c.transactionLock.Unlock()

	num, err := c.fileHandle.WriteAt([]byte(" "+payload+"\n"), c.dbSize)

	if err != nil {
		log.Println("Error occurred when writing bytes with writeAt", err)
		return err
	}

	c.dbSize += int64(num)
	return nil
}

//todo: create a routine for garbage collection

type Record struct {
	Name string `json:"name"`
	Time int64  `json:"time"`
}

func GetRecord() string {

	payload, err := json.Marshal(&Record{Name: "Test record", Time: time.Now().UnixNano()})

	if err != nil {
		log.Println("Failed to marshal json in GetRecord()")
		return ""
	}
	return string(payload)
}
