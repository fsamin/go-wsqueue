package wsqueue

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"github.com/satori/go.uuid"
)

//Server is a server
type Server struct {
	Router      *mux.Router
	RoutePrefix string
	AdminQueue  *Queue
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

//NewServer init a new WSQueue server
func NewServer(router *mux.Router, routePrefix, username, password string) *Server {
	s := &Server{
		Router:      router,
		RoutePrefix: routePrefix,
	}
	adminQueueOptions := &QueueOptions{
	/*ACL: &QueueACL{
		[]QueueACE{
			QueueACEDigest{QueueACE{QUEUE_ACL_SCHEME_DIGEST, QUEUE_ACL_PERM_ADMIN}, username, password},
		},
	},*/
	}
	adminQueue := createAdminQueue(s, adminQueueOptions)
	s.RegisterQueue(adminQueue)
	return s
}

func (s *Server) NewQueue(topic string) (*Queue, error) {
	queue := &Queue{
		Topic: topic,
	}
	return queue, nil
}

func (s *Server) RegisterQueue(q *Queue) {
	log.Printf("Register queue %s on route %s", q.Topic, s.RoutePrefix+"/wsqueue/"+q.Topic)

	q.mutex = &sync.Mutex{}
	q.wsConnections = make(map[string][]*Conn)

	handler := createHandler(q)
	s.Router.HandleFunc(s.RoutePrefix+"/wsqueue/"+q.Topic, handler)
}

//CreateQueue create queue
func (s *Server) CreateQueue(topic string) *Queue {
	queue, _ := s.NewQueue(topic)
	s.RegisterQueue(queue)
	return queue
}

func createHandler(q *Queue) func(w http.ResponseWriter, r *http.Request) {
	log.Printf("Creating handler for %s", q.Topic)

	return func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Print("upgrade:", err)
			return
		}

		q.mutex.Lock()
		conn := &Conn{
			ID:     uuid.NewV1().String(),
			WSConn: c,
		}
		if q.wsConnections[conn.ID] == nil {
			q.wsConnections[conn.ID] = []*Conn{conn}
		} else {
			q.wsConnections[conn.ID] = append(q.wsConnections[conn.ID], conn)
		}
		q.mutex.Unlock()

		if q.OpenedConnectionCallback != nil {
			go q.OpenedConnectionCallback(conn)
		}

		defer c.Close()
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				if q.ClosedConnectionCallback != nil {
					go q.ClosedConnectionCallback(conn)
				}
				break
			}

			if q.OnMessageCallback != nil {
				var parsedMessage Message
				if e := json.Unmarshal(message, &parsedMessage); err != nil {
					log.Println(e.Error())
				}
				go q.OnMessageCallback(conn, parsedMessage)
			}
		}

	}
}
