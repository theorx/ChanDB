package Signal

/**
This is used for signaling the readStream that there has been new
activity on the database and it's time to start reading again
*/
type Signal struct {
	signal chan bool
}

func CreateSignal() *Signal {
	return &Signal{
		signal: make(chan bool, 1),
	}
}

func (s *Signal) Channel() <-chan bool {
	return s.signal
}

func (s *Signal) Signal() bool {
	select {
	case s.signal <- true:
		return true
	default:
		return false
	}
}
