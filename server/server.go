package server

import (
	"context"
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"

	"github.com/moodbase/TxForesight/log/fmtlog"
	"github.com/moodbase/TxForesight/mps"
	"github.com/moodbase/TxForesight/mps/mpsclient"
	"github.com/moodbase/TxForesight/txfpool"
)

type Server struct {
	ethCli *ethclient.Client
	mpsCli *mpsclient.Client

	ctx    context.Context
	cancel context.CancelFunc
	feedCh chan *mps.FeedPacket

	chainConfig *params.ChainConfig
	pool        txfpool.Pool
}

func New(ethEndpoint, mpsEndpoint string) (*Server, error) {
	ethCli, err := ethclient.Dial(ethEndpoint)
	if err != nil {
		return nil, err
	}
	mpsCli, err := mpsclient.New(mpsEndpoint)
	if err != nil {
		return nil, err
	}
	mpsCli.SubscribeTopicNewTx()
	mpsCli.SubscribeTopicBlockedTxHashes()
	ctx, cancel := context.WithCancel(context.Background())
	pool := txfpool.New()

	return &Server{
		ethCli: ethCli,
		mpsCli: mpsCli,

		ctx:    ctx,
		cancel: cancel,
		feedCh: make(chan *mps.FeedPacket, 8),

		pool: pool,
	}, nil
}

func (s *Server) Start() {
	go s.mpsCli.DrainLoop(s.feedCh)
	s.PacketLoop()
}

func (s *Server) Stop() {
	s.cancel()
	s.mpsCli.Close()
	s.ethCli.Close()
}

func (s *Server) PacketLoop() {
	for {
		select {
		case packet := <-s.feedCh:
			switch packet.Type {
			case mps.FeedTypeChainConfig:
				err := json.Unmarshal(packet.Data, &s.chainConfig)
				if err != nil {
					fmtlog.Info("err", err)
					fmtlog.Info("invalid chain config", packet.Data)
				} else {
					fmtlog.Info("received chain config:", s.chainConfig)
				}
				s.pool.SetSigner(types.LatestSigner(s.chainConfig))
			case mps.FeedTypeTransactions:
				var txs types.Transactions
				err := json.Unmarshal(packet.Data, &txs)
				if err != nil {
					fmtlog.Info("invalid transactions", packet.Data)
				} else {
					fmtlog.Info("received transactions:", txs)
				}
				err = s.pool.Feed(txs)
				if err != nil {
					fmtlog.Info("feed txs error", err)
					fmtlog.Info("retry feed txs")
					s.feedCh <- packet
				}
			case mps.FeedTypeBlockedTxHashes:
				var hashes []common.Hash
				err := json.Unmarshal(packet.Data, &hashes)
				if err != nil {
					fmtlog.Info("invalid blocked tx hashes", packet.Data)
				} else {
					fmtlog.Info("received blocked tx hashes:", hashes)
				}
				s.pool.Block(hashes)
			case mps.FeedTypeResponse:
				var resp mps.ResponsePacket
				err := json.Unmarshal(packet.Data, &resp)
				if err != nil {
					fmtlog.Info("invalid response", packet.Data)
				} else {
					fmtlog.Info("received response:", resp)
				}
			default:
				fmtlog.Info("unknown packet type", packet.Type, packet.Data)
			}
		case <-s.ctx.Done():
			fmtlog.Info("packet handle loop exit")
			return
		}
	}
}
