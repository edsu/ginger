package ginger

import (
	"crypto/md5"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/eikeon/dynamodb"
	"github.com/eikeon/ginger/queue"
)

var DB dynamodb.DynamoDB

type CollectionItem struct {
	CollectionName string `db:"HASH"`
	URL            string `db:"RANGE"`
	AddedOn        string
	RequestedOn    string
	// Last fetched on
	FetchedOn string
}

func (ci *CollectionItem) Put() error {
	return DB.PutItem("collectionitem", DB.ToItem(ci))
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
	URLHash     string `db:"HASH"`
	RequestedOn string
	FetchedOn   string
	URL         string
	// Response
	StatusCode    int
	ContentLength int64
}

func (req *Fetch) Put() error {
	return DB.PutItem("fetch", DB.ToItem(req))
}

func (req *Fetch) Update() error {
	return req.Put()
}

type FetchResponse struct {
}

type Collection struct {
	Name        string `db:"HASH"`
	RequestedBy string
}

func (c *Collection) Add(URL string, requestedBy string) error {
	now := time.Now().Format(time.RFC3339Nano)
	f := &CollectionItem{CollectionName: c.Name, URL: URL, AddedOn: now, RequestedOn: now} // TODO: requestedBy
	f.Put()
	return nil
}

func (c *Collection) Put() error {
	return DB.PutItem("collection", DB.ToItem(c))
}

func (c *Collection) Items() (fetch []*CollectionItem) {
	response, err := DB.Scan("collectionitem")
	if err != nil {
		log.Fatal(err)
	}
	for _, i := range response.Items {
		fetch = append(fetch, DB.FromItem("collectionitem", i).(*CollectionItem))
	}
	return
}

func (c *Collection) Requested() (requested []*CollectionItem) {
	for _, fetch := range c.Items() {
		if fetch.RequestedOn != "" {
			requested = append(requested, fetch)
		}
	}
	return
}

func (c *Collection) Fetched() (fetched []*CollectionItem) {
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

func NewMemoryGinger(dynamo bool) *Ginger {
	if dynamo {
		DB = dynamodb.NewDynamoDB()
	} else {
		DB = dynamodb.NewMemoryDB()
	}
	fetch, err := DB.Register("fetch", (*Fetch)(nil))
	if err != nil {
		panic(err)
	}
	collection, err := DB.Register("collection", (*Collection)(nil))
	if err != nil {
		panic(err)
	}
	collectionitem, err := DB.Register("collectionitem", (*CollectionItem)(nil))
	if err != nil {
		panic(err)
	}
	if err := DB.CreateTable(fetch); err != nil {
		panic(err)
	}
	if err := DB.CreateTable(collection); err != nil {
		panic(err)
	}
	if err := DB.CreateTable(collectionitem); err != nil {
		panic(err)
	}

	// wait until all tables are active
	for _, name := range []string{"fetch", "collection", "collectionitem"} {
		for {
			if description, err := DB.DescribeTable(name); err != nil {
				log.Println("DescribeTable err:", err)
			} else {
				log.Println(description.Table.TableStatus)
				if description.Table.TableStatus == "ACTIVE" {
					break
				}
			}
			time.Sleep(time.Second)
		}
	}

	return &Ginger{}
}

func (g *Ginger) Collections() (collection []*Collection) {
	response, err := DB.Scan("collection")
	if err != nil {
		log.Fatal(err)
	}
	for _, i := range response.Items {
		collection = append(collection, DB.FromItem("collection", i).(*Collection))
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
	if r, err := DB.GetItem("collection", DB.ToKey(&Collection{Name: name})); err == nil {
		return DB.FromItem("collection", r.Item).(*Collection), nil
	} else {
		return nil, err
	}
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
	response, err := DB.Scan("collectionitem")
	if err != nil {
		log.Fatal(err)
	}
	for _, i := range response.Items {
		requests.Send(DB.FromItem("collectionitem", i).(*CollectionItem))
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
		fetch.StatusCode = r.StatusCode
		fetch.ContentLength = r.ContentLength
		fetch.Put()
		collectionitem.RequestedOn = ""
		collectionitem.FetchedOn = now
		collectionitem.Update()
	}
}
