package candle

import (
	"fmt"
	"github.com/shopspring/decimal"
	"sync"
	"time"
)

type Store struct {
	mu              *sync.Mutex
	interval        time.Duration
	candles         []*Candle
	lastCandleStart time.Time
	lastCandleEnd   time.Time
}

//
// CreateStore instantiates a new candle store that will hold candles of the specified duration
// interval. For example, one might instantiate a 1-minute candle store, a 5-minute candle store,
// and a 15-minute candle store.
//
func CreateStore(interval time.Duration, initialCandle *Candle) (*Store, error) {
	//
	// Instantiate the new candle store.
	//
	o := &Store{
		mu:       &sync.Mutex{},
		interval: interval,
		candles:  make([]*Candle, 0),
	}

	//
	// Add the initial candle to the new candle store.
	//
	err := o.appendCandle(initialCandle)
	if err != nil {
		return nil, err
	}

	return o, nil
}

//
// Previous retrieves the last closed-out candle from the candle store. If one does not exist, it
// simply returns nil.
//
func (o *Store) Previous() *Candle {
	o.mu.Lock()
	defer o.mu.Unlock()

	pos := len(o.candles) - 2

	if pos < 0 {
		return nil
	}

	return o.candles[pos]
}

//
// Current retrieves the current in-progress candle from the candle store. If one does not exist, it
// simply returns nil.
//
func (o *Store) Current() *Candle {
	o.mu.Lock()
	defer o.mu.Unlock()

	pos := len(o.candles) - 1

	if pos < 0 {
		return nil
	}

	return o.candles[pos]
}

//
// Append calculates a new trade into the most recently-created candle in the candle store. If the
// time of the trade does not fall within the timespan of said candle, an error will occur.
//
func (o *Store) Append(time time.Time, amt decimal.Decimal) (bool, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	//
	// Validate that we are not trying to append to a historical, closed-out candle in the candle
	// store.
	//
	if time.Before(o.lastCandleStart) {
		return false, fmt.Errorf("cannot modify closed-out candles in candle store")
	}

	//
	// Figure out if we need to create a new candle and do so if necessary.
	//
	if time.After(o.lastCandleEnd) {
		if err := o.appendNewCandle(o.lastCandleEnd, amt); err != nil {
			return false, err
		}

		return true, nil
	}

	//
	// If we get this far, we simply need to append to the most recent candle in the store. Grab the
	// last candle in the candle store – which we have, at this point, validated is the one we must
	// append the trade to – and actually update it.
	//
	curCandle := o.candles[len(o.candles)-1]

	if err := curCandle.Append(time, amt); err != nil {
		return false, err
	}

	return false, nil
}

//
// appendNewCandle creates a brand-new candle with the provided initial values and adds it to the
// candle store.
//
func (o *Store) appendNewCandle(start time.Time, amt decimal.Decimal) error {
	candle := CreateCandle(start, o.interval, amt)

	return o.appendCandle(candle)
}

//
// appendCandle adds the provided candle to the tip of the candle store. If the provided candle does
// not start after final instant of the previous candle in the candle store, an error will occur.
//
func (o *Store) appendCandle(candle *Candle) error {
	//
	// Ensure that the candle is of the same duration as those held by the candle store.
	//
	if candle.duration != o.interval {
		return fmt.Errorf(
			"cannot append candle of duration %s to candle store of %s candles",
			candle.duration, o.interval,
		)
	}

	//
	// Actually append the candle to the candle store.
	//
	o.candles = append(o.candles, candle)
	o.lastCandleStart = candle.start
	o.lastCandleEnd = candle.start.Add(o.interval)

	return nil
}
