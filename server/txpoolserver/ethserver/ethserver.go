package ethserver

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"

	"github.com/moodbase/TxForesight/mps"
	"github.com/moodbase/TxForesight/mps/mpsclient"
	"github.com/moodbase/TxForesight/txfpool"
)

type ETHServer struct {
	ethCli *ethclient.Client
	mpsCli *mpsclient.Client

	ctx    context.Context
	cancel context.CancelFunc
	feedCh chan *mps.FeedPacket

	chainConfig *params.ChainConfig
	pool        txfpool.ETHPool
}

func New(ethEndpoint, mpsEndpoint string, pool txfpool.ETHPool) (*ETHServer, error) {
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

	return &ETHServer{
		ethCli: ethCli,
		mpsCli: mpsCli,

		ctx:    ctx,
		cancel: cancel,
		feedCh: make(chan *mps.FeedPacket, 8),

		pool: pool,
	}, nil
}

func (s *ETHServer) Start() {
	go s.mpsCli.DrainLoop(s.feedCh)
	s.packetLoop()
}

func (s *ETHServer) Stop() {
	s.cancel()
	s.mpsCli.Close()
	s.ethCli.Close()
}

func (s *ETHServer) packetLoop() {
	for {
		select {
		case packet := <-s.feedCh:
			switch packet.Type {
			case mps.FeedTypeChainConfig:
				err := json.Unmarshal(packet.Data, &s.chainConfig)
				if err != nil {
					slog.Error("invalid chain config", "err", err, "data", packet.Data)
				} else {
					slog.Info("received chain config", "config", s.chainConfig)
				}
				s.pool.SetSigner(types.LatestSigner(s.chainConfig))
			case mps.FeedTypeTransactions:
				var txs types.Transactions
				err := json.Unmarshal(packet.Data, &txs)
				if err != nil {
					slog.Error("invalid transactions", "err", err, "data", packet.Data)
				} else {
					slog.Info("received transactions", "len", len(txs))
				}
				err = s.pool.Feed(txs)
				if err != nil {
					slog.Error("feed txs failed", "err", err)
					slog.Info("retry feed txs")
					s.feedCh <- packet
				}
			case mps.FeedTypeBlockedTxHashes:
				var hashes []common.Hash
				err := json.Unmarshal(packet.Data, &hashes)
				if err != nil {
					slog.Error("invalid blocked tx hashes", "err", err, "data", packet.Data)
				} else {
					slog.Info("received blocked tx hashes:", "len", hashes)
				}
				s.pool.Block(hashes)
			case mps.FeedTypeResponse:
				var resp mps.ResponsePacket
				err := json.Unmarshal(packet.Data, &resp)
				if err != nil {
					slog.Info("invalid response", "err", err, "data", packet.Data)
				} else {
					slog.Info("received response:", "resp", resp)
				}
			default:
				slog.Info("unknown packet type", "type", packet.Type, "data", packet.Data)
			}
		case <-s.ctx.Done():
			slog.Info("packet handle loop exit")
			return
		}
	}
}
