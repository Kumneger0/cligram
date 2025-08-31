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

func (m *Manager) GetUserChats(ctx context.Context, isBot bool) ([]types.UserInfo, error) {
	dialogsSlice, err := m.getAllDialogs(ctx)
	if err != nil {
		return nil, types.NewTelegramError(types.ErrorCodeGetMessagesFailed, "failed to get dialogs", err)
	}

	if dialogsSlice == nil {
		return []types.UserInfo{}, nil
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
			userLastSeenStatus := getUserOnlineStatus(tgUser.Status)
			user := convertTGUserToUserInfo(tgUser)
			unreadCount := getUnreadCount(userPeerIDs, tgUser.ID)
			user.IsOnline = userLastSeenStatus.IsOnline
			user.LastSeen = userLastSeenStatus.LastSeen
			user.UnreadCount = unreadCount
			users = append(users, *user)
		}
	}

	return users, nil
}

func (m *Manager) GetChannels(ctx context.Context, isBroadCast bool) ([]types.ChannelInfo, error) {
	dialogsSlice, err := m.getAllDialogs(ctx)
	if err != nil {
		return nil, types.NewTelegramError(types.ErrorCodeGetMessagesFailed, "failed to get dialogs", err)
	}

	if dialogsSlice == nil {
		return []types.ChannelInfo{}, nil
	}

	var channels []types.ChannelInfo
	for _, chatClass := range dialogsSlice.Chats {
		if channel, ok := chatClass.(*tg.Channel); ok && channel.Broadcast == isBroadCast {
			channelInfo := convertTGChannelToChannelInfo(channel)
			channels = append(channels, *channelInfo)
		}
	}

	return channels, nil
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
		Username:          nil,
		ID:                peer.ID,
		AccessHash:        "",
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
		return nil, types.NewGetMessagesError(errors.New(inputPeer.String()))
	}

	msgs, users, err := shared.GetMessageAndUserClasses(history)
	if err != nil {
		return nil, err
	}

	var userInfo *types.UserInfo
	var channel *types.ChannelInfo

	areWeInUserModeOrBotMode := peer.ChatType == types.BotChat || peer.ChatType == types.UserChat

	if areWeInUserModeOrBotMode {
		ID, _ := strconv.ParseInt(peer.ID, 10, 64)
		userInfo = getUser(users, ID)
	} else {
		channel, err = m.GetChannelInfo(ctx, peer)
		if err != nil {
			slog.Error(err.Error())
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
				userInfo = convertTGUserToUserInfo(user)
			}
		}
	}
	return userInfo
}

func (m *Manager) UserInfoFromPeerClass(ctx context.Context, peerClass *tg.PeerUser) *types.UserInfo {
	userClasses, err := m.client.GetAPI().UsersGetUsers(ctx, []tg.InputUserClass{
		&tg.InputUser{
			UserID:     peerClass.UserID,
			AccessHash: peerClass.UserID,
		},
	})
	if err != nil {
		return nil
	}
	if len(userClasses) == 0 {
		return nil
	}

	if user, ok := userClasses[0].(*tg.User); !ok {
		return convertTGUserToUserInfo(user)

	}

	return nil
}

func (m *Manager) GetUserChatsCmd(ctx context.Context, isBot bool) tea.Cmd {
	return func() tea.Msg {
		users, err := m.GetUserChats(ctx, isBot)
		return types.UserChatsMsg{
			Response: &users,
			Err:      err,
		}
	}
}

func (m *Manager) GetChannelsCmd(ctx context.Context, isBroadCast bool) tea.Cmd {
	return func() tea.Msg {
		channels, err := m.GetChannels(ctx, isBroadCast)
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

func (m *Manager) getAllDialogs(ctx context.Context) (*tg.MessagesDialogsSlice, error) {
	dialogs, err := m.client.GetAPI().MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
		Limit:      1000,
		OffsetPeer: &tg.InputPeerSelf{},
	})

	if err != nil {
		return nil, err
	}

	if dialogsSlice, ok := dialogs.(*tg.MessagesDialogsSlice); ok {
		return dialogsSlice, nil
	}
	return nil, fmt.Errorf("failed to get dialogs")
}

func convertTGUserToUserInfo(tgUser *tg.User) *types.UserInfo {
	return &types.UserInfo{
		FirstName:  tgUser.FirstName,
		LastName:   tgUser.LastName,
		Username:   tgUser.Username,
		IsBot:      tgUser.Bot,
		PeerID:     strconv.FormatInt(tgUser.ID, 10),
		AccessHash: strconv.FormatInt(tgUser.AccessHash, 10),
		IsTyping:   false,
		IsOnline:   false,
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
