package ginger_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/bmizerany/assert"
	"github.com/eikeon/ginger"
	"github.com/eikeon/ginger/db"
	"github.com/eikeon/ginger/queue"
)

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
	requests := queue.NewChannelQueue(nil)
	responses := queue.NewChannelQueue(nil)
	mdb := &db.MemoryDB{}
	mdb.CreateTable("fetchresponse", []db.AttributeDefinition{}, db.KeySchema{})
	g := ginger.NewGinger(requests, responses, mdb)
	err := g.Add("http://www.eikeon.com/")

	if err != nil {
		t.Error("unable to fetch http://eikeon.com/")
	}

	go ginger.Worker(requests, responses)
	go ginger.Persister(responses, mdb)

	time.Sleep(1 * time.Second)
	if responses, err := mdb.Scan("fetchresponse"); err == nil {
		for _, fr := range responses {
			if fr.(ginger.FetchResponse).Request.URL.String() == "http://www.eikeon.com/" {
				goto found
			}
		}
	} else {
		t.Log(err)
	}
	t.Error("Didn't find expected result")
found:
}
