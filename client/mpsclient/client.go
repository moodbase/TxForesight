package mpsclient

import (
	"fmt"
	"github.com/ethereum/go-ethereum/log"
	"github.com/gorilla/websocket"
	"github.com/moodbase/TxForesight/mps"
	"net/url"
	"time"
)

type Client struct {
	conn *websocket.Conn
}

func New(addr string) (*Client, error) {
	u := url.URL{Scheme: "ws", Host: addr, Path: "/"}
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, err
	}
	return &Client{conn}, nil
}

func (c *Client) Close() {
	c.conn.Close()
}

// DrainLoop relay packets from conn
func (c *Client) DrainLoop(ch chan<- *mps.FeedPacket) {
	for {
		var packet mps.FeedPacket
		err := c.conn.ReadJSON(&packet)
		if err != nil {
			if !websocket.IsCloseError(err) {
				fmt.Println(err)
			} else {
				fmt.Println("conn close")
				c.conn.Close()
			}
			return
		}
		ch <- &packet
	}
}

func (c *Client) subscribeTopic(t mps.Topic) {
	req := mps.RequestPacket{
		Op:    mps.ClientOptSubscribe,
		Id:    int(time.Now().Unix()),
		Topic: t,
	}
	err := c.conn.WriteJSON(req)
	if err != nil {
		log.Error(err.Error())
	}
}
func (c *Client) SubscribeTopicNewTx() {
	c.subscribeTopic(mps.TopicNewTx)
}
func (c *Client) SubscribeTopicBlockedTxHashes() {
	c.subscribeTopic(mps.TopicBlockedTxHashes)
}

func (c *Client) unsubscribeTopic(t mps.Topic) error {
	req := mps.RequestPacket{
		Op:    mps.ClientOptUnsubscribe,
		Id:    int(time.Now().Unix()),
		Topic: t,
	}
	return c.conn.WriteJSON(req)
}
func (c *Client) UnsubscribeTopicNewTx() error {
	return c.unsubscribeTopic(mps.TopicNewTx)
}
func (c *Client) UnsubscribeTopicBlockedTxHashes() error {
	return c.unsubscribeTopic(mps.TopicBlockedTxHashes)
}
