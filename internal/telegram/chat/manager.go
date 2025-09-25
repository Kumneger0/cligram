package chat

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gotd/td/tg"

	"github.com/kumneger0/cligram/internal/telegram/shared"
	"github.com/kumneger0/cligram/internal/telegram/types"
)

type Manager struct {
	client APIClient
}

type APIClient interface {
	GetAPI() *tg.Client
	Context() context.Context
}

func NewManager(client APIClient) types.ChatManager {
	return &Manager{
		client: client,
	}
}
func (m *Manager) GetUserChats(ctx context.Context, isBot bool, offsetDate, offsetID int) (types.GetUserChatsResult, error) {
	dialogsSlice, err := m.getAllDialogs(ctx, offsetDate, offsetID)
	if err != nil {
		return types.GetUserChatsResult{
			Data:       nil,
			OffsetDate: 0,
			OffsetID:   0,
		}, types.NewTelegramError(types.ErrorCodeGetMessagesFailed, "failed to get dialogs", err)
	}

	if dialogsSlice == nil {
		return types.GetUserChatsResult{
			Data:       []types.UserInfo{},
			OffsetDate: 0,
			OffsetID:   0,
		}, nil
	}

	var userPeerIDs []userPeerIDUnreadCount
	for _, userClass := range dialogsSlice.Dialogs {
		if dialog, ok := userClass.(*tg.Dialog); ok {
			if peer, ok := dialog.Peer.(*tg.PeerUser); ok {
				userPeerIDs = append(userPeerIDs, userPeerIDUnreadCount{
					unreadCount: dialog.UnreadCount,
					peerID:      peer.UserID,
				})
			}
		}
	}
	var users []types.UserInfo
	for _, userClass := range dialogsSlice.Users {
		if tgUser, ok := userClass.(*tg.User); ok && tgUser.Bot == isBot {
			user := shared.ConvertTGUserToUserInfo(tgUser)
			unreadCount := getUnreadCount(userPeerIDs, tgUser.ID)
			user.UnreadCount = unreadCount
			users = append(users, *user)
		}
	}

	return types.GetUserChatsResult{
		Data:       users,
		OffsetDate: dialogsSlice.OffsetDate,
		OffsetID:   dialogsSlice.OffsetID,
	}, nil
}

func (m *Manager) GetChannels(ctx context.Context, isBroadCast bool, offsetDate, offsetID int) (types.GetChannelsResult, error) {
	dialogsSlice, err := m.getAllDialogs(ctx, offsetDate, offsetID)
	if err != nil {
		return types.GetChannelsResult{
			Data:       nil,
			OffsetDate: 0,
			OffsetID:   0,
		}, types.NewTelegramError(types.ErrorCodeGetMessagesFailed, "failed to get dialogs", err)
	}

	if dialogsSlice == nil {
		return types.GetChannelsResult{
			Data:       []types.ChannelInfo{},
			OffsetDate: 0,
			OffsetID:   0,
		}, nil
	}

	var channels []types.ChannelInfo
	for _, chatClass := range dialogsSlice.Chats {
		if channel, ok := chatClass.(*tg.Channel); ok && channel.Broadcast == isBroadCast {
			channelInfo := convertTGChannelToChannelInfo(channel)
			channels = append(channels, *channelInfo)
		}
	}

	return types.GetChannelsResult{
		Data:       channels,
		OffsetDate: dialogsSlice.OffsetDate,
		OffsetID:   dialogsSlice.OffsetID,
	}, nil
}

func (m *Manager) GetChannelInfo(ctx context.Context, peer types.Peer) (*types.ChannelInfo, error) {
	channelID, err := strconv.ParseInt(peer.ID, 10, 64)
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}
	accessHash, err := strconv.ParseInt(peer.AccessHash, 10, 64)
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}
	channelPeer := &tg.InputChannel{
		ChannelID:  channelID,
		AccessHash: accessHash,
	}
	chatClass, err := m.client.GetAPI().ChannelsGetChannels(ctx, []tg.InputChannelClass{channelPeer})
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}
	messageChatClass, ok := chatClass.(*tg.MessagesChats)
	if !ok {
		slog.Error("we have failed to get the type for the chat")
		return nil, errors.New("we have failed to get the type for the chat")
	}
	if len(messageChatClass.Chats) == 0 {
		slog.Error("we have failed to get any channels")
		return nil, errors.New("we have failed to get any channels")
	}

	chat, ok := messageChatClass.Chats[0].(*tg.Channel)

	if !ok {
		slog.Error("we have faild to cast they type")
		return nil, errors.New("we have faild to cast they type")
	}

	return &types.ChannelInfo{
		ChannelTitle:      chat.Title,
		Username:          &chat.Username,
		ID:                peer.ID,
		AccessHash:        strconv.FormatInt(chat.AccessHash, 10),
		IsCreator:         chat.Creator,
		IsBroadcast:       chat.Broadcast,
		ParticipantsCount: &chat.ParticipantsCount,
	}, nil
}

