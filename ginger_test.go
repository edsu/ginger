package ginger_test

import (
	"testing"
	"time"

	"github.com/bmizerany/assert"
	"github.com/eikeon/ginger"
	"github.com/eikeon/ginger/db"
	"github.com/eikeon/ginger/queue"
)

func TestGinger(t *testing.T) {
	requests := queue.NewChannelQueue(nil)
	responses := queue.NewChannelQueue(nil)
	mdb := &db.MemoryDB{}
	mdb.CreateTable("fetchresponse", []db.AttributeDefinition{}, db.KeySchema{})
	g := ginger.NewGinger(requests, responses, mdb)
	c, err := g.AddCollection("testCollection", "me")
	assert.Equal(t, err, nil)
	err = c.Add("http://www.eikeon.com/", "me")

	if err != nil {
		t.Error("unable to fetch http://eikeon.com/")
	}

	ginger.Qer(mdb, requests)

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
