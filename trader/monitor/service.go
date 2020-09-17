package monitor

import (
  "flag"
  "fmt"
  "github.com/lukehollenback/goose/constants"
  "github.com/lukehollenback/goose/exchange"
  "github.com/lukehollenback/goose/trader/candle"
  "github.com/lukehollenback/goose/trader/writer"
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

  client exchange.Client

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
      o.backtestStart, err = time.Parse("2006-01-02 03:04", *cfgBacktestStart)
      if err != nil {
        logger.Fatalf("Failed to instantiate. Backtest start timestamp could not be parsed. (Error: %s)", err)
      }

      o.backtestEnd, err = time.Parse("2006-01-02 03:04", *cfgBacktestEnd)
      if err != nil {
        logger.Fatalf("Failed to instantiate. Backtest end timestamp could not be parsed. (Error: %s)", err)
      }

      logger.Printf("Enabled backtesting. (Start: %s, End: %s)", o.backtestStart, o.backtestEnd)
    }
  })

  return o
}

//
// SetAsset tells the Monitor Service which asset it should subscribe to and watch.
//
func (o *Service) SetAsset(asset string) {
  o.asset = asset
  o.market = o.client.RetrieveSymbol(asset, "USD")
}

//
// SetClient tells the Monitor Service which client instance it should use to communicate with the
// relevant exchange's REST API (e.g. for loading historical data).
//
func (o *Service) SetClient(client exchange.Client) {
  o.client = client
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
  //
  // Execute the appropriate monitor.
  //
  if o.backtest {
    o.backtestTrades()
  } else {
    o.monitorLiveTrades()
  }

  //
  // Send the signal that we have shut down.
  //
  o.chStopped <- true
}

//
// backtestTrades loads historical handles from the relevant exchange API and converts them directly
// into candles that it can produce.
//
func (o *Service) backtestTrades() {
  //
  // Retrieve, process, and produce historical candles from the relevant exchange's API for the
  // configured backtest period.
  //
  for s, e, c := o.obtainBacktestCursors(nil); c; s, e, c = o.obtainBacktestCursors(s) {
    //
    // Log some debug info.
    //
    logger.Printf("Loading historical candles from %s through %s.", s, e)

    //
    // Load historical candles.
    //
    oneMinResp, err := o.client.RetrieveCandles(o.market, exchange.OneMinute, *s, *e, 1000)
    if err != nil {
      logger.Fatalf("Failed to load historical one minute candles. (Error: %s)", err)
    }

    fiveMinResp, err := o.client.RetrieveCandles(o.market, exchange.FiveMinute, *s, *e, 1000)
    if err != nil {
      logger.Fatalf("Failed to load historical five minute candles. (Error: %s)", err)
    }

    fifteenMinResp, err := o.client.RetrieveCandles(o.market, exchange.FifteenMinute, *s, *e, 1000)
    if err != nil {
      logger.Fatalf("Failed to load historical fifteen minute candles. (Error: %s)", err)
    }

    //
    // Log some debug info.
    //
    logger.Printf(
      "Loaded %d one minute, %d five minute, and %d fifteen minute candles.",
      len(oneMinResp.Candles()), len(fiveMinResp.Candles()), len(fifteenMinResp.Candles()),
    )

    //
    // Process and produce historical candles.
    //
    fiveMinIndex := 0
    fifteenMinIndex := 0

    for _, v := range oneMinResp.Candles() {
      //
      // Write the one minute candle's close price.
      //
      _ = writer.Instance().Write(*(v.EndTime()), writer.ClosingPrice, *(v.Close()))

      //
      // Create all of the necessary candles and then "produce" them to any handlers that are
      // registered and waiting for them.
      //
      // NOTE ~> We will always have a one minute candle. Occasionally we'll have a five minute
      //  candle or a fifteen minute candle.
      //
      candles := &candle.Candles{
        OneMin: candle.CreateFullCandle(*(v.StartTime()), candle.OneMin, *(v.Open()), *(v.Close()), *(v.High()), *(v.Low()), *(v.Volume()), decimal.NewFromInt(int64(*(v.Count())))),
      }

      if fiveMinResp.Candles()[fiveMinIndex].EndTime().Equal(*v.EndTime()) {
        v := fiveMinResp.Candles()[fiveMinIndex]
        candles.FiveMin = candle.CreateFullCandle(*(v.StartTime()), candle.OneMin, *(v.Open()), *(v.Close()), *(v.High()), *(v.Low()), *(v.Volume()), decimal.NewFromInt(int64(*(v.Count()))))

        fiveMinIndex++
      }

      if fifteenMinResp.Candles()[fifteenMinIndex].EndTime().Equal(*v.EndTime()) {
        v := fifteenMinResp.Candles()[fifteenMinIndex]
        candles.FifteenMin = candle.CreateFullCandle(*(v.StartTime()), candle.OneMin, *(v.Open()), *(v.Close()), *(v.High()), *(v.Low()), *(v.Volume()), decimal.NewFromInt(int64(*(v.Count()))))

        fifteenMinIndex++
      }

      o.processClosedCandles(candles)
    }
  }

  //
  // Log some debug info.
  //
  logger.Printf("Backtesting has completed.")
}

//
// obtainBacktestCursors initializes or slides the start and end timestamp cursors being used to
// retrieve historical candles. Slides the window by twelve hours each call.
//
func (o *Service) obtainBacktestCursors(prevStart *time.Time) (*time.Time, *time.Time, bool) {
  var start time.Time
  var end time.Time
  var cont = true

  //
  // Prime or update the head cursor.
  //
  if prevStart == nil {
    start = o.backtestStart
  } else {
    start = prevStart.Add(constants.TwelveHours)
  }

  //
  // Update the tail cursor and see if we are at the end of our backtest period.
  //
  end = start.Add(constants.TwelveHours).Add(-1 * time.Nanosecond)

  if end.After(o.backtestEnd) {
    end = o.backtestEnd
    cont = false
  }

  //
  // Return the new head cursor, the new tail cursor, and a sentinel indicating whether or not we
  // have reached the end of our backtest period.
  //
  return &start, &end, cont
}

//
// monitorLiveTrades actually monitors trades as received from the relevant exchange's websocket
// feed in realtime to produce candles.
//
func (o *Service) monitorLiveTrades() {
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
      /*go */handler(candles.OneMin)
    }
  }

  if candles.FiveMin != nil {
    for _, handler := range o.onFiveMinCandleCloseHandlers {
      /*go */handler(candles.FiveMin)
    }
  }

  if candles.FifteenMin != nil {
    for _, handler := range o.onFifteenMinCandleCloseHandlers {
      /*go */handler(candles.FifteenMin)
    }
  }

  for _, handler := range o.onCandleCloseHandlers {
    /*go */handler()
  }
}
