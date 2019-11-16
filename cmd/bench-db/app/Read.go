package Benchmark

import (
	"io"
	"log"
	"strconv"
	"time"
)

func (b *Benchmark) benchRead(records int, seed int) time.Duration {
	b.truncate()
	db := b.setupAndSeedDB(records+seed, 10000)
	defer b.deferClose(db)

	start := time.Now().UnixNano()

	counter := 0

	for {
		_, err := db.Read()
		if err == io.EOF {
			break
		}
		counter++
		if counter >= records {
			break
		}
	}

	if counter != records {
		log.Fatalln("Bench read "+strconv.Itoa(records)+" mil failed, only counted:", counter)
	}

	totalTime := time.Duration(time.Now().UnixNano() - start)

	//add to the global stats
	b.totalReadsPerSec += int(float64(seed20MIL) / totalTime.Seconds())
	b.totalBenchTime += totalTime

	return totalTime
}
