package movingaverages

import (
  "flag"
  "fmt"
  "github.com/logrusorgru/aurora"
  "github.com/lukehollenback/goose/structs/evictingqueue"
  "github.com/lukehollenback/goose/trader/broker"
  "github.com/lukehollenback/goose/trader/candle"
  "github.com/lukehollenback/goose/trader/monitor"
  "github.com/shopspring/decimal"
  "log"
  "sync"
)

const (
  LogPrefix = "≪moving-averages≫"
)

var (
  o           *Algo
  once        sync.Once
  cfgPeriod   *int
  cfgLongLen  *int
  cfgShortLen *int
  cfgExp      *bool

  invalidAvg = decimal.NewFromInt(-1)
  one        = decimal.NewFromInt(1)
  two        = decimal.NewFromInt(2)
)

func init() {
  //
  // Register and parse configuration flags.
  //
  cfgPeriod = flag.Int(
    "ma-period",
    5,
    fmt.Sprintf(
      "The period length (in minutes) that the %s algorithm should watch. Valid values are 1, 5, and 15.",
      LogPrefix,
    ),
  )

  cfgLongLen = flag.Int(
    "ma-long-length",
    15,
    fmt.Sprintf(
      "The length (in periods) of the long moving average for use by the %s algorithm.",
      LogPrefix,
    ),
  )

  cfgShortLen = flag.Int(
    "ma-short-length",
    5,
    fmt.Sprintf(
      "The length (in periods) of the short moving average for use by the %s algorithm.",
      LogPrefix,
    ),
  )

  cfgExp = flag.Bool(
    "ma-exp",
    false,
    fmt.Sprintf(
      "Enables exponantial/weighted moving averages for the %s algorithm.",
      LogPrefix,
    ),
  )
}

type Algo struct {
  candles *evictingqueue.EvictingQueue // Holds references to the most recent one-minute candles that have been provided to the algorithm.

  shortLen decimal.Decimal // Length of the short-duration moving average.
  longLen  decimal.Decimal // Length of the long-duration moving average.
  lastSignal broker.Signal // The last signal that was fired by the algorithm.

  smaShort     decimal.Decimal // Most-recently-calculated short-duration moving average.
  smaShortPrev decimal.Decimal // Previously-calculated short-duration moving average.
  smaLong      decimal.Decimal // Most-recently-calculated long-duration  moving average.
  smaLongPrev  decimal.Decimal // Previously-calculated long-duration moving average.

  emaEnabled            bool            // Whether or not to use the exponential moving average instead of the simple moving average.
  emaExpSmoothingFactor decimal.Decimal // Calculated smoothing factor for use when using exponential/weighted moving averages.
  emaShort              decimal.Decimal // Most-recently-calculated short-duration exponential moving average.
  emaShortPrev          decimal.Decimal // Previously-calculated short-duration exponential moving average.
  emaLong               decimal.Decimal // Most-recently-calculated long-duration exponential moving average.
  emaLongPrev           decimal.Decimal // Previously-calculated long-duration exponential moving average.
}

//
// Init initializes the algorithm and registers its signal handlers with the Trade Monitor Service.
// Trade algorithms can only be initialized once – subsequent calls will simply return their
// singleton instance.
//
func Init() *Algo {
  once.Do(func() {
    //
    // Instantiate the algorithm.
    //
    o = &Algo{
      candles: evictingqueue.New(*cfgLongLen),

      shortLen: decimal.NewFromInt(int64(*cfgShortLen)),
      longLen:  decimal.NewFromInt(int64(*cfgLongLen)),
      lastSignal: broker.None,

      smaShort:     invalidAvg,
      smaShortPrev: invalidAvg,
      smaLong:      invalidAvg,
      smaLongPrev:  invalidAvg,

      emaEnabled:   *cfgExp,
      emaShort:     invalidAvg,
      emaShortPrev: invalidAvg,
      emaLong:      invalidAvg,
      emaLongPrev:  invalidAvg,
    }

    //
    // Calculate the exponential smoothing factor.
    //
    // NOTE ~> EMA Smoothing Factor = 2 ÷ (number of time periods + 1)
    //
    o.emaExpSmoothingFactor = two.Div(o.longLen.Add(one))

    //
    // Register the correct candle close listener for the configured period length.
    //
    if *cfgPeriod == 1 {
      monitor.Instance().RegisterOneMinCandleCloseHandler(o.candleCloseHandler)
    } else if *cfgPeriod == 5 {
      monitor.Instance().RegisterFiveMinCandleCloseHandler(o.candleCloseHandler)
    } else if *cfgPeriod == 15 {
      monitor.Instance().RegisterFifteenMinCandleCloseHandler(o.candleCloseHandler)
    } else {
      // TODO ~> Throw an error.
    }

    //
    // Log some debug info.
    //
    log.Printf(
      "%s Initialized. (Period = %d minutes, Long MA = %s periods, Short MA = %s periods, Exponential = %t, "+
          "Exponential Smoothing Factor = %s).",
      LogPrefix, *cfgPeriod, o.longLen, o.shortLen, o.emaEnabled, o.emaExpSmoothingFactor,
    )
  })

  return o
}

