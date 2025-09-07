package types

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gotd/td/tg"
)

type TelegramClient interface {
	Auth(ctx context.Context) error
	SendMessage(ctx context.Context, req SendMessageRequest) tea.Cmd
	GetMessages(ctx context.Context, req GetMessagesRequest) tea.Cmd
	GetUserChats(ctx context.Context, chatType ChatType, offsetDate, offsetID int) tea.Cmd
	GetUserChannels(ctx context.Context, isBroadCast bool, offsetDate, offsetID int) tea.Cmd
	DeleteMessage(ctx context.Context, req DeleteMessageRequest) (DeleteMessageResponse, error)
	EditMessage(ctx context.Context, req EditMessageRequest) error
	ForwardMessages(ctx context.Context, req ForwardMessagesRequest) error
	MarkMessagesAsRead(ctx context.Context, req MarkAsReadRequest) error
	SetUserTyping(ctx context.Context, req SetTypingRequest) error
	SearchUsers(ctx context.Context, query string) tea.Cmd
	GetAllStories(ctx context.Context) tea.Cmd
}

type MessageSender interface {
	SendText(ctx context.Context, peer Peer, text string, replyTo *int) tea.Cmd
	SendMedia(ctx context.Context, peer Peer, filePath string, caption string, replyTo *int) tea.Cmd
}
type UserManager interface {
	GetUserInfo(ctx context.Context, userID int64) (*UserInfo, error)
	GetUserStatus(ctx context.Context, userID int64) (*UserStatus, error)
	SearchUsers(ctx context.Context, query string) ([]UserInfo, error)
}

type ChatManager interface {
	GetUserChats(ctx context.Context, isBot bool, offsetDate, offsetID int) (GetUserChatsResult, error)
	GetChannels(ctx context.Context, isBroadCast bool, offsetDate, offsetID int) (GetChannelsResult, error)
	GetChannelInfo(ctx context.Context, peer Peer) (*ChannelInfo, error)
	GetChatHistory(ctx context.Context, peer Peer, limit int, offsetID *int) ([]FormattedMessage, error)
	GetUserChatsCmd(ctx context.Context, isBot bool, offsetDate, offsetID int) tea.Cmd
	GetChannelsCmd(ctx context.Context, isBroadCast bool, offsetDate, offsetID int) tea.Cmd
	GetChatHistoryCmd(ctx context.Context, peer Peer, limit int, offsetID *int) tea.Cmd
	UserInfoFromPeerClass(ctx context.Context, peer *tg.PeerUser) *UserInfo
}

type SessionManager interface {
	GetSessionStorage() SessionStorage
	EnsureSessionDir() error
}

type SessionStorage interface {
	Load() ([]byte, error)
	Store(data []byte) error
}
