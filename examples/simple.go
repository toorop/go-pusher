package main

// Display on console live Trades and Orders book from Bitstamp
// run with
// 		go run simple.go

import (
	"github.com/toorop/go-pusher"
	"log"
)

const (
	APP_KEY = "de504dc5763aeef9ff52" // bitstamp
)

func main() {
	pusherClient, err := pusher.NewClient(APP_KEY)
	if err != nil {
		log.Fatalln(err)
	}
	// Subscribe
	err = pusherClient.Subscribe("live_trades")
	if err != nil {
		log.Println("Subscription error : ", err)
	}

	err = pusherClient.Subscribe("order_book")
	if err != nil {
		log.Println("Subscription error : ", err)
	}

	// test
	err = pusherClient.Subscribe("order_book")
	if err != nil {
		log.Println("Subscription error : ", err)
	}

	// Bind events
	dataChannelTrade, err := pusherClient.Bind("data")
	tradeChannelTrade, err := pusherClient.Bind("trade")

	for {
		select {
		case dataEvt := <-dataChannelTrade:
			log.Println("ORDER BOOK: " + dataEvt.Data)
		case tradeEvt := <-tradeChannelTrade:
			log.Println("TRADE: " + tradeEvt.Data)
		}
	}
}
