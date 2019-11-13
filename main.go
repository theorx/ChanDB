package main

import (
	"github.com/theorx/goDB/ChanDB"
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

	err = db.Close()

	if err != nil {
		log.Println("Database closing failed:", err)
	}
}
