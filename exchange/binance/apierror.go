package binance

import "fmt"

//
// APIError implements the exchange.APIError interface for errors returned from Binance.US API
// calls.
//
type APIError struct {
  Code    int    `json:"code"`
  Message string `json:"msg"`
}

func (o *APIError) ErrorCode() int {
  return o.Code
}

func (o *APIError) ErrorMessage() string {
  return o.Message
}

func (o *APIError) Error() string {
  return fmt.Sprintf(
    "the Binance.US endpoint returned an API error (code: %d, message: %s)",
    o.ErrorCode(), o.ErrorMessage(),
  )
}

//
// populated returns whether or not the structure appears to actually hold an error. This is useful
// when determining whether or not the deserialized response payload was actually an error that fit
// into the structure's model or not.
//
func (o *APIError) populated() bool {
  return o.Code != 0 && o.Message != ""
}