package main

import (
	"github.com/theorx/ChanDB/pkg/ChanDB"
	"log"
)

func main() {
	log.SetFlags(log.Lshortfile)

	db, err := ChanDB.CreateDatabase(&ChanDB.Settings{
		DBFile:                           "data/main_db.txt",
		GCFile:                           "data/gc_db.txt",
		WriteOnlyFile:                    "data/write_db.txt",
		SyncSyscallIntervalMilliseconds:  100,
		GarbageCollectionIntervalSeconds: 2,
		LogFunction:                      log.Println,
	})

	if err != nil {
		log.Println(err)

	}

	db.Write("test 44")

	err = db.Close()

	if err != nil {
		log.Println("Database closing failed:", err)
	}
	log.Println("Closed the database..")
}
