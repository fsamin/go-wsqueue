package wsqueue

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

//Client is the wqueue entrypoint
type Client struct {
	Protocol string
	Host     string
	Route    string
	dialer   *websocket.Dialer
	conn     *websocket.Conn
}

type wsqueueType string

const (
	topic wsqueueType = "topic"
	queue             = "queue"
)

//Subscribe aims to connect to a Topic
func (c *Client) Subscribe(q string) (chan Message, chan error, error) {
	Logfunc("Subcribing to Topic %s", q)
	chanMessage := make(chan Message)
	chanError := make(chan error)
	go c.handler(q, chanMessage, chanError, topic)
	return chanMessage, chanError, nil
}

//Listen aims to connect to a Queue
func (c *Client) Listen(q string) (chan Message, chan error, error) {
	Logfunc("Listening to Queue %s", q)
	chanMessage := make(chan Message)
	chanError := make(chan error)
	go c.handler(q, chanMessage, chanError, queue)
	return chanMessage, chanError, nil
}

func (c *Client) connect(q string, t wsqueueType) error {
	var url = fmt.Sprintf("%s://%s%swsqueue/%s/%s", c.Protocol, c.Host, c.Route, string(t), q)
	c.dialer = websocket.DefaultDialer
	c.dialer.HandshakeTimeout = 1 * time.Second

	Logfunc("Dialing %s", url)
	var err error
	c.conn, _, err = c.dialer.Dial(url, http.Header{})
	return err
}

func (c *Client) reconnect(q string, t wsqueueType, nbRetry int) error {
	var i = 0
	var f = NewFibonacci()
	for {
		if i != -1 && i > nbRetry {
			Logfunc("Unable to connect to %s : %s", string(t), q)
			return fmt.Errorf("Unable to connect to %s : %s", string(t), q)
		}
		if c.conn == nil {
			i++
			if err := c.connect(q, t); err != nil {
				Logfunc("Waiting before retry connection to %s : %s", string(t), q)
				f.WaitForIt(time.Second)
			}
		} else {
			return nil
		}
	}
}

func (c *Client) handler(q string, chanMessage chan Message, chanError chan error, t wsqueueType) {
	Logfunc("Handling message on %s : %s", string(t), q)
	for {
		if e := c.reconnect(q, t, 100); e != nil {
			chanError <- e
			close(chanError)
			close(chanMessage)
			break
		}
		_, p, e := c.conn.ReadMessage()
		if e != nil {
			c.conn = nil
			chanError <- e
			if !websocket.IsUnexpectedCloseError(e, websocket.CloseMessage) {
				close(chanMessage)
				close(chanMessage)
				break
			}
		} else {
			message := &Message{}
			if err := json.Unmarshal(p, message); err != nil {
				log.Println(err)
			}
			message.Header["received"] = time.Now().String()
			//FIXME: Ack
			chanMessage <- *message
		}
	}
}
