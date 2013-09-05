package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/eikeon/ginger"
	"github.com/stathat/go"
	"launchpad.net/goamz/aws"
	"launchpad.net/~prudhvikrishna/goamz/sqs"
)

func fetcher(q *sqs.Queue) {
	count := 0
	count_throttled := 0
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for {
	receiveMessage:
		delay := 30 + r.Intn(100)
		resp, err := q.ReceiveMessage([]string{"All"}, 1, delay)
		if err != nil {
			log.Println("Error receiving message:", err)
			time.Sleep(time.Second)
			goto receiveMessage
		}
		if len(resp.Messages) == 0 {
			log.Println("No messages in queue. We're done.")
			break
		}
		for _, message := range resp.Messages {
			var msg struct {
				Message string
			}
			if err = json.NewDecoder(strings.NewReader(message.Body)).Decode(&msg); err != nil {
				log.Println(err)
			} else {
				url := msg.Message
				log.Println("url:", url)

				if f, err := ginger.NewFetch(url); err == nil {
					if f.NumFetchesLast(time.Second) < 1 {
						f.Fetch()
						count += 1
						const N = 100
						if count%N == 0 {
							go stathat.PostEZCount("gingerfetch", "eikeon@eikeon.com", N)
						}
						if err := f.Put(); err != nil {
							log.Println("Error putting fetch:", err)
						}
					deleteMessage:
						_, err = q.DeleteMessage(message.ReceiptHandle)
						if err != nil {
							log.Println("error deleting message:", err)
							time.Sleep(100 * time.Millisecond)
							goto deleteMessage
						}
					} else {
						count_throttled += 1
						log.Println("throttling:", url)
						const N = 100
						if count_throttled%N == 0 {
							go stathat.PostEZCount("gingerfetch (throttled)", "eikeon@eikeon.com", N)
						}

					}
				} else {
					log.Println("could not create new fetch:", err)
				}
			}
		}
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	_ = ginger.NewMemoryGinger(true)

auth:
	auth, err := aws.EnvAuth()
	if err != nil {
		log.Println("EnvAuth Failed:", err)
		time.Sleep(time.Second)
		goto auth
	}

	ssqs := sqs.New(auth, aws.USEast)

getQueue:
	q, err := ssqs.GetQueue("ginger-fetchrequests-test")
	if err != nil {
		log.Println("error getting queue:", err)
		time.Sleep(10 * time.Second)
		goto getQueue
	}

	const C = 64

	var wg sync.WaitGroup
	wg.Add(C)
	for i := 0; i < C; i++ {
		go func() {
			fetcher(q)
			wg.Done()

		}()
	}
	wg.Wait()
}
