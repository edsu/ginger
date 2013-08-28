package ginger_test

import (
	"testing"
	"time"

	"github.com/eikeon/ginger"
	"github.com/eikeon/ginger/queue"
)

func TestGinger(t *testing.T) {
	requests := queue.NewChannelQueue(nil)
	ginger.NewMemoryGinger(false)

	if f, err := ginger.NewFetch("http://www.eikeon.com/"); err == nil {
		f.URL = "http://www.eikeon.com/"
		f.Fetch()
		f.Put()
	} else {
		t.Error("unable to add fetch for http://eikeon.com/")
	}

	ginger.Qer(requests)

	go ginger.Worker(requests)

	time.Sleep(1 * time.Second)
	if response, err := ginger.DB.Scan("fetch", nil); err == nil {
		for _, i := range response.Items {
			f := ginger.DB.FromItem("fetch", i).(*ginger.Fetch)
			if f.URL == "http://www.eikeon.com/" {
				if f.StatusCode != 0 {
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
