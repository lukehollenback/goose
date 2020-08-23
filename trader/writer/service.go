package writer

import (
  "encoding/csv"
  "flag"
  "fmt"
  "github.com/lukehollenback/goose/constants"
  "github.com/shopspring/decimal"
  "log"
  "os"
  "sync"
  "time"
)

const (
  Name         = "≪writer-service≫"
  TimestampKey = "Timestamp"
)

var (
  o    *Service
  once sync.Once
  logger *log.Logger

  cfgOutputDir *string
)

func init() {
  //
  // Initialize the logger.
  //
  logger = log.New(log.Writer(), fmt.Sprintf(constants.LogPrefixFmt, Name), log.Ldate | log.Ltime | log.Lmsgprefix)

  //
  // Determine the current working directory. If that cannot be done for some reason, we are in a
  // critical failure state.
  //
  workingDir, err := os.Getwd()
  if err != nil {
    logger.Fatalf("Failed to determine the current working directory. (Error: %s)", err)
  }

  //
  // Register and parse configuration flags.
  //
  cfgOutputDir = flag.String(
    "writer-dir",
    workingDir,
    fmt.Sprintf(
      "The directory %s service should output CSV files with performance data to.",
      Name,
    ),
  )
}

//
// Service represents a service instance.
//
type Service struct {
  mu         *sync.Mutex
  chKill     chan bool
  chStopped  chan bool
  outputDir  string
  outputFile *os.File
  writer     *csv.Writer
}

//
// Instance returns a singleton instance of the service.
//
func Instance() *Service {
  once.Do(func() {
    o = &Service{
      mu:        &sync.Mutex{},
      outputDir: *cfgOutputDir,
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
  // Create the output CSV file.
  //
  var err error

  outputFilePath := o.outputDir + "/goose.csv"
  o.outputFile, err = os.Create(outputFilePath)
  if err != nil {
    o.chStopped <- true

    return o.chStopped, err
  }

  logger.Printf("Outputing CSV to %s.", outputFilePath)

  //
  // Create the CSV writer and use it to write out the header row.
  //
  o.writer = csv.NewWriter(o.outputFile)

  err = o.writer.Write([]string{TimestampKey, ClosingPrice.String(), GrossMockEarnings.String()})
  if err != nil {
    o.chStopped <- true

    return o.chStopped, err
  }

  //
  // Fire off a goroutine as the executor for the service.
  //
  go o.service()

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
  // Return the "stopped" channel that the caller can block on if they need to know that the
  // service has completely shutdown.
  //
  return o.chStopped, nil
}

//
// Write outputs the provided data point to the current CSV output file.
//
// NOTE ~> This method logs its own failures, but also returns them in case the caller wants to
//  pivot on them as well.
//
func (o *Service) Write(timestamp time.Time, category Type, value decimal.Decimal) error {
  var err error

  if category == ClosingPrice {
    err = o.writer.Write([]string{timestamp.String(), value.String(), nil})
  } else if category == GrossMockEarnings {
    err = o.writer.Write([]string{timestamp.String(), nil, value.String()})
  }
  
  if err != nil {
    logger.Printf(
      "Failed to write out data point. (Timestamp: %s, Category: %s, Value: %s) (Error: %s)",
      timestamp, category, value, err,
    )
  }

  return err
}

//
// service executes the top-level logic of the service. It is intended to be spun off into its own
// goroutine when the service is started.
//
func (o *Service) service() {
  //
  // Yield indefinitely.
  //
  <-o.chKill

  //
  // Flush the CSV writer's buffer to the output file.
  //
  o.writer.Flush()

  //
  // Close the handle on the output file.
  //
  err := o.outputFile.Close()
  if err != nil {
    logger.Printf("Failed to close handle on output file. (Error: %s)", err)
  }

  //
  // Send the signal that we have shut down.
  //
  o.chStopped <- true
}
