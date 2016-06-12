package wsqueue

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/phayes/freeport"
	"github.com/stretchr/testify/assert"
)

func TestCheckACLShouldAuthorizeEveryoneWhenACEIsSetToWord(t *testing.T) {
	var port = freeport.GetPort()
	wait := make(chan bool, 1)

	//setup ACL
	acl := ACL{
		&ACEWorld{},
	}

	//setup server
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.True(t, checkACL(acl, w, r), "check should return true")
		wait <- true
	}
	http.HandleFunc(fmt.Sprintf("/%d", port), handler)
	//Run the server
	t.Logf("Starting test server on port %d", port)
	go http.ListenAndServe(fmt.Sprintf(":%d", port), nil)

	//Run the test
	t.Logf("Calling the test server")
	client := http.DefaultClient
	res, err := client.Get(fmt.Sprintf("http://localhost:%d/%d", port, port))
	assert.NoError(t, err)
	assert.Equal(t, 200, res.StatusCode, "status code should be 200")

	<-wait
}

func TestCheckACLShouldAuthorizeLocalIPWhenACEIsSetLocalIP(t *testing.T) {
	var port = freeport.GetPort()
	wait := make(chan bool, 1)

	//setup ACL
	acl := ACL{
		&ACEIP{"localhost0"},
	}

	//setup server
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.True(t, checkACL(acl, w, r), "check should return true")
		wait <- true
	}
	http.HandleFunc(fmt.Sprintf("/%d", port), handler)
	//Run the server
	t.Logf("Starting test server on port %d", port)
	go http.ListenAndServe(fmt.Sprintf(":%d", port), nil)

	//Run the test
	t.Logf("Calling the test server")
	client := http.DefaultClient
	req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:%d/%d", port, port), nil)
	req.Header.Set("X-Forwarded-For", "localhost0")
	res, err := client.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, res.StatusCode, "status code should be 200")

	<-wait
}

func TestCheckACLShouldUnauthorizeLocalIPWhenACEIPIsSetToFooBar(t *testing.T) {
	var port = freeport.GetPort()
	wait := make(chan bool, 2)

	//setup ACL
	acl := ACL{
		&ACEIP{"FooBar"},
	}

	//setup server
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.False(t, checkACL(acl, w, r), "check should return false")
		wait <- true
	}
	http.HandleFunc(fmt.Sprintf("/%d", port), handler)
	//Run the server
	t.Logf("Starting test server on port %d", port)
	go http.ListenAndServe(fmt.Sprintf(":%d", port), nil)

	//Run the test
	t.Logf("Calling the test server")
	client := http.DefaultClient
	req, _ := http.NewRequest("GET", fmt.Sprintf("http://localhost:%d/%d", port, port), nil)
	req.Header.Set("X-Forwarded-For", "localhost")
	res, err := client.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode, "status code should be "+string(http.StatusUnauthorized))
	wait <- true

	<-wait
	<-wait
}

func TestCheckACLShouldAuthorizeFooBarWhenACEDigetIsSetToFooBar(t *testing.T) {
	var port = freeport.GetPort()
	wait := make(chan bool, 1)

	//setup ACL
	acl := ACL{
		&ACEDigest{Username: "Foo", Password: "Bar"},
	}

	//setup server
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.True(t, checkACL(acl, w, r), "check should return true")
		wait <- true
	}
	http.HandleFunc(fmt.Sprintf("/%d", port), handler)
	//Run the server
	t.Logf("Starting test server on port %d", port)
	go http.ListenAndServe(fmt.Sprintf(":%d", port), nil)

	//Run the test
	t.Logf("Calling the test server")
	client := http.DefaultClient
	req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:%d/%d", port, port), nil)
	req.SetBasicAuth("Foo", "Bar")
	res, err := client.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, res.StatusCode, "status code should be 200")

	<-wait
}

func TestCheckACLShouldUnauthorizeFooBarWhenACEDigetIsSetToXXXXX(t *testing.T) {
	var port = freeport.GetPort()
	wait := make(chan bool, 2)

	//setup ACL
	acl := ACL{
		&ACEDigest{Username: "XXX", Password: "XXX"},
	}

	//setup server
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.False(t, checkACL(acl, w, r), "check should return false")
		wait <- true
	}
	http.HandleFunc(fmt.Sprintf("/%d", port), handler)
	//Run the server
	t.Logf("Starting test server on port %d", port)
	go http.ListenAndServe(fmt.Sprintf(":%d", port), nil)

	//Run the test
	t.Logf("Calling the test server")
	client := http.DefaultClient
	req, _ := http.NewRequest("GET", fmt.Sprintf("http://localhost:%d/%d", port, port), nil)
	req.SetBasicAuth("Foo", "Bar")
	res, err := client.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode, "status code should be "+string(http.StatusUnauthorized))
	wait <- true

	<-wait
	<-wait
}
