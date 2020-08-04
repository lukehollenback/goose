package hammerstar

import (
  "fmt"
  "github.com/logrusorgru/aurora"
  "github.com/lukehollenback/goose/structs/evictingqueue"
  "github.com/lukehollenback/goose/trader/candle"
  "github.com/lukehollenback/goose/trader/monitor"
  "github.com/shopspring/decimal"
  "log"
  "sync"
)

var (
  o    *Algo
  once sync.Once
)

const (
  candleCount = 3
)

type Algo struct {
  oneMinCandles *evictingqueue.EvictingQueue // Holds references to the most recent one-minute candles that have been provided to the algorithm.
}

//
// Init initializes the algorithm and registers its signal handlers with the Trade Monitor Service.
// Trade algorithms can only be initialized once – subsequent calls will simply return their
// singleton instance.
//
func Init() *Algo {
  once.Do(func() {
    o = &Algo{
      oneMinCandles: evictingqueue.New(candleCount),
    }

    monitor.Instance().RegisterOneMinCandleCloseHandler(o.onOneMinCandleClose)
  })

  log.Print("Initialized the Candlestar algorithm.")

  return o
}

func (o *Algo) onOneMinCandleClose(candle *candle.Candle) {
  o.oneMinCandles.Add(candle)

  //
  // Check to see if a hammer is detected.
  //
  if o.isHammerScenario() {
    msg := fmt.Sprintf(
      "≪Candlestar≫ Hammer detected and confirmed. %s signal!",
      aurora.Bold(aurora.Green("Buy")),
    )

    log.Print(msg)
  }
}

func (o *Algo) isHammerScenario() bool {
  //
  // If the algorithm is not yet warmed up with enough data, simply return false and wait for the
  // next candle to close.
  //
  if o.oneMinCandles.Len() != candleCount {
    log.Printf(
      "≪Candlestar≫ Skipping signal detection. Only %d/%d necessary candles have closed "+
          "so far.",
      o.oneMinCandles.Len(),
      candleCount,
    )

    return false
  }

  //
  // Confirm potential bottom-out.
  //
  middleCandleRaw, _ := o.oneMinCandles.Get(1)
  middleCandle := middleCandleRaw.(*candle.Candle)

  tailIsAtLeastDoubleBody := middleCandle.BodySize().Mul(decimal.NewFromInt(2)).GreaterThanOrEqual(middleCandle.TailSize())
  closeIsAboveOpen := middleCandle.CloseAmt().GreaterThanOrEqual(middleCandle.OpenAmt())

  lastCandleRaw, _ := o.oneMinCandles.Get(2)
  lastCandle := lastCandleRaw.(*candle.Candle)

  closeIsAboveLow := lastCandle.CloseAmt().GreaterThan(middleCandle.LowAmt())

  return tailIsAtLeastDoubleBody && closeIsAboveOpen && closeIsAboveLow
}
