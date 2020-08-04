package candle

import (
  "fmt"
  "sync"
  "time"

  "github.com/logrusorgru/aurora"
  "github.com/shopspring/decimal"
)

var One = decimal.NewFromInt(1)

//
// Candle represents a snapshot of match data.
//
type Candle struct {
  mu       *sync.Mutex
  start    time.Time
  duration time.Duration
  open     decimal.Decimal
  close    decimal.Decimal
  high     decimal.Decimal
  low      decimal.Decimal
  avg      decimal.Decimal
  total    decimal.Decimal
  cnt      decimal.Decimal
}

//
// CreateCandle instantiates a new candle struct.
//
func CreateCandle(
	start time.Time,
	duration time.Duration,
	firstAmt decimal.Decimal,
) *Candle {
	o := &Candle{
		mu:       &sync.Mutex{},
		start:    start,
		duration: duration,
		open:     firstAmt,
		close:    firstAmt,
		high:     firstAmt,
		low:      firstAmt,
		avg:      firstAmt,
		total:    firstAmt,
		cnt:      One,
	}

	return o
}

//
// CreateFullCandle instantiates a new candle struct with each specific paramter specified. This can
// be useful when it is necessary to instantiate a cande that matches a known historical candle that
// has been provided by an financial exchange.
//
func CreateFullCandle(
    start time.Time,
    duration time.Duration,
    open decimal.Decimal,
    close decimal.Decimal,
    high decimal.Decimal,
    low decimal.Decimal,
    avg decimal.Decimal,
    total decimal.Decimal,
    cnt decimal.Decimal,
) *Candle {
  o := &Candle{
    mu:       &sync.Mutex{},
    start:    start,
    duration: duration,
    open:     open,
    close:    close,
    high:     high,
    low:      low,
    avg:      avg,
    total:    total,
    cnt:      cnt,
  }

  return o
}

//
// Append calculates a given transaction into the candle (if possible). It is expected that the
// provided transaction occurred within the window in time that the candle represents a snapshot of.
//
func (o *Candle) Append(time time.Time, amt decimal.Decimal) error {
  o.mu.Lock()
  defer o.mu.Unlock()

  //
  // Make sure the provided transaction is valid for this candle.
  //
  if time.After(o.start.Add(o.duration)) {
    return fmt.Errorf(
      "cannot append transaction from %s to %s candle starting at %s",
      time, o.duration, o.start,
    )
  }

  //
  // Update the necessary fields of the candle.
  //
  o.close = amt

  if amt.GreaterThan(o.high) {
    o.high = amt
  }

  if amt.LessThan(o.low) {
    o.low = amt
  }

  o.total = o.total.Add(amt)
  o.cnt = o.cnt.Add(One)
  o.avg = o.total.Div(o.cnt)

  return nil
}

//
// End returns the ending instant of time of the candle.
//
func (o *Candle) End() time.Time {
  return o.start.Add(o.duration)
}

//
// OpenAmt returns the currency amount that the candle opened at.
//
func (o *Candle) OpenAmt() decimal.Decimal {
  return o.open
}

//
// CloseAmt returns the currency amount that the candle closed at.
//
func (o *Candle) CloseAmt() decimal.Decimal {
  return o.close
}

// TODO ~> Document.
func (o *Candle) HighAmt() decimal.Decimal {
  return o.high
}

// TODO ~> Document.
func (o *Candle) LowAmt() decimal.Decimal {
  return o.low
}

// TODO ~> Document.
func (o *Candle) BodyTop() decimal.Decimal {
  if o.close.GreaterThan(o.open) {
    return o.close
  }

  return o.open
}

// TODO ~> Document.
func (o *Candle) BodyBottom() decimal.Decimal {
  if o.close.LessThan(o.open) {
    return o.close
  }

  return o.open
}

// TODO ~> Document.
func (o *Candle) BodySize() decimal.Decimal {
  return o.BodyTop().Sub(o.BodyBottom())
}

// TODO ~> Document.
func (o *Candle) WickSize() decimal.Decimal {
  // NOTE ~> We intentionally do not take the absolute value after subtracting here because doing so
  //  could mask a critical issue in our code. If the returned value is ever negative, something is
  //  wrong with how the candle's high/low/open/close are being tracked.

  return o.high.Sub(o.BodyTop())
}

// TODO ~> Document.
func (o *Candle) TailSize() decimal.Decimal {
  // NOTE ~> We intentionally do not take the absolute value after subtracting here because doing so
  //  could mask a critical issue in our code. If the returned value is ever negative, something is
  //  wrong with how the candle's high/low/open/close are being tracked.

  return o.BodyBottom().Sub(o.low)
}

func (o *Candle) String() string {
  var arrow aurora.Value

  if o.close.GreaterThan(o.open) {
    arrow = aurora.Green("▲")
  } else if o.close.LessThan(o.open) {
    arrow = aurora.Red("▼")
  } else {
    arrow = aurora.Blue("◼")
  }

  return fmt.Sprintf(
    "%s (O: %-8s  C: %-8s  H: %-8s  L: %-8s  A: %-22s  C: %-5s  S: %s)",
    arrow, o.open, o.close, o.high, o.low, o.avg, o.cnt, o.start,
  )
}