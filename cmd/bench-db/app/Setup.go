package Benchmark

import (
	"bytes"
	"github.com/theorx/ChanDB/pkg/ChanDB"
	"io"
	"log"
	"os"
	"time"
)

const (
	seedNone    int    = 0
	seed1MIL    int    = 1000000
	seed5MIL           = seed1MIL * 5
	seed20MIL          = seed1MIL * 20
	seed50MIL          = seed1MIL * 50
	seed100MIL         = seed1MIL * 100
	bench1MIL   int    = 1000000
	bench5MIL          = bench1MIL * 5
	bench20MIL         = bench1MIL * 20
	bench50MIL         = bench1MIL * 50
	bench100MIL        = bench1MIL * 100
	DataStr     string = "twitchtwitchtwitchtwitchtwitch"
)

type MixedResults struct {
	TotalTime    time.Duration
	WriteTime    time.Duration
	ReadTime     time.Duration
	WriteRecords int
	ReadRecords  int
}

func (b *Benchmark) setupAndSeedDB(records int, gcSeconds int) ChanDB.Database {
	seedingRequired := true
	for num, file := range b.cacheMap {
		if num == records {
			seedingRequired = false
			dbFH, err := os.OpenFile(b.dbFile, os.O_RDWR|os.O_CREATE, 0644)

			if err != nil {
				log.Fatalln("Loading database file failed while trying to use cache", err)
			}

			cacheFH, err := os.OpenFile(file, os.O_RDONLY, 0644)

			if err != nil {
				log.Fatalln("Failed loading cache file while trying to use cache", err)
			}

			err = dbFH.Truncate(0)

			if err != nil {
				log.Fatalln("Truncating db file before applying cache failed", err)
			}

			num, err := io.Copy(dbFH, cacheFH)

			log.Println("Used cache and copied:", num, file)

			if err != nil {
				log.Fatalln("Failed to copy cache data to db file", err)
			}
		}
	}

	db, err := ChanDB.CreateDatabase(&ChanDB.Settings{
		DBFile:                           b.dbFile,
		GCFile:                           b.gcFile,
		WriteOnlyFile:                    b.writeFile,
		SyncSyscallIntervalMilliseconds:  100,
		GarbageCollectionIntervalSeconds: gcSeconds,
		LogFunction:                      log.Println,
	})

	if err != nil {
		log.Fatalln("Failed to create the database:", err)
	}

	if seedingRequired {
		for i := 0; i < records; i++ {
			err := db.Write(DataStr)

			if err != nil {
				log.Fatalln("Failed writing while seeding the database:", err)
			}
		}
	}

	return db
}

func (b *Benchmark) truncate() {

	files := []string{b.dbFile, b.gcFile, b.writeFile}
	for _, file := range files {
		log.Println("Truncating file", file)
		f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE, 0644)

		if err != nil {
			log.Fatalln("Failed to open file", file, err)
		}

		err = f.Truncate(0)

		if err != nil {
			log.Fatalln("Failed to truncate", file)
		}
	}
}

func (b *Benchmark) generateSeedCaches() {

	b.cacheMap[seed20MIL] = b.benchDir + "/cache/20M.txt"
	b.cacheMap[seed50MIL] = b.benchDir + "/cache/50M.txt"
	b.cacheMap[seed100MIL] = b.benchDir + "/cache/100M.txt"

	for records, name := range b.cacheMap {

		fh, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE, 0644)

		if err != nil {
			log.Fatalln("Failed to create cache file for", records, "with name", name)
		}

		h := &ChanDB.Header{
			Records: int64(records),
			Version: "benchmark",
		}

		err = h.Write(fh)

		fh.Seek(ChanDB.HeaderBytes, io.SeekStart)

		if err != nil {
			log.Fatalln("Failed to write header to the cache file")
		}

		statInfo, err := fh.Stat()

		if err != nil {
			log.Fatalln("Failed to get stat info of cache file", name)
		}

		if statInfo.Size() < int64(records) {
			//we do the seeding
			var buffer bytes.Buffer
			for i := 0; i < records; i++ {
				buffer.WriteString(" " + DataStr + "\n")
			}
			_, err := fh.Write(buffer.Bytes())
			buffer.Reset()

			if err != nil {
				log.Fatalln("Failed to write cache file for", name)
			}
		}
		log.Println("Cache file created / updated for", name)
		fh.Close()
	}
}

func (b *Benchmark) deferClose(db ChanDB.Database) {
	err := db.Close()
	if err != nil {
		log.Fatalln("Failed db close in defer..", err)
	}
}
