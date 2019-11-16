package ChanDB

import (
	"time"
)

type Stream interface {
	Stream() <-chan string
	Close() error
}

type stream struct {
	out        chan string
	dbManager  *manager
	killSignal chan bool
	done       chan bool
	isOpen     bool
}

func createStream(manager *manager) *stream {
	instance := &stream{
		dbManager:  manager,
		out:        make(chan string),
		killSignal: make(chan bool, 1),
		done:       make(chan bool, 1),
		isOpen:     true,
	}

	go instance.streamRoutine()

	return instance
}

func (s *stream) streamRoutine() {
	readChan := s.dbManager.mainDB.streamReads()
	for {
		select {
		case <-s.killSignal:
			s.done <- true
			return
		case data, ok := <-readChan:
			if ok == false {
				return
			}
			s.out <- data
		}
	}
}

func (s *stream) Close() error {
	//already closed
	if s.isOpen == false {
		return nil
	}

	s.isOpen = false
	//send close signal to the reading routine
	s.killSignal <- true

	//handle reading from the out channel if it is not empty
	for {
		select {
		case data, _ := <-s.out:
			if len(data) > 0 {
				err := s.dbManager.mainDB.write(data)
				if err != nil {
					s.dbManager.log("Failed writing back from the stream", data, err)
				}
			}
		case <-s.done:
			s.dbManager.log("A reading stream has been closed!")

			close(s.out) //close the channel after the stream has finished
			return nil
		default:
			time.Sleep(time.Microsecond)
		}
	}
}

func (s *stream) Stream() <-chan string {
	return s.out
}
