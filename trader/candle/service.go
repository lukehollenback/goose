package candle

import (
	"log"
	"sync"
)

var (
	o    *Service
	once sync.Once
)

//
// Service represents a candle store service instance.
//
type Service struct {
	chKill    chan bool
	chStopped chan bool
}

//
// Instance returns a singleton instance of the candle store service.
//
func Instance() *Service {
	once.Do(func() {
		o = &Service{}
	})

	return o
}

//
// Start fires up the service. It is up to the caller to not call this multiple times in a row
// without stopping the service and waiting for full termination in between. A channel that can be
// blocked on for a "true" value – which indicates that start up is complete – is returned.
//
func (o *Service) Start() (<-chan bool, error) {
	//
	// (Re)initialize our instance variables.
	//
	o.chKill = make(chan bool, 1)
	o.chStopped = make(chan bool, 1)

	//
	// Return our "started" channel in case the caller wants to block on it and log some debug info.
	//
	chStarted := make(chan bool, 1)
	chStarted <- true

	log.Printf("The candle store service has started.")

	return chStarted, nil
}

//
// Stop tells the service to shut down. It is up to the caller to not call this multiple times in
// a row without starting the service first. A channel that can be blocked on for a "true" value –
// which indiciates that shut down is complete – is returned.
//
func (o *Service) Stop() (<-chan bool, error) {
	//
	// Log some debug info.
	//
	log.Printf("The candle store service is stopping...")

	//
	// Tell the goroutines that were spun off by the service to shutdown.
	//
	o.chKill <- true

	//
	// Return the "stopped" channel that the caller can block on if they need to know that the
	// service has completely shutdown.
	//
	o.chStopped <- true

	return o.chStopped, nil
}
