package exchange

import (
  "github.com/shopspring/decimal"
  "time"
)

//
// Candle generically provides an interface to objects that represent candlesticks (a.k.a. klines)
// provided in a response from a call to an exchange's API endpoint.
//
type Candle interface {

  //
  // StartTime returns a pointer to the structure representing the opening instant of the candle.
  //
  StartTime() *time.Time

  //
  // EndTime returns a pointer to the structure representing the closing instant of the candle. As
  // an example, a one minute candle might start at 2020/8/25 00:00:00:000000000 and end at
  // 2020.8.25 00:00:00:999999999.
  //
  EndTime() *time.Time

  //
  // Open returns a pointer to the structure representing the opening price of the candle.
  //
  Open() *decimal.Decimal

  //
  // High returns a pointer to the structure representing the high price of the candle.
  //
  High() *decimal.Decimal

  //
  // Low returns a pointer to the structure representing the low price of the candle.
  //
  Low() *decimal.Decimal

  //
  // Close returns a pointer to the structure representing the closing price of the candle.
  //
  Close() *decimal.Decimal

  //
  // Volume returns a pointer to the structure representing the trade volume of the candle.
  //
  Volume() *decimal.Decimal

  //
  // Open returns a pointer to the transaction count of the candle.
  //
  Count() *int

}
