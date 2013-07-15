package ginger_test

import (
	"encoding/json"
	"errors"
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
		return err
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
		return err
	}
	return nil
}

type table struct {
	responses []ginger.FetchResponse
}

// an in memory ginger.DB implementation
type db struct {
	tables map[string]*table
}

func (b *db) CreateTable(name string, attributeDefinitions []ginger.AttributeDefinition, keySchema ginger.KeySchema) {
	if b.tables == nil {
		b.tables = make(map[string]*table)
	}
	b.tables[name] = &table{}
}

func (b *db) Put(tableName string, r ginger.FetchResponse) error {
	if b.tables == nil {
		return errors.New("no tables")
	}
	t, ok := b.tables[tableName]
	if !ok {
		t = &table{}
		b.tables[tableName] = t
	}
	t.responses = append(t.responses, r)
	return nil
}

func TestFetch(t *testing.T) {
	u, err := url.Parse("http://www.eikeon.com/")
	if err != nil {
		t.Error(err)
	}
	req := &ginger.FetchRequest{u}
	response := req.Fetch()
	assert.Equal(t, response.StatusCode, 200)
}

func TestAdd(t *testing.T) {
	requests := &queue{make(chan string, 100)}
	responses := &queue{make(chan string, 100)}
	db := &db{}
	db.CreateTable("fetchresponse", []ginger.AttributeDefinition{}, ginger.KeySchema{})
	g := ginger.NewGinger(requests, responses, db)
	err := g.Add("http://www.eikeon.com/")

	if err != nil {
		t.Error("unable to fetch http://eikeon.com/")
	}

	go ginger.Worker(requests, responses)
	go ginger.Persister(responses, db)

	time.Sleep(1 * time.Second)
	for _, fr := range db.tables["fetchresponse"].responses {
		if fr.Request.URL.String() == "http://www.eikeon.com/" {
			goto found
		}
	}
	t.Error("Didn't find expected result")
found:
}
