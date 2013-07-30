package ginger

import (
	"crypto/md5"
	"errors"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/eikeon/ginger/db"
	"github.com/eikeon/ginger/queue"
)

var DB db.DB

type CollectionItem struct {
	CollectionName string
	URL            string
	AddedOn        string
	RequestedOn    string
	// Last fetched on
	FetchedOn string
}

func (ci *CollectionItem) Put() error {
	return DB.Put("collectionitem", *ci)
}

func (ci *CollectionItem) Update() error {
	return ci.Put()
}

func urlHash(URL string) string {
	h := md5.New()
	io.WriteString(h, URL)
	return string(h.Sum(nil))
}

type Resource struct {
	URLHash     string
	AddedOn     string
	UpdatedOn   string
	URL         string
	Collections []string
}

type Fetch struct {
	URLHash     string
	RequestedOn string
	FetchedOn   string
	URL         string
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
	now := time.Now().Format(time.RFC3339Nano)
	f := &CollectionItem{CollectionName: c.Name, URL: URL, AddedOn: now, RequestedOn: now} // TODO: requestedBy
	f.Put()
	return nil
}

func (c *Collection) Put() error {
	return DB.Put("collection", *c)
}

func (c *Collection) Items() (fetch []CollectionItem) {
	items, err := DB.Scan("collectionitem")
	if err != nil {
		log.Fatal(err)
	}
	for _, i := range items {
		fetch = append(fetch, i.(CollectionItem))
	}
	return
}

func (c *Collection) Requested() (requested []CollectionItem) {
	for _, fetch := range c.Items() {
		if fetch.RequestedOn != "" {
			requested = append(requested, fetch)
		}
	}
	return
}

func (c *Collection) Fetched() (fetched []CollectionItem) {
	for _, fetch := range c.Items() {
		if fetch.FetchedOn != "" {
			fetched = append(fetched, fetch)
		}
	}
	return
}

type Ginger struct {
	cond *sync.Cond // a rendezvous point for goroutines waiting for or announcing state changed
}

func NewMemoryGinger() *Ginger {
	DB = &db.MemoryDB{}
	DB.CreateTable("fetch", []db.AttributeDefinition{}, db.KeySchema{db.KeySchemaElement{"URL", "String"}})
	DB.CreateTable("collection", []db.AttributeDefinition{}, db.KeySchema{db.KeySchemaElement{"Name", "String"}})
	DB.CreateTable("collectionitem", []db.AttributeDefinition{}, db.KeySchema{db.KeySchemaElement{"Name", "String"}, db.KeySchemaElement{"URL", "String"}})
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

func (g *Ginger) GetCollection(name string) (*Collection, error) {
	for _, c := range g.Collections() {
		if c.Name == name {
			return &c, nil
		}
	}
	return nil, errors.New("Collection not found")
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
	items, err := DB.Scan("collectionitem")
	if err != nil {
		log.Fatal(err)
	}
	for _, i := range items {
		requests.Send(i)
	}
}

func Worker(requests queue.Queue) {
	for {
		var collectionitem CollectionItem
		err := requests.Receive(&collectionitem)
		if err != nil {
			log.Println("Done fetching")
			break
		}
		r, err := http.Get(collectionitem.URL)
		if err != nil {
			log.Fatal(err)
		}
		now := time.Now().Format(time.RFC3339Nano)
		fetch := &Fetch{URLHash: urlHash(collectionitem.URL), URL: collectionitem.URL, FetchedOn: now} // TODO: requestedBy
		fetch.Response = &FetchResponse{r.StatusCode, r.ContentLength}
		fetch.Put()
		collectionitem.RequestedOn = ""
		collectionitem.FetchedOn = now
		collectionitem.Update()
	}
}
