package pusher

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

// ErrEvent error on event
const ErrEvent = "ErrEvent"

// Client is a pusher client
type Client struct {
	ws                 *websocket.Conn
	Events             chan *Event
	Stop               chan bool
	Errors             chan error
	subscribedChannels *subscribedChannels
	binders            map[string]chan *Event
	m                  sync.Mutex
}

func (c *Client) sendError(err error) {
	select {
	case c.Errors <- err:
	default:
		log.Println(err)
	}
}

// heartbeat send a ping frame to server each - TODO reconnect on disconnect
func (c *Client) heartbeat() {
	for !c.Stopped() {
		if err := websocket.Message.Send(c.ws, `{"event":"pusher:ping","data":"{}"}`); err != nil {
			c.sendError(err)
		}
		time.Sleep(HEARTBEAT_RATE * time.Second)
	}
}

// listen to Pusher server and process/dispatch recieved events
func (c *Client) listen() {
	for !c.Stopped() {
		var event Event
		err := websocket.JSON.Receive(c.ws, &event)
		if err != nil {
			if c.Stopped() {
				// Normal termination (ws Receive returns error when ws is
				// closed by other goroutine)
				c.sendError(fmt.Errorf("Listen error and c stopped : %s", err))
				return
			}
			c.m.Lock()
			ch, ok := c.binders[ErrEvent]
			c.m.Unlock()
			if ok {
				ch <- &Event{
					Event: ErrEvent,
					Data:  err.Error()}
			}
			// no matter what error happened, will log again and again
			// so return
			// err eg:
			// 1. use of closed network connection
			// 2. EOF (network error)
			// 3. ...
			c.sendError(fmt.Errorf("Listen error : %s", err))
			return
		}
		switch event.Event {
		case "pusher:ping":
			websocket.Message.Send(c.ws, `{"event":"pusher:pong","data":"{}"}`)
		case "pusher:pong":
		case "pusher:error":
			c.sendError(fmt.Errorf("Event error received : %s", event.Data))
		default:
			c.m.Lock()
			ch, ok := c.binders[event.Event]
			c.m.Unlock()
			if ok {
				ch <- &event
			}
		}

	}
}

// Subscribe ..to a channel
func (c *Client) Subscribe(channel string) (err error) {
	// Already subscribed ?
	if c.subscribedChannels.contains(channel) {
		err = fmt.Errorf("Channel %s already subscribed", channel)
		return
	}
	err = websocket.Message.Send(c.ws, fmt.Sprintf(`{"event":"pusher:subscribe","data":{"channel":"%s"}}`, channel))
	if err != nil {
		return
	}
	err = c.subscribedChannels.add(channel)
	return
}

// Unsubscribe from a channel
func (c *Client) Unsubscribe(channel string) (err error) {
	// subscribed ?
	if !c.subscribedChannels.contains(channel) {
		err = fmt.Errorf("Client isn't subscrived to %s", channel)
		return
	}
	err = websocket.Message.Send(c.ws, fmt.Sprintf(`{"event":"pusher:unsubscribe","data":{"channel":"%s"}}`, channel))
	if err != nil {
		return
	}
	// Remove channel from subscribedChannels slice
	c.subscribedChannels.remove(channel)
	return
}

// Bind an event
func (c *Client) Bind(evt string) (dataChannel chan *Event, err error) {
	// Already binded
	c.m.Lock()
	_, ok := c.binders[evt]
	c.m.Unlock()
	if ok {
		err = fmt.Errorf("Event %s already binded", evt)
		return
	}
	// New data channel
	dataChannel = make(chan *Event, EVENT_CHANNEL_BUFF_SIZE)
	c.m.Lock()
	c.binders[evt] = dataChannel
	c.m.Unlock()
	return
}

// Unbind a event
func (c *Client) Unbind(evt string) {
	c.m.Lock()
	delete(c.binders, evt)
	c.m.Unlock()
}

// Stopped checks, in a non-blocking way, if client has been closed.
func (c *Client) Stopped() bool {
	select {
	case <-c.Stop:
		return true
	default:
		return false
	}
}

// Close the underlying Pusher connection (websocket)
func (c *Client) Close() error {
	// Closing the Stop channel "broadcasts" the stop signal.
	close(c.Stop)
	return c.ws.Close()
}

// NewCustomClient return a custom client
func NewCustomClient(appKey, host, scheme string) (*Client, error) {
	ws, err := NewWSS(appKey, host, scheme)
	if err != nil {
		return nil, err
	}
	sChannels := new(subscribedChannels)
	sChannels.channels = make([]string, 0)
	pClient := Client{
		ws:                 ws,
		Events:             make(chan *Event, EVENT_CHANNEL_BUFF_SIZE),
		Stop:               make(chan bool),
		Errors:             make(chan error),
		subscribedChannels: sChannels,
		binders:            make(map[string]chan *Event)}
	go pClient.heartbeat()
	go pClient.listen()
	return &pClient, nil
}

// NewWSS return a websocket connexion
func NewWSS(appKey, host, scheme string) (ws *websocket.Conn, err error) {
	origin := "http://localhost/"
	url := scheme + "://" + host + "/app/" + appKey + "?protocol=" + PROTOCOL_VERSION
	ws, err = websocket.Dial(url, "", origin)
	if err != nil {
		return nil, err
	}
	var resp = make([]byte, 11000) // Pusher max message size is 10KB
	n, err := ws.Read(resp)
	if err != nil {
		return nil, err
	}
	var eventStub EventStub
	err = json.Unmarshal(resp[0:n], &eventStub)
	if err != nil {
		return nil, err
	}
	switch eventStub.Event {
	case "pusher:error":
		var ewe EventError
		err = json.Unmarshal(resp[0:n], &ewe)
		if err != nil {
			return nil, err
		}
		return nil, ewe
	case "pusher:connection_established":
		return ws, nil
	}
	return nil, errors.New("Ooooops something wrong happen")
}

// NewClient initialize & return a Pusher client
func NewClient(appKey string) (*Client, error) {
	return NewCustomClient(appKey, "ws.pusherapp.com:443", "wss")
}
