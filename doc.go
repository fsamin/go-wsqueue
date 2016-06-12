/*
Package wsqueue provides a framework over gorilla/mux and gorilla/websocket to operate
kind of AMQP but over websockets. It offers a Server, a Client and two king of messaging
protocols : Topics and Queues.

Topics : Publish & Subscribe Pattern

Publish and subscribe semantics are implemented by Topics. When you publish a message it
goes to all the subscribers who are interested - so zero to many subscribers will receive
a copy of the message. Only subscribers who had an active subscription at the time the broker
receives the message will get a copy of the message.

Start a server and handle a topic

    //Server side
    r := mux.NewRouter()
	s := wsqueue.NewServer(r, "")
	q := s.CreateTopic("myTopic")
    http.Handle("/", r)
	go http.ListenAndServe("0.0.0.0:9000", r)

    ...
    //Publish a message
    q.Publish("This is a message")

Start a client and listen on a topic

    //Client slide
    go func() {
		c := &wsqueue.Client{
			Protocol: "ws",
			Host:     "localhost:9000",
			Route:    "/",
		}
		cMessage, cError, err := c.Subscribe("myTopic")
		if err != nil {
			panic(err)
		}
		for {
			select {
			case m := <-cMessage:
				fmt.Println(m.String())
			case e := <-cError:
				fmt.Println(e.Error())
			}
		}
	}()


Queues : Work queues Pattern

The main idea behind Work Queues (aka: Task Queues) is to avoid doing a resource-intensive
task immediately and having to wait for it to complete. Instead we schedule the task to be
done later. We encapsulate a task as a message and send it to the queue. A worker process
running in the background will pop the tasks and eventually execute the job. When you run
many workers the tasks will be shared between them. Queues implement load balancer semantics.
A single message will be received by exactly one consumer. If there are no consumers available
at the time the message is sent it will be kept until a consumer is available that can process
the message. If a consumer receives a message and does not acknowledge it before closing then
the message will be redelivered to another consumer. A queue can have many consumers with messages
load balanced across the available consumers.

Examples

see samples/queue/main.go, samples/topic/main.go


*/
package wsqueue
