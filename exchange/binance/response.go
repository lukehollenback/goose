package binance

import (
  "net/http"
)

//
// Response implements the exchange.Response interface for wrapped responses from the Binance.US
// API.
//
type Response struct {
  response *http.Response
  candles []*Candle
}

func (o *Response) Raw() *http.Response {
  return o.response
}

func (o *Response) Candles() []*Candle {
  return o.candles
}
