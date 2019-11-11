package ChanDB

import (
	"log"
	"testing"
)

func TestDB(t *testing.T) {

	db := &database{
		storageFile:              "../db_test.txt",
		syncIntervalMilliseconds: 100,
	}

	err := db.loadDatabase()

	log.Print(err)

	db.write("Payload 1")
	db.write("Payload 1")
	db.write("Payload 1")
	db.write("Payload 1")
	db.write("Payload 1")
	db.write("Payload 1")
	err = db.truncate()
	log.Println(err)
	db.write("next element, should be in the beginning of the file, but its not")

	log.Println(db.close())
}
