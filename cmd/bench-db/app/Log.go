package Benchmark

import (
	"log"
	"os"
	"strconv"
	"time"
)

func (b *Benchmark) openLog() {
	fh, err := os.OpenFile(b.logFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		log.Fatalln("Failed to open the log", err)
	}
	b.logFileHandle = fh
}

func (b *Benchmark) writeLog(line string) {
	_, err := b.logFileHandle.WriteString(line + "\n")
	if err != nil {
		log.Fatalln("Failed to write log, aborting:", err)
	}
}

func (b *Benchmark) writeStats(line string, duration time.Duration, records int) {
	b.writeLog(line + "\t\t - " + "Op/s: " + strconv.Itoa(int(float64(records)/duration.Seconds())/1000) + " K\t" + duration.String())
}

func (b *Benchmark) writeStatsMixed(line string, result *MixedResults) {
	b.writeLog(
		line + "\t\t - " + "Op/s: " +
			strconv.Itoa(int(float64(result.ReadRecords)/result.ReadTime.Seconds())/1000+
				int(float64(result.WriteRecords)/result.WriteTime.Seconds())/1000) +
			" K\t" + result.TotalTime.String(),
	)
	b.writeLog("\t Reads/s: " + strconv.Itoa(int(float64(result.ReadRecords)/result.ReadTime.Seconds())/1000) + " K")
	b.writeLog("\t Writes/s: " + strconv.Itoa(int(float64(result.WriteRecords)/result.WriteTime.Seconds())/1000) + " K")
}
