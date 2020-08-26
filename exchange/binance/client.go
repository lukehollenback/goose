package binance

import (
  "encoding/json"
  "fmt"
  "github.com/lukehollenback/goose/exchange"
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
    CandlesUrl,
    symbol,
    interval,
    start.UnixNano()/1000000,
    end.UnixNano()/1000000,
    limit,
  )

  //
  // Make a request to the endpoint.
  //
  req, err := http.NewRequest("GET", url, nil)
  if err != nil {
    return nil, err
  }

  req.Header.Add(ApiKeyHeader, o.apiKey)

  resp, err := o.httpClient.Do(req)
  if err != nil {
    return nil, err
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
  body, err := ioutil.ReadAll(resp.Body)
  if err != nil {
    return wrappedResp, err
  }

  //
  // Check the response for API errors.
  //
  var apiErr *APIError

  _ = json.Unmarshal(body, &apiErr)

  if apiErr.populated() {
    return wrappedResp, apiErr
  }

  //
  // Parse the response
  //
  var candles []*Candle

  err = json.Unmarshal(body, &candles)
  if err != nil {
    return wrappedResp, err
  }

  //
  // Finish packing the wrapped response and return it.
  //
  wrappedResp.candles = candles

  return wrappedResp, nil
}
