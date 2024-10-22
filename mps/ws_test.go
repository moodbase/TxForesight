package mps

import (
	"github.com/ethereum/go-ethereum/log"
	"testing"
)

func TestWSServer(t *testing.T) {
	wss := newWS(":7856", log.New())
	err := wss.ListenAndServe()
	if err != nil {
		t.Error(err)
	}
}
