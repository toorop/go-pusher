package pusher

import (
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"
)

type client struct {
	ws                 *websocket.Conn
	Events             chan *Event
	Stop               chan bool
	subscribedChannels []string
	binders            map[string]chan *Event
}

// NewClient initialize & return a Pusher client
func NewClient(appKey string) (*client, error) {
	origin := "http://localhost/"
	url := "wss://ws.pusherapp.com:443/app/" + appKey + "?protocol=" + PROTOCOL_VERSION
	ws, err := websocket.Dial(url, "", origin)
	if err != nil {
		return nil, err
	}
	var resp = make([]byte, 11000) // Pusher max message size is 10KB
	n, err := ws.Read(resp)
	if err != nil {
		return nil, err
	}
	var event Event
	err = json.Unmarshal(resp[0:n], &event)
	if err != nil {
		return nil, err
	}
	switch event.Event {
	case "pusher:error":
		var data eventError
		err = json.Unmarshal([]byte(event.Data), &data)
		if err != nil {
			return nil, err
		}
		err = errors.New(fmt.Sprintf("Pusher return error : code : %d, message %s", data.code, data.message))
		return nil, err
	case "pusher:connection_established":
		pClient := client{ws, make(chan *Event, EVENT_CHANNEL_BUFF_SIZE), make(chan bool), make([]string, 0), make(map[string]chan *Event)}
		go pClient.heartbeat()
		go pClient.listen()
		return &pClient, nil
	}
	return nil, errors.New("Ooooops something wrong happen")
}

// heartbeat send a ping frame to server each - TODO reconnect on disconnect
func (c *client) heartbeat() {
	for {
		websocket.Message.Send(c.ws, `{"event":"pusher:ping","data":"{}"}`)
		time.Sleep(HEARTBEAT_RATE * time.Second)
	}
}

// listen to Pusher server and process/dispatch recieved events
func (c *client) listen() {
	for {
		var event Event
		err := websocket.JSON.Receive(c.ws, &event)
		if err != nil {
			log.Println("Listen error : ", err)
		} else {
			//log.Println(event)
			switch event.Event {
			case "pusher:ping":
				websocket.Message.Send(c.ws, `{"event":"pusher:pong","data":"{}"}`)
			case "pusher:pong":
			case "pusher:error":
				log.Println("Event error recieved: ", event.Data)
			default:
				_, ok := c.binders[event.Event]
				if ok {
					c.binders[event.Event] <- &event
				}
			}
		}
	}
}

// Subsribe to a channel
func (c *client) Subscribe(channel2sub string) (err error) {
	//log.Println("Subscribing to channel " + channel2sub + "...")
	// Already subscribed ?
	if c.isSubscribedTo(channel2sub) {
		err = errors.New(fmt.Sprintf("Subscription to %s channel2sub already done.", channel2sub))
		return
	}
	err = websocket.Message.Send(c.ws, fmt.Sprintf(`{"event":"pusher:subscribe","data":{"channel":"%s"}}`, channel2sub))
	if err != nil {
		return
	}
	//ch = channel{channel2sub}
	c.subscribedChannels = append(c.subscribedChannels, channel2sub)
	//log.Println("Subscribed to channel " + channel2sub)
	return
}

// Bind an event
func (c *client) Bind(ev string) (dataChannel chan *Event, err error) {
	// Alerady binded
	_, ok := c.binders[ev]
	if ok {
		err = errors.New(fmt.Sprintf("Event %s already binded", ev))
		return
	}
	// New data channel
	dataChannel = make(chan *Event, EVENT_CHANNEL_BUFF_SIZE)
	c.binders[ev] = dataChannel
	return
}

// Helpers
// isSubscribedTo check if a channel is already subscribed
func (c *client) isSubscribedTo(channelName string) bool {
	for _, s := range c.subscribedChannels {
		if s == channelName {
			return true
		}
	}
	return false
}
