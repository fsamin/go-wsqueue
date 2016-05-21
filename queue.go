package wsqueue

import (
	"encoding/json"
	"errors"
	"sync"
	"time"
)

const (
	//MaxUint is the maximum uint on your platform
	maxUint = ^uint(0)
	//MinUint is the min uint on your platform
	minUint = 0
	//MaxInt is the max int on your platform
	maxInt = int(maxUint >> 1)
	//MinInt is the min int on your platform
	minInt = -maxInt - 1
)

//Queue implements load balancer semantics. A single message will be received by
// exactly one consumer. If there are no consumers available at the time the
// message is sent it will be kept until a consumer is available that can process
// the message. If a consumer receives a message and does not acknowledge it
// before closing then the message will be redelivered to another consumer.
// A queue can have many consumers with messages load balanced across the
// available consumers.
type Queue struct {
	Options               *Options `json:"options,omitempty"`
	Queue                 string   `json:"topic,omitempty"`
	newConsumerHandler    func(*Conn)
	consumerExitedHandler func(*Conn)
	ackHandler            func(*Conn, *Message) error
	mutex                 *sync.RWMutex
	wsConnections         map[ConnID]*Conn
	acks                  map[*Message]bool
	lb                    *loadBalancer
	store                 StorageDriver
	storeOptions          *StorageOptions
	stopQueue             chan bool
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
		acks:          make(map[*Message]bool),
	}
	q.lb = &loadBalancer{queue: q, counter: make(map[ConnID]int)}
	q.newConsumerHandler = newConsumerHandler(q)
	q.consumerExitedHandler = consumerExitedHandler(q)
	q.ackHandler = ackHandler(q)
	q.store = NewStack()
	q.stopQueue = make(chan bool, 1)
	return q, nil
}

//RegisterQueue register
func (s *Server) RegisterQueue(q *Queue) {
	Logfunc("Register queue %s on route %s", q.Queue, s.RoutePrefix+"/wsqueue/queue/"+q.Queue)
	handler := createHandler(
		q.mutex,
		&q.wsConnections,
		&q.newConsumerHandler,
		&q.consumerExitedHandler,
		&q.ackHandler,
	)
	q.store.Open(q.storeOptions)
	q.handle(5)
	s.Router.HandleFunc(s.RoutePrefix+"/wsqueue/queue/"+q.Queue, handler)
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

	var minCounter = maxInt
	for id := range lb.queue.wsConnections {
		counter := lb.counter[id]
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
	return nil, errors.New("No connection available")
}

//Send send a message
func (q *Queue) Send(data interface{}) error {
	m, e := newMessage(data)
	if e != nil {
		return e
	}
	if len(q.wsConnections) == 0 {
		q.store.Push(m)
		return nil
	}
	q.send(m)
	return nil
}

func (q *Queue) send(m *Message) {
	connID, err := q.lb.next()
	if err != nil {
		Logfunc("Error while sending to %s : %s", *connID, err.Error())
		q.store.Push(m)
	} else {
		conn := q.wsConnections[*connID]
		q.acks[m] = false
		b, _ := json.Marshal(m)
		q.mutex.Lock()
		err := conn.WSConn.WriteMessage(1, b)
		q.mutex.Unlock()
		if err != nil {
			Logfunc("Error while sending to %s : %s", *connID, err.Error())
			q.store.Push(m)
		}
	}
}

func (q *Queue) handle(interval int64) {
	var cont = true
	go func(c *bool) {
		for *c {
			time.Sleep(time.Duration(interval) * time.Second)
			data := q.store.Pop()
			if data != nil {
				m := data.(*Message)
				q.send(m)
			}
		}
	}(&cont)
	go func(c *bool) {
		var b bool
		//FIXME: test
		b = <-q.stopQueue
		cont = *c && b
	}(&cont)
}

func newConsumerHandler(q *Queue) func(*Conn) {
	return func(c *Conn) {
		q.mutex.Lock()
		q.lb.counter[c.ID] = 0
		//Reinitiatlisation du load balancer
		for id := range q.lb.counter {
			q.lb.counter[id] = 0
		}
		q.mutex.Unlock()
	}
}

func consumerExitedHandler(q *Queue) func(*Conn) {
	return func(c *Conn) {
		q.mutex.Lock()
		delete(q.lb.counter, c.ID)
		q.mutex.Unlock()
	}
}

func ackHandler(q *Queue) func(*Conn, *Message) error {
	return func(c *Conn, m *Message) error {
		q.mutex.Lock()
		q.acks[m] = true
		q.mutex.Unlock()
		return nil
	}
}
