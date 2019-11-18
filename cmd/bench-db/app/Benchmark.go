package Benchmark

import (
	"log"
	"os"
	"strconv"
	"time"
)

type Benchmark struct {
	cacheMap          map[int]string
	logFileHandle     *os.File
	logFile           string
	benchDir          string
	totalWritesPerSec int
	totalReadsPerSec  int
	totalStreamScore  int
	totalBenchTime    time.Duration
	dbFile            string
	gcFile            string
	writeFile         string
}

func CreateBenchmark(benchDirectory string, logFile string) *Benchmark {

	bm := &Benchmark{
		benchDir: benchDirectory,
		logFile:  logFile,
		cacheMap: make(map[int]string),
	}

	bm.dbFile = benchDirectory + "/db_file.txt"
	bm.gcFile = benchDirectory + "/gc_file.txt"
	bm.writeFile = benchDirectory + "/write_file.txt"

	err := os.MkdirAll(benchDirectory+"/cache/", 0774)

	if err != nil {
		log.Fatalln("Failed to create benchmark and cache directories:", err)
	}

	return bm
}

/**
The main benchmark that is being executed with `go run cmd/bench-db/bench-db.go`
*/
func (b *Benchmark) Run() {
	b.openLog()
	b.writeLog(blue("### Started benchmark - ") + green(time.Now().String()) + blue(" ###"))
	b.generateSeedCaches()

	b.writeLog(blue("### Stream benchmarks ----> "))
	b.writeStats(yellow("Stream #1 5M 10R 50M seed"), b.benchStream(bench5MIL, 10, seed50MIL), bench5MIL)
	b.writeStats(yellow("Stream #2 20M 100R 50M seed"), b.benchStream(bench20MIL, 100, seed50MIL), bench20MIL)
	b.writeStats(yellow("Stream #3 20M 10R 50M seed"), b.benchStream(bench20MIL, 10, seed50MIL), bench20MIL)
	b.writeStats(yellow("Stream #4 5M 100R 20M seed"), b.benchStream(bench5MIL, 100, seed20MIL), bench5MIL)
	b.writeLog(blue("### Stream score ") + " " + green(strconv.Itoa(b.totalStreamScore/1000)+" K"))

	b.writeLog(blue("### Regular benchmarks ----> "))
	b.writeStats(yellow("#1 20M writes no seed"), b.benchWrite(bench20MIL, seedNone), bench20MIL)
	b.writeStats(yellow("#2 20M reads no seed"), b.benchRead(bench20MIL, seedNone), bench20MIL)
	b.writeStats(yellow("#3 20M writes 100M seed"), b.benchWrite(bench20MIL, seed100MIL), bench20MIL)
	b.writeStats(yellow("#4 20M reads 100M seed"), b.benchRead(bench20MIL, seed100MIL), bench20MIL)
	b.writeStatsMixed(yellow("#5 5M w50-r50 50M seed"), b.benchMixed(bench5MIL, 50, 50, seed50MIL))
	b.writeStatsMixed(yellow("#6 20M w50-r50 50M seed"), b.benchMixed(bench20MIL, 50, 50, seed50MIL))
	b.writeStatsMixed(yellow("#7 20M w25-r75 50M seed"), b.benchMixed(bench20MIL, 25, 75, seed50MIL))
	b.writeStatsMixed(yellow("#8 20M w75-r25 50M seed"), b.benchMixed(bench20MIL, 75, 25, seed50MIL))
	b.writeStatsMixed(yellow("#9 20M w90-r10 50M seed"), b.benchMixed(bench20MIL, 90, 10, seed50MIL))
	b.writeStatsMixed(yellow("#10 20M w10-r90 50M seed"), b.benchMixed(bench20MIL, 10, 90, seed50MIL))
	b.writeStatsMixed(yellow("#11 20M w50-r50 no seed"), b.benchMixed(bench20MIL, 50, 50, seedNone))
	b.writeStatsMixed(yellow("#12 20M w75-r25 no seed"), b.benchMixed(bench20MIL, 75, 25, seedNone))
	b.writeStatsMixed(yellow("#13 20M w90-r10 no seed"), b.benchMixed(bench20MIL, 90, 10, seedNone))

	b.writeLog(green("\n\nBenchmark results"))
	b.writeLog("Read score:" + green(strconv.Itoa(b.totalReadsPerSec/1000)+" K"))
	b.writeLog("Write score:" + red(strconv.Itoa(b.totalWritesPerSec/1000)+" K"))
	b.writeLog("Total score:" + yellow(strconv.Itoa((b.totalWritesPerSec+b.totalReadsPerSec)/1000)+" K"))
	b.writeLog("Stream score:" + green(strconv.Itoa(b.totalStreamScore/1000)+" K"))
	b.writeLog("\nBenchmark time:" + b.totalBenchTime.String())
	b.writeLog(blue("### Finished benchmark - ") + green(time.Now().String()) + blue(" ###\n\n"))
}
