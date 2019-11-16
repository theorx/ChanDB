package ChanDB

import (
	"errors"
	"sync"
)

const (
	normalMode int = 1
	gcMode     int = 2
)

type LogFunction func(v ...interface{})

type Settings struct {
	/**
	Files where database data is being stored
	*/
	DBFile        string
	GCFile        string
	WriteOnlyFile string

	/**
	Sync syscall will be called for each database instance
	*/
	SyncSyscallIntervalMilliseconds  int
	GarbageCollectionIntervalSeconds int
	LogFunction                      LogFunction
}

type Database interface {
	Read() (string, error)
	Write(string) error
	Length() int64
	ReadStream() Stream
	Truncate() error
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

	if settings.GarbageCollectionIntervalSeconds < 1 {
		settings.GarbageCollectionIntervalSeconds = 1
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
	m.mode = normalMode
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
