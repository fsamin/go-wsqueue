package wsqueue

import (
	"encoding/json"
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

func (c *Client) connect(q string) error {
	c.dialer = websocket.DefaultDialer
	log.Println("Dialing " + c.Protocol + "://" + c.Host + c.Route + "wsqueue/" + q)

	var err error
	c.conn, _, err = c.dialer.Dial(c.Protocol+"://"+c.Host+c.Route+"wsqueue/"+q, http.Header{})
	return err
}

func (c *Client) OpenQueue(q string) (chan Message, chan error, error) {
	e := c.connect(q)
	if e != nil {
		return nil, nil, e
	}
	chanMessage := make(chan Message)
	chanError := make(chan error)
	go c.handler(q, chanMessage, chanError)
	return chanMessage, chanError, nil
}

func (c *Client) handler(q string, chanMessage chan Message, chanError chan error) {
	for {
		_, p, e := c.conn.ReadMessage()
		if e != nil {
			if websocket.IsUnexpectedCloseError(e, websocket.CloseMessage) {
				chanError <- e
				time.Sleep(30 * time.Second)
				for err := c.connect(q); err != nil; c.connect(q) {
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
			chanMessage <- *message
		}
	}
}
