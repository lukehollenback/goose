package monitor

import (
  "flag"
  "fmt"
  "github.com/lukehollenback/goose/constants"
  "github.com/lukehollenback/goose/trader/candle"
  "github.com/shopspring/decimal"
  "log"
  "sync"
  "time"

  ws "github.com/gorilla/websocket"
  coinbasepro "github.com/preichenberger/go-coinbasepro/v2"
)

const (
  Name = "≪monitor-service≫"
)

var (
  o      *Service
  once   sync.Once
  logger *log.Logger

  cfgBacktest      *bool
  cfgBacktestStart *string
  cfgBacktestEnd   *string
)

func init() {
  //
  // Initialize the logger.
  //
  logger = log.New(log.Writer(), fmt.Sprintf(constants.LogPrefixFmt, Name), log.Ldate|log.Ltime|log.Lmsgprefix)

  //
  // Register and parse configuration flags.
  //
  cfgBacktest = flag.Bool(
    "backtest",
    false,
    "Whether or not to run as a backtest. Does NOT enable mocking NOR disable real trading by default. Be "+
        "careful.",
  )

  cfgBacktestStart = flag.String(
    "backtest-start",
    "2006-01-02 03:04",
    "The desired backtest start timestamp.",
  )

  cfgBacktestEnd = flag.String(
    "backtest-end",
    "2006-01-02 03:04",
    "The desired backtest end timestamp.",
  )
}

//
// Service represents a match monitor service instance.
//
type Service struct {
  mu        *sync.Mutex
  chKill    chan bool
  chStopped chan bool

  backtest      bool
  backtestStart time.Time
  backtestEnd   time.Time

  state state
  conn  *ws.Conn

  asset  string
  market string

  onOneMinCandleCloseHandlers     []func(*candle.Candle)
  onFiveMinCandleCloseHandlers    []func(*candle.Candle)
  onFifteenMinCandleCloseHandlers []func(*candle.Candle)
  onCandleCloseHandlers           []func()
}

//
// Instance returns a singleton instance of the match monitor service.
//
func Instance() *Service {
  once.Do(func() {
    var err error

    //
    // Instantiate the structure.
    //
    o = &Service{
      mu: &sync.Mutex{},

      backtest: *cfgBacktest,

      state: disconnected,

      onOneMinCandleCloseHandlers:     make([]func(*candle.Candle), 0),
      onFiveMinCandleCloseHandlers:    make([]func(*candle.Candle), 0),
      onFifteenMinCandleCloseHandlers: make([]func(*candle.Candle), 0),
      onCandleCloseHandlers:           make([]func(), 0),
    }

    //
    // Parse the backtest start and end timestamps if backtesting has been enabled.
    //
    if o.backtest {
      o.backtestStart, err = time.Parse("", *cfgBacktestStart)
      if err != nil {
        log.Fatalf("Failed to instantiate. Backtest start timestamp could not be parsed. (Error: %s)", err)
      }

      o.backtestEnd, err = time.Parse("", *cfgBacktestEnd)
      if err != nil {
        log.Fatalf("Failed to instantiate. Backtest end timestamp could not be parsed. (Error: %s)", err)
      }

      log.Printf("Enabled backtesting. (Start: %s, End: %s)", o.backtestStart, o.backtestEnd)
    }
  })

  return o
}

//
// SetAsset tells the Monitor Service which asset it should subscribe to and watch.
//
func (o *Service) SetAsset(asset string) {
  o.asset = asset
  o.market = fmt.Sprintf("%s-USD", asset)
}

//
// RegisterOneMinCandleCloseHandler registers a signal handler to be executed whenever a one minute
// candle closes out.
//
func (o *Service) RegisterOneMinCandleCloseHandler(handler func(*candle.Candle)) {
  o.mu.Lock()
  defer o.mu.Unlock()

  o.onOneMinCandleCloseHandlers = append(o.onOneMinCandleCloseHandlers, handler)
}

//
// RegisterFiveMinCandleCloseHandler registers a signal handler to be executed whenever a five
// minute candle closes out.
//
func (o *Service) RegisterFiveMinCandleCloseHandler(handler func(*candle.Candle)) {
  o.mu.Lock()
  defer o.mu.Unlock()

  o.onFiveMinCandleCloseHandlers = append(o.onFiveMinCandleCloseHandlers, handler)
}

//
// RegisterFifteenMinCandleCloseHandler registers a signal handler to be executed whenever a
// fifteen minute candle closes out.
//
func (o *Service) RegisterFifteenMinCandleCloseHandler(handler func(*candle.Candle)) {
  o.mu.Lock()
  defer o.mu.Unlock()

  o.onFifteenMinCandleCloseHandlers = append(o.onFifteenMinCandleCloseHandlers, handler)
}

//
// RegisterOnClose registers a signal handler to be executed whenever any candles close out and
// other signal handlers have been fired off.
//
func (o *Service) RegisterCandleCloseHandler(handler func()) {
  o.mu.Lock()
  defer o.mu.Unlock()

  o.onCandleCloseHandlers = append(o.onCandleCloseHandlers, handler)
}

