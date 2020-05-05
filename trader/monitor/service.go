package monitor

import (
	"log"
	"sync"

	ws "github.com/gorilla/websocket"
	coinbasepro "github.com/preichenberger/go-coinbasepro/v2"
)

var (
	o    *Service
	once sync.Once
)

//
// Service represents a match monitor service instance.
//
type Service struct {
	state     state
	conn      *ws.Conn
	chKill    chan bool
	chStopped chan bool
}

//
// Instance returns a singleton instance of the match monitor service.
//
func Instance() *Service {
	once.Do(func() {
		o = &Service{
			state: disconnected,
		}
	})

	return o
}

//
// Start fires up the service. It is up to the caller to not call this multiple times in a row
// without stopping the service and waiting for full termination in between. A channel that can be
// blocked on for a "true" value – which indicates that start up is complete – is returned.
//
func (o *Service) Start() (<-chan bool, error) {
	//
	// (Re)initialize our instance variables.
	//
	o.chKill = make(chan bool, 1)
	o.chStopped = make(chan bool, 1)

	//
	// Fire off a goroutine as the executor for the service.
	//
	go o.monitor()

	//
	// Return our "started" channel in case the caller wants to block on it and log some debug info.
	//
	chStarted := make(chan bool, 1)
	chStarted <- true

	log.Printf("The match monitor service has started.")

	return chStarted, nil
}

//
// Stop tells the service to shut down. It is up to the caller to not call this multiple times in
// a row without starting the service first. A channel that can be blocked on for a "true" value –
// which indiciates that shut down is complete – is returned.
//
func (o *Service) Stop() (<-chan bool, error) {
	//
	// Log some debug info.
	//
	log.Printf("The match monitor service is stopping...")

	//
	// Tell the goroutines that were spun off by the service to shutdown.
	//
	o.chKill <- true

	//
	// Return the "stopped" channel that the caller can block on if they need to know that the
	// service has completely shutdown.
	//
	return o.chStopped, nil
}

//
// Monitor connects to the Coinbase Pro websocket feed and monitors it for trade events so that it
// can determine when to buy or sell currency.
//
func (o *Service) monitor() {
	var err error

	//
	// Connect to the Coinbase Pro websocket feed so that we can monitor network events that occur.
	//
	var wsDialer ws.Dialer

	o.conn, _, err = wsDialer.Dial("wss://ws-feed.pro.coinbase.com", nil)
	if err != nil {
		log.Fatalf("Could not connect to the Coinbase Pro websocket feed. (Error: %s)", err.Error())
	}

	o.state = connected

	//
	// Subscribe to heartbeat messages and trade messages over the Coinbase Pro websocket feed.
	//
	subscribe := coinbasepro.Message{
		Type: "subscribe",
		Channels: []coinbasepro.MessageChannel{
			coinbasepro.MessageChannel{
				Name: "heartbeat",
				ProductIds: []string{
					"BTC-USD",
				},
			},
			coinbasepro.MessageChannel{
				Name: "matches",
				ProductIds: []string{
					"BTC-USD",
				},
			},
		},
	}

	if err := o.conn.WriteJSON(subscribe); err != nil {
		log.Fatalf(
			"Could not subscribe to specific messages from the Coinbase Pro websocket feed. (Error: %s)",
			err.Error(),
		)
	}

	//
	// Begin monitoring and processing messages from the Coinbase Pro websocket feed.
	//
	cont := true

	for cont {
		chMsg := make(chan *coinbasepro.Message, 1)
		chErr := make(chan error, 1)

		go o.readNextMessage(chMsg, chErr)

		select {
		case <-o.chKill:
			cont = false

			break

		case err := <-chErr:
			log.Fatalf(
				"Could not read the next JSON message from the Coinbase Pro websocket feed. (Error: %s)",
				err.Error(),
			)

			cont = false

			break

		case msg := <-chMsg:
			o.handleMessage(msg)

			break
		}
	}

	//
	// Close our websocket connection.
	//
	o.conn.Close()

	o.state = disconnected

	//
	// Send the signal that we have shut down.
	//
	o.chStopped <- true
}

func (o *Service) readNextMessage(chMsg chan<- *coinbasepro.Message, chErr chan<- error) {
	msg := &coinbasepro.Message{}

	if err := o.conn.ReadJSON(msg); err != nil {
		chErr <- err
	}

	chMsg <- msg
}

func (o *Service) handleMessage(msg *coinbasepro.Message) {
	if msg.Type == "subscriptions" {
		o.state = subscribed

		log.Printf("Successfully subscribed to relevant Coinbase Pro websocket channels.")
	} else if o.state == subscribed {
		if msg.Type == "match" {
			log.Printf("%s", msg.Price)
		}
	}
}
