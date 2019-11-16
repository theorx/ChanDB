package Benchmark

import (
	"sync/atomic"
	"time"
)

func (b *Benchmark) benchStream(records int, readers int, seed int) time.Duration {
	b.truncate()
	db := b.setupAndSeedDB(seed, 10000)
	defer b.deferClose(db)

	counter := int64(0)
	start := time.Now().UnixNano()
	for i := 0; i < readers; i++ {
		go func() {
			stream := db.ReadStream()
			for range stream.Stream() {
				atomic.AddInt64(&counter, 1)
			}
		}()
	}

	for {
		time.Sleep(time.Millisecond * 50)
		if atomic.LoadInt64(&counter) >= int64(records) {
			break
		}
	}

	duration := time.Duration(time.Now().UnixNano() - start)
	b.totalStreamScore += int(float64(records) / duration.Seconds())

	return duration
}
