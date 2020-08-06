package main

import (
	"github.com/lukehollenback/goose/trader/algos/movingaverages"
	"github.com/lukehollenback/goose/trader/broker"
	"log"
	"os"
	"os/signal"

	"github.com/lukehollenback/goose/trader/candle"
	"github.com/lukehollenback/goose/trader/monitor"
)

func main() {
	//
	// Register a kill signal handler with the operating system so that we can gracefully shutdown if
	// necessary.
	//
	osInterrupt := make(chan os.Signal, 1)

	signal.Notify(osInterrupt, os.Interrupt)

	//
	// Start up all necessary services.
	//
	chCandleStarted, err := candle.Instance().Start()
	if err != nil {
		log.Fatalf("Failed to start the match monitor service. (Error: %s)", err)
	}

	chBrokerStarted, err := broker.Instance().Start()
	if err != nil {
		log.Fatalf("Failed to start the broker service. (Error: %s)", err)
	}

	chMonitorStarted, err := monitor.Instance().Start()
	if err != nil {
		log.Fatalf("Failed to start the match monitor service. (Error: %s)", err)
	}

	<-chCandleStarted
	<-chBrokerStarted
	<-chMonitorStarted

	//
	// Start the desired algorithm(s).
	//
	movingaverages.Init()

	//
	// Block until we are shut down by the operating system.
	//
	<-osInterrupt

	log.Print("An operating system interrupt has been received. Shutting down all services...")

	//
	// Stop all running services.
	//
	chMonitorStopped, err := monitor.Instance().Stop()
	if err != nil {
		log.Fatalf("Failed to stop the match monitor service. (Error: %s)", err)
	}

	chBrokerStopped, err := broker.Instance().Stop()
	if err != nil {
		log.Fatalf("Failed to stop the broker service. (Error: %s)", err)
	}

	chCandleStopped, err := candle.Instance().Stop()
	if err != nil {
		log.Fatalf("Failed to stop the match monitor service. (Error: %s)", err)
	}

	<-chMonitorStopped
	<-chBrokerStopped
	<-chCandleStopped

	//
	// Wrap everything up.
	//
	log.Print("Goodbye.")
}
