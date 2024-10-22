package client

import (
	"sync"
	"testing"
	"time"
)

func TestSubscribe(t *testing.T) {
	c, err := New("localhost:7856")
	if err != nil {
		t.Error(err)
	}
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		c.TakeLoop()
		wg.Done()
	}()
	c.SubscribeTopicNewTx()
	go func() {
		<-time.After(10 * time.Second)
		c.UnsubscribeTopicNewTx()
	}()
	wg.Wait()
}
