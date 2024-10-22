package mps

type FeedType int

const (
	FeedTypeChainConfig FeedType = iota
	FeedTypeTransactions
	FeedTypeBlockedTxHashes
	FeedTypeResponse
)

// FeedPacket 服务器向客户端推送的数据类型
type FeedPacket struct {
	T    FeedType `json:"type"`
	Data any      `json:"data"`
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
