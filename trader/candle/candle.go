package candle

import (
	"log"
	"sync"
	"time"

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
// Append calculates a given transaction into the candle (if possible). It is expected that the
// provided transaction occured within the window in time that the candle represents a snapshot of.
//
func (o *Candle) Append(time time.Time, amt decimal.Decimal) {
	o.mu.Lock()
	defer o.mu.Unlock()

	//
	// Make sure the provided transaction is valid for this candle.
	//
	if time.After(o.start.Add(o.duration)) {
		log.Fatalf(
			"Cannot append transaction from %s to %s candle starting at %s.",
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
}
