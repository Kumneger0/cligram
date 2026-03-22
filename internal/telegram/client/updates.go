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

	dispatcher.OnReadHistoryOutbox(func(ctx context.Context, e tg.Entities, u *tg.UpdateReadHistoryOutbox) error {
		peerClass := u.GetPeer()
		var PeerType types.ChatType
		var peerID string
		switch peer := peerClass.(type) {
		case *tg.PeerUser:
			peerID = strconv.FormatInt(peer.UserID, 10)
			PeerType = types.UserChat
		case *tg.PeerChannel:
			peerID = strconv.FormatInt(peer.ChannelID, 10)
			PeerType = types.ChannelChat
		case *tg.PeerChat:
			peerID = strconv.FormatInt(peer.ChatID, 10)
			PeerType = types.GroupChat
		default:
			slog.Warn("unknown peer type", "peer", peerClass)
			return nil
		}

		maxID := u.GetMaxID()
		notification := types.Notification{
			ReadHistoryOutbox: &types.ReadHistoryOutboxNotification{
				PeerID:   peerID,
				MaxID:    maxID,
				PeerType: PeerType,
			},
		}

		select {
		case updateChannel <- notification:
		default:
			slog.Warn("update channel is full, dropping read history outbox notification")
		}
		return nil
	})

	dispatcher.OnNewMessage(func(ctx context.Context, e tg.Entities, update *tg.UpdateNewMessage) error {
		msg, ok := update.Message.(*tg.Message)
		if !ok {
			return nil
		}
		var peerClass tg.PeerClass
		if p, ok := msg.GetFromID(); ok {
			peerClass = p
		} else if msg.Out {
			if p := msg.GetPeerID(); p != nil {
				peerClass = p
			}
		}

		if peerClass == nil {
			if p := msg.GetPeerID(); p != nil {
				peerClass = p
			}
		}

		if peerClass != nil {
			var fromID string
			switch peer := peerClass.(type) {
			case *tg.PeerUser:
				fromID = strconv.FormatInt(peer.UserID, 10)
			case *tg.PeerChannel:
				fromID = strconv.FormatInt(peer.ChannelID, 10)
			case *tg.PeerChat:
				fromID = strconv.FormatInt(peer.ChatID, 10)
			default:
				slog.Warn("unknown peer type", "peer", peerClass)
				return nil
			}

			notification := types.Notification{
				NewMessage: &types.NewMessageNotification{
					ID:      msg.GetID(),
					FromID:  fromID,
					Message: msg,
				},
			}

			select {
			case updateChannel <- notification:
			default:
				slog.Warn("update channel is full, dropping message")
			}
		} else {
			slog.Error("could not determine peer from message")
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
