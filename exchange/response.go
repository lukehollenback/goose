package exchange

import "net/http"

//
// Response generically provides an interface to an object that represents a response from a call to
// an exchange's API endpoint.
//
type Response interface {

  //
  // Raw provides the raw HTTP response from the endpoint call that was made.
  //
  Raw() *http.Response

  //
  // Candles provides a slice of the candles returned from the endpoint call that was made (if there
  // were any).
  //
  Candles() []*Candle

}