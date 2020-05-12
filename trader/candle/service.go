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
  oneMinStore     *Store
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
  // Return the "stopped" channel that the caller can block on if they need to know that the
  // service has completely shutdown.
  //
  chStopped := make(chan bool, 1)
  chStopped <- true

  return chStopped, nil
}

//
// Init (re)initializes the candle store service's candle stores with the provided initial candles.
// It should be called prior to processing any new trades into the service to seed the candle stores
// with current candle values.
//
func (o *Service) Init(oneMinCandle *Candle, fiveMinCandle *Candle, fifteenMinCandle *Candle) error {
  o.mu.Lock()
  defer o.mu.Unlock()

  var err error

  //
  // Initialize the fifteen-minute candle store.
  //
  if o.oneMinStore, err = CreateStore(OneMin, oneMinCandle); err != nil {
    return err
  }

  //
  // Initialize the fifteen-minute candle store.
  //
  if o.fifteenMinStore, err = CreateStore(FifteenMin, fifteenMinCandle); err != nil {
    return err
  }

  //
  // Initialize the five-minute candle store.
  //
  if o.fiveMinStore, err = CreateStore(FiveMin, fiveMinCandle); err != nil {
    return err
  }

  return nil
}

//
// Append adds the provided trade to all of the necessary candle stores.
//
func (o *Service) Append(time time.Time, amt decimal.Decimal) error {
  o.mu.Lock()
  defer o.mu.Unlock()

  //
  // Ensure that the necessary candle stores have been initialized.
  //
  if o.oneMinStore == nil || o.fiveMinStore == nil || o.fifteenMinStore == nil {
    return errors.New(
      "cannot append trade before the candle store service's candle stores have been " +
          "initialized",
    )
  }

  //
  // Append the trade to the fifteen-minute candle store.
  //
  createdNewCandle, err := o.oneMinStore.Append(time, amt)
  if err != nil {
    return err
  } else if createdNewCandle {
    prevCandle := o.oneMinStore.Previous()

    log.Printf("1 Min ↝ %s", prevCandle)
  }

  //
  // Append the trade to the five-minute candle store.
  //
  createdNewCandle, err = o.fiveMinStore.Append(time, amt)
  if err != nil {
    return err
  } else if createdNewCandle {
    prevCandle := o.fiveMinStore.Previous()

    log.Printf("5 Min ↝ %s", prevCandle)
  }

  //
  // Append the trade to the fifteen-minute candle store.
  //
  createdNewCandle, err = o.fifteenMinStore.Append(time, amt)
  if err != nil {
    return err
  } else if createdNewCandle {
    prevCandle := o.fifteenMinStore.Previous()

    log.Printf("15 Min ↝ %s", prevCandle)
  }

  return nil
}
