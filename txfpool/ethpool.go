package txfpool

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
	"sync"
	"time"
)

type ETHPool interface {
	SetSigner(signer types.Signer)

	Feed(txs []*types.Transaction) error
	Block(hashes []common.Hash)

	//Pend(hashes []common.Hash)
	//Queue(hashes []common.Hash)
	//
	//Pending() []*types.Transaction
	//Queuing() []*types.Transaction

	All() []*types.Transaction
}

type TxfETHPool struct {
	lock           sync.RWMutex
	all            []*types.Transaction
	signer         types.Signer
	untilSignerSet chan struct{}
	//pending []*types.Transaction
	//queuing []*types.Transaction
}

func New() *TxfETHPool {
	return &TxfETHPool{
		all:            make([]*types.Transaction, 0, 256),
		untilSignerSet: make(chan struct{}),
	}
}

func (p *TxfETHPool) SetSigner(signer types.Signer) {
	p.signer = signer
	close(p.untilSignerSet)
}

func (p *TxfETHPool) Feed(txs []*types.Transaction) error {
	select {
	case <-time.After(3 * time.Second):
		return errors.New("signer not set")
	case <-p.untilSignerSet:
	}
	for i, tx := range txs {
		from, err := types.Sender(p.signer, tx)
		if err != nil {
			fmt.Println("tx sender error:", err)
			continue
		}
		fmt.Println(i, tx.Hash(), tx.Nonce(), tx.Gas(), tx.GasPrice(), from, tx.To(), tx.Value())
	}
	p.lock.Lock()
	defer p.lock.Unlock()
	p.all = append(p.all, txs...)
	return nil
}

func (p *TxfETHPool) Block(hashes []common.Hash) {
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

func (p *TxfETHPool) All() []*types.Transaction {
	p.lock.RLock()
	defer p.lock.RUnlock()
	all := make([]*types.Transaction, len(p.all))
	copy(all, p.all)
	return all
}
