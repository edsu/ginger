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

var DB db.DB

type Fetch struct {
	URL         *url.URL
	RequestedOn string
	Response    *FetchResponse
}

func (req *Fetch) Put() error {
	return DB.Put("fetch", *req)
}

func (req *Fetch) Update() error {
	return req.Put()
}

type FetchResponse struct {
	StatusCode    int
	ContentLength int64
}

type Collection struct {
	Name        string
	RequestedBy string
}

func (c *Collection) Add(URL string, requestedBy string) error {
	u, err := url.Parse(URL)
	if err == nil {
		t := time.Now()
		f := &Fetch{u, t.Format(time.RFC3339Nano), nil}
		f.Put()
	}
	return err
}

func (c *Collection) Put() error {
	return DB.Put("collection", *c)
}

type Ginger struct {
	cond *sync.Cond // a rendezvous point for goroutines waiting for or announcing state changed
}

func NewMemoryGinger() *Ginger {
	DB = &db.MemoryDB{}
	DB.CreateTable("fetch", []db.AttributeDefinition{}, db.KeySchema{})
	DB.CreateTable("collection", []db.AttributeDefinition{}, db.KeySchema{})
	return &Ginger{}
}

func (g *Ginger) Collections() (collection []Collection) {
	items, err := DB.Scan("collection")
	if err != nil {
		log.Fatal(err)
	}
	for _, i := range items {
		collection = append(collection, i.(Collection))
	}
	return
}

func (g *Ginger) AddCollection(name string, requestedBy string) (*Collection, error) {
	c := &Collection{name, requestedBy}
	if err := c.Put(); err != nil {
		return nil, err
	}
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

func Qer(requests queue.Queue) {
	items, err := DB.Scan("fetch")
	if err != nil {
		log.Fatal(err)
	}
	for _, i := range items {
		requests.Send(i)
	}
}

func Worker(requests queue.Queue) {
	for {
		var fetch Fetch
		err := requests.Receive(&fetch)
		if err != nil {
			log.Println("Done fetching")
			break
		}
		r, err := http.Get(fetch.URL.String())
		if err != nil {
			log.Fatal(err)
		}
		fetch.Response = &FetchResponse{r.StatusCode, r.ContentLength}
		fetch.Update()
	}
}