//
// DeInit disables the algorithm so that it can be re-initialized again in the future if necessary.
// This should really only be used to support tests.
//
func DeInit() {
  //
  // Reset the singleton lock.
  //
  //goland:noinspection GoVetCopyLock
  once = *(new(sync.Once))

  //
  // Unregister candle close handlers
  //
  // TODO ~> This.
  //
}

//
// candleCloseHandler is this algorithm's "candle close handler". It adds the newly-closed candle
// that is provided to it to the algorithm's data structure, updates calculated moving averages
// based on historically-handled candles.
//
func (o *Algo) candleCloseHandler(newCandle *candle.Candle) {
  //
  // Add the new candle to the evicting queue of candles known by the algorithm's instance. If the
  // evicting queue is already full, the oldest entry will be evicted.
  //
  o.candles.Add(newCandle)

  //
  // We are going to need an int64 representation of the current candles queue.
  //
  candlesCurLen := int64(o.candles.Len())

  //
  // Calculate the short moving averages.
  //
  if candlesCurLen >= o.shortLen.IntPart() {
    //
    // Calculate the short SMA.
    //
    o.smaShortPrev = o.smaShort
    o.smaShort = o.calculateSimpleMovingAverage(o.shortLen)

    //
    // Calculate the short EMA. To accurately do this, we must wait one extra period after the
    // calculation of the SMA.
    //
    // NOTE ~> If this is the first EMA that we are calculating, we must prime it with the most
    //  recently-calculated SMA.
    //
    if candlesCurLen > o.shortLen.IntPart() {
      if o.emaShort == invalidAvg {
        o.emaShortPrev = o.smaShort
      } else {
        o.emaShortPrev = o.emaShort
      }

      o.emaShort = o.calculateExponentialMovingAverage(newCandle.CloseAmt(), o.emaShortPrev)
    }
  }

  //
  // Calculate the long moving averages.
  //
  if candlesCurLen >= o.longLen.IntPart() {
    //
    // Calculate the short SMA.
    //
    o.smaLongPrev = o.smaLong
    o.smaLong = o.calculateSimpleMovingAverage(o.longLen)

    //
    // Calculate the long EMA. To accurately do this, we must wait one extra period after the
    // calculation of the SMA.
    //
    // NOTE ~> If this is the first EMA that we are calculating, we must prime it with the most
    //  recently-calculated SMA.
    //
    if candlesCurLen > o.longLen.IntPart() {
      if o.emaLong == invalidAvg {
        o.emaLongPrev = o.smaLong
      } else {
        o.emaLongPrev = o.emaLong
      }

      o.emaLong = o.calculateExponentialMovingAverage(newCandle.CloseAmt(), o.emaLongPrev)
    }
  }

  //
  // Determine if a signal should be fired given the above-calculated moving averages. If we do not
  // yet have a calculation for both moving averages, we must skip this step as the algorithm is not
  // warmed up enough.
  //
  maShort, maShortPrev, maLong, maLongPrev := o.getMovingAverages()
  log.Printf("S %s, SP %s, L %s, LP %s", maShort, maShortPrev, maLong, maLongPrev)

  if maShort.Equal(invalidAvg) || maShortPrev.Equal(invalidAvg) ||
      maLong.Equal(invalidAvg) || maLongPrev.Equal(invalidAvg) {
    log.Printf(
      "%s Not warmed up yet (%d/%s data points collected). One or all moving averages has"+
          " not yet been calculated.",
      LogPrefix,
      o.candles.Len(),
      o.longLen.Add(one),
    )
  } else {
    //
    // Determine if a cross-over has occurred. If one has, determine if the short moving average is
    // now above the long moving average (indicating a "buy" opportunity), or vice-versa (indicating
    // a "sell" opportunity). Otherwise, if no cross-over has occurred, it can be assumed that
    // the current position should be "held".
    //
    shortAboveLong := o.smaShort.GreaterThan(maLong)
    shortAboveLongPrev := o.smaShortPrev.GreaterThan(maLongPrev)

    shortBelowLong := o.smaShort.LessThan(maLong)
    shortBelowLongPrev := o.smaShortPrev.LessThan(maLongPrev)

    if shortAboveLong == shortAboveLongPrev && shortBelowLong == shortBelowLongPrev {
      // TODO ~> Signal that current position should be held.
    } else {
      if shortAboveLong {
        log.Printf(
          "%s Short SMA (%s) has crossed ABOVE long SMA (%s). This is a %s signal (at %s)!",
          LogPrefix, o.smaShort, o.smaLong, aurora.Bold(aurora.Green("BUY")), newCandle.CloseAmt(),
        )

        o.emitSignal(broker.UptrendDetected, newCandle)
      } else if shortBelowLong {
        log.Printf(
          "%s Short SMA (%s) has crossed BELOW long SMA (%s). This is a %s signal (at %s)!",
          LogPrefix, o.smaShort, o.smaLong, aurora.Bold(aurora.Red("SELL")), newCandle.CloseAmt(),
        )

        o.emitSignal(broker.DowntrendDetected, newCandle)
      }
    }
  }
}

