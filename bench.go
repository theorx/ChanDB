package main

import (
	"github.com/theorx/goDB/ChanDB"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	logFile      string = "benchLog.txt"
	dbFile       string = "bench/db_file.txt"
	gcFile       string = "bench/gc_file.txt"
	writeFile    string = "bench/write_file.txt"
	SEED_1_MIL   int    = 1000000
	SEED_5_MIL   int    = SEED_1_MIL * 5
	SEED_20_MIL  int    = SEED_1_MIL * 20
	SEED_50_MIL  int    = SEED_1_MIL * 50
	SEED_100_MIL int    = SEED_1_MIL * 100
	DataStr      string = "twitchtwitchtwitchtwitchtwitch"
)

var logFH *os.File = nil
var cacheMap map[int]string = make(map[int]string)

type MixedResults struct {
	TotalTime    time.Duration
	WriteTime    time.Duration
	ReadTime     time.Duration
	WriteRecords int
	ReadRecords  int
}

var totalWritesPerSec = 0
var totalReadsPerSec = 0
var totalBenchTime time.Duration = 0

func main() {
	openLog()
	writeLog("### Started benchmark - " + time.Now().String() + " ###")
	generateSeedCaches()

	writeStatsMixed("#1 5M w50-r50 50M seed", benchMixed(SEED_5_MIL, 50, 50, SEED_50_MIL))
	writeStats("#2 20M writes", benchWrite20MIL(), SEED_20_MIL)
	writeStats("#3 20M reads", benchRead20MIL(), SEED_20_MIL)
	writeStats("#4 20M writes 100M seed", benchWrite20MIL100MSeed(), SEED_20_MIL)
	writeStats("#5 20M reads 100M seed", benchRead20MIL100MSeed(), SEED_20_MIL)
	writeStatsMixed("#6 20M w50-r50 50M seed", benchMixed(SEED_20_MIL, 50, 50, SEED_50_MIL))
	writeStatsMixed("#7 20M w25-r75 50M seed", benchMixed(SEED_20_MIL, 25, 75, SEED_50_MIL))
	writeStatsMixed("#8 20M w75-r25 50M seed", benchMixed(SEED_20_MIL, 75, 25, SEED_50_MIL))
	writeStatsMixed("#9 20M w90-r10 50M seed", benchMixed(SEED_20_MIL, 90, 10, SEED_50_MIL))
	writeStatsMixed("#10 20M w10-r90 50M seed", benchMixed(SEED_20_MIL, 10, 90, SEED_50_MIL))
	writeStatsMixed("#11 20M w50-r50 no seed", benchMixed(SEED_20_MIL, 50, 50, 0))
	writeStatsMixed("#12 20M w75-r25 no seed", benchMixed(SEED_20_MIL, 75, 25, 0))
	writeStatsMixed("#13 20M w90-r10 no seed", benchMixed(SEED_20_MIL, 90, 10, 0))

	writeLog("\n\nBenchmark results - ")
	writeLog("Read score:" + strconv.Itoa(totalReadsPerSec))
	writeLog("Write score:" + strconv.Itoa(totalWritesPerSec))
	writeLog("\nBenchmark time:" + totalBenchTime.String())
	writeLog("### Finished benchmark - " + time.Now().String() + " ###\n\n")
}

func generateSeedCaches() {

	cacheMap[SEED_20_MIL] = "bench/cache/20M.txt"
	cacheMap[SEED_50_MIL] = "bench/cache/50M.txt"
	cacheMap[SEED_100_MIL] = "bench/cache/100M.txt"

	for records, name := range cacheMap {

		fh, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE, 0644)

		if err != nil {
			log.Fatalln("Failed to create cache file for", records, "with name", name)
		}

		statInfo, err := fh.Stat()

		if err != nil {
			log.Fatalln("Failed to get stat info of cache file", name)
		}

		if statInfo.Size() < int64(records) {
			//we do the seeding
			_, err := fh.WriteString(strings.Repeat(" "+DataStr+"\n", records))
			if err != nil {
				log.Fatalln("Failed to write cache file for", name)
			}
		}
		log.Println("Cache file created / updated for", name)
	}
}

