package main

import (
	"github.com/theorx/goDB/ChanDB"
	"log"
	"strconv"
	"sync"
)

func main() {

	db, err := ChanDB.CreateDatabase(&ChanDB.Settings{
		DBFile:                           "db_file.txt",
		GCFile:                           "gc_file.txt",
		WriteOnlyFile:                    "wo_file.txt",
		SyncSyscallIntervalMilliseconds:  100,
		GarbageCollectionIntervalSeconds: 25,
		LogFunction:                      nil,
	})

	if err != nil {
		log.Println("Failed to create db", err)
		return
	}

	//write 1 mil records

	err = db.Truncate()

	if err != nil {
		log.Println("Failed to truncate", err)
		return
	}

	for i := 0; i < 500000; i++ {
		err := db.Write(strconv.Itoa(i))
		if err != nil {
			log.Println("Failed writing to the db", err)
		}
	}

	log.Println("Finished writing..")

	wg := &sync.WaitGroup{}

	workers := 400

	counters := make([]int, workers)

	data := make(map[string]bool)

	wl := &sync.Mutex{}

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func(index int) {
			defer wg.Done()
			for {
				msg, err := db.Read()

				if err != nil {
					//log.Println("Error in routine", index)
					return
				}

				wl.Lock()
				_, exists := data[msg]

				if exists {
					log.Println("Collision detected!")
					log.Println(msg, data)
					return
				}

				data[msg] = true
				wl.Unlock()

				counters[index]++
			}
		}(i)
	}

	wg.Wait()

	total := 0

	log.Println(len(data))

	for _, count := range counters {
		total += count
	}

	log.Println("Total of all counters", total)

	log.Println(db.Length())
}
