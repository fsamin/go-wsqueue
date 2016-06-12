package wsqueue

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewMessageShouldBeString(t *testing.T) {
	data := "test message"
	msg, err := newMessage(data)
	assert.NoError(t, err)
	assert.NotNil(t, msg)
	assert.Equal(t, msg.ContentType(), "string")
}

func TestNewMessageShouldBeJSON(t *testing.T) {
	data := map[string]interface{}{
		"data": "test message",
	}
	msg, err := newMessage(data)
	assert.NoError(t, err)
	assert.NotNil(t, msg)
	assert.Equal(t, msg.ContentType(), "application/json")
	assert.Equal(t, msg.ApplicationType(), "map[string]interface {}")
}

func TestNewMessageShouldBeJSONEvenIfPointer(t *testing.T) {
	data := map[string]interface{}{
		"data": "test message",
	}
	msg, err := newMessage(&data)
	assert.NoError(t, err)
	assert.NotNil(t, msg)
	assert.Equal(t, msg.ContentType(), "application/json")
	assert.Equal(t, msg.ApplicationType(), "map[string]interface {}")
}
