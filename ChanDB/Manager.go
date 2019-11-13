package ChanDB

import (
	"errors"
	"sync"
	"time"
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

	//set-up database instances

	m.mainDB = &database{
		storageFile:              m.settings.DBFile,
		syncIntervalMilliseconds: m.settings.SyncSyscallIntervalMilliseconds,
		log:                      m.settings.LogFunction,
	}

	err := m.mainDB.loadDatabase()

	if err != nil {
		return err
	}

	m.writeDB = &database{
		storageFile:              m.settings.WriteOnlyFile,
		syncIntervalMilliseconds: m.settings.SyncSyscallIntervalMilliseconds,
		log:                      m.settings.LogFunction,
	}

	err = m.writeDB.loadDatabase()

	if err != nil {
		return err
	}

	m.gcDB = &database{
		storageFile:              m.settings.GCFile,
		syncIntervalMilliseconds: m.settings.SyncSyscallIntervalMilliseconds,
		log:                      m.settings.LogFunction,
	}

	err = m.gcDB.loadDatabase()

	if err != nil {
		return err
	}

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

func (m *manager) Length() int64 {

	return m.mainDB.length()
}

func (m *manager) Close() error {
	m.readLock.Lock()
	m.writeLock.Lock()
	defer m.readLock.Unlock()
	defer m.writeLock.Unlock()

	for {
		time.Sleep(time.Millisecond * 10)
		if m.mode == normalMode {
			break
		}
	}

	m.gcQuitSignal <- true

	errorMessage := ""
	err := m.mainDB.close()

	if err != nil {
		errorMessage += err.Error() + ";"
	}

	err = m.writeDB.close()

	if err != nil {
		errorMessage += err.Error() + ";"
	}

	err = m.gcDB.close()

	if err != nil {
		errorMessage += err.Error() + ";"
	}

	if len(errorMessage) > 0 {
		return errors.New(errorMessage)
	}

	return nil
}
