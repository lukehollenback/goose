package binance

import (
  "encoding/json"
  "fmt"
  "github.com/lukehollenback/goose/exchange"
  "io"
  "io/ioutil"
  "net/http"
  "time"
)

//
// Client implements the exchange.Client interface for the Binance.US API.
//
type Client struct {
  apiKey    string
  apiSecret string
  httpClient *http.Client
}

func NewClient() *Client {
  return &Client{
    httpClient: &http.Client{},
  }
}

func (o *Client) Auth(key string, secret string) (exchange.Response, error) {
  o.apiKey = key
  o.apiSecret = secret

  return nil, nil
}

func (o Client) RetrieveCandles(
    symbol string,
    interval exchange.Interval,
    start time.Time,
    end time.Time,
    limit int,
) (exchange.Response, error) {
  //
  // Build request URL.
  //
  url := fmt.Sprintf(
    "%s?symbol=%s&interval=%s&startTime=%d&endTime=%d&limit=%d",
    CandlesURL,
    symbol,
    interval,
    start.UnixNano()/1000000,
    end.UnixNano()/1000000,
    limit,
  )

  //
  // Make the endpoint request and handle any errors along the way.
  //
  resp, err := o.request("GET", url, nil)
  if err != nil {
    return resp, err
  }

  //
  // Parse the response
  //
  var candles []*Candle

  err = json.Unmarshal(resp.body, &candles)
  if err != nil {
    return resp, err
  }

  //
  // Finish packing the wrapped response and return it.
  //
  resp.candles = candles

  return resp, nil
}

//
// request makes the specified request to the Binance.US API and returns a wrapped response (parsed
// as much as generically possible) and/or an error if something went wrong.
//
func (o *Client) request(method string, url string, body io.Reader) (*Response, error) {
  //
  // Make a request to the endpoint.
  //
  req, err := http.NewRequest("GET", url, nil)
  if err != nil {
    return nil, err
  }

  req.Header.Add(APIKeyHeader, o.apiKey)

  resp, err := o.httpClient.Do(req)
  if err != nil {
    return nil, err
  }

  //
  // Make sure the error code was valid.
  //
  if resp.StatusCode != 200 {
    return nil,exchange.NewHTTPError(resp.StatusCode)
  }

  //
  // Begin wrapping the response in the standard response structure.
  //
  wrappedResp := &Response{
    response: resp,
  }

  //
  // Read the response.
  //
  respBody, err := ioutil.ReadAll(resp.Body)
  if err != nil {
    return wrappedResp, err
  }

  wrappedResp.body = respBody

  //
  // Check the response for API errors.
  //
  var apiErr *APIError

  _ = json.Unmarshal(respBody, &apiErr)

  if apiErr.populated() {
    return wrappedResp, apiErr
  }

  //
  // Return the wrapped response.
  //
  return wrappedResp, nil
}