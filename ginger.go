package ginger

import (
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/eikeon/dynamodb"
	"github.com/eikeon/ginger/queue"
)

var DB dynamodb.DynamoDB

func urlHash(URL string) string {
	h := md5.New()
	io.WriteString(h, URL)
	return fmt.Sprintf("%x", h.Sum(nil))
}

type FetchRequest struct {
	Host        string `db:"HASH"`
	URLHash     string
	URL         string
	RequestedOn string `db:"RANGE"`
	RequestedBy string
}

func NewFetchRequest(URL string) (*FetchRequest, error) {
	if u, err := url.Parse(URL); err == nil {
		now := time.Now().Format(time.RFC3339Nano)
		return &FetchRequest{Host: u.Host, URL: URL, RequestedOn: now}, nil
	} else {
		return nil, err
	}
}

type Fetch struct {
	Host          string `db:"HASH"`
	URLHash       string
	URL           string
	FetchedOn     string `db:"RANGE"`
	StatusCode    int
	ContentLength int64
}

func NewFetch(URL string) (*Fetch, error) {
	if u, err := url.Parse(URL); err == nil {
		now := time.Now().Format(time.RFC3339Nano)
		return &Fetch{Host: u.Host, URL: URL, FetchedOn: now}, nil
	} else {
		return nil, err
	}
}

func (req *FetchRequest) Put() error {
	_, err := DB.PutItem("fetchrequest", DB.ToItem(req), nil)
	return err
}

func (req *FetchRequest) Update() error {
	return req.Put()
}

func (c *Ginger) Add(URL string, requestedBy string) error {
	if f, err := NewFetchRequest(URL); err == nil {
		err := f.Put()
		if err != nil {
			return err
		}
	} else {
		return err
	}

	return nil
}

func (c *Ginger) Items() (fetch []*FetchRequest) {
	response, err := DB.Scan("fetchrequest", nil)
	if err != nil {
		log.Fatal(err)
	}
	for _, i := range response.Items {
		fetch = append(fetch, DB.FromItem("fetchrequest", i).(*FetchRequest))
	}
	return
}

func (c *Ginger) Requested() (requested []*FetchRequest) {
	for _, fetch := range c.Items() {
		if fetch.RequestedOn != "" {
			requested = append(requested, fetch)
		}
	}
	return
}

func (c *Ginger) Fetched() (fetch []*Fetch) {
	response, err := DB.Scan("fetch", nil)
	if err != nil {
		log.Fatal(err)
	}
	for _, i := range response.Items {
		fetch = append(fetch, DB.FromItem("fetch", i).(*Fetch))
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
	fetchrequest, err := DB.Register("fetchrequest", (*FetchRequest)(nil))
	if err != nil {
		panic(err)
	}
	fetch, err := DB.Register("fetch", (*Fetch)(nil))
	if err != nil {
		panic(err)
	}
	pt := dynamodb.ProvisionedThroughput{ReadCapacityUnits: 1024, WriteCapacityUnits: 1024}
	if _, err := DB.CreateTable("fetchrequest", fetchrequest.AttributeDefinitions, fetchrequest.KeySchema, pt, nil); err != nil {
		log.Println(err)
	}
	if _, err := DB.CreateTable("fetch", fetch.AttributeDefinitions, fetch.KeySchema, dynamodb.ProvisionedThroughput{ReadCapacityUnits: 1, WriteCapacityUnits: 1}, nil); err != nil {
		log.Println(err)
	}

	// wait until all tables are active
	for _, name := range []string{"fetchrequest", "fetch"} {
		for {
			if description, err := DB.DescribeTable(name, nil); err != nil {
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
	response, err := DB.Scan("fetchrequest", nil)
	if err != nil {
		log.Fatal(err)
	}
	for _, i := range response.Items {
		requests.Send(DB.FromItem("fetchrequest", i).(*FetchRequest))
	}
}

func Worker(requests queue.Queue) {
	for {
		var fetchrequest FetchRequest
		err := requests.Receive(&fetchrequest)
		if err != nil {
			log.Println("Done fetching")
			break
		}
		r, err := http.Get(fetchrequest.URL)
		if err != nil {
			log.Fatal(err)
		}
		if fetch, err := NewFetch(fetchrequest.URL); err == nil {
			fetch.StatusCode = r.StatusCode
			fetch.ContentLength = r.ContentLength
			DB.PutItem("fetch", DB.ToItem(fetch), nil)
		} else {
			log.Println(err)
		}
		DB.DeleteItem("fetchrequest", DB.ToKey(&fetchrequest), nil)
	}
}
