package wsqueue

import (
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"sync"
	"time"
)

//Queue implements load balancer semantics. A single message will be received by
// exactly one consumer. If there are no consumers available at the time the
// message is sent it will be kept until a consumer is available that can process
// the message. If a consumer receives a message and does not acknowledge it
// before closing then the message will be redelivered to another consumer.
// A queue can have many consumers with messages load balanced across the
// available consumers.
type Queue struct {
	Options                *Options `json:"options,omitempty"`
	Queue                  string   `json:"topic,omitempty"`
	newConsumerCallback    func(*Conn)
	consumerExitedCallback func(*Conn)
	ackCallback            func(*Conn, Message) error
	mutex                  *sync.Mutex
	wsConnections          map[ConnID]*Conn
	chanMessage            chan *Message
	acks                   map[*Message]bool
	lb                     *loadBalancer
}

//CreateQueue create queue
func (s *Server) CreateQueue(name string, bufferSize int) *Queue {
	q, _ := s.newQueue(name, bufferSize)
	s.RegisterQueue(q)
	return q
}

func (s *Server) newQueue(name string, bufferSize int) (*Queue, error) {
	q := &Queue{
		Queue:         name,
		mutex:         &sync.Mutex{},
		wsConnections: make(map[ConnID]*Conn),
		chanMessage:   make(chan *Message, bufferSize),
		acks:          make(map[*Message]bool),
	}
	q.lb = &loadBalancer{queue: q, counter: make(map[ConnID]int)}
	return q, nil
}

//RegisterQueue register
func (s *Server) RegisterQueue(q *Queue) {
	log.Printf("Register queue %s on route %s", q.Queue, s.RoutePrefix+"/wsqueue/queue/"+q.Queue)
	handler := createHandler(
		q.mutex,
		&q.wsConnections,
		&q.newConsumerCallback,
		&q.consumerExitedCallback,
		&q.ackCallback,
	)
	s.Router.HandleFunc(s.RoutePrefix+"/wsqueue/queue/"+q.Queue, handler)
	go q.retry(5)
}

type loadBalancer struct {
	queue   *Queue
	counter map[ConnID]int
}

func (lb *loadBalancer) next() (ConnID, error) {
	lb.queue.mutex.Lock()
	defer lb.queue.mutex.Unlock()

	if len(lb.queue.wsConnections) == 0 {
		log.Println("no connections...")
		return ConnID(""), errors.New("No connection available")
	}

	var minCounter = -1
	for id := range lb.queue.wsConnections {
		c := lb.counter[id]
		log.Println("Nb message sent to " + string(id) + " : " + strconv.Itoa(c))
		if minCounter < c && len(lb.queue.wsConnections) > 1 {
			log.Println("don't send to " + string(id) + "(" + strconv.Itoa(c) + ", " + strconv.Itoa(minCounter))
			minCounter = c
		} else {
			c++
			lb.counter[id] = c
			return id, nil
		}
	}
	return ConnID(""), errors.New("No connection available")
}

//Send send a message
func (q *Queue) Send(data interface{}) error {
	m, e := newMessage(data)
	if e != nil {
		return e
	}
	if len(q.wsConnections) == 0 {
		q.chanMessage <- m
		return nil
	}
	return q.send(m)
}

func (q *Queue) send(m *Message) error {
	connID, err := q.lb.next()
	if err != nil {
		q.chanMessage <- m
	} else {
		log.Println("Send to connID : " + connID)
		conn := q.wsConnections[connID]
		q.acks[m] = false
		b, _ := json.Marshal(m)
		q.mutex.Lock()
		conn.WSConn.WriteMessage(1, b)
		q.mutex.Unlock()
	}
	return nil
}

func (q *Queue) retry(interval int64) {
	for m := range q.chanMessage {
		time.Sleep(time.Duration(interval) * time.Second)
		q.send(m)
	}
}
