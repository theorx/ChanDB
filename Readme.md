# ChanDB

## Author
* Lauri Orgla `theorx@hotmail.com`

## Description
*FIFO (First in, first out) database for storing content from a channel
 (go channel) persistently. Meant to be used as a persistent queue.*
*This database is a GO library, that can be embedded within any go project.
 No searching or indexing features are implemented, when you read a record 
 from the database, then the record gets deleted.*

## API Overview

```go

type Database interface {
    /* Reading will discard the record from the database */
	Read() (string, error) 
    /* Returns the number of active records from the database */
	Length() int64
	/* Writes data to the database */
	Write(string) error 
    /* Opens a channel for streaming records from the database, once record is receive it will no longer be stored in the database*/
	ReadStream() Stream 
    /* Truncates the database contents */
	Truncate() error
    /* Closes database instance and makes sure, that all of the data is persisted to the disk, cleans up */
	Close() error
}


```

# Usage

*__Important:__ In order to make sure that all of the data persists `db.Close()` has to be called after db instance is not needed or program exists.*

### Database settings

* type LogFunction func(v ...interface{})  <- Compatible with log.Println


```go

type Settings struct {
	/**
	Files where database data is being stored
	All of these files are required for the database to operate properly
	*/
	DBFile string
	/**
	Garbage collection file, used for temporarily storing all active records while deleting all
	records that are marked for deletion
	*/
	GCFile string
	/**
	Write-only file that will be used while garbage collection is active. All of the writes are applied to
	write-only file while gc is in progress and after gc has finished, all of the write-only database records
	will be written to the main database
	*/
	WriteOnlyFile string
	/**
	Sync syscall will be called for each database instance, minimum value is 100ms
	*/
	SyncSyscallIntervalMilliseconds int
	/**
	Minimum value for GC interval is 10 seconds
	*/
	GarbageCollectionIntervalSeconds int
	/**
	Log function, compatible with log package (log.Println)
	*/
	LogFunction LogFunction
}
```

### Creating database instance

```go

	db, err := ChanDB.CreateDatabase(&ChanDB.Settings{
		DBFile:                           "db_file.txt",
		GCFile:                           "gc_file.txt",
		WriteOnlyFile:                    "wo_file.txt",
		SyncSyscallIntervalMilliseconds:  1000,
		GarbageCollectionIntervalSeconds: 300,
		LogFunction:                      log.Println,
	})


```

### Basic operations

```go

//Writing to the database

	err := db.Write(payload)

	if err != nil {
		//handle the error
	}


//Reading from the database

	data, err := db.Read()

	if err != nil {
		//handle the error
	}

	log.Println(data)

//Truncating database

	err := db.Truncate()

	if err != nil {
		//handle error
	}

//Get the number of active records

	length := db.Length()
	

//Closing the database

	err := db.Close()

	if err != nil {
		//handle error
	}


```


### Streaming reads from the database
*Streams are go channels that provide data from the database via channel interface*
*Once you receive a record from the channel, it is already deleted from the database*

```go

//writing some records to the database first
	db.Write("test1")
	db.Write("test2")
	db.Write("test3")
	db.Write("test4")
	db.Write("test5")
	db.Write("test6")
	db.Write("test7")
	db.Write("test8")

	stream := db.ReadStream()

	for msg := range stream.Stream() {
		log.Println(msg)
	}
	err := stream.Close()

	if err != nil {
		//handle error
	}

// output:

$ go run main.go 
23:04:07 test1
23:04:07 test2
23:04:07 test3
23:04:07 test4
23:04:07 test5
23:04:07 test6
23:04:07 test7
23:04:07 test8


```



### Benchmarking 

*Go get and go install the library:*
* `go get -u github.com/theorx/ChanDB/cmd/bench-db`
* `go install github.com/theorx/ChanDB/cmd/bench-db`

```bash

$ bench-db -h 
Usage of bench-db:
  -bench-dir string
    	Benchmark directory to store temporary files (default "bench")
  -log string
    	Log file path (default "benchLog.txt")


```

* running bench-db requires approx 8 gigs free ram and 10 gigabytes of
 free disk space to complete the benchmark. The RAM requirement is related
 to some kind of bug where RAM is not being freed up after writing benchmark
 cache files to the disk
 

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