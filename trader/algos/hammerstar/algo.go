package main

import (
  "github.com/lukehollenback/goose/structs/evictingqueue"
  "github.com/lukehollenback/goose/trader/candle"
  "github.com/lukehollenback/goose/trader/monitor"
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
  oneMinCandles  *evictingqueue.EvictingQueue // Holds references to the most recent one-minute candles that have been provided to the algorithm.
}

//
// Init initializes the algorithm and registers its signal handlers with the Trade Monitor Service.
// Trade algorithms can only be initialized once – subsequent calls will simply return their
// singleton instance.
//
func Init() *Algo {
  once.Do(func() {
    o = &Algo{
      oneMinCandles:  evictingqueue.New(candleCount),
    }

    monitor.Instance().RegisterOneMinCandleCloseHandler(o.onOneMinCandleClose)
  })

  return o
}

func (o *Algo) onOneMinCandleClose(candle *candle.Candle) {
  o.oneMinCandles.Add(candle)
}