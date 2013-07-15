package ginger

import (
	"log"
	"net/http"
	"net/url"

	"github.com/eikeon/ginger/db"
	"github.com/eikeon/ginger/queue"
)

type FetchRequest struct {
	URL *url.URL
}

func (req *FetchRequest) Fetch() *FetchResponse {
	response, err := http.Get(req.URL.String())
	if err != nil {
		log.Fatal(err)
	}
	return &FetchResponse{response.StatusCode, response.ContentLength, req}
}

type FetchResponse struct {
	StatusCode    int
	ContentLength int64
	Request       *FetchRequest
}

type Ginger struct {
	requests  queue.Queue
	responses queue.Queue
	db        db.DB
}

func NewGinger(requests, responses queue.Queue, db db.DB) *Ginger {
	return &Ginger{requests, responses, db}
}

func (g *Ginger) Greeting() string {
	return "Hello, world!"
}

func (g *Ginger) Add(URL string) error {
	u, err := url.Parse(URL)
	if err == nil {
		g.requests.Send(&FetchRequest{u})
	}
	return err
}

func Worker(requests, responses queue.Queue) {
	for {
		var request FetchRequest
		err := requests.Receive(&request)
		if err != nil {
			log.Println("Done fetching")
			break
		}
		response := request.Fetch()
		responses.Send(response)
	}
}

func Persister(responses queue.Queue, db db.DB) {
	for {
		var response FetchResponse
		err := responses.Receive(&response)
		if err != nil {
			log.Println("done persisting")
			break
		}
		db.Put("fetchresponse", response)
	}
}
