package mps

import (
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/params"

	"github.com/moodbase/TxForesight/log"
)

type txSub interface {
	SubscribeTransactions(ch chan<- core.NewTxsEvent, reorgs bool) event.Subscription
}
type blockchain interface {
	SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription
	Config() *params.ChainConfig
}

// MPS (Message Pushing Service) is designed to be used as a service embedded into go-ethereum.
type MPS struct {
	pool txSub
	bc   blockchain

	txch            chan core.NewTxsEvent
	blkch           chan core.ChainHeadEvent
	stop            chan struct{}
	txSubscription  event.Subscription
	blkSubscription event.Subscription

	ws     *wsServer
	logger log.Logger
}

func New(pool txSub, bc blockchain, logger log.Logger) *MPS {
	as := &MPS{
		pool: pool,
		bc:   bc,

		txch:  make(chan core.NewTxsEvent),
		blkch: make(chan core.ChainHeadEvent),
		stop:  make(chan struct{}),

		ws:     newWS(":7856", logger, bc.Config()),
		logger: logger,
	}
	return as
}

func (s *MPS) subscribeEvents() {
	s.txSubscription = s.pool.SubscribeTransactions(s.txch, true)
	s.blkSubscription = s.bc.SubscribeChainHeadEvent(s.blkch)
}
func (s *MPS) unSubscribeEvents() {
	s.txSubscription.Unsubscribe()
	s.blkSubscription.Unsubscribe()
}

func (s *MPS) listen() error {
	return s.ws.ListenAndServe()
}

func (s *MPS) loop() {
	for {
		select {
		case txe := <-s.txch:
			s.logger.Debug("new txs", "len", len(txe.Txs))
			s.ws.DispatchNewTxsEvent(txe)
		case blke := <-s.blkch:
			s.logger.Debug("new blocks", "tx len", len(blke.Block.Transactions()))
			s.ws.DispatchChainHeadEvent(blke)
		case <-s.stop:
			return
		}
	}
}

func (s *MPS) Start() error {
	s.logger.Debug("### start mps server ###")
	s.subscribeEvents()
	go s.loop()
	go s.listen()
	return nil
}
func (s *MPS) Stop() error {
	s.logger.Debug("stop MPS")
	s.unSubscribeEvents()
	close(s.stop)
	err := s.ws.Shutdown()
	if err != nil {
		return err
	}
	return nil
}
