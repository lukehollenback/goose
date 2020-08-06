package broker

//
// Signal is an enum that represents a trend or scenario that an algorithm has detected and would
// like to communicate with the Broker Service so that it can make an informed decision about
// whether or not to change it's position.
//
type Signal int

const (
  None Signal = iota
  UptrendDetected
  DowntrendDetected
)
