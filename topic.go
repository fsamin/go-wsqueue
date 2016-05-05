package wsqueue

import (
	"encoding/json"
	"log"
	"sync"
)

//Topic implements publish and subscribe semantics. When you publish a message
//it goes to all the subscribers who are interested - so zero to many
//subscribers will receive a copy of the message. Only subscribers who had an
//active subscription at the time the broker receives the message will get a
//copy of the message.
type Topic struct {
	Options                  *Options                   `json:"options,omitempty"`
	Topic                    string                     `json:"topic,omitempty"`
	OpenedConnectionCallback func(*Conn)                `json:"-"`
	ClosedConnectionCallback func(*Conn)                `json:"-"`
	OnMessageCallback        func(*Conn, Message) error `json:"-"`
	mutex                    *sync.Mutex
	wsConnections            map[ConnID]*Conn
}

//CreateTopic create topic
func (s *Server) CreateTopic(topic string) *Topic {
	t, _ := s.newTopic(topic)
	s.RegisterTopic(t)
	return t
}

func (s *Server) newTopic(topic string) (*Topic, error) {
	t := &Topic{
		Topic:         topic,
		mutex:         &sync.Mutex{},
		wsConnections: make(map[ConnID]*Conn),
	}
	return t, nil
}

//RegisterTopic register
func (s *Server) RegisterTopic(t *Topic) {
	log.Printf("Register queue %s on route %s", t.Topic, s.RoutePrefix+"/wsqueue/topic/"+t.Topic)
	handler := createHandler(
		t.mutex,
		&t.wsConnections,
		&t.OpenedConnectionCallback,
		&t.ClosedConnectionCallback,
		&t.OnMessageCallback,
	)
	s.Router.HandleFunc(s.RoutePrefix+"/wsqueue/topic/"+t.Topic, handler)
}

func (t *Topic) publish(m Message) error {
	t.mutex.Lock()
	b, _ := json.Marshal(m)
	for _, conn := range t.wsConnections {
		conn.WSConn.WriteMessage(1, b)
	}
	t.mutex.Unlock()
	return nil
}

//Publish send message to everyone
func (t *Topic) Publish(data interface{}) error {
	m, e := newMessage(data)
	if e != nil {
		return e
	}
	return t.publish(*m)
}
