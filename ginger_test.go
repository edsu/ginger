package ginger_test

import (
	"net/http"
	"testing"

	"github.com/bmizerany/assert"
	"github.com/eikeon/ginger"
)

func TestFetch(t *testing.T) {
	req := &ginger.FetchRequest{}
	assert.Equal(t, ginger.Fetch(req), &ginger.FetchResponse{&http.Response{}})
}
