package wsqueue

import (
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"sync"
	"time"
)

const (
	MaxUint = ^uint(0)
	MinUint = 0
	MaxInt  = int(MaxUint >> 1)
	MinInt  = -MaxInt - 1
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
	ackCallback            func(*Conn, *Message) error
	mutex                  *sync.RWMutex
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
		mutex:         &sync.RWMutex{},
		wsConnections: make(map[ConnID]*Conn),
		chanMessage:   make(chan *Message, bufferSize),
		acks:          make(map[*Message]bool),
	}
	q.lb = &loadBalancer{queue: q, counter: make(map[ConnID]int)}
	q.newConsumerCallback = newConsumerCallback(q)
	q.consumerExitedCallback = consumerExitedCallback(q)
	q.ackCallback = ackCallback(q)
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

func (lb *loadBalancer) next() (*ConnID, error) {
	lb.queue.mutex.Lock()
	defer lb.queue.mutex.Unlock()

	if len(lb.queue.wsConnections) == 0 {
		return nil, errors.New("No connection available")
	}

	var minCounter = MaxInt
	for id := range lb.queue.wsConnections {
		counter := lb.counter[id]
		log.Println("Nb message sent to " + string(id) + " : " + strconv.Itoa(counter) + ", " + strconv.Itoa(minCounter))
		if counter < minCounter {
			minCounter = counter
		}
	}

	for id := range lb.queue.wsConnections {
		c := lb.counter[id]
		if c == minCounter {
			c++
			lb.counter[id] = c
			return &id, nil
		}
	}
	log.Println("Any solution")
	return nil, errors.New("No connection available")
}

//Send send a message
func (q *Queue) Send(data interface{}) error {
	log.Println("new message in mailbox")
	m, e := newMessage(data)
	if e != nil {
		return e
	}
	if len(q.wsConnections) == 0 {
		log.Println("no consumer, i will wait")
		q.chanMessage <- m
		return nil
	}
	q.send(m)
	return nil
}

func (q *Queue) send(m *Message) {
	connID, err := q.lb.next()
	if err != nil {
		q.chanMessage <- m
	} else {
		conn := q.wsConnections[*connID]
		q.acks[m] = false
		b, _ := json.Marshal(m)
		q.mutex.Lock()
		err := conn.WSConn.WriteMessage(1, b)
		q.mutex.Unlock()
		if err != nil {
			log.Println("Error while sending to "+*connID, err.Error())
			q.chanMessage <- m
		}
	}
}

func (q *Queue) retry(interval int64) {
	for m := range q.chanMessage {
		time.Sleep(time.Duration(interval) * time.Second)
		q.send(m)
	}
}

func newConsumerCallback(q *Queue) func(*Conn) {
	return func(c *Conn) {
		log.Println("New consumer : " + c.ID)
		q.mutex.Lock()
		q.lb.counter[c.ID] = 0
		for id := range q.lb.counter {
			q.lb.counter[id] = 0
		}
		q.mutex.Unlock()
	}
}

func consumerExitedCallback(q *Queue) func(*Conn) {
	return func(c *Conn) {
		log.Println("Lost consumer : " + c.ID)
		q.mutex.Lock()
		delete(q.lb.counter, c.ID)
		q.mutex.Unlock()
	}
}

func ackCallback(q *Queue) func(*Conn, *Message) error {
	return func(c *Conn, m *Message) error {
		q.mutex.Lock()
		q.acks[m] = true
		q.mutex.Unlock()
		return nil
	}
}
