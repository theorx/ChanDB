package main

import (
	"github.com/theorx/goDB/ChanDB"
	"log"
	"strconv"
	"sync"
	"time"
)

func main() {
	log.SetFlags(log.Lshortfile)

	db, err := ChanDB.CreateDatabase(&ChanDB.Settings{
		DBFile:                           "main_db.txt",
		GCFile:                           "gc_db.txt",
		WriteOnlyFile:                    "write_db.txt",
		SyncSyscallIntervalMilliseconds:  1000,
		GarbageCollectionIntervalSeconds: 1,
	})

	if err != nil {
		log.Println(err)
	}

	wg := &sync.WaitGroup{}

	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			time.Sleep(time.Microsecond)
			db.Write("Record numero#" + strconv.Itoa(i))
		}

	}()

	go func() {
		defer wg.Done()
		time.Sleep(time.Microsecond * 5000)

		for i := 0; i < 1000; i++ {
			time.Sleep(time.Microsecond)
			_, err := db.Read()

			if err != nil {
				log.Println("FOOKING ERROR MATE", err)
			}
		}

	}()

	wg.Wait()

	err = db.Close()

	if err != nil {
		log.Println("Database closing failed:", err)
	}

}
