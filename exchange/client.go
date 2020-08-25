package exchange

import (
  "time"
)

//
// Client generically provides an interface to an object that can be used to interact with a
// cryptocurrency exchange's regular REST API. Normally, this is the client used to do things
// like place orders, check balances, and retrieve historical trade data.
//
// Whenever an endpoint fails – whether due to a system failure, an HTTP error, or an API error –
// the error component of the response will be non-nil and, if at all possible, the response payload
// that was received will be returned.
//
type Client interface {

  //
  // Auth provides the relevant exchange's API key and secret to the client. Some implementations
  // will actually authenticate against the API, in which case a meaningful response will
  // be returned. Others will simply store the information for use in headers, in which case the
  // returned response future will be meaningless.
  //
  Auth(key string, secret string) (*Response, error)

  //
  // RetrieveCandles retrieves candles of the specified interval for the specified ticker symbol
  // within the specified time range. A maximum of the specified limit of candles will be returned
  // (note that many exchanges impose a hard maximum on the limit – usually at around 1000 candles).
  //
  RetrieveCandles(symbol string, interval Interval, start time.Time, end time.Time, limit int) (*Response, error)
}
