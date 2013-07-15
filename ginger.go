package ginger

import (
	"log"
	"net/http"
	"net/url"
	"sync"

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
	cond      *sync.Cond // a rendezvous point for goroutines waiting for or announcing state changed
}

func NewGinger(requests, responses queue.Queue, db db.DB) *Ginger {
	return &Ginger{requests, responses, db, nil}
}

type Collection struct {
	Name        string
	RequestedBy string
}

func (g *Ginger) Collections() (collection []Collection) {
	items, err := g.db.Scan("collection")
	if err != nil {
		log.Fatal(err)
	}
	for _, i := range items {
		collection = append(collection, i.(Collection))
	}
	return
}

func (g *Ginger) AddCollection(name string, requestedBy string) error {
	g.db.Put("collection", Collection{name, requestedBy})
	g.StateChanged()
	return nil
}

func (g *Ginger) Add(URL string) error {
	u, err := url.Parse(URL)
	if err == nil {
		g.requests.Send(&FetchRequest{u})
	}
	return err
}

func (m *Ginger) getStateCond() *sync.Cond {
	if m.cond == nil {
		m.cond = sync.NewCond(&sync.Mutex{})
	}
	return m.cond
}

func (m *Ginger) StateChanged() {
	c := m.getStateCond()
	c.L.Lock()
	c.Broadcast()
	c.L.Unlock()
}

func (m *Ginger) WaitStateChanged() {
	c := m.getStateCond()
	c.L.Lock()
	c.Wait()
	c.L.Unlock()
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