func (m *Manager) GetChatHistory(ctx context.Context, peer types.Peer, limit int, offsetID *int) ([]types.FormattedMessage, error) {
	inputPeer, err := convertPeerToInputPeer(peer)
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}

	req := &tg.MessagesGetHistoryRequest{
		Peer:  inputPeer,
		Limit: limit,
	}
	if offsetID != nil {
		req.OffsetID = *offsetID
	}

	history, err := m.client.GetAPI().MessagesGetHistory(ctx, req)
	if err != nil {
		slog.Error(err.Error())
		return nil, types.NewGetMessagesError(err)
	}

	msgs, users, err := shared.GetMessageAndUserClasses(history)
	if err != nil {
		return nil, err
	}

	var userInfo *types.UserInfo
	var channel *types.ChannelInfo

	areWeInUserModeOrBotMode := peer.ChatType == types.BotChat || peer.ChatType == types.UserChat

	if areWeInUserModeOrBotMode {
		if id, err := strconv.ParseInt(peer.ID, 10, 64); err == nil {
			userInfo = getUser(users, id)
		} else {
			slog.Error("invalid peer.ID", "peerID", peer.ID, "err", err)
		}
	}

	var formattedMessages []types.FormattedMessage
	for _, msg := range msgs {
		msg, ok := msg.(*tg.Message)
		if !ok {
			continue
		}
		if areWeInUserModeOrBotMode {
			formattedMessages = append(formattedMessages, *shared.FormatMessage(msg, userInfo, msgs))
		} else if peer.ChatType == types.GroupChat {
			fromID, ok := msg.FromID.(*tg.PeerUser)
			if ok {
				userInfo := getUser(users, fromID.UserID)
				formattedMessages = append(formattedMessages, *shared.FormatMessage(msg, userInfo, msgs))
			} else {
				formattedMessages = append(formattedMessages, *shared.FormatMessage(msg, channel, msgs))
			}
		} else {
			formattedMessages = append(formattedMessages, *shared.FormatMessage(msg, channel, msgs))
		}
	}

	slices.Reverse(formattedMessages)
	return formattedMessages, nil
}

func getUser(users []tg.UserClass, peerID int64) *types.UserInfo {
	var userInfo *types.UserInfo
	for _, userClass := range users {
		if user, ok := userClass.(*tg.User); ok {
			if user.ID == peerID {
				userInfo = shared.ConvertTGUserToUserInfo(user)
			}
		}
	}
	return userInfo
}

func (m *Manager) UserInfoFromPeerClass(ctx context.Context, peerClass *tg.PeerUser) *types.UserInfo {
	userClasses, err := m.client.GetAPI().UsersGetUsers(ctx, []tg.InputUserClass{
		&tg.InputUser{UserID: peerClass.UserID},
	})
	if err != nil {
		return nil
	}

	if len(userClasses) == 0 {
		return nil
	}

	if user, ok := userClasses[0].(*tg.User); !ok {
		return shared.ConvertTGUserToUserInfo(user)
	}
	return nil
}

func (m *Manager) GetUserChatsCmd(ctx context.Context, isBot bool, offsetDate, offsetID int) tea.Cmd {
	return func() tea.Msg {
		users, err := m.GetUserChats(ctx, isBot, offsetDate, offsetID)
		return types.UserChatsMsg{
			Response: &users,
			Err:      err,
		}
	}
}

func (m *Manager) GetChannelsCmd(ctx context.Context, isBroadCast bool, offsetDate, offsetID int) tea.Cmd {
	return func() tea.Msg {
		channels, err := m.GetChannels(ctx, isBroadCast, offsetDate, offsetID)
		if isBroadCast {
			return types.ChannelsMsg{
				Response: &channels,
				Err:      err,
			}
		}
		return types.GroupsMsg{
			Response: &channels,
			Err:      err,
		}
	}
}

func (m *Manager) GetChatHistoryCmd(ctx context.Context, peer types.Peer, limit int, offsetID *int) tea.Cmd {
	return func() tea.Msg {
		messages, err := m.GetChatHistory(ctx, peer, limit, offsetID)
		if err != nil {
			return types.GetMessagesMsg{
				Messages: [50]types.FormattedMessage{},
				Err:      err,
			}
		}

		var msgArray [50]types.FormattedMessage
		copy(msgArray[:], messages)
		return types.GetMessagesMsg{
			Messages: msgArray,
			Err:      nil,
		}
	}
}

// Helper functions and types

type userPeerIDUnreadCount struct {
	unreadCount int
	peerID      int64
}

func getUnreadCount(u []userPeerIDUnreadCount, peerID int64) int {
	for _, p := range u {
		if p.peerID == peerID {
			return p.unreadCount
		}
	}
	return 0
}

type CligramGetDialogsResponse struct {
	Chats      []tg.ChatClass
	Users      []tg.UserClass
	Dialogs    []tg.DialogClass
	OffsetDate int
	OffsetID   int
}

