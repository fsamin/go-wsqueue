package wsqueue

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckACLShouldAuthorizeEveryoneWhenACEIsSetToWord(t *testing.T) {
	var testID = 0

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
	http.HandleFunc(fmt.Sprintf("/%d", testID), handler)
	//Run the server
	go http.ListenAndServe(fmt.Sprintf(":5880%d", testID), nil)

	//Run the test
	t.Logf("Calling the test server")
	client := http.DefaultClient
	client.Get(fmt.Sprintf("http://localhost:5880%d/%d", testID, testID))

	<-wait
}

func TestCheckACLShouldAuthorizeLocalIPWhenACEIsSetLocalIP(t *testing.T) {
	var testID = 1
	wait := make(chan bool, 1)

	//setup ACL
	acl := ACL{
		&ACEIP{"127.0.0.1"},
	}

	//setup server
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.True(t, checkACL(acl, w, r), "check should return true")
		wait <- true
	}
	http.HandleFunc(fmt.Sprintf("/%d", testID), handler)
	//Run the server
	go http.ListenAndServe(fmt.Sprintf(":5880%d", testID), nil)

	//Run the test
	t.Logf("Calling the test server")
	client := http.DefaultClient
	req, _ := http.NewRequest("GET", fmt.Sprintf("http://localhost:5880%d/%d", testID, testID), nil)
	req.Header.Set("X-Forwarded-For", "127.0.0.1")
	client.Do(req)

	<-wait
}

func TestCheckACLShouldUnauthorizeLocalIPWhenACEIPIsSetToFooBar(t *testing.T) {
	var testID = 2
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
	http.HandleFunc(fmt.Sprintf("/%d", testID), handler)
	//Run the server
	go http.ListenAndServe(fmt.Sprintf(":5880%d", testID), nil)

	//Run the test
	t.Logf("Calling the test server")
	client := http.DefaultClient
	req, _ := http.NewRequest("GET", fmt.Sprintf("http://localhost:5880%d/%d", testID, testID), nil)
	req.Header.Set("X-Forwarded-For", "127.0.0.1")
	res, _ := client.Do(req)
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode, "status code should be "+string(http.StatusUnauthorized))
	wait <- true

	<-wait
	<-wait
}

func TestCheckACLShouldAuthorizeFooBarWhenACEDigetIsSetToFooBar(t *testing.T) {
	var testID = 3
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
	http.HandleFunc(fmt.Sprintf("/%d", testID), handler)
	//Run the server
	go http.ListenAndServe(fmt.Sprintf(":5880%d", testID), nil)

	//Run the test
	t.Logf("Calling the test server")
	client := http.DefaultClient
	req, _ := http.NewRequest("GET", fmt.Sprintf("http://localhost:5880%d/%d", testID, testID), nil)
	req.SetBasicAuth("Foo", "Bar")
	client.Do(req)

	<-wait
}

func TestCheckACLShouldUnauthorizeFooBarWhenACEDigetIsSetToXXXXX(t *testing.T) {
	var testID = 4
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
	http.HandleFunc(fmt.Sprintf("/%d", testID), handler)
	//Run the server
	go http.ListenAndServe(fmt.Sprintf(":5880%d", testID), nil)

	//Run the test
	t.Logf("Calling the test server")
	client := http.DefaultClient
	req, _ := http.NewRequest("GET", fmt.Sprintf("http://localhost:5880%d/%d", testID, testID), nil)
	req.SetBasicAuth("Foo", "Bar")
	res, _ := client.Do(req)
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode, "status code should be "+string(http.StatusUnauthorized))
	wait <- true

	<-wait
	<-wait
}
