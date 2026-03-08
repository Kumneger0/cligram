package telegram

import (
	"context"
	"sync"

	"github.com/kumneger0/cligram/internal/telegram/client"
	"github.com/kumneger0/cligram/internal/telegram/types"
)

var (
	once    sync.Once
	Cligram *client.Client
)

func NewClient(ctx context.Context, updateChannel chan types.Notification, telegramAPIID, telegramAPIHash string) (*client.Client, error) {
	var err error
	once.Do(func() {
		c, e := client.NewClientFromEnv(ctx, updateChannel, telegramAPIID, telegramAPIHash)
		Cligram = c
		err = e
	})
	return Cligram, err
}