func writeStats(line string, duration time.Duration, records int) {
	writeLog(line + "\t\t\t\t\t - " + duration.String())
	writeLog("\t Op/s: " + strconv.Itoa(int(float64(records)/duration.Seconds())))
	writeLog("\t Operations: " + strconv.Itoa(int(records)))
}
func writeStatsMixed(line string, result *MixedResults) {
	writeLog(line + "\t\t\t\t\t - " + result.TotalTime.String())
	writeLog("\t Reads/s: " + strconv.Itoa(int(float64(result.ReadRecords)/result.ReadTime.Seconds())))
	writeLog("\t Writes/s: " + strconv.Itoa(int(float64(result.WriteRecords)/result.WriteTime.Seconds())))
	writeLog("\t Operations: " + strconv.Itoa(int(result.ReadRecords+result.WriteRecords)))
}
func benchMixed(records int, writePercent int, readPercent int, seed int) *MixedResults {

	truncate()
	db := setupAndSeedDB(seed, 10000)
	defer deferClose(db)

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
		//write

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
		//read
		for i := 0; i < readRecords; {
			_, err := db.Read()

			if err != nil {
				fails++
			} else {
				i++
			}

			if fails == 50 { //threshold for failing, takes approx 5 seconds to fail
				log.Fatalln("Failed reading for 50 times in benchMixed()", records, writePercent, readPercent, seed, err)
			}
		}

		readTotal = time.Duration(time.Now().UnixNano() - startRead)
	}()

	wg.Wait()

	totalTime := time.Duration(time.Now().UnixNano() - start)

	//add to the global stats
	totalWritesPerSec += int(float64(writeRecords) / writeTotal.Seconds())
	totalReadsPerSec += int(float64(readRecords) / readTotal.Seconds())
	totalBenchTime += totalTime

	return &MixedResults{
		TotalTime:    totalTime,
		WriteTime:    writeTotal,
		ReadTime:     readTotal,
		WriteRecords: writeRecords,
		ReadRecords:  readRecords,
	}
}
func benchRead20MIL100MSeed() time.Duration {
	truncate()
	db := setupAndSeedDB(SEED_100_MIL, 10000)
	defer deferClose(db)

	start := time.Now().UnixNano()

	counter := 0

	for {
		_, err := db.Read()
		if err == io.EOF {
			break
		}
		counter++

		if counter >= SEED_20_MIL {
			break
		}
	}

	if counter != SEED_20_MIL {
		log.Fatalln("Bench read 20 mil failed, only counted:", counter)
	}

	totalTime := time.Duration(time.Now().UnixNano() - start)

	//add to the global stats
	totalReadsPerSec += int(float64(SEED_20_MIL) / totalTime.Seconds())
	totalBenchTime += totalTime

	return totalTime
}
func benchWrite20MIL100MSeed() time.Duration {
	truncate()
	db := setupAndSeedDB(SEED_100_MIL, 10000)
	defer deferClose(db)

	start := time.Now().UnixNano()

	for i := 0; i < SEED_20_MIL; i++ {
		err := db.Write(DataStr)
		if err != nil {
			log.Fatalln("Failed writing benchWrite20MIL:", err)
		}
	}

	totalTime := time.Duration(time.Now().UnixNano() - start)

	//add to the global stats
	totalWritesPerSec += int(float64(SEED_20_MIL) / totalTime.Seconds())
	totalBenchTime += totalTime

	return totalTime
}
func benchWrite20MIL() time.Duration {
	truncate()
	db := setupAndSeedDB(0, 10000)
	defer deferClose(db)

	start := time.Now().UnixNano()

	for i := 0; i < SEED_20_MIL; i++ {
		err := db.Write(DataStr)
		if err != nil {
			log.Fatalln("Failed writing benchWrite20MIL:", err)
		}
	}

	totalTime := time.Duration(time.Now().UnixNano() - start)

	//add to the global stats
	totalWritesPerSec += int(float64(SEED_20_MIL) / totalTime.Seconds())
	totalBenchTime += totalTime

	return totalTime
}
func benchRead20MIL() time.Duration {
	truncate()
	db := setupAndSeedDB(SEED_20_MIL, 10000)
	defer deferClose(db)

	start := time.Now().UnixNano()

	counter := 0

	for {
		_, err := db.Read()
		if err == io.EOF {
			break
		}
		counter++
	}

	if counter != SEED_20_MIL {
		log.Fatalln("Bench read 20 mil failed, only counted:", counter)
	}

	totalTime := time.Duration(time.Now().UnixNano() - start)

	//add to the global stats
	totalReadsPerSec += int(float64(SEED_20_MIL) / totalTime.Seconds())
	totalBenchTime += totalTime

	return totalTime
}
func deferClose(db ChanDB.Database) {
	err := db.Close()
	if err != nil {
		log.Fatalln("Failed db close in defer..", err)
	}
}
func setupAndSeedDB(records int, gcSeconds int) ChanDB.Database {
	seedingRequired := true
	for num, file := range cacheMap {
		if num == records {
			seedingRequired = false
			dbFH, err := os.OpenFile(dbFile, os.O_RDWR|os.O_CREATE, 0644)

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

			_, err = io.Copy(dbFH, cacheFH)

			if err != nil {
				log.Fatalln("Failed to copy cache data to db file", err)
			}
		}
	}

	db, err := ChanDB.CreateDatabase(&ChanDB.Settings{
		DBFile:                           dbFile,
		GCFile:                           gcFile,
		WriteOnlyFile:                    writeFile,
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
func truncate() {

	files := []string{dbFile, gcFile, writeFile}
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
func openLog() {
	fh, err := os.OpenFile(logFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		log.Fatalln("Failed to open the log", err)
	}
	logFH = fh
}
func writeLog(line string) {
	_, err := logFH.WriteString(line + "\n")

	if err != nil {
		log.Fatalln("Failed to write log, aborting:", err)
	}
}
