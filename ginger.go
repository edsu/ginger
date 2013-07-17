package ginger

import (
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/eikeon/ginger/db"
	"github.com/eikeon/ginger/queue"
)

type FetchRequest struct {
	db          db.DB
	URL         *url.URL
	RequestedOn string
}

func (req *FetchRequest) Fetch() *FetchResponse {
	response, err := http.Get(req.URL.String())
	if err != nil {
		log.Fatal(err)
	}
	return &FetchResponse{response.StatusCode, response.ContentLength, req}
}

func (req *FetchRequest) Put() error {
	return req.db.Put("fetchrequest", *req)
}

type FetchResponse struct {
	StatusCode    int
	ContentLength int64
	Request       *FetchRequest
}

type Collection struct {
	db          db.DB
	Name        string
	RequestedBy string
}

func (c *Collection) Add(URL string, requestedBy string) error {
	u, err := url.Parse(URL)
	if err == nil {
		t := time.Now()
		f := &FetchRequest{c.db, u, t.Format(time.RFC3339Nano)}
		f.Put()
	}
	return err
}

func (c *Collection) Put() error {
	return c.db.Put("collection", *c)
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

func (g *Ginger) AddCollection(name string, requestedBy string) (*Collection, error) {
	c := &Collection{g.db, name, requestedBy}
	c.Put()
	g.StateChanged()
	return c, nil
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

func Qer(db db.DB, requests queue.Queue) {
	items, err := db.Scan("fetchrequest")
	if err != nil {
		log.Fatal(err)
	}
	for _, i := range items {
		requests.Send(i)
	}
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
