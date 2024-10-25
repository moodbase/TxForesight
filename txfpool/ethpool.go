package txfpool

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/moodbase/TxForesight/mps"
	"log/slog"
	"math/big"
	"sync"
)

type ETHPoolTx struct {
	Raw      *types.Transaction `json:"raw"`
	Hash     common.Hash        `json:"hash"`
	Nonce    uint64             `json:"nonce"`
	UnixTime int64              `json:"unixTime"`
	Gas      uint64             `json:"gas"`
	GasPrice *big.Int           `json:"gasPrice"`
	From     *common.Address    `json:"from"`
	To       *common.Address    `json:"to"`
	Value    *big.Int           `json:"value"`
}

type ETHPool interface {
	Feed(txs *mps.TxsWithSender)
	Block(hashes []common.Hash)

	//Pend(hashes []common.Hash)
	//Queue(hashes []common.Hash)
	//
	//Pending() []*types.Transaction
	//Queuing() []*types.Transaction

	All(page, pageSize int) (selected []*ETHPoolTx, total int)
}

type TxfETHPool struct {
	lock sync.RWMutex
	all  []*ETHPoolTx
	//pending []*types.Transaction
	//queuing []*types.Transaction
}

func NewTxfETHPool() *TxfETHPool {
	return &TxfETHPool{
		all: make([]*ETHPoolTx, 0, 256),
	}
}

func (p *TxfETHPool) Feed(txsWithSender *mps.TxsWithSender) {
	txs := make([]*ETHPoolTx, len(txsWithSender.Txs))
	for i, tx := range txsWithSender.Txs {
		txs[i] = &ETHPoolTx{
			Raw:      tx,
			Hash:     tx.Hash(),
			Nonce:    tx.Nonce(),
			UnixTime: tx.Time().Unix(),
			Gas:      tx.Gas(),
			GasPrice: tx.GasPrice(),
			From:     txsWithSender.Senders[i],
			To:       tx.To(),
			Value:    tx.Value(),
		}
	}
	p.lock.Lock()
	defer p.lock.Unlock()
	p.all = append(p.all, txs...)
}

func (p *TxfETHPool) Block(hashes []common.Hash) {
	toRm := make(map[common.Hash]bool, len(hashes))
	for _, hash := range hashes {
		toRm[hash] = true
	}

	p.lock.Lock()
	defer p.lock.Unlock()
	lenPool := len(p.all)
	offset := 0
	for i := 0; i < lenPool; i++ {
		p.all[i-offset] = p.all[i]
		if toRm[p.all[i].Hash] {
			offset++
		}
	}
	p.all = p.all[:len(p.all)-offset]
	slog.Info("new block rm transactions from pool", "size", lenPool, "removed", offset, "remain", len(p.all))
}

func pageInfo(page, pageSize, total int) (start, end int) {
	if page < 1 {
		page = 1
	}
	start = (page - 1) * pageSize
	if start > total {
		start = total
		return start, start
	}
	end = page * pageSize
	if end > total {
		end = total
	}
	return start, end
}

func (p *TxfETHPool) All(page, pageSize int) (selected []*ETHPoolTx, total int) {
	total = len(p.all)
	start, end := pageInfo(page, pageSize, total)
	if start == end {
		return make([]*ETHPoolTx, 0), total
	}
	p.lock.RLock()
	selected = make([]*ETHPoolTx, end-start)
	copy(selected, p.all)
	p.lock.RUnlock()
	return selected, total
}
