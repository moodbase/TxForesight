package ethpool

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/moodbase/TxForesight/mps"
	"log/slog"
	"math/big"
	"slices"
	"sync"
)

type PoolTx struct {
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

func (tx *PoolTx) String() string {
	from := "_"
	if tx.From != nil {
		from = tx.From.String()
	}
	to := "_"
	if tx.To != nil {
		to = tx.To.String()
	}
	var value int64 = 0
	if tx.Value != nil {
		value = tx.Value.Int64()
	}
	return fmt.Sprintf("tx: nonce=%d from=%s value=%d to=%s", tx.Nonce, from, value, to)
}

type Pool interface {
	Feed(txs *mps.TxsWithSender)
	Block(hashes []common.Hash)

	//Pend(hashes []common.Hash)
	//Queue(hashes []common.Hash)
	//
	//Pending() []*types.Transaction
	//Queuing() []*types.Transaction

	All(page, pageSize int) (selected []*PoolTx, total int)
}

type TxfPool struct {
	lock sync.RWMutex
	all  []*PoolTx
	//pending []*types.Transaction
	//queuing []*types.Transaction
}

func NewTxfPool() *TxfPool {
	return &TxfPool{
		all: make([]*PoolTx, 0, 256),
	}
}

func (p *TxfPool) Feed(txsWithSender *mps.TxsWithSender) {
	txs := make([]*PoolTx, len(txsWithSender.Txs))
	for i, tx := range txsWithSender.Txs {
		txs[i] = &PoolTx{
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
	// NOTE: the time of transactions is not guaranteed to be in order
	// we may sort it when necessary
	p.all = append(p.all, txs...)
}

func (p *TxfPool) Block(hashes []common.Hash) {
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

func (p *TxfPool) All(page, pageSize int) (selected []*PoolTx, total int) {
	total = len(p.all)
	start, end := pageInfo(page, pageSize, total)
	if start == end {
		return make([]*PoolTx, 0), total
	}
	p.lock.RLock()
	selected = make([]*PoolTx, end-start)
	// respond latest transactions first
	// the order of transactions is assumed to be in the order of timestamp,
	// so we simply selected from tail to head and reverse it
	copy(selected, p.all[total-end:total-start])
	p.lock.RUnlock()
	slices.Reverse(selected)
	return selected, total
}
