package Benchmark

import (
	"log"
	"sync"
	"time"
)

func (b *Benchmark) benchMixed(records int, writePercent int, readPercent int, seed int) *MixedResults {
	b.truncate()
	db := b.setupAndSeedDB(seed, 10000)
	defer b.deferClose(db)

	if readPercent+writePercent > 100 {
		log.Fatalln("Invalid call to benchMixed, read and write percentage total is over 100 percent")
	}

	writeRecords := (records / 100) * writePercent
	readRecords := (records / 100) * readPercent

	wg := &sync.WaitGroup{}
	wg.Add(2)

	start := time.Now().UnixNano()

	readTotal := time.Duration(0)
	writeTotal := time.Duration(0)

	go func() {
		defer wg.Done()
		startWrite := time.Now().UnixNano()

		for i := 0; i < writeRecords; i++ {
			err := db.Write(DataStr)

			if err != nil {
				log.Fatalln("Failed writing in benchMixed()", records, writePercent, readPercent, seed, err)
			}
		}
		writeTotal = time.Duration(time.Now().UnixNano() - startWrite)
	}()

	go func() {
		defer wg.Done()
		fails := 0
		startRead := time.Now().UnixNano()

		for i := 0; i < readRecords; {
			_, err := db.Read()

			if err != nil {
				fails++
			} else {
				i++
			}

			if fails == 50 { //threshold for failing, takes approx 5 seconds to fail
				log.Fatalln("Failed reading for 50 times in benchMixed()", i, records, writePercent, readPercent, seed, err)
			}
		}

		readTotal = time.Duration(time.Now().UnixNano() - startRead)
	}()

	wg.Wait()

	totalTime := time.Duration(time.Now().UnixNano() - start)

	//add to the global stats
	b.totalWritesPerSec += int(float64(writeRecords) / writeTotal.Seconds())
	b.totalReadsPerSec += int(float64(readRecords) / readTotal.Seconds())
	b.totalBenchTime += totalTime

	return &MixedResults{
		TotalTime:    totalTime,
		WriteTime:    writeTotal,
		ReadTime:     readTotal,
		WriteRecords: writeRecords,
		ReadRecords:  readRecords,
	}
}
