package Signal

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