//
// Start implements the Service interface's described method.
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
// service connects to the Coinbase Pro websocket feed and monitors it for trade events so that it
// can determine when to buy or sell currency.
//
func (o *Service) service() {
  var err error

  //
  // Connect to the Coinbase Pro websocket feed so that we can monitor network events that occur.
  //
  var wsDialer ws.Dialer

  o.state = connecting

  o.conn, _, err = wsDialer.Dial("wss://ws-feed.pro.coinbase.com", nil)
  if err != nil {
    log.Fatalf("Could not connect to the Coinbase Pro websocket feed. (Error: %s)", err.Error())
  }

  o.state = connected

  //
  // Subscribe to heartbeat messages and trade messages over the Coinbase Pro websocket feed.
  //
  subscribe := coinbasepro.Message{
    Type: "subscribe",
    Channels: []coinbasepro.MessageChannel{
      coinbasepro.MessageChannel{
        Name: "heartbeat",
        ProductIds: []string{
          o.market,
        },
      },
      coinbasepro.MessageChannel{
        Name: "matches",
        ProductIds: []string{
          o.market,
        },
      },
    },
  }

  if err := o.conn.WriteJSON(subscribe); err != nil {
    log.Fatalf(
      "Could not subscribe to specific messages from the Coinbase Pro websocket feed. (Error: %s)",
      err.Error(),
    )
  }

  //
  // Begin monitoring and processing messages from the Coinbase Pro websocket feed.
  //
  cont := true

  for cont {
    chMsg := make(chan *coinbasepro.Message, 1)
    chErr := make(chan error, 1)

    go o.readNextMessage(chMsg, chErr)

    select {
    case <-o.chKill:
      cont = false

      break

    case msg := <-chMsg:
      o.handleMessage(msg)

      break

    case err := <-chErr:
      log.Fatalf(
        "Could not read the next JSON message from the Coinbase Pro websocket feed. (Error: %s)",
        err.Error(),
      )
    }
  }

  //
  // Close our websocket connection.
  //
  err = o.conn.Close()
  if err != nil {
    log.Fatalf("Failed to close websocket connection to Coinbase Pro. (Error: %s)", err)
  }

  o.state = disconnected

  //
  // Send the signal that we have shut down.
  //
  o.chStopped <- true
}

func (o *Service) readNextMessage(chMsg chan<- *coinbasepro.Message, chErr chan<- error) {
  msg := &coinbasepro.Message{}

  if err := o.conn.ReadJSON(msg); err != nil {
    chErr <- err
  }

  chMsg <- msg
}

func (o *Service) handleMessage(msg *coinbasepro.Message) {
  if o.state == connected {
    if msg.Type == "subscriptions" {
      //
      // Move the trade monitor service into a "subscribed" state – indicating that it has
      // successfully received acknowledgement from the Coinbase Pro websocket API that it has
      // subscribed to the necessary message channnels.
      //
      o.state = subscribed

      logger.Printf("Successfully subscribed to relevant Coinbase Pro websocket channels (Market: %s).", o.market)
    }
  } else if o.state == subscribed {
    if msg.Type == "last_match" {
      //
      // Extract the trade time and price from the message.
      //
      time := msg.Time.Time()
      amt, err := decimal.NewFromString(msg.Price)
      if err != nil {
        log.Fatalf("Failed to parse price from message. (Message: %+v) (Error: %s)", msg, err)
      }

      //
      // Initialize the Candle Store Service with the last trade as stated by the message.
      //
      oneMinCandle := candle.CreateCandle(time, candle.OneMin, amt)
      fiveMinCandle := candle.CreateCandle(time, candle.FiveMin, amt)
      fifteenMinCandle := candle.CreateCandle(time, candle.FifteenMin, amt)

      if err := candle.Instance().Init(oneMinCandle, fiveMinCandle, fifteenMinCandle); err != nil {
        log.Fatalf("Failed to initialize the Candle Store Service. (Error: %s)", err)
      }

      //
      // Move the Trade Monitor Service into a "ready" state – indicating that it is now fully ready
      // to begin monitoring and processing trades received from the Coinbase Pro websocket API.
      //
      o.state = ready
    }
  } else if o.state == ready {
    if msg.Type == "match" {
      //
      // Extract the trade time and price from the message.
      //
      time := msg.Time.Time()
      amt, err := decimal.NewFromString(msg.Price)
      if err != nil {
        log.Fatalf("Failed to parse price from message. (Message: %+v) (Error: %s)", msg, err)
      }

      //
      // Provide the trade to the candle store service.
      //
      closedCandles, err := candle.Instance().Append(time, amt)
      if err != nil {
        log.Fatalf("Failed to provide the trade to the Candle Store Service. (Error: %s)", err)
      }

      //
      // Process any candles that were closed out.
      //
      go o.processClosedCandles(closedCandles)
    }
  }
}

//
// processClosedCandles fires off any necessary signal handlers given the closed out candles
// provided.
//
func (o *Service) processClosedCandles(candles *candle.Candles) {
  o.mu.Lock()
  defer o.mu.Unlock()

  // NOTE ~> We must lock because we will be iterating slices that are members of the instance.

  //
  // Make sure candles were actually closed out.
  //
  if candles.OneMin == nil && candles.FiveMin == nil && candles.FifteenMin == nil {
    return
  }

  //
  // Fire off necessary signal handlers.
  //
  if candles.OneMin != nil {
    for _, handler := range o.onOneMinCandleCloseHandlers {
      go handler(candles.OneMin)
    }
  }

  if candles.FiveMin != nil {
    for _, handler := range o.onFiveMinCandleCloseHandlers {
      go handler(candles.FiveMin)
    }
  }

  if candles.FifteenMin != nil {
    for _, handler := range o.onFifteenMinCandleCloseHandlers {
      go handler(candles.FifteenMin)
    }
  }

  for _, handler := range o.onCandleCloseHandlers {
    go handler()
  }
}
