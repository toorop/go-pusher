package main

// Display on console live Trades and Orders book from Bitstamp
// run with
// 		go run client.go

import (
	"go-pusher"
	"log"
	"time"
)

const (
	APP_KEY = "de504dc5763aeef9ff52" // bitstamp
)

func main() {

	INIT:
	log.Println("init...")
	pusherClient, err := pusher.NewClient(APP_KEY)
	// if you need to connect to custom endpoint
	//pusherClient, err := pusher.NewCustomClient(APP_KEY, "localhost:8080", "ws")
	if err != nil {
		log.Fatalln(err)
	}
	// Subscribe
	err = pusherClient.Subscribe("live_trades")
	if err != nil {
		log.Println("Subscription error : ", err)
	}

	log.Println("first subscribe done")

	err = pusherClient.Subscribe("order_book")
	if err != nil {
		log.Println("Subscription error : ", err)
	}

	// test subcride to and already subscribed channel
	err = pusherClient.Subscribe("order_book")
	if err != nil {
		log.Println("Subscription error : ", err)
	}

	err = pusherClient.Subscribe("foo")
	if err != nil {
		log.Println("Subscription error : ", err)
	}
	log.Println("Subscribed to foo")

	err = pusherClient.Unsubscribe("foo")
	if err != nil {
		log.Println("Unsubscription error : ", err)
	}
	log.Println("Unsubscibed from foo")

	// Bind events
	dataChannelTrade, err := pusherClient.Bind("data")
	if err != nil {
		log.Println("Bind error: ", err)
	}
	log.Println("Binded to 'data' event")
	tradeChannelTrade, err := pusherClient.Bind("trade")
	if err != nil {
		log.Println("Bind error: ", err)
	}
	log.Println("Binded to 'trade' event")

	// Test bind/unbind
	_, err = pusherClient.Bind("foo")
	if err != nil {
		log.Println("Bind error: ", err)
	}
	pusherClient.Unbind("foo")

	// Test bind err
	errChannel, err := pusherClient.Bind(pusher.ErrEvent)
	if err != nil {
		log.Println("Bind error: ", err)
	}
	log.Println("Binded to 'ErrEvent' event")

	log.Println("init done")

	for {
		select {
		case dataEvt := <-dataChannelTrade:
			log.Println("ORDER BOOK: " + dataEvt.Data)
		case tradeEvt := <-tradeChannelTrade:
			log.Println("TRADE: " + tradeEvt.Data)
		case errEvt := <-errChannel:
			log.Println("ErrEvent: " + errEvt.Data)
			pusherClient.Close()
			time.Sleep(time.Second)
			goto INIT
		}
	}
}
