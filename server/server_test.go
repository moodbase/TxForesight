package server

import (
	"testing"
	"time"
)

func TestRunServer(t *testing.T) {
	s, err := New("http://localhost:8545", "localhost:7856")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		<-time.After(60 * time.Second)
		s.Stop()
	}()
	s.Start()

}
