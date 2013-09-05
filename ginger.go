package ginger

import (
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/eikeon/dynamodb"
	"github.com/eikeon/ginger/queue"
	"github.com/eikeon/sns"
)

var DB dynamodb.DynamoDB
var SNS sns.SNS

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
	Error         string
}

func NewFetch(URL string) (*Fetch, error) {
	if u, err := url.Parse(URL); err == nil {
		now := time.Now().Format(time.RFC3339Nano)
		return &Fetch{Host: u.Host, URL: URL, FetchedOn: now}, nil
	} else {
		return nil, err
	}
}

func (f *Fetch) NumFetchesLast(d time.Duration) int {
	now := time.Now()
	start := now.Add(-d)
	f1 := start.Format(time.RFC3339Nano)
	f2 := now.Format(time.RFC3339Nano)
	conditions := dynamodb.KeyConditions{"Host": {[]dynamodb.AttributeValue{{"S": f.Host}}, "EQ"}, "FetchedOn": {[]dynamodb.AttributeValue{{"S": f1}, {"S": f2}}, "BETWEEN"}}
	if qr, err := DB.Query("fetch", &dynamodb.QueryOptions{KeyConditions: conditions, Select: "COUNT"}); err == nil {
		return qr.Count
	} else {
		log.Println("query error:", err)
	}
	return 0
}

func (f *Fetch) Fetch() {
	c := http.DefaultTransport
	req, err := http.NewRequest("GET", f.URL, nil)
	if err != nil {
		f.Error = err.Error()
	}
	f.FetchedOn = time.Now().Format(time.RFC3339Nano)
	if r, err := c.RoundTrip(req); err != nil {
		f.Error = err.Error()
	} else {
		f.StatusCode = r.StatusCode
		if _, err := ioutil.ReadAll(r.Body); err == nil {
			r.Body.Close()
		} else {
			log.Println("ReadAll err:", err)
		}
		f.ContentLength = r.ContentLength
	}
}

func (f *Fetch) Put() error {
	_, err := DB.PutItem("fetch", DB.ToItem(f), nil)
	return err
}

func (req *FetchRequest) Put() error {
	_, err := DB.PutItem("fetchrequest", DB.ToItem(req), nil)
	return err
}

func (req *FetchRequest) Update() error {
	return req.Put()
}

var options = url.Values{"TopicArn": []string{"arn:aws:sns:us-east-1:966103638140:ginger-test"}}

func (c *Ginger) Add(URL string, requestedBy string) error {
	if _, err := SNS.Publish(URL, options); err != nil {
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
	SNS = sns.NewSNS()
	if dynamo {
		DB = dynamodb.NewDynamoDB()
	} else {
		DB = dynamodb.NewMemoryDB()
	}
	fetch, err := DB.Register("fetch", (*Fetch)(nil))
	if err != nil {
		panic(err)
	}
	pt := dynamodb.ProvisionedThroughput{ReadCapacityUnits: 8192, WriteCapacityUnits: 8192}
	if _, err := DB.CreateTable("fetch", fetch.AttributeDefinitions, fetch.KeySchema, pt, nil); err != nil {
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
