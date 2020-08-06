package broker

//
// position is an enum that represents the current state of the Broker Service. It is used to
// indicate what the Broker Service is currently doing.
//
type position int

const (
  offline position = iota // The Broker Service is not currently running.
  buying                  // A position is being entered.
  selling                 // A position is being exited.
  holding                 // The current position is being held in hopes of a gain.
  waiting                 // No position is currently held, nor is one being acquired.
)
