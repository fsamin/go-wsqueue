# go-wsqueue

WIP

## Introduction

- Golang API over gorilla/mux and gorilla/websocket
- Like queue over websocket

## Getting started

`$ go get  github.com/fsamin/go-wsqueue`

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
	q := s.CreateQueue("topic1")

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
		cMessage, cError, err := c.OpenQueue("topic1")
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
