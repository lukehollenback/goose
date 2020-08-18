package main

import (
	"flag"
	"fmt"
	"github.com/lukehollenback/goose/trader/algos/movingaverages"
	"github.com/lukehollenback/goose/trader/broker"
	"github.com/shopspring/decimal"
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
	// Register and parse global flags.
	//
	// NOTE ~> Individual services and algorithms might register their own flags in their package
	//  initialization functions. This, however, is the only place where they are parsed.
	//
	cfgAsset := flag.String(
		"asset",
		"BTC",
		fmt.Sprintf("The asset that should be traded."),
	)

	cfgMock := flag.Bool(
		"mock",
		false,
		fmt.Sprintf("Whether or not mock trading should be enabled."),
	)

	cfgMockAmt := flag.Int64(
		"mock-amount",
		1000,
		fmt.Sprintf("The initial amount of USD to fund the mock trader with."),
	)

	cfgMockFee := flag.Float64(
		"mock-fee",
		0.00075,
		fmt.Sprintf("The maker/taker fee that each mock trade costs to execute."),
	)

	flag.Parse()

	//
	// Start up the Candle Service
	//
	chCandleStarted, err := candle.Instance().Start()
	if err != nil {
		log.Fatalf("Failed to start the match monitor service. (Error: %s)", err)
	}

	//
	// Start up the Broker Service.
	//
	broker.Instance().SetAsset(*cfgAsset)

	if *cfgMock {
		broker.Instance().EnableMockTrading(decimal.NewFromInt(*cfgMockAmt), decimal.NewFromFloat(*cfgMockFee))
	}

	chBrokerStarted, err := broker.Instance().Start()
	if err != nil {
		log.Fatalf("Failed to start the broker service. (Error: %s)", err)
	}

	//
	// Start up the Monitor Service.
	//
	monitor.Instance().SetAsset(*cfgAsset)
	chMonitorStarted, err := monitor.Instance().Start()
	if err != nil {
		log.Fatalf("Failed to start the match monitor service. (Error: %s)", err)
	}

	//
	// Wait for all services to finish starting up.
	//
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
