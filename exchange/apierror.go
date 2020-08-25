package exchange

//
// APIError generically provides an interface to objects that represent a first-class error provided
// in the response of a request against a cryptocurrency exchange's API.
//
type APIError interface {

  //
  // Code returns the actual error code provided by the API (if there was one).
  //
  Code() int

  //
  // Message returns the actual error message provided by the API (if there was one).
  //
  Message() string

}
