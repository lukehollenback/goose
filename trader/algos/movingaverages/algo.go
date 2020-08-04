package movingaverages

import (
  "github.com/logrusorgru/aurora"
  "github.com/lukehollenback/goose/structs/evictingqueue"
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
  o    *Algo
  once sync.Once

  invalidAvg = decimal.NewFromInt(-1)
)

type Algo struct {
  candles     *evictingqueue.EvictingQueue // Holds references to the most recent one-minute candles that have been provided to the algorithm.
  maShortLen  decimal.Decimal              // Length of the short-duration moving average.
  maShort     decimal.Decimal              // Most-recently-calculated short-duration moving average.
  maShortPrev decimal.Decimal              // Previously-calculated short-duration moving average.
  maLongLen   decimal.Decimal              // Length of the long-duration moving average.
  maLong      decimal.Decimal              // Most-recently-calculated long-duration  moving average.
  maLongPrev  decimal.Decimal              // Previously-calculated long-duration moving average.
}

//
// Init initializes the algorithm and registers its signal handlers with the Trade Monitor Service.
// Trade algorithms can only be initialized once – subsequent calls will simply return their
// singleton instance.
//
func Init() *Algo {
  once.Do(func() {
    o = &Algo{
      candles:     evictingqueue.New(5),
      maShortLen:  decimal.NewFromInt(1),
      maShort:     invalidAvg,
      maShortPrev: invalidAvg,
      maLongLen:   decimal.NewFromInt(5),
      maLong:      invalidAvg,
      maLongPrev:  invalidAvg,
    }

    monitor.Instance().RegisterOneMinCandleCloseHandler(o.onOneMinCandleClose)

    log.Printf("Initialized the %s algorithm.", LogPrefix)
  })

  return o
}

//
// onOneMinCandleClose is this algorithm's "candle close handler". It adds the newly-closed candle
// that is provided to it to the algorithm's data structure, updates calculated moving averages
// based on historically-handled candles.
//
func (o *Algo) onOneMinCandleClose(newCandle *candle.Candle) {
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
  // Calculate the short moving average.
  //
  if candlesCurLen >= o.maShortLen.IntPart() {
    o.maShortPrev = o.maShort
    o.maShort = o.calculateAverage(o.maShortLen)
  }

  //
  // Calculate the long moving average.
  //
  if candlesCurLen >= o.maLongLen.IntPart() {
    o.maLongPrev = o.maLong
    o.maLong = o.calculateAverage(o.maLongLen)
  }

  //
  // Determine if a signal should be fired given the above-calculated moving averages. If we do not
  // yet have a calculation for both moving averages, we must skip this step as the algorithm is not
  // warmed up enough.
  //
  if o.maShort.Equal(invalidAvg) || o.maShortPrev.Equal(invalidAvg) ||
      o.maLong.Equal(invalidAvg) || o.maLongPrev.Equal(invalidAvg) {
    log.Printf(
      "%s Not warmed up yet (%d/%s data points collected). One or all moving averages has"+
          " not yet been calculated.",
      LogPrefix,
      o.candles.Len(),
      o.maLongLen,
    )
  } else {
    //
    // Determine if a cross-over has occurred. If one has, determine if the short moving average is
    // now above the long moving average (indicating a "buy" opportunity), or vice-versa (indicating
    // a "sell" opportunity). Otherwise, if no cross-over has occurred, it can be assumed that
    // the current position should be "held".
    //
    shortAboveLong := o.maShort.GreaterThan(o.maLong)
    shortAboveLongPrev := o.maShortPrev.GreaterThan(o.maLongPrev)

    if shortAboveLong == shortAboveLongPrev {
      // TODO ~> Signal that current position should be held.
    } else {
      if shortAboveLong {
        // TODO ~> Signal that position should be bought.

        log.Printf(
          "%s Short SMA (%s) has crossed above long SMA (%s). This is a %s signal (at %s)!",
          LogPrefix, o.maShort, o.maLong, aurora.Bold(aurora.Green("buy")), newCandle.CloseAmt(),
        )
      } else {
        // TODO ~> Signal that position should be sold.

        log.Printf(
          "%s Short SMA (%s) has crossed above long SMA (%s). This is a %s signal (at %s)!",
          LogPrefix, o.maShort, o.maLong, aurora.Bold(aurora.Red("sell")), newCandle.CloseAmt(),
        )
      }
    }
  }
}

//
// calculateAverage calculates and returns a "moving average" against however many recently-handled
// candles are specified to be looked at.
//
func (o *Algo) calculateAverage(lookback decimal.Decimal) decimal.Decimal {
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
