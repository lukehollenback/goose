package binance

import (
  "github.com/lukehollenback/goose/exchange"
  "net/http"
)

//
// Response implements the exchange.Response interface for wrapped responses from the Binance.US
// API.
//
type Response struct {
  response *http.Response
  body     []byte
  candles  []*Candle
}

func (o *Response) Raw() *http.Response {
  return o.response
}

func (o *Response) Body() []byte {
  return o.body
}

func (o *Response) Candles() []exchange.Candle {
  ret := make([]exchange.Candle, len(o.candles))

  for i, v := range o.candles {
    ret[i] = v
  }

  return ret
}
