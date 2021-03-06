package wsqueue

import (
	"encoding/json"
	"os"
	"reflect"
	"strconv"
	"time"

	"github.com/satori/go.uuid"
)

type Header map[string]string

//Message message
type Message struct {
	Header Header `json:"metadata,omitempty"`
	Body   string `json:"data"`
}

func newMessage(data interface{}) (*Message, error) {
	m := Message{
		Header: make(map[string]string),
		Body:   "",
	}

	switch data.(type) {
	case string, *string:
		m.Header["content-type"] = "string"
		m.Body = data.(string)
	case int, *int, int32, *int32, int64, *int64:
		m.Header["content-type"] = "int"
		m.Body = strconv.Itoa(data.(int))
	case bool, *bool:
		m.Header["content-type"] = "bool"
		m.Body = strconv.FormatBool(data.(bool))
	default:
		m.Header["content-type"] = "application/json"
		if reflect.TypeOf(data).Kind() == reflect.Ptr {
			m.Header["application-type"] = reflect.ValueOf(data).Elem().Type().String()
		} else {
			m.Header["application-type"] = reflect.ValueOf(data).Type().String()
		}
		b, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		m.Body = string(b)
	}
	m.Header["id"] = uuid.NewV1().String()
	m.Header["date"] = time.Now().String()
	m.Header["host"], _ = os.Hostname()
	return &m, nil
}

func (m *Message) String() string {
	var s string
	s = "\n---HEADER---"
	for k, v := range m.Header {
		s = s + "\n" + k + ":" + v
	}
	s = s + "\n---BODY---"
	s = s + "\n" + m.Body
	return s
}

//ID returns message if
func (m *Message) ID() string {
	return m.Header["id"]
}

//ContentType returns content-type
func (m *Message) ContentType() string {
	return m.Header["content-type"]
}

//ApplicationType returns application-type. Empty if content-type is not application/json
func (m *Message) ApplicationType() string {
	return m.Header["application-type"]
}
