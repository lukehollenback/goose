package candle

import (
  "errors"
  "fmt"
  "github.com/lukehollenback/goose/constants"
  "github.com/lukehollenback/goose/trader/writer"
  "github.com/shopspring/decimal"
  "log"
  "sync"
  "time"
)

const (
  Name = "≪candle-service≫"
)

var (
  o      *Service
  once   sync.Once
  logger *log.Logger
)

func init() {
  //
  // Initialize the logger.
  //
  logger = log.New(log.Writer(), fmt.Sprintf(constants.LogPrefixFmt, Name), log.Ldate | log.Ltime | log.Lmsgprefix)
}

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
// Start implements the Service interface's described method.
//
func (o *Service) Start() (<-chan bool, error) {
  o.mu.Lock()
  defer o.mu.Unlock()

  //
  // Return our "started" channel in case the caller wants to block on it and log some debug info.
  //
  chStarted := make(chan bool, 1)
  chStarted <- true

  logger.Printf("Started.")

  return chStarted, nil
}

//
// Stop implements the Service interface's described method.
//
func (o *Service) Stop() (<-chan bool, error) {
  o.mu.Lock()
  defer o.mu.Unlock()

  //
  // Log some debug info.
  //
  logger.Printf("Stopping...")

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
// Append adds the provided trade to all of the necessary candle stores. Returns a structure holding
// references to any candles that were closed out by the append.
//
func (o *Service) Append(time time.Time, amt decimal.Decimal) (*Candles, error) {
  o.mu.Lock()
  defer o.mu.Unlock()

  //
  // Ensure that the necessary candle stores have been initialized.
  //
  if o.oneMinStore == nil || o.fiveMinStore == nil || o.fifteenMinStore == nil {
    return nil, errors.New(
      "cannot append trade before the candle store service's candle stores have been " +
          "initialized",
    )
  }

  //
  // Create a new structure to hold references to any candles that were closed out by this append.
  //
  closedCandles := &Candles{}

  //
  // Append the trade to the one-minute candle store. We also report one-minute candle closes to
  // the Writer Service so that it can track the moving price of the asset being traded against
  // any other data points it is tracking.
  //
  createdNewCandle, err := o.oneMinStore.Append(time, amt)
  if err != nil {
    return nil, err
  } else if createdNewCandle {
    closedCandles.OneMin = o.oneMinStore.Previous()

    _ = writer.Instance().Write(closedCandles.OneMin.End(), writer.ClosingPrice, closedCandles.OneMin.CloseAmt())

    logger.Printf("1 Min ↝ %s", closedCandles.OneMin)
  }

  //
  // Append the trade to the five-minute candle store.
  //
  createdNewCandle, err = o.fiveMinStore.Append(time, amt)
  if err != nil {
    return nil, err
  } else if createdNewCandle {
    closedCandles.FiveMin = o.fiveMinStore.Previous()

    logger.Printf("5 Min ↝ %s", closedCandles.FiveMin)
  }

  //
  // Append the trade to the fifteen-minute candle store.
  //
  createdNewCandle, err = o.fifteenMinStore.Append(time, amt)
  if err != nil {
    return nil, err
  } else if createdNewCandle {
    closedCandles.FifteenMin = o.fifteenMinStore.Previous()

    logger.Printf("15 Min ↝ %s", closedCandles.FifteenMin)
  }

  return closedCandles, nil
}
