package main

import (
	"flag"
	Benchmark "github.com/theorx/ChanDB/cmd/bench-db/app"
)

var (
	BenchmarkDir = flag.String("bench-dir", "bench", "Benchmark directory to store temporary files")
	LogFile      = flag.String("log", "benchLog.txt", "Log file path")
)

func main() {
	flag.Parse()

	bench := Benchmark.CreateBenchmark(*BenchmarkDir, *LogFile)
	bench.Run()
}
