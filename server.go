package wsqueue

import (
	"encoding/json"
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
	Router      *mux.Router
	RoutePrefix string
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
	return s
}

func createHandler(
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
			for _, ace := range options.ACL {
				switch ace.Scheme() {
				case ACLSSchemeWorld:
					Logfunc("Connection Authorized")
				case ACLSSchemeIP:

				}
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
