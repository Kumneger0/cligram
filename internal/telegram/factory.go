package telegram

import (
	"context"
	"sync"

	"github.com/kumneger0/cligram/internal/telegram/client"
	"github.com/kumneger0/cligram/internal/telegram/types"
)

var (
	once sync.Once
)

var Cligram *client.Client

func NewClient(ctx context.Context, updateChannel chan types.Notification) (*client.Client, error) {
	var err error
	once.Do(func() {
		c, e := client.NewClientFromEnv(ctx, updateChannel)
		Cligram = c
		err = e
	})
	return Cligram, err
}

type PeerInfo struct {
	PeerID     string         `json:"peerId"`
	AccessHash string         `json:"accessHash"`
	ChatType   types.ChatType `json:"chatType"`
}

type PeerInfoParams struct {
	PeerID                      string         `json:"peerId"`
	AccessHash                  string         `json:"accessHash"`
	UserFirstNameOrChannelTitle string         `json:"title"`
	ChatType                    types.ChatType `json:"chatType"`
}

type ChannelAndGroupInfo = types.ChannelInfo

func GetChannelID(c ChannelAndGroupInfo) string {
	return c.ID
}
