package client

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/updates"
	"github.com/gotd/td/tg"
	"github.com/kumneger0/cligram/internal/config"
	cligramNotifation "github.com/kumneger0/cligram/internal/notification"
	"github.com/kumneger0/cligram/internal/telegram/shared"

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
			peer, ok := peerClass.(*tg.PeerUser)
			if !ok {
				return nil
			}
			userInfo, err := shared.GetUserInfo(ctx, *Cligram.API(), peer.UserID)
			if err != nil {
				return types.NewTelegramError(types.ErrorCodeUserNotFound, err.Error(), nil)
			}

			accessHash, err := strconv.ParseInt(userInfo.AccessHash, 10, 64)

			if err != nil {
				return nil
			}

			req := &tg.MessagesGetHistoryRequest{
				Peer: &tg.InputPeerUser{
					UserID:     peer.UserID,
					AccessHash: accessHash,
				},
				Limit: 50,
			}

			history, err := Cligram.API().MessagesGetHistory(ctx, req)
			if err != nil {
				slog.Error(err.Error())
				return types.NewGetMessagesError(errors.New(peerClass.String()))
			}

			msgs, _, err := shared.GetMessageAndUserClasses(history)

			if err != nil {
				slog.Error(err.Error())
				return nil
			}

			formattedMessage := shared.FormatMessage(msg, userInfo, msgs)
			notification := types.Notification{
				NewMessage: &types.NewMessageNotification{
					Message: *formattedMessage,
					User:    userInfo,
				},
			}

			sendNewMessageNotification(userInfo.FirstName, " Sent You New Message", formattedMessage.Content)
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

func sendNewMessageNotification(sender string, title string, message string) {
	var notificationTitle string
	var notificationMessage string
	userNotificationConfig := config.GetConfig().Notifications
	if *userNotificationConfig.Enabled {
		if *userNotificationConfig.ShowMessagePreview {
			notificationTitle = fmt.Sprint(sender, title)
			notificationMessage = message
		} else {
			notificationTitle = sender
			notificationMessage = title
		}
		cligramNotifation.Notify(notificationTitle, notificationMessage)
	}
}
