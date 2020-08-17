package movingaverages

import (
  "github.com/lukehollenback/goose/trader/broker"
  "github.com/lukehollenback/goose/trader/candle"
  "github.com/shopspring/decimal"
  "testing"
  "time"
)

const (
  FiveMinutes = 5 * time.Minute
)

var (
  now = time.Now()
)

//
// seedAlgo generates the exact number of candles required to get the algorithm warmed up. In order
// to ensure a constant and known algorithm state, all candles are given the exact same close value
// of 5000.
//
func seedAlgo() {
  for i := 0; i < 15; i++ {
    // NOTE ~> We do not care about the timestamp each of these candles are tagged too. We are not
    //  testing candle stores here.

    data := candle.CreateCandle(now, FiveMinutes, decimal.NewFromInt(10000))

    o.candleCloseHandler(data)
  }
}

func TestFiveMinuteSMAFiveOverFifteen(t *testing.T) {
  //
  // Initialize the algorithm.
  //
  // NOTE ~> We must first forcefully reset the singleton lock in case a previous test already tried
  //  to initialize the algorithm.
  //
  DeInit()
  Init()

  //
  // Seed the algorithm and verify that it is in the expected constant state.
  //
  seedAlgo()

  //
  // Simulate a short-over-long crossover and validate that averages were calculated properly and
  // signals were determined properly.
  //
  data := candle.CreateCandle(now, FiveMinutes, decimal.NewFromInt(19000))

  o.candleCloseHandler(data)

  if !o.smaShort.Equal(decimal.NewFromInt(11800)) {
    t.Errorf("Expected SMA Short to be 11,800 but was instead %s.", o.smaShort)
  }

  if !o.smaLong.Equal(decimal.NewFromInt(10600)) {
    t.Errorf("Expected SMA Long to be 10,600 but was instead %s.", o.smaLong)
  }

  if o.lastSignal != broker.UptrendDetected {
    t.Errorf(
      "Expected an \"Uptrend Detected\" (signal %d) signal to have been fired, but instead a %d signal was.",
      broker.UptrendDetected, o.lastSignal,
    )
  }
}

func TestFiveMinuteSMAFiveUnderFifteen(t *testing.T) {
  //
  // Initialize the algorithm.
  //
  // NOTE ~> We must first forcefully reset the singleton lock in case a previous test already tried
  //  to initialize the algorithm.
  //
  DeInit()
  Init()

  //
  // Seed the algorithm and verify that it is in the expected constant state.
  //
  seedAlgo()

  //
  // Simulate a short-under-long crossover and validate that averages were calculated properly and
  // signals were determined properly.
  //
  data := candle.CreateCandle(now, FiveMinutes, decimal.NewFromInt(1000))

  o.candleCloseHandler(data)

  if !o.smaShort.Equal(decimal.NewFromInt(8200)) {
    t.Errorf("Expected SMA Short to be 8,200 but was instead %s.", o.smaShort)
  }

  if !o.smaLong.Equal(decimal.NewFromInt(9400)) {
    t.Errorf("Expected SMA Long to be 9,400 but was instead %s.", o.smaLong)
  }

  if o.lastSignal != broker.DowntrendDetected {
    t.Errorf(
      "Expected an \"Downtrend Detected\" (signal %d) signal to have been fired, but instead a %d signal was.",
      broker.DowntrendDetected, o.lastSignal,
    )
  }
}

func TestFiveMinuteEMAFiveOverFifteen(t *testing.T) {
  //
  // Initialize the algorithm.
  //
  // NOTE ~> We must first forcefully reset the singleton lock in case a previous test already tried
  //  to initialize the algorithm.
  //
  DeInit()
  Init()
}
