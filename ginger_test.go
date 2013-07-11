package ginger_test

import (
	"encoding/json"
	"errors"
	"log"
	"net/url"
	"testing"
	"time"

	"github.com/bmizerany/assert"
	"github.com/eikeon/ginger"
)

// channel based queue implementation
type queue struct {
	messages chan string
}

func (q *queue) Send(message interface{}) error {
	b, err := json.Marshal(message)
	if err != nil {
		log.Fatal(err)
	}
	q.messages <- string(b)
	return nil
}

func (q *queue) Receive(i interface{}) error {
	message := <-q.messages
	if message == "" {
		return errors.New("empty")
	}
	if err := json.Unmarshal([]byte(message), &i); err != nil {
		//log.Fatal(err) TODO: a FetchResponse we can round trip
		log.Println("ERROR:", err)
	}
	return nil
}

func TestFetch(t *testing.T) {
	u, err := url.Parse("http://www.eikeon.com/")
	if err != nil {
		t.Error(err)
	}
	req := &ginger.FetchRequest{u}
	response := req.Fetch()
	assert.Equal(t, response.Response.StatusCode, 200)
}

func TestGinger(t *testing.T) {
	requests := &queue{make(chan string, 100)}
	responses := &queue{make(chan string, 100)}
	results := make(ginger.Results)
	g := ginger.NewGinger(requests, responses, &results)

	u, err := url.Parse("http://www.eikeon.com/")
	if err != nil {
		t.Error(err)
	}
	g.Add(u)

	go ginger.Worker(requests, responses)
	go ginger.Persister(responses, &results)

	time.Sleep(1 * time.Second)
	_, ok := results["http://www.eikeon.com/"]
	assert.Equal(t, ok, true)
}
