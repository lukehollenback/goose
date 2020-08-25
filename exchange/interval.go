package exchange

//
// Interval is an enum that represents various kline/candlestick intervals that can be retrieved
// from an exchange's historical data endpoints.
//
type Interval int

const (
  OneMinute Interval = iota
  ThreeMinute
  FiveMinute
  FifteenMinute
  ThirtyMinute
  OneHour
  TwoHour
  FourHour
  SixHour
  EightHour
  TwelveHour
  OneDay
  ThreeDay
  OneWeek
  OneMonth
)

func (o Interval) String() string {
  return [...]string{"1m", "3m", "5m", "15m", "30m", "1h", "2h", "4h", "6h", "8h", "12h", "1d", "3d", "1w", "1M"}[o]
}
