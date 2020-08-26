package exchange

import "fmt"

//
// HTTPError represents an error due to non-200 response from an API endpoint. When dealing with
// cryptocurrency exchange APIs, such a response almost always means that something critically wrong
// has occurred.
//
type HTTPError struct {
  statusCode int
}

func NewHTTPError(statusCode int) *HTTPError {
  return &HTTPError{
    statusCode: statusCode,
  }
}

func (o *HTTPError) StatusCode() int {
  return o.statusCode
}

func (o *HTTPError) Error() string {
  return fmt.Sprintf("server responded with a %d status code", o.statusCode)
}
