package ginger_test

import (
	//"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/bmizerany/assert"
	"github.com/eikeon/ginger"
)

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
	g := ginger.NewGinger()
	go g.Fetcher()
	go g.Persister()
	u, err := url.Parse("http://www.eikeon.com/")
	if err != nil {
		t.Error(err)
	}
	g.Add(u)
	time.Sleep(1 * time.Second)
	_, ok := g.Results["http://www.eikeon.com/"]
	assert.Equal(t, ok, true)
}
