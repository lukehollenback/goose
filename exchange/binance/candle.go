package binance

import (
  "encoding/json"
  "fmt"
  "github.com/shopspring/decimal"
  "time"
)

// NOTE ~> According to https://tinyurl.com/y4eywj46, the structure of the arrays returned from
//  the Binance.US candlestick endpoint are as follows:
//
//  [0]  1499040000000,      // Open time
//  [1]  "0.01634790",       // Open
//  [2]  "0.80000000",       // High
//  [3]  "0.01575800",       // Low
//  [4]  "0.01577100",       // Close
//  [5]  "148976.11427815",  // Volume
//  [6]  1499644799999,      // Close time
//  [7]  "2434.19055334",    // Quote asset volume
//  [8]  308,                // Number of trades
//  [9]  "1756.87402397",    // Taker buy base asset volume
//  [10] "28.46694368",      // Taker buy quote asset volume
//  [11] "17928899.62484339" // Ignore.

const (
  StartTimeIndex               = 0
  OpenIndex                    = 1
  HighIndex                    = 2
  LowIndex                     = 3
  CloseIndex                   = 4
  VolumeIndex                  = 5
  EndTimeIndex                 = 6
  QuoteAssetVolumeIndex        = 7
  CountIndex                   = 8
  TakerBuyBaseAssetVolumeIndex = 9
  TakeBuyQuoteAssetVolumeIndex = 10
)

//
// Response implements the exchange.Candle interface for candlesticks (a.k.a. klines) provided by
// the Binance.US API.
//
type Candle struct {
  start  time.Time
  end    time.Time
  open   decimal.Decimal
  high   decimal.Decimal
  low    decimal.Decimal
  close  decimal.Decimal
  volume decimal.Decimal
  count  int
}

//
// UnmarshalJSON implements the json.Unmarshaller interface for Candle structures so that that JSON
// arrays provided by the Binance.US API that represent them can be properly unmarshalled.
//
func (o *Candle) UnmarshalJSON(data []byte) error {
  //
  // Unmarshall the provided JSON string into a raw interface array.
  //
  // NOTE ~> Unknown numbers always come in as float64 types when unmarshalled. Thus, we are going
  //  to need to expect such values and cast them accordingly.
  //
  var raw []interface{}

  err := json.Unmarshal(data, &raw)
  if err != nil {
    return err
  }

  //
  // Parse the start and end time values of the candle.
  //
  startRaw, ok := raw[StartTimeIndex].(float64)
  if !ok {
    return fmt.Errorf("failed to assert type of start (open) time (%+v)", raw[StartTimeIndex])
  }

  o.start = time.Unix(0, int64(startRaw))

  endRaw, ok := raw[EndTimeIndex].(float64)
  if !ok {
    return fmt.Errorf("failed to assert type of end (close) time (%+v)", raw[EndTimeIndex])
  }

  o.end = time.Unix(0, int64(endRaw))

  //
  // Parse the open, high, low, and close values of the candle.
  //
  openRaw, ok := raw[OpenIndex].(string)
  if !ok {
    return fmt.Errorf("failed to assert type of open (%+v)", raw[OpenIndex])
  }

  o.open, err = decimal.NewFromString(openRaw)
  if err != nil {
    return err
  }

  highRaw, ok := raw[HighIndex].(string)
  if !ok {
    return fmt.Errorf("failed to assert type of high (%+v)", raw[HighIndex])
  }

  o.high, err = decimal.NewFromString(highRaw)
  if err != nil {
    return err
  }

  lowRaw, ok := raw[LowIndex].(string)
  if !ok {
    return fmt.Errorf("failed to assert type of low (%+v)", raw[LowIndex])
  }

  o.low, err = decimal.NewFromString(lowRaw)
  if err != nil {
    return err
  }

  closeRaw, ok := raw[CloseIndex].(string)
  if !ok {
    return fmt.Errorf("failed to assert type of close (%+v)", raw[CloseIndex])
  }

  o.close, err = decimal.NewFromString(closeRaw)
  if err != nil {
    return err
  }

  //
  // Parse the volume value of the candle.
  //
  volumeRaw, ok := raw[VolumeIndex].(string)
  if !ok {
    return fmt.Errorf("failed to assert type of volume (%+v)", raw[VolumeIndex])
  }

  o.volume, err = decimal.NewFromString(volumeRaw)
  if err != nil {
    return err
  }

  //
  // Parse the count value of the candle.
  //
  countRaw, ok := raw[CountIndex].(float64)
  if !ok {
    return fmt.Errorf("failed to assert type of count (%+v)", raw[CountIndex])
  }

  o.count = int(countRaw)

  //
  // Return the candle struct.
  //
  return nil
}
