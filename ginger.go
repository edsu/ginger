package ginger

import (
	"net/url"
	"net/http"
)

type FetchRequest struct {
	URL url.URL
}

type FetchResponse struct {
	Response *http.Response
}

type Ginger struct {
}

func (g *Ginger) Greeting() string {
	return "Hello, world!"
}

func Fetch(req *FetchRequest) *FetchResponse {
	return &FetchResponse{&http.Response{}}
}
