package candle

//
// Candles holds a single candle reference for each granularity of candle. Each reference may be
// nil if not relevant for the use case.
//
type Candles struct {
  OneMin     *Candle
  FiveMin    *Candle
  FifteenMin *Candle
}
