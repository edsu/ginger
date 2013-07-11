package ginger

import (
	"encoding/json"
	"errors"
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

type Queue struct {
	messages chan string
}

func (q *Queue) SendMessage(message interface{}) error {
	b, err := json.Marshal(message)
	if err != nil {
		log.Fatal(err)
	}
	q.messages <- string(b)
	return nil
}

func (q *Queue) ReceiveMessage(i interface{}) error {
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

func NewGinger() *Ginger {
	return &Ginger{Queue{make(chan string, 100)}, Queue{make(chan string, 100)}, make(Results)}
}

func (g *Ginger) Greeting() string {
	return "Hello, world!"
}

func (g *Ginger) Add(url *url.URL) {
	fr := FetchRequest{url}
	g.requests.SendMessage(fr)
}

func (g *Ginger) Fetcher() {
	for {
		var request FetchRequest
		err := g.requests.ReceiveMessage(&request)
		if err != nil {
			log.Println("Done fetching")
			break
		}
		response := request.Fetch()
		g.responses.SendMessage(response)
	}
}

func (g *Ginger) Persister() {
	for {
		var response FetchResponse
		err := g.responses.ReceiveMessage(&response)
		if err != nil {
			log.Println("done persisting")
			break
		}
		g.Results[response.Response.Request.URL.String()] = Result{response.Response.StatusCode, response.Response.ContentLength}
	}
}
