# ChanDB

## Author

## Description

## API Overview

```go

type Database interface {
	Read() (string, error)
	Write(string) error
	Length() int64
	ReadStream() Stream
	Truncate() error
	Close() error
}


```

# Usage



### Creating database instance

### Basic operations

### Streaming reads from the database

### Benchmarking 

* Go get and go install the library `go get -u github.com/theorx/ChanDB/cmd/bench-db`





## Benchmarks

```bash


### Stream benchmarks ----> 
Stream #1 5M 10R 50M seed		 - Op/s: 743 K	6.727770838s
Stream #2 20M 100R 50M seed		 - Op/s: 810 K	24.669382641s
Stream #3 20M 10R 50M seed		 - Op/s: 821 K	24.342919792s
Stream #4 5M 100R 20M seed		 - Op/s: 775 K	6.446157793s
### Stream score  3151 K
### Regular benchmarks ----> 
#1 20M writes no seed		 - Op/s: 1139 K	17.547591677s
#2 20M reads no seed		 - Op/s: 1267 K	15.785111447s
#3 20M writes 100M seed		 - Op/s: 1098 K	18.212119309s
#4 20M reads 100M seed		 - Op/s: 1180 K	16.937341788s
#5 5M w50-r50 50M seed		 - Op/s: 1254 K	4.03087532s
	 Reads/s: 620 K
	 Writes/s: 634 K
#6 20M w50-r50 50M seed		 - Op/s: 1172 K	17.073796581s
	 Reads/s: 585 K
	 Writes/s: 587 K
#7 20M w25-r75 50M seed		 - Op/s: 1519 K	17.016391482s
	 Reads/s: 881 K
	 Writes/s: 638 K
#8 20M w75-r25 50M seed		 - Op/s: 1477 K	16.917839574s
	 Reads/s: 591 K
	 Writes/s: 886 K
#9 20M w90-r10 50M seed		 - Op/s: 1562 K	17.814496179s
	 Reads/s: 552 K
	 Writes/s: 1010 K
#10 20M w10-r90 50M seed		 - Op/s: 1625 K	15.859597618s
	 Reads/s: 1134 K
	 Writes/s: 491 K
#11 20M w50-r50 no seed		 - Op/s: 1281 K	15.681679353s
	 Reads/s: 637 K
	 Writes/s: 644 K
#12 20M w75-r25 no seed		 - Op/s: 1496 K	16.669375876s
	 Reads/s: 597 K
	 Writes/s: 899 K
#13 20M w90-r10 no seed		 - Op/s: 1674 K	17.223696112s
	 Reads/s: 629 K
	 Writes/s: 1045 K


Benchmark results
Read score:8678 K
Write score:9077 K
Total score:17756 K
Stream score:3151 K

Benchmark time:3m26.769912316s


```