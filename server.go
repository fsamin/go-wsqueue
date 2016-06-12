package wsqueue

import (
	"encoding/json"
	"expvar"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"github.com/satori/go.uuid"
)

//Logfunc is a function that logs the provided message with optional
//fmt.Sprintf-style arguments. By default, logs to the default log.Logger.
//setting it to nil can be used to disable logging for this package.
//This doesn’t enforce a coupling with any specific external package
//and is already widely supported by existing loggers.
var Logfunc = log.Printf

//Warnfunc is a function that logs the provided message with optional
//fmt.Sprintf-style arguments. By default, logs to the default log.Logger.
//setting it to nil can be used to disable logging for this package.
//This doesn’t enforce a coupling with any specific external package
//and is already widely supported by existing loggers.
var Warnfunc = log.Printf

//Server is a server
type Server struct {
	Router          *mux.Router
	RoutePrefix     string
	QueuesCounter   *expvar.Int
	TopicsCounter   *expvar.Int
	ClientsCounter  *expvar.Int
	MessagesCounter *expvar.Int
}

//StorageDriver is in-memory Stack or Redis server
type StorageDriver interface {
	Open(options *Options)
	Push(data interface{})
	Pop() interface{}
}

//Options is options on topic or queues
type Options struct {
	ACL     ACL            `json:"acl,omitempty"`
	Storage StorageOptions `json:"storage,omitempty"`
}

//StorageOptions is a collection of options, see storage documentation
type StorageOptions map[string]interface{}

//ConnID a a connection ID
type ConnID string

//Conn is a conn
type Conn struct {
	ID     ConnID
	WSConn *websocket.Conn
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

//NewServer init a new WSQueue server
func NewServer(router *mux.Router, routePrefix string) *Server {
	s := &Server{
		Router:      router,
		RoutePrefix: routePrefix,
	}
	router.HandleFunc(routePrefix+"/vars", varsHandler)
	if routePrefix != "" {
		routePrefix = "." + routePrefix
	}
	s.QueuesCounter = expvar.NewInt("wsqueue" + routePrefix + ".stats.queues.counter")
	s.TopicsCounter = expvar.NewInt("wsqueue" + routePrefix + ".stats.topics.counter")
	s.ClientsCounter = expvar.NewInt("wsqueue" + routePrefix + ".stats.clients.counter")
	s.MessagesCounter = expvar.NewInt("wsqueue" + routePrefix + ".stats.messages.counter")

	return s
}
func (s *Server) createHandler(
	mutex *sync.RWMutex,
	wsConnections *map[ConnID]*Conn,
	openedConnectionCallback *func(*Conn),
	closedConnectionCallback *func(*Conn),
	onMessageCallback *func(*Conn, *Message) error,
	options *Options,
) func(
	w http.ResponseWriter,
	r *http.Request,
) {
	return func(w http.ResponseWriter, r *http.Request) {

		if options != nil && len(options.ACL) > 0 {
			if !checkACL(options.ACL, w, r) {
				Warnfunc("Not Authorized by ACL")
				w.Write([]byte("Not Authorized by ACL"))
				return
			}
		}

		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			Warnfunc("Cannot upgrade connection %s", err.Error())
			w.Write([]byte(fmt.Sprintf("Cannot upgrade connection %s", err.Error())))
			w.WriteHeader(426)
			return
		}

		mutex.Lock()
		conn := &Conn{
			ID:     ConnID(uuid.NewV4().String()),
			WSConn: c,
		}
		if (*wsConnections)[conn.ID] != nil {
			(*wsConnections)[conn.ID].WSConn.Close()
			(*closedConnectionCallback)((*wsConnections)[conn.ID])
		}
		(*wsConnections)[conn.ID] = conn
		mutex.Unlock()

		if (*openedConnectionCallback) != nil {
			go (*openedConnectionCallback)(conn)
		}

		s.ClientsCounter.Add(1)

		defer c.Close()
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				mutex.Lock()
				delete(*wsConnections, conn.ID)
				mutex.Unlock()
				if (*closedConnectionCallback) != nil {
					(*closedConnectionCallback)(conn)
				}
				s.ClientsCounter.Add(-1)
				break
			}

			if (*onMessageCallback) != nil {
				var parsedMessage Message
				if e := json.Unmarshal(message, &parsedMessage); err != nil {
					Warnfunc("Cannot Unmarshall message", e.Error())
				}
				(*onMessageCallback)(conn, &parsedMessage)
			}
		}

	}
}

func varsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintf(w, "{\n")
	first := true
	expvar.Do(func(kv expvar.KeyValue) {
		if !first {
			fmt.Fprintf(w, ",\n")
		}
		first = false
		fmt.Fprintf(w, "%q: %s", kv.Key, kv.Value)
	})
	fmt.Fprintf(w, "\n}\n")
}
