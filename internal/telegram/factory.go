package telegram

import (
	"context"

	"github.com/kumneger0/cligram/internal/telegram/client"
	"github.com/kumneger0/cligram/internal/telegram/types"
)

var (
	Cligram *client.Client
)

func NewClient(ctx context.Context, updateChannel chan types.Notification, telegramAPIID, telegramAPIHash, account string) (*client.Client, error) {
	return client.NewClientFromEnv(ctx, updateChannel, telegramAPIID, telegramAPIHash, account)
}
