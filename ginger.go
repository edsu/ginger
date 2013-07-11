package ginger

import (
	"log"
	"net/http"
	"net/url"
)

type FetchRequest struct {
	URL *url.URL
}

func (req *FetchRequest) Fetch() *FetchResponse {
	response, err := http.Get(req.URL.String())
	if err != nil {
		log.Fatal(err)
	}
	return &FetchResponse{response}
}

type FetchResponse struct {
	Response *http.Response
}

type Queue interface {
	Send(message interface{}) error
	Receive(message interface{}) error
}

type Result struct {
	StatusCode    int
	ContentLength int64
}
type Results map[string]Result

type Ginger struct {
	requests  Queue
	responses Queue
	Results   Results
}

func NewGinger(requests, responses Queue) *Ginger {
	return &Ginger{requests, responses, make(Results)}
}

func (g *Ginger) Greeting() string {
	return "Hello, world!"
}

func (g *Ginger) Add(url *url.URL) {
	g.requests.Send(&FetchRequest{url})
}

func (g *Ginger) Fetcher() {
	for {
		var request FetchRequest
		err := g.requests.Receive(&request)
		if err != nil {
			log.Println("Done fetching")
			break
		}
		response := request.Fetch()
		g.responses.Send(response)
	}
}

func (g *Ginger) Persister() {
	for {
		var response FetchResponse
		err := g.responses.Receive(&response)
		if err != nil {
			log.Println("done persisting")
			break
		}
		g.Results[response.Response.Request.URL.String()] = Result{response.Response.StatusCode, response.Response.ContentLength}
	}
}