func (m *Manager) getAllDialogs(ctx context.Context, offsetDate, offsetID int) (*CligramGetDialogsResponse, error) {
	var allChats []tg.ChatClass
	var allUsers []tg.UserClass
	var allDialogs []tg.DialogClass
	offsetPeer := &tg.InputPeerEmpty{}
	dialogs, err := m.client.GetAPI().MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
		OffsetDate: offsetDate,
		OffsetID:   offsetID,
		OffsetPeer: offsetPeer,
		Limit:      100,
	})
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}

	switch d := dialogs.(type) {
	case *tg.MessagesDialogs:
		allChats = append(allChats, d.Chats...)
		allUsers = append(allUsers, d.Users...)
		allDialogs = append(allDialogs, d.Dialogs...)
		return &CligramGetDialogsResponse{
			Chats:      allChats,
			Users:      allUsers,
			Dialogs:    allDialogs,
			OffsetDate: -1,
			OffsetID:   -1,
		}, nil

	case *tg.MessagesDialogsSlice:
		allChats = append(allChats, d.Chats...)
		allUsers = append(allUsers, d.Users...)
		allDialogs = append(allDialogs, d.Dialogs...)
		if len(d.Messages) == 0 {
			return &CligramGetDialogsResponse{
				Chats:      allChats,
				Users:      allUsers,
				Dialogs:    allDialogs,
				OffsetDate: -1,
				OffsetID:   -1,
			}, nil
		}
		last := d.Messages[len(d.Messages)-1]

		switch msg := last.(type) {
		case *tg.Message:
			return &CligramGetDialogsResponse{
				Chats:      allChats,
				Users:      allUsers,
				Dialogs:    allDialogs,
				OffsetDate: msg.Date,
				OffsetID:   msg.ID,
			}, nil
		default:
			return &CligramGetDialogsResponse{
				Chats:   allChats,
				Users:   allUsers,
				Dialogs: allDialogs,
			}, nil
		}

	default:
		return &CligramGetDialogsResponse{
			Chats:      allChats,
			Users:      allUsers,
			Dialogs:    allDialogs,
			OffsetDate: -1,
			OffsetID:   -1,
		}, nil
	}
}

func convertTGChannelToChannelInfo(channel *tg.Channel) *types.ChannelInfo {
	return &types.ChannelInfo{
		ChannelTitle:      channel.Title,
		Username:          &channel.Username,
		ID:                strconv.FormatInt(channel.ID, 10),
		AccessHash:        strconv.FormatInt(channel.AccessHash, 10),
		IsCreator:         channel.Creator,
		IsBroadcast:       channel.Broadcast,
		ParticipantsCount: &channel.ParticipantsCount,
		HasStories:        false,
	}
}
func convertPeerToInputPeer(peer types.Peer) (tg.InputPeerClass, error) {
	peerID, err := strconv.ParseInt(peer.ID, 10, 64)
	if err != nil {
		return nil, types.NewInvalidPeerError(peer.ID)
	}

	accessHash, err := strconv.ParseInt(peer.AccessHash, 10, 64)
	if err != nil {
		return nil, types.NewInvalidPeerError(peer.AccessHash)
	}

	switch peer.ChatType {
	case types.UserChat, types.BotChat:
		return &tg.InputPeerUser{
			UserID:     peerID,
			AccessHash: accessHash,
		}, nil
	case types.ChannelChat, types.GroupChat:
		return &tg.InputPeerChannel{
			ChannelID:  peerID,
			AccessHash: accessHash,
		}, nil
	default:
		return nil, types.NewTelegramError(types.ErrorCodeInvalidPeer, "unsupported chat type", nil)
	}
}

type userOnlineStatus struct {
	IsOnline bool
	LastSeen *string
}

func getUserOnlineStatus(status tg.UserStatusClass) *userOnlineStatus {
	if status == nil {
		return &userOnlineStatus{
			IsOnline: false,
			LastSeen: nil,
		}
	}
	switch s := status.(type) {
	case *tg.UserStatusOnline:
		lastSeen := "online"
		return &userOnlineStatus{
			IsOnline: true,
			LastSeen: &lastSeen,
		}
	case *tg.UserStatusOffline:
		lastSeen := calculateLastSeenHumanReadable(s.WasOnline)
		return &userOnlineStatus{
			IsOnline: false,
			LastSeen: &lastSeen,
		}
	default:
		lastSeen := "last seen long time ago"
		return &userOnlineStatus{
			IsOnline: false,
			LastSeen: &lastSeen,
		}
	}
}

func calculateLastSeenHumanReadable(wasOnline int) string {
	lastSeenTime := time.Unix(int64(wasOnline), 0)
	currentTime := time.Now()
	diff := currentTime.Sub(lastSeenTime)

	if diff.Seconds() < 60 {
		return "last seen just now"
	}
	if diff.Hours() < 24 {
		return fmt.Sprintf("last seen at %s", lastSeenTime.Format("03:04 PM"))
	}
	if diff.Hours() < 48 {
		return fmt.Sprintf("last seen yesterday at %s", lastSeenTime.Format("03:04 PM"))
	}

	return fmt.Sprintf("last seen on %s", lastSeenTime.Format("02/01/2006 03:04 PM"))
}
