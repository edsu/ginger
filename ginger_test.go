package ginger_test

import (
	"testing"
	"time"

	"github.com/bmizerany/assert"
	"github.com/eikeon/ginger"
	"github.com/eikeon/ginger/queue"
)

func TestGinger(t *testing.T) {
	requests := queue.NewChannelQueue(nil)
	g := ginger.NewMemoryGinger()
	c, err := g.AddCollection("testCollection", "me")
	assert.Equal(t, err, nil)
	err = c.Add("http://www.eikeon.com/", "me")

	if err != nil {
		t.Error("unable to fetch http://eikeon.com/")
	}

	ginger.Qer(requests)

	go ginger.Worker(requests)

	time.Sleep(1 * time.Second)
	if responses, err := ginger.DB.Scan("fetch"); err == nil {
		for _, r := range responses {
			f := r.(ginger.Fetch)
			if f.URL == "http://www.eikeon.com/" {
				if f.Response != nil && f.Response.StatusCode != 0 {
					goto found
				}
			}
		}
	} else {
		t.Log(err)
	}
	t.Error("Didn't find expected result")
found:
}
