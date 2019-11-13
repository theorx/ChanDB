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
		DBFile:                           "data/main_db.txt",
		GCFile:                           "data/gc_db.txt",
		WriteOnlyFile:                    "data/write_db.txt",
		SyncSyscallIntervalMilliseconds:  100,
		GarbageCollectionIntervalSeconds: 2,
	})

	if err != nil {
		log.Println(err)
	}

	wg := &sync.WaitGroup{}

	wg.Add(2)

	amount := 20000000

	go func() {
		defer wg.Done()
		for i := 0; i < amount; i++ {
			db.Write("test#" + strconv.Itoa(i))
		}

	}()

	go func() {
		defer wg.Done()

		for i := 0; i < 2750000; {
			_, err := db.Read()

			if err != nil {
				//	log.Println("FOOKING ERROR MATE", err)
			} else {
				i++
				//	log.Println("Successful read", msg)
			}
		}

	}()

	log.Println("Waiting for routines to finish")
	wg.Wait()
	log.Println("Waiting done!")

	time.Sleep(time.Second * 3)

	err = db.Close()

	if err != nil {
		log.Println("Database closing failed:", err)
	}
}