//
// emitSignal fires off the specified signal, as triggered by the specified candle, to the Broker
// Service so that it can act upon it.
//
func (o *Algo) emitSignal(signal broker.Signal, candle *candle.Candle) {
  //
  // Actually emit the signal to the Broker Service.
  //
  broker.Instance().Signal(signal, candle.CloseAmt())

  //
  // Cache the just-emitted signal in case we want to refer back to it at any point (e.g. in tests
  // or user interfaces).
  //
  o.lastSignal = signal
}

//
// getMovingAverages returns the proper short and long moving average values depending on how the
// algorithm is configured.
//
func (o *Algo) getMovingAverages() (maShort decimal.Decimal, maShortPrev decimal.Decimal,
    maLong decimal.Decimal, maLongPrev decimal.Decimal) {
  if o.emaEnabled {
    return o.emaShort, o.emaShortPrev, o.emaLong, o.emaLongPrev
  }

  return o.smaShort, o.smaShortPrev, o.smaLong, o.smaLongPrev
}

//
// calculateSimpleMovingAverage calculates and returns a "simple moving average" (SMA) against
// however many recently-handled candles are specified to be looked at.
//
func (o *Algo) calculateSimpleMovingAverage(lookback decimal.Decimal) decimal.Decimal {
  first := o.candles.Len() - 1
  last := int64(o.candles.Len()) - lookback.IntPart()
  cur, _ := o.candles.Get(first)
  avg := cur.(*candle.Candle).CloseAmt()

  // NOTE ~> We add the first candle into the average before the loop, so we must skip it in the
  //  loop so as not to double-weight its value.

  for i := first - 1; int64(i) >= last; i-- {
    cur, _ := o.candles.Get(i)

    avg = avg.Add(cur.(*candle.Candle).CloseAmt())
  }

  return avg.Div(lookback)
}

//
// calculateExponentialMovingAverage calculates and returns an "exponential moving average" (EMA)
// data point for the provided just-closed period.
//
func (o *Algo) calculateExponentialMovingAverage(
    curCloseAmt decimal.Decimal,
    prevEMA decimal.Decimal,
) decimal.Decimal {
  // NOTE ~> EMA = (closing price - previous day's EMA) × smoothing constant as a decimal
  //  + previous day's EMA

  return curCloseAmt.Sub(prevEMA).Mul(o.emaExpSmoothingFactor).Add(prevEMA)
}
