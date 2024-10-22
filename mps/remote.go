package mps

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/gorilla/websocket"
	"github.com/moodbase/TxForesight/log"
	"github.com/pkg/errors"
	"io"
	"sync"
)

var ErrConnClosed = errors.New("ws conn closed")

var supportTopics map[Topic]bool

func init() {
	supportTopics = map[Topic]bool{
		TopicNewTx:           true,
		TopicBlockedTxHashes: true,
	}
}

// Remote represents one conn instance
type Remote struct {
	c          *websocket.Conn
	subscribed map[Topic]bool

	feedCh chan struct {
		packet FeedPacket
		errCh  chan error
	}
	stopCh    chan struct{}
	closeOnce sync.Once

	logger log.Logger
}

func NewRemote(c *websocket.Conn, logger log.Logger) *Remote {
	return &Remote{
		c:          c,
		subscribed: make(map[Topic]bool),
		feedCh: make(chan struct {
			packet FeedPacket
			errCh  chan error
		}),
		stopCh: make(chan struct{}),
		logger: logger,
	}
}

// feed put packet into conn
func (r *Remote) feed(packet FeedPacket) chan error {
	errCh := make(chan error)
	r.feedCh <- struct {
		packet FeedPacket
		errCh  chan error
	}{packet, errCh}
	return errCh
}

func (r *Remote) FeedChainConfig(config *params.ChainConfig) error {
	return <-r.feed(FeedPacket{
		Type: FeedTypeChainConfig,
		Data: config,
	})
}

// FeedResponse respond to client requests
func (r *Remote) FeedResponse(id int, ok bool, msg string) error {
	return <-r.feed(FeedPacket{
		Type: FeedTypeResponse,
		Data: ResponsePacket{id, ok, msg},
	})
}

// FeedNewTx relay new tx event to clients
func (r *Remote) FeedNewTx(txs types.Transactions) error {
	return <-r.feed(FeedPacket{
		Type: FeedTypeTransactions,
		Data: txs,
	})
}

// FeedBlockedTxHash relay new blocked tx hash to clients
func (r *Remote) FeedBlockedTxHash(hashes []common.Hash) error {
	return <-r.feed(FeedPacket{
		Type: FeedTypeBlockedTxHashes,
		Data: hashes,
	})
}

func (r *Remote) sendLoop() {
	for {
		select {
		case <-r.stopCh:
			break
		case m := <-r.feedCh:
			err := r.c.WriteJSON(m.packet)
			m.errCh <- err
		}
	}
}

func (r *Remote) onSubscribe(topic Topic, id int) error {
	if supportTopics[topic] {
		r.subscribed[topic] = true
		return r.FeedResponse(id, true, "subscribed topic: "+string(topic))
	}
	return r.FeedResponse(id, false, "unknown topic :"+string(topic))
}

// onUnsubscribe always respond ok
func (r *Remote) onUnsubscribe(topic Topic, id int) error {
	delete(r.subscribed, topic)
	return r.FeedResponse(id, true, "unsubscribed topic (unchecked): "+string(topic))
}

func (r *Remote) RecvLoop() error {
	for {
		var req RequestPacket
		err := r.c.ReadJSON(&req)
		if err != nil {
			r.logger.Error("ws Request decode", "err", err, "remote", r.c.RemoteAddr())
			err = r.FeedResponse(-1, false, err.Error())
			if err != nil {
				return err
			}
		}
		switch req.Op {
		case ClientOptSubscribe:
			err = r.onSubscribe(req.Topic, req.Id)
			if err != nil {
				return err
			}
		case ClientOptUnsubscribe:
			err = r.onUnsubscribe(req.Topic, req.Id)
			if err != nil {
				return err
			}
		default:
			r.logger.Error("Announce MPS: unsupported request type", "remote", r.c.RemoteAddr())
		}
	}
}

// Serve blocked until conn closed
func (r *Remote) Serve(cfg *params.ChainConfig) error {
	go r.sendLoop()
	err := r.FeedChainConfig(cfg)
	if err != nil {
		return err
	}
	err = r.RecvLoop()
	if !errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
		return ErrConnClosed
	}
	return nil
}

func (r *Remote) Stop() {
	// TODO: optimize
	r.closeOnce.Do(func() {
		r.logger.Debug("Closing websocket connection", "remote", r.c.RemoteAddr())
		r.c.Close()
		close(r.stopCh)
	})
}
