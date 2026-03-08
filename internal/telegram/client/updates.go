package client

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/updates"
	"github.com/gotd/td/tg"
	"github.com/kumneger0/cligram/internal/config"
	"github.com/kumneger0/cligram/internal/telegram/shared"

	cligramNotification "github.com/kumneger0/cligram/internal/notification"
	"github.com/kumneger0/cligram/internal/telegram/types"
)

func newUpdateHandler(updateChannel chan types.Notification) telegram.UpdateHandler {
	dispatcher := tg.NewUpdateDispatcher()
	dispatcher.OnNewChannelMessage(func(ctx context.Context, e tg.Entities, update *tg.UpdateNewChannelMessage) error {
		msg, ok := update.Message.(*tg.Message)
		if !ok {
			return nil
		}

		if peerClass, ok := msg.GetFromID(); ok {
			slog.Debug("received channel message", "peer", peerClass)
			return nil
		}
		return nil
	})

	dispatcher.OnNewMessage(func(ctx context.Context, e tg.Entities, update *tg.UpdateNewMessage) error {
		msg, ok := update.Message.(*tg.Message)
		if !ok {
			return nil
		}
		if peerClass, ok := msg.GetFromID(); ok {
			var fromID string
			notificationContent := "You have Received New Message"

			switch peer := peerClass.(type) {
			case *tg.PeerUser:
				fromID = strconv.FormatInt(peer.UserID, 10)
				userInfo, err := shared.GetUserInfo(ctx, *Cligram.API(), peer.UserID)
				if err != nil {
					slog.Error(err.Error())
				} else if userInfo != nil {
					notificationContent = fmt.Sprintf("%s Sent You New Message", userInfo.FirstName)
				}
			case *tg.PeerChannel:
				fromID = strconv.FormatInt(peer.ChannelID, 10)
			case *tg.PeerChat:
				fromID = strconv.FormatInt(peer.ChatID, 10)
			default:
				return nil
			}

			notification := types.Notification{
				NewMessage: &types.NewMessageNotification{
					ID:      msg.GetID(),
					FromID:  fromID,
					Message: msg,
				},
			}

			sendNewMessageNotification(" ", notificationContent, msg.Message)
			select {
			case updateChannel <- notification:
			default:
				slog.Warn("update channel is full, dropping message")
			}
		}
		return nil
	})

	dispatcher.OnUserTyping(func(ctx context.Context, e tg.Entities, update *tg.UpdateUserTyping) error {
		userID := update.UserID
		userInfo, err := shared.GetUserInfo(ctx, *Cligram.API(), userID)
		if err != nil {
			return types.NewTelegramError(types.ErrorCodeUserNotFound, err.Error(), nil)
		}

		userInfo.IsOnline = true
		userInfo.IsTyping = true
		notification := types.Notification{
			UserTyping: &types.UserTypingNotification{
				User: *userInfo,
			},
		}

		select {
		case updateChannel <- notification:
		default:
			slog.Warn("update channel is full, dropping typing notification")
		}
		return nil
	})

	dispatcher.OnUserStatus(func(ctx context.Context, e tg.Entities, update *tg.UpdateUserStatus) error {
		userID := update.UserID
		userInfo, err := shared.GetUserInfo(ctx, *Cligram.API(), userID)
		if err != nil {
			return types.NewTelegramError(types.ErrorCodeUserNotFound, err.Error(), nil)
		}

		if userInfo == nil {
			return types.NewUserNotFoundError(userID)
		}

		var lastSeen time.Time
		var isOnline bool

		switch status := update.Status.(type) {
		case *tg.UserStatusOffline:
			lastSeen = time.Unix(int64(status.WasOnline), 0)
			isOnline = false
		case *tg.UserStatusOnline:
			lastSeen = time.Now()
			isOnline = true
		default:
			lastSeen = time.Time{}
			isOnline = false
		}

		userInfo.IsOnline = isOnline

		notification := types.Notification{
			UserStatus: &types.UserStatusNotification{
				UserInfo: *userInfo,
				Status: types.UserStatus{
					IsOnline: isOnline,
					LastSeen: lastSeen,
				},
			},
		}

		select {
		case updateChannel <- notification:
		default:
			slog.Warn("update channel is full, dropping status notification")
		}
		return nil
	})

	return updates.New(updates.Config{
		Handler: dispatcher,
	})
}

func sendNewMessageNotification(sender string, titleSuffix string, message string) {
	cfg := config.GetConfig().Notifications
	enabled := true
	if cfg.Enabled != nil {
		enabled = *cfg.Enabled
	}
	if !enabled {
		return
	}
	showPreview := true
	if cfg.ShowMessagePreview != nil {
		showPreview = *cfg.ShowMessagePreview
	}
	var notificationTitle, notificationMessage string
	if showPreview {
		notificationTitle = fmt.Sprintf("%s %s", sender, titleSuffix)
		notificationMessage = message
	} else {
		notificationTitle = sender
		notificationMessage = titleSuffix
	}
	cligramNotification.Notify(notificationTitle, notificationMessage)
}
