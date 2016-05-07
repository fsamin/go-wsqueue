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
}

type Options struct {
	ACL *QueueACL `json:"acl,omitempty"`
}

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
func NewServer(router *mux.Router, routePrefix, username, password string) *Server {
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
) func(
	w http.ResponseWriter,
	r *http.Request,
) {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Print("upgrade:", err)
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
				delete(*wsConnections, conn.ID)
				if (*closedConnectionCallback) != nil {
					go (*closedConnectionCallback)(conn)
				}
				break
			}

			if (*onMessageCallback) != nil {
				var parsedMessage Message
				if e := json.Unmarshal(message, &parsedMessage); err != nil {
					log.Println(e.Error())
				}
				go (*onMessageCallback)(conn, &parsedMessage)
			}
		}

	}
}
