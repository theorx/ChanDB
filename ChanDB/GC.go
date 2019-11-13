package ChanDB

import (
	"io"
	"log"
	"os"
	"time"
)

func (m *manager) garbageCollectRoutine() {
	timeElapsed := time.Duration(0)
	for {
		select {
		case <-m.gcQuitSignal:

			return
		default:
			timeElapsed += time.Millisecond * 250
			time.Sleep(time.Millisecond * 250)

			if timeElapsed < time.Second*time.Duration(m.settings.GarbageCollectionIntervalSeconds) {
				continue
			}
			timeElapsed = 0
		}
		m.garbageCollect()
		m.writeBackDataToMainDB()
	}
}

func (m *manager) writeBackDataToMainDB() {
	for {
		msg, err := m.writeDB.read(false)

		if err == io.EOF {
			break
		}

		err = m.mainDB.write(msg)
		if err != nil {
			log.Println("GC final state error - writeBackDataToMainDB has failed:", err)
			return
		}
	}

	err := m.writeDB.truncate()

	if err != nil {
		log.Println("GC writeDB.truncate has failed:", err)
	}
}

func (m *manager) garbageCollect() {
	m.readLock.Lock()
	defer m.readLock.Unlock()

	m.switchToGCMode()

	err := m.gcDB.truncate()

	if err != nil {
		log.Print("GC failed, gcDB.truncate() error", err)
		return
	}

	err = m.moveRecordsToGCDB()

	if err != nil {
		log.Println("GC failed, moveRecordsToGCDB has failed:", err)
		return
	}

	err = m.moveGCDataToMainDB()

	if err != nil {
		log.Println("GC Failed, moveGCDataToMainDB has failed:", err)
		return
	}

	m.switchToNormalMode()
}

func (m *manager) moveGCDataToMainDB() error {
	err := m.mainDB.close()
	if err != nil {
		return err
	}

	err = m.gcDB.close()
	if err != nil {
		return err
	}

	err = os.Rename(m.settings.GCFile, m.settings.DBFile)

	if err != nil {
		return err
	}

	err = m.mainDB.loadDatabase()

	if err != nil {
		return err
	}

	err = m.gcDB.loadDatabase()
	if err != nil {
		return err
	}

	return nil
}

func (m *manager) moveRecordsToGCDB() error {
	for {
		msg, err := m.mainDB.read(false)

		if err == io.EOF {
			return nil
		}

		if err != nil {
			return err
		}

		err = m.gcDB.write(msg)

		if err != nil {
			return err
		}
	}
}

func (m *manager) switchToNormalMode() {
	m.writeLock.Lock()
	defer m.writeLock.Unlock()

	m.mode = NormalMode
}

func (m *manager) switchToGCMode() {
	m.writeLock.Lock()
	defer m.writeLock.Unlock()

	m.mode = GCMode
}
