package main

import (
	"encoding/json"
	"log"
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
	for {
	receiveMessage:
		resp, err := q.ReceiveMessage([]string{"All"}, 1, 3600)
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
					d := time.Second
					if f.NumFetchesLast(d) < 1 {
						if err := f.Fetch(); err != nil {
							log.Println("err:", err)
						} else {
							count += 1
							const N = 100
							if count%N == 0 {
								stathat.PostEZCount("gingerfetch", "eikeon@eikeon.com", N)
							}
						}
					} else {
						count_throttled += 1
						log.Println("throttling:", url)
						const N = 100
						if count_throttled%N == 0 {
							stathat.PostEZCount("gingerfetch (throttled)", "eikeon@eikeon.com", N)
						}

					}
				} else {
					log.Println(err)
					continue
				}
			}
		deleteMessage:
			_, err = q.DeleteMessage(message.ReceiptHandle)
			if err != nil {
				log.Println("error deleting message:", err)
				time.Sleep(100 * time.Millisecond)
				goto deleteMessage
			}
		}
	}
}

func main() {
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

	const C = 16

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
