package ethserver

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/moodbase/TxForesight/client/mpsclient"
	"github.com/moodbase/TxForesight/mps"
	"github.com/moodbase/TxForesight/txfpool/ethpool"
)

type ETHServer struct {
	ethCli *ethclient.Client
	mpsCli *mpsclient.Client

	ctx    context.Context
	cancel context.CancelFunc
	feedCh chan *mps.FeedPacket

	chainConfigJsonData []byte
	pool                ethpool.Pool
}

func New(ethEndpoint, mpsEndpoint string, pool ethpool.Pool) (*ETHServer, error) {
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

func (s *ETHServer) ChainConfig() []byte {
	return s.chainConfigJsonData
}

func (s *ETHServer) packetLoop() {
	for {
		select {
		case packet := <-s.feedCh:
			switch packet.Type {
			case mps.FeedTypeChainConfig:
				s.chainConfigJsonData = packet.Data
				slog.Info("received chain config json data", "data[unverified]", string(s.chainConfigJsonData))
			case mps.FeedTypeTransactions:
				var txsWithSender mps.TxsWithSender
				err := json.Unmarshal(packet.Data, &txsWithSender)
				if err != nil {
					slog.Error("invalid transactions", "err", err, "data", packet.Data)
				} else {
					slog.Info("received transactions", "len", len(txsWithSender.Txs))
				}
				s.pool.Feed(&txsWithSender)
			case mps.FeedTypeBlockedTxHashes:
				var hashes []common.Hash
				err := json.Unmarshal(packet.Data, &hashes)
				if err != nil {
					slog.Error("invalid blocked tx hashes", "err", err, "data", packet.Data)
				} else {
					slog.Info("received blocked tx hashes:", "len", len(hashes))
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
