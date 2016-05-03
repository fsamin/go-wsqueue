package wsqueue

import (
	"encoding/json"
	"os"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

//Queue is a queue
type Queue struct {
	Options                  *QueueOptions              `json:"options,omitempty"`
	Topic                    string                     `json:"topic,omitempty"`
	OpenedConnectionCallback func(*Conn)                `json:"-"`
	ClosedConnectionCallback func(*Conn)                `json:"-"`
	OnMessageCallback        func(*Conn, Message) error `json:"-"`
	mutex                    *sync.Mutex
	wsConnections            map[string][]*Conn
}

type QueueOptions struct {
	ACL *QueueACL `json:"acl,omitempty"`
}

//Conn is a conn
type Conn struct {
	ID     string
	WSConn *websocket.Conn
}

type Message struct {
	Header map[string]string `json:"metadata,omitempty"`
	Body   string            `json:"data"`
}

func (m *Message) String() string {
	var s string
	s = "\n---HEADER---"
	for k, v := range m.Header {
		s = s + "\n" + k + ":" + v
	}
	s = s + "\n---BODY---"
	s = s + "\n" + m.Body
	return s
}

func (q *Queue) publish(m Message) error {
	q.mutex.Lock()
	b, _ := json.Marshal(m)
	for _, conns := range q.wsConnections {
		for _, conn := range conns {
			conn.WSConn.WriteMessage(1, b)
		}
	}
	q.mutex.Unlock()
	return nil
}

//Publish send message to everyone
func (q *Queue) Publish(data interface{}) error {
	m := Message{
		Header: make(map[string]string),
		Body:   "",
	}

	switch data.(type) {
	case string, *string:
		m.Header["content-type"] = "string"
		m.Body = data.(string)
	case int, *int, int32, *int32, int64, *int64:
		m.Header["content-type"] = "int"
		m.Body = strconv.Itoa(data.(int))
	case bool, *bool:
		m.Header["content-type"] = "bool"
		m.Body = strconv.FormatBool(data.(bool))
	default:
		m.Header["content-type"] = "application/json"
		if reflect.TypeOf(data).Kind() == reflect.Ptr {
			m.Header["application-type"] = reflect.ValueOf(data).Elem().Type().String()
		} else {
			m.Header["application-type"] = reflect.ValueOf(data).Type().String()
		}
		b, err := json.Marshal(data)
		if err != nil {
			return err
		}
		m.Body = string(b)
	}
	m.Header["date"] = time.Now().String()
	m.Header["host"], _ = os.Hostname()
	return q.publish(m)
}
