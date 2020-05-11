package monitor

type state int

const (
  disconnected state = iota // The monitor service has not yet attempted to establish a connection to the Coinbase Pro websocket API.
  connecting                // The monitor service is attempting to establish a connection to the Coinbase Pro websocket API.
  connected                 // The monitor service has connected to the Coinbase Pro websocket API.
  subscribed                // The monitor service has successfully subscribed to necessary message channels of the Coinbase Pro websocket API.
  ready                     // The monitor service has successfully initialized the candle store service with the most recent known trade and is thus ready to start processing new trade messages.
)
