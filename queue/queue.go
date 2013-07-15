package queue

import (
	"encoding/json"
	"errors"
)

type Queue interface {
	Send(message interface{}) error
	Receive(message interface{}) error
}

// channel based queue implementation
type ChannelQueue struct {
	messages chan string
}

func NewChannelQueue(messages chan string) Queue {
	if messages == nil {
		messages = make(chan string, 100)
	}
	return &ChannelQueue{messages}
}

func (q *ChannelQueue) Send(message interface{}) error {
	b, err := json.Marshal(message)
	if err != nil {
		return err
	}
	q.messages <- string(b)
	return nil
}

func (q *ChannelQueue) Receive(i interface{}) error {
	message := <-q.messages
	if message == "" {
		return errors.New("empty")
	}
	if err := json.Unmarshal([]byte(message), &i); err != nil {
		return err
	}
	return nil
}
