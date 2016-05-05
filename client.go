package wsqueue

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	Protocol string
	Host     string
	Route    string
	dialer   *websocket.Dialer
	conn     *websocket.Conn
}

type wsqueueType int

const (
	topic wsqueueType = iota
	queue
)

func (c *Client) connect(q string, t wsqueueType) error {
	var suffix string
	switch t {
	case topic:
		suffix = "topic"
	case queue:
		suffix = "queue"
	}

	var url = fmt.Sprintf("%s://%s%swsqueue/%s/%s", c.Protocol, c.Host, c.Route, suffix, q)
	c.dialer = websocket.DefaultDialer

	log.Println("Dialing " + url)
	var err error
	c.conn, _, err = c.dialer.Dial(url, http.Header{})
	return err
}

func (c *Client) Subscribe(q string) (chan Message, chan error, error) {
	e := c.connect(q, topic)
	if e != nil {
		return nil, nil, e
	}
	chanMessage := make(chan Message)
	chanError := make(chan error)
	go c.handler(q, chanMessage, chanError, topic)
	return chanMessage, chanError, nil
}

func (c *Client) Listen(q string) (chan Message, chan error, error) {
	e := c.connect(q, queue)
	if e != nil {
		return nil, nil, e
	}
	chanMessage := make(chan Message)
	chanError := make(chan error)
	go c.handler(q, chanMessage, chanError, queue)
	return chanMessage, chanError, nil
}

func (c *Client) handler(q string, chanMessage chan Message, chanError chan error, t wsqueueType) {
	for {
		_, p, e := c.conn.ReadMessage()
		if e != nil {
			if websocket.IsUnexpectedCloseError(e, websocket.CloseMessage) {
				chanError <- e
				time.Sleep(30 * time.Second)
				for err := c.connect(q, t); err != nil; c.connect(q, t) {
					chanError <- e
				}
			} else {
				close(chanMessage)
				break
			}
		} else {
			message := &Message{}
			if err := json.Unmarshal(p, message); err != nil {
				log.Println(err)
			}
			message.Header["received"] = time.Now().String()
			//Ack
			chanMessage <- *message
		}
	}
}
