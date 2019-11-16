package Benchmark

import (
	"log"
	"time"
)

func (b *Benchmark) benchWrite(records int, seed int) time.Duration {
	b.truncate()
	db := b.setupAndSeedDB(seed, 10000)
	defer b.deferClose(db)

	start := time.Now().UnixNano()

	for i := 0; i < records; i++ {
		err := db.Write(DataStr)
		if err != nil {
			log.Fatalln("Failed writing benchWrite20MIL:", err)
		}
	}

	totalTime := time.Duration(time.Now().UnixNano() - start)

	//add to the global stats
	b.totalWritesPerSec += int(float64(records) / totalTime.Seconds())
	b.totalBenchTime += totalTime

	return totalTime
}
