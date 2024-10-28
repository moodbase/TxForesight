package server

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/moodbase/TxForesight/txfpool/ethpool"
	"log/slog"
	"net/http"
	"sync"

	"github.com/moodbase/TxForesight/server/txpoolserver/ethserver"
)

type ChainTag string

const (
	ETH ChainTag = "eth"
)

type Server struct {
	r            *gin.Engine
	httpListener *http.Server

	ethPools       map[ChainTag]ethpool.Pool
	ethPoolServers map[ChainTag]*ethserver.ETHServer

	wg sync.WaitGroup
}

func New(listenAddr string) *Server {
	r := gin.Default()

	s := &Server{
		r: r,
		httpListener: &http.Server{
			Addr:    listenAddr,
			Handler: r,
		},
		ethPools:       make(map[ChainTag]ethpool.Pool),
		ethPoolServers: make(map[ChainTag]*ethserver.ETHServer),
	}
	s.Register()
	return s
}

func (s *Server) Start() error {
	for tag, server := range s.ethPoolServers {
		slog.Info("start eth txPool server", "chain", tag)
		go server.Start()
		s.wg.Add(1)
	}
	if len(s.ethPoolServers) == 0 {
		return errors.New("no txPool server registered")
	}
	go s.httpListen()
	return nil
}

func (s *Server) Stop() {
	for tag, ethServer := range s.ethPoolServers {
		go func() {
			ethServer.Stop()
			slog.Info("stopped eth txPool server", "chain", tag)
			s.wg.Done()
		}()
	}
	s.httpShutdown()
	s.wg.Wait()
}

func (s *Server) registerETHServer(tag ChainTag, rpcAddr, mpsAddr string) error {
	pool := ethpool.NewTxfPool()
	ethServer, err := ethserver.New(rpcAddr, mpsAddr, pool)
	if err != nil {
		return err
	}
	s.ethPoolServers[tag] = ethServer
	s.ethPools[tag] = pool
	return nil
}

func (s *Server) Register() {
	err := s.registerETHServer(ETH, "http://localhost:8545", "localhost:7856")
	if err != nil {
		slog.Error("failed to register eth server", "err", err)
	}
	for tag, _ := range s.ethPoolServers {
		s.routeETH(tag)
	}
}
