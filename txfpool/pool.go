package txfpool

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"sync"
)

type Pool interface {
	Feed(txs []*types.Transaction)
	Block(hashes []common.Hash)

	//Pend(hashes []common.Hash)
	//Queue(hashes []common.Hash)
	//
	//Pending() []*types.Transaction
	//Queuing() []*types.Transaction

	All() []*types.Transaction
}

type TxfPool struct {
	lock sync.RWMutex
	all  []*types.Transaction
	//pending []*types.Transaction
	//queuing []*types.Transaction
}

func New() *TxfPool {
	return &TxfPool{
		all: make([]*types.Transaction, 0, 256),
	}
}

func (p *TxfPool) Feed(txs []*types.Transaction) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.all = append(p.all, txs...)
}

func (p *TxfPool) Block(hashes []common.Hash) {
	p.lock.Lock()
	defer p.lock.Unlock()
	lenPool := len(p.all)
	offset := 0
	for i := 0; i < lenPool; i++ {
		p.all[i-offset] = p.all[i]
		if p.all[i].Hash() == hashes[i] {
			offset++
		}
	}
	p.all = p.all[:len(p.all)-offset]
	fmt.Printf("new block(%d txs) rm %d tx from pool, left:%d \n", len(hashes), offset, len(p.all))
}

func (p *TxfPool) All() []*types.Transaction {
	p.lock.RLock()
	defer p.lock.RUnlock()
	all := make([]*types.Transaction, len(p.all))
	copy(all, p.all)
	return all
}
