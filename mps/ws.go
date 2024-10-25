package mps

import (
	"context"
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/gorilla/websocket"
	"github.com/moodbase/TxForesight/log"
	"net/http"
	"sync"
)

var upgrader = websocket.Upgrader{}

type wsServer struct {
	srv      *http.Server
	conns    map[string]*Remote
	connLock sync.RWMutex

	logger log.Logger

	chainConfig *params.ChainConfig
	signer      types.Signer
}

func newWS(addr string, logger log.Logger, chainConfig *params.ChainConfig) *wsServer {
	w := &wsServer{
		conns:       make(map[string]*Remote),
		logger:      logger,
		chainConfig: chainConfig,
		signer:      types.LatestSigner(chainConfig),
	}
	srv := &http.Server{
		Addr:    addr,
		Handler: w,
	}
	w.srv = srv
	return w
}

func (s *wsServer) DispatchNewTxsEvent(e core.NewTxsEvent) {
	s.connLock.RLock()
	defer s.connLock.RUnlock()
	senders := make([]*common.Address, len(e.Txs))
	for i, tx := range e.Txs {
		from, err := types.Sender(s.signer, tx)
		if err != nil {
			s.logger.Error("tx sender parse error:", err)
			continue
		}
		senders[i] = &from
	}
	for addr, conn := range s.conns {
		if !conn.subscribed[TopicNewTx] {
			continue
		}
		err := conn.FeedNewTx(TxsWithSender{e.Txs, senders})
		if err != nil {
			s.logger.Error(err.Error(), "addr", addr)
		}
	}
}

func (s *wsServer) DispatchChainHeadEvent(e core.ChainHeadEvent) {
	s.connLock.RLock()
	defer s.connLock.RUnlock()
	hashes := make([]common.Hash, len(e.Block.Transactions()))
	for i, tx := range e.Block.Transactions() {
		hashes[i] = tx.Hash()
	}
	for addr, conn := range s.conns {
		if !conn.subscribed[TopicBlockedTxHashes] {
			continue
		}
		err := conn.FeedBlockedTxHash(hashes)
		if err != nil {
			s.logger.Error(err.Error(), "addr", addr)
		}
	}
}

func (s *wsServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("wsServer upgrade:", err)
	}
	s.logger.Info("new conn:", "addr", c.RemoteAddr())
	conn := NewRemote(c, s.logger)
	s.AddConn(conn)
}

func (s *wsServer) AddConn(r *Remote) {
	s.connLock.Lock()
	defer s.connLock.Unlock()
	s.conns[r.c.RemoteAddr().String()] = r

	go func() {
		err := r.Serve(s.chainConfig)
		if err != nil {
			if !errors.Is(err, ErrConnClosed) {
				s.RemoveConn(r, false)
				s.logger.Error(err.Error())
			} else {
				s.RemoveConn(r, true)
				s.logger.Info(err.Error())
			}
		} else {
			s.logger.Warn("r goroutine stops strangely, it will never happen under normal cases")
		}
	}()
}

func (s *wsServer) RemoveConn(conn *Remote, closeConn bool) {
	if closeConn {
		conn.Stop()
	}
	s.connLock.Lock()
	defer s.connLock.Unlock()
	delete(s.conns, conn.c.RemoteAddr().String())
}

func (s *wsServer) ListenAndServe() error {
	return s.srv.ListenAndServe()
}

func (s *wsServer) Shutdown() error {
	s.connLock.Lock()
	defer s.connLock.Unlock()
	for _, conn := range s.conns {
		conn.Stop()
	}
	return s.srv.Shutdown(context.Background())
}
