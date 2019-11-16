package ChanDB

type Stream interface {
	Stream() <-chan string
	Close() error
}

type stream struct {
	out        chan string
	dbManager  *manager
	killSignal chan bool
	isOpen     bool
}

func createStream(manager *manager) *stream {
	instance := &stream{
		dbManager:  manager,
		out:        make(chan string),
		killSignal: make(chan bool),
		isOpen:     true,
	}

	instance.streamRoutine()

	return instance
}

func (s *stream) streamRoutine() {
	for {
		select {
		case <-s.killSignal:
			return
		case data, ok := <-s.dbManager.mainDB.streamReads():
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
	//close the reading routine
	s.killSignal <- true

	//handle reading from the out channel if it is not empty
	select {
	case data, _ := <-s.out:
		if len(data) > 0 {
			return s.dbManager.mainDB.write(data)
		}
	default:
	}

	return nil
}

func (s *stream) Stream() <-chan string {
	return s.out
}
