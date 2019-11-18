package ChanDB

import (
	"errors"
	"sync"
)

const (
	initialMode int = 0
	normalMode  int = 1
	gcMode      int = 2
)

type LogFunction func(v ...interface{})

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

type manager struct {
	settings     *Settings
	readLock     *sync.Mutex
	writeLock    *sync.Mutex
	mainDB       *database
	gcDB         *database
	writeDB      *database
	mode         int
	gcQuitSignal chan bool
	log          LogFunction
	streams      []Stream
}

func CreateDatabase(settings *Settings) (*manager, error) {

	if len(settings.DBFile) == 0 {
		return nil, errors.New("no DBFile given in Settings")
	}

	if len(settings.GCFile) == 0 {
		return nil, errors.New("no GCFile given in Settings")
	}

	if len(settings.WriteOnlyFile) == 0 {
		return nil, errors.New("no WriteOnlyFile given in Settings")
	}

	if settings.GarbageCollectionIntervalSeconds < 10 {
		settings.GarbageCollectionIntervalSeconds = 10
	}

	if settings.SyncSyscallIntervalMilliseconds < 100 {
		settings.SyncSyscallIntervalMilliseconds = 100
	}

	if settings.LogFunction == nil {
		//set the default logging function
		settings.LogFunction = func(v ...interface{}) {
		}
	}

	mgr := &manager{
		settings: settings,
		log:      settings.LogFunction,
	}

	return mgr, mgr.init()
}

func (m *manager) init() error {
	//initialize values
	m.writeLock = &sync.Mutex{}
	m.readLock = &sync.Mutex{}
	m.gcQuitSignal = make(chan bool, 0)

	//todo: optimize the code repetitions for creating the databases
	//set-up database instances

	instance, err := createDatabase(m.settings.DBFile, m.settings.SyncSyscallIntervalMilliseconds, m.settings.LogFunction)
	if err != nil {
		return err
	}

	m.mainDB = instance

	instance, err = createDatabase(m.settings.WriteOnlyFile, m.settings.SyncSyscallIntervalMilliseconds, m.settings.LogFunction)
	if err != nil {
		return err
	}

	m.writeDB = instance

	instance, err = createDatabase(m.settings.GCFile, m.settings.SyncSyscallIntervalMilliseconds, m.settings.LogFunction)
	if err != nil {
		return err
	}

	m.gcDB = instance

	go m.garbageCollectRoutine()

	//database successfully running
	m.mode = normalMode

	return nil
}

func (m *manager) Write(payload string) (err error) {
	m.writeLock.Lock()

	if m.mode == gcMode {
		err = m.writeDB.write(payload)
	} else {
		err = m.mainDB.write(payload)
	}
	m.writeLock.Unlock()

	return err
}

func (m *manager) Read() (string, error) {
	m.readLock.Lock()
	result, err := m.mainDB.read(true)
	m.readLock.Unlock()

	return result, err
}

func (m *manager) Truncate() error {
	m.readLock.Lock()
	m.writeLock.Lock()
	defer m.readLock.Unlock()
	defer m.writeLock.Unlock()

	return joinErrors(m.mainDB.truncate(), m.gcDB.truncate(), m.writeDB.truncate())
}

func (m *manager) Length() int64 {

	return m.mainDB.length()
}

func (m *manager) Close() error {
	if m.mode == initialMode {
		return errors.New("database is not running, CreateDatabase must have failed")
	}

	//acquire locks
	m.readLock.Lock()
	m.writeLock.Lock()
	defer m.readLock.Unlock()
	defer m.writeLock.Unlock()

	//first close all of the reading streams before acquiring locks
	for _, stream := range m.streams {
		err := stream.Close()
		if err != nil {
			m.log("Failed closing stream", err)
		}
	}

	m.gcQuitSignal <- true

	return joinErrors(m.mainDB.close(), m.writeDB.close(), m.gcDB.close())
}

func (m *manager) ReadStream() Stream {

	stream := createStream(m)
	m.streams = append(m.streams, stream) //store the stream in the array

	return stream
}

func joinErrors(v ...error) error {
	errorState := false
	errorString := ""
	for _, err := range v {
		if err != nil {
			errorState = true
			errorString += err.Error() + "; "
		}
	}

	if errorState {
		return errors.New(errorString)
	}
	return nil
}
