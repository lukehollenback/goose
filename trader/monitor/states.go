package monitor

type state int

const (
	disconnected state = iota
	connecting
	connected
	subscribed
)
