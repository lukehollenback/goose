package broker

import (
  "fmt"
  "github.com/logrusorgru/aurora"
  "github.com/shopspring/decimal"
  "log"
  "sync"
)

var (
  o    *Service
  once sync.Once
)

//
// Service represents a service instance.
//
type Service struct {
  mu          *sync.Mutex
  chKill      chan bool
  chStopped   chan bool
  position    position
  mockUSD     decimal.Decimal
  mockUSDInit decimal.Decimal
  mockUSDGain decimal.Decimal
  mockBTC     decimal.Decimal
}

//
// Instance returns a singleton instance of the service.
//
func Instance() *Service {
  once.Do(func() {
    o = &Service{
      mu:          &sync.Mutex{},
      position:    offline,
      mockUSD:     decimal.NewFromInt(100),
      mockUSDInit: decimal.NewFromInt(100),
      mockUSDGain: decimal.Zero,
      mockBTC:     decimal.Zero,
    }
  })

  return o
}

//
// Start fires up the service. It is up to the caller to not call this multiple times in a row
// without stopping the service and waiting for full termination in between. A channel that can be
// blocked on for a "true" value – which indicates that start up is complete – is returned.
//
func (o *Service) Start() (<-chan bool, error) {
  o.mu.Lock()
  defer o.mu.Unlock()

  //
  // (Re)initialize our instance variables.
  //
  o.chKill = make(chan bool, 1)
  o.chStopped = make(chan bool, 1)

  //
  // Fire off a goroutine as the executor for the service.
  //
  go o.service()

  //
  // Adjust the tracked position (a.k.a. state) of the service to indicate that it is now running.
  //
  o.position = waiting

  //
  // Return our "started" channel in case the caller wants to block on it and log some debug info.
  //
  chStarted := make(chan bool, 1)
  chStarted <- true

  log.Printf("The trade broker service has started.")

  return chStarted, nil
}

//
// Stop tells the service to shut down. It is up to the caller to not call this multiple times in
// a row without starting the service first. A channel that can be blocked on for a "true" value –
// which indicates that shut down is complete – is returned.
//
func (o *Service) Stop() (<-chan bool, error) {
  o.mu.Lock()
  defer o.mu.Unlock()

  //
  // Log some debug info.
  //
  log.Printf("The trade broker service is stopping...")

  //
  // Tell the goroutines that were spun off by the service to shutdown.
  //
  o.chKill <- true

  //
  // Adjust the tracked position (a.k.a. state) of the service to indicate that it is no longer
  // running.
  //
  o.position = offline

  //
  // Return the "stopped" channel that the caller can block on if they need to know that the
  // service has completely shutdown.
  //
  return o.chStopped, nil
}

//
// Signal tells the Broker Service that a trend or scenario has been detected by an algorithm so
// it can decide if it wants to enter or exit a position.
//
func (o *Service) Signal(signal Signal, price decimal.Decimal) {
  // TODO ~> Flesh this mechanism out quite a bit. For now, we just immediately pretend to execute
  //  a trade so that we can see how we are doing.

  if signal == UptrendDetected && o.position == waiting {
    o.mockBTC = o.mockUSD.Div(price)
    o.mockUSD = decimal.Zero
    o.position = holding
  } else if signal == DowntrendDetected && o.position == holding {
    o.mockUSD = o.mockBTC.Mul(price)
    o.mockBTC = decimal.Zero
    o.position = waiting
    o.mockUSDGain = o.mockUSD.Sub(o.mockUSDInit)
  }

  log.Printf(
    "Mock trade executed! Current holdings are %s and %s. Total running gain/loss is %s.",
    aurora.Bold(aurora.Yellow(fmt.Sprintf("%s BTC", o.mockBTC))),
    aurora.Bold(aurora.Green(fmt.Sprintf("%s USD", o.mockUSD))),
    aurora.Bold(fmt.Sprintf("%s USD", o.mockUSD)),
  )
}

//
// service executes the top-level logic of the service. It is intended to be spun off into its own
// goroutine when the service is started.
//
func (o *Service) service() {
  //
  // Yield indefinitely.
  //
  // TODO ~> Maintain authentication with the Coinbase Pro API and execute trades.
  //
  <-o.chKill

  //
  // Send the signal that we have shut down.
  //
  o.chStopped <- true
}
