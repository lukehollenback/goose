package trader

//
// Service generically provides an interface to any isolated service in the software.
//
type Service interface {

  //
  // Start fires up the service. It is up to the caller to not call this multiple times in a row
  // without stopping the service and waiting for full termination in between. A channel that can be
  // blocked on for a "true" value – which indicates that start up is complete – is returned.
  //
  Start() (<-chan bool, error)

  //
  // Stop tells the service to shut down. It is up to the caller to not call this multiple times in
  // a row without starting the service first. A channel that can be blocked on for a "true" value –
  // which indicates that shut down is complete – is returned.
  //
  Stop() (<-chan bool, error)

}
