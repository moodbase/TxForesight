package mps

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type FeedType int

const (
	FeedTypeChainConfig FeedType = iota
	FeedTypeTransactions
	FeedTypeBlockedTxHashes
	FeedTypeResponse // response to client subscription requests
)

// TxsWithSender is a wrapper of transactions and their senders,
// used to send txs packet to mps client
type TxsWithSender struct {
	Txs     types.Transactions `json:"txs"`
	Senders []*common.Address  `json:"senders"`
}

// FeedPacket is the packet sent to mps client
type FeedPacket struct {
	Type FeedType `json:"type"`
	Data []byte   `json:"data"`
}

type RequestPacket struct {
	Id    int       `json:"id"`
	Op    ClientOpt `json:"opt"`
	Topic Topic     `json:"topic"`
}

type ResponsePacket struct {
	// quest id
	Id      int    `json:"id"`
	Ok      bool   `json:"ok"`
	Message string `json:"message"`
}

type ClientOpt int

const (
	ClientOptSubscribe ClientOpt = iota
	ClientOptUnsubscribe
)

type Topic string

const (
	TopicNewTx           Topic = "newTx"
	TopicBlockedTxHashes       = "blockedTxHashes"
)
