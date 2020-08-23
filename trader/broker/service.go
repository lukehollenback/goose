package broker

import (
  "fmt"
  "github.com/logrusorgru/aurora"
  "github.com/lukehollenback/goose/constants"
  "github.com/lukehollenback/goose/trader/writer"
  "github.com/shopspring/decimal"
  "log"
  "sync"
  "time"
)

const (
  Name = "≪broker-service≫"
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
// Service represents a service instance.
//
type Service struct {
  mu        *sync.Mutex
  chKill    chan bool
  chStopped chan bool
  asset     string
  position  position

  isMockTrading bool
  mockTradeFee  decimal.Decimal
  mockUSD       decimal.Decimal
  mockUSDInit   decimal.Decimal
  mockUSDGain   decimal.Decimal
  mockBTC       decimal.Decimal
}

//
// Instance returns a singleton instance of the service.
//
func Instance() *Service {
  once.Do(func() {
    o = &Service{
      mu:            &sync.Mutex{},
      position:      offline,
      isMockTrading: false,
    }
  })

  return o
}

//
// EnableMockTrading turns on the mock trade executor and funds it with the provided initial amount
// of capital.
//
func (o *Service) EnableMockTrading(initUSDHolding decimal.Decimal, tradeFee decimal.Decimal) {
  o.mu.Lock()
  defer o.mu.Unlock()

  o.mockUSD = initUSDHolding
  o.mockUSDInit = initUSDHolding
  o.mockUSDGain = decimal.Zero
  o.isMockTrading = true
}

//
// SetAsset tells the Broker Service which asset it should be trading. This should normally be the
// same asset that is being monitored by the Monitor Service.
//
func (o *Service) SetAsset(asset string) {
  o.asset = asset
}

//
// DisableMockTrading turns off the mock trade executor.
//
func (o *Service) DisableMockTrading() {
  o.mu.Lock()
  defer o.mu.Unlock()

  o.isMockTrading = false
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
  // Validate that necessary configurations have been provided.
  //
  // TODO ~> This.

  //
  // (Re)initialize our instance variables.
  //
  o.chKill = make(chan bool, 1)
  o.chStopped = make(chan bool, 1)

  //
  // Adjust the tracked position (a.k.a. state) of the service to indicate that it is now running.
  //
  o.position = waiting

  //
  // Return our "started" channel in case the caller wants to block on it and log some debug info.
  //
  chStarted := make(chan bool, 1)
  chStarted <- true

  logger.Printf("Started.")

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
  logger.Printf("Stopping...")

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
  // Send the signal that we have shut down.
  //
  o.chStopped <- true

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
func (o *Service) Signal(signal Signal, price decimal.Decimal, timestamp time.Time) {
  o.mu.Lock()
  defer o.mu.Unlock()

  // TODO ~> Flesh this mechanism out quite a bit. For now, we just immediately pretend to execute
  //  a trade so that we can see how we are doing.

  //
  // Depending on the signal that came in, enter or exit a position.
  //
  gainMsg := ""
  feeMsg := ""

  if signal == UptrendDetected && o.position == waiting {
    //
    // Calculate the transaction fee.
    //
    fee := o.mockUSD.Mul(o.mockTradeFee)

    //
    // Execute the mock transaction and enter the new position.
    //
    o.mockBTC = o.mockUSD.Div(price).Sub(fee)
    o.mockUSD = decimal.Zero
    o.position = holding

    //
    // Build out a message explaining how much was spent on transaction fees during this mock trade.
    //
    feeMsg = fmt.Sprintf("Fees were %s.", aurora.Bold(aurora.Blue(fmt.Sprintf("%s USD", fee))))
  } else if signal == DowntrendDetected && o.position == holding {
    //
    // Calculate the transaction fee.
    //
    fee := o.mockBTC.Mul(o.mockTradeFee)

    //
    // Execute the mock transaction and exit the current position.
    //
    o.mockUSD = o.mockBTC.Mul(price).Sub(fee)
    o.mockBTC = decimal.Zero
    o.position = waiting
    o.mockUSDGain = o.mockUSD.Sub(o.mockUSDInit)

    //
    // Build out a message explaining how much was spent on transaction fees during this mock trade.
    //
    feeMsg = fmt.Sprintf("Fees were %s.", aurora.Bold(aurora.Blue(fmt.Sprintf("%s %s", fee, o.asset))))

    //
    // Since we are now holding USD again, build out a message that explains the current running
    // USD gain/loss. Also, notify the Writer Service so that it can track the data point if it
    // cares.
    //
    if o.mockUSDGain.GreaterThan(decimal.Zero) {
      gainMsg = fmt.Sprintf(
        "Total running gain/loss is %s.",
        aurora.Bold(aurora.Green(fmt.Sprintf("%s USD", o.mockUSDGain))),
      )
    } else {
      gainMsg = fmt.Sprintf(
        "Total running gain/loss is %s.",
        aurora.Bold(aurora.Red(fmt.Sprintf("%s USD", o.mockUSDGain))),
      )
    }

    _ = writer.Instance().Write(timestamp, writer.GrossMockEarnings, o.mockUSDGain)
  }

  //
  // Log details about the current position now that the mock trade has been executed.
  //
  logger.Printf(
    "Mock trade executed! Current holdings are %s and %s. %s %s",
    aurora.Bold(aurora.Yellow(fmt.Sprintf("%s %s", o.mockBTC, o.asset))),
    aurora.Bold(aurora.Green(fmt.Sprintf("%s USD", o.mockUSD))),
    feeMsg,
    gainMsg,
  )
}