package candle

import (
  "errors"
  "github.com/shopspring/decimal"
  "log"
  "sync"
  "time"
)

var (
  o    *Service
  once sync.Once
)

//
// Service represents a candle store service instance.
//
type Service struct {
  mu              *sync.Mutex
  chKill          chan bool
  chStopped       chan bool
  fiveMinStore    *Store
  fifteenMinStore *Store
}

//
// Instance returns a singleton instance of the candle store service.
//
func Instance() *Service {
  once.Do(func() {
    o = &Service{
      mu: &sync.Mutex{},
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
// which indicates that shut down is complete – is returned.
//
func (o *Service) Stop() (<-chan bool, error) {
  o.mu.Lock()
  defer o.mu.Unlock()

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

//
// Init (re)initializes the candle store service's candle stores with the provided initial candles.
// It should be called prior to processing any new trades into the service to seed the candle stores
// with current candle values.
//
func (o *Service) Init(fiveMinCandle *Candle, fifteenMinCandle *Candle) error {
  o.mu.Lock()
  defer o.mu.Unlock()

  var err error

  //
  // Initialize the fifteen-minute candle store.
  //
  fifteenMin := 15 * time.Minute

  if o.fifteenMinStore, err = CreateStore(fifteenMin, fifteenMinCandle); err != nil {
    return err
  }

  //
  // Initialize the five-minute candle store.
  //
  fiveMin := 5 * time.Minute

  if o.fiveMinStore, err = CreateStore(fiveMin, fiveMinCandle); err != nil {
    return err
  }

  return nil
}

func (o *Service) Append(time time.Time, amt decimal.Decimal) error {
  o.mu.Lock()
  defer o.mu.Unlock()

  //
  // Ensure that the necessary candle stores have been initialized.
  //
  if o.fiveMinStore == nil || o.fifteenMinStore == nil {
    return errors.New(
      "cannot append trade before the candle store service's candle stores have been " +
          "initialized",
    )
  }

  //
  // Append the trade to the five-minute candle store.
  //
  if err := o.fiveMinStore.Append(time, amt); err != nil {
    return err
  }

  //
  // Append the trade to the fifteen-minute candle store.
  //
  if err := o.fifteenMinStore.Append(time, amt); err != nil {
    return err
  }

  return nil
}
