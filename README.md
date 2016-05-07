# go-wsqueue

WIP

## Introduction

- Golang API over gorilla/mux and gorilla/websocket
- Kink of AMQP but over websockets

### Publish & Subscribe Pattern

Publish and subscribe semantics are implemented by Topics.

When you publish a message it goes to all the subscribers who are interested - so zero to many subscribers will receive a copy of the message. Only subscribers who had an active subscription at the time the broker receives the message will get a copy of the message.

### Work queues Pattern

The main idea behind Work Queues (aka: Task Queues) is to avoid doing a resource-intensive task immediately and having to wait for it to complete. Instead we schedule the task to be done later. We encapsulate a task as a message and send it to the queue. A worker process running in the background will pop the tasks and eventually execute the job. When you run many workers the tasks will be shared between them.

Queues implement load balancer semantics. A single message will be received by exactly one consumer. If there are no consumers available at the time the message is sent it will be kept until a consumer is available that can process the message. If a consumer receives a message and does not acknowledge it before closing then the message will be redelivered to another consumer. A queue can have many consumers with messages load balanced across the available consumers.

## Getting started

`$ go get  github.com/fsamin/go-wsqueue`

### Publish & Subscribe

Run this with 3 terminals :

`go run main.go -server`

`go run main.go -client1`

`go run main.go -client2`


main.go :

```
package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/fsamin/go-wsqueue"
	"github.com/gorilla/mux"

	"github.com/jmcvetta/randutil"
)

var fServer = flag.Bool("server", false, "Run server")
var fClient1 = flag.Bool("client1", false, "Run client #1")
var fClient2 = flag.Bool("client2", false, "Run client #2")

func main() {
	flag.Parse()
	forever := make(chan bool)

	if *fServer {
		server()
	}
	if *fClient1 {
		client("1")
	}
	if *fClient2 {
		client("2")
	}

	<-forever
}

func server() {
	r := mux.NewRouter()
	s := wsqueue.NewServer(r, "", "", "")
	q := s.CreateTopic("topic1")

	q.OpenedConnectionCallback = func(c *wsqueue.Conn) {
		log.Println("Welcome " + c.ID)
		q.Publish("Welcome " + c.ID)
	}

	q.ClosedConnectionCallback = func(c *wsqueue.Conn) {
		log.Println("Bye bye " + c.ID)
	}
	http.Handle("/", r)
	go http.ListenAndServe("0.0.0.0:9000", r)

	//Start send message to queue
	go func() {
		for {
			time.Sleep(5 * time.Second)
			s, _ := randutil.AlphaString(10)
			q.Publish("message from goroutine 1 : " + s)
		}
	}()

	go func() {
		for {
			time.Sleep(10 * time.Second)
			s, _ := randutil.AlphaString(10)
			q.Publish("message from goroutine 2 : " + s)
		}
	}()
}

func client(ID string) {
	//Connect a client
	go func() {
		c := &wsqueue.Client{
			Protocol: "ws",
			Host:     "localhost:9000",
			Route:    "/",
		}
		cMessage, cError, err := c.Subscribe("topic1")
		if err != nil {
			panic(err)
		}
		for {
			select {
			case m := <-cMessage:
				log.Println("\n\n********* Client " + ID + " *********" + m.String() + "\n******************")
			case e := <-cError:
				log.Println("\n\n********* Client " + ID + "  *********" + e.Error() + "\n******************")
			}
		}
	}()
}

```

### Load balanced Work Queues

Run this with 3 terminals :

`go run main.go -server`

`go run main.go -client1`

`go run main.go -client2`


main.go :

```
package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/fsamin/go-wsqueue"
	"github.com/gorilla/mux"

	"github.com/jmcvetta/randutil"
)

var fServer = flag.Bool("server", false, "Run server")
var fClient1 = flag.Bool("client1", false, "Run client #1")
var fClient2 = flag.Bool("client2", false, "Run client #2")

func main() {
	flag.Parse()
	forever := make(chan bool)

	if *fServer {
		server()
	}
	if *fClient1 {
		client("1")
	}
	if *fClient2 {
		client("2")
	}

	<-forever
}

func server() {
	r := mux.NewRouter()
	s := wsqueue.NewServer(r, "", "", "")
	q := s.CreateQueue("queue1", 10)

	http.Handle("/", r)
	go http.ListenAndServe("0.0.0.0:9000", r)

	//Start send message to queue
	go func() {
		for {
			time.Sleep(5 * time.Second)
			s, _ := randutil.AlphaString(10)
			q.Send("message from goroutine 1 : " + s)
		}
	}()

	go func() {
		for {
			time.Sleep(6 * time.Second)
			s, _ := randutil.AlphaString(10)
			q.Send("message from goroutine 2 : " + s)
		}
	}()
}

func client(ID string) {
	//Connect a client
	go func() {
		c := &wsqueue.Client{
			Protocol: "ws",
			Host:     "localhost:9000",
			Route:    "/",
		}
		cMessage, cError, err := c.Listen("queue1")
		if err != nil {
			panic(err)
		}
		for {
			select {
			case m := <-cMessage:
				log.Println("\n\n********* Client " + ID + " *********" + m.String() + "\n******************")
			case e := <-cError:
				log.Println("\n\n********* Client " + ID + "  *********" + e.Error() + "\n******************")
			}
		}
	}()
}

```
