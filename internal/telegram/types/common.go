package types // nolint:revive

import (
	"time"
)

type ChatType string

const (
	UserChat    ChatType = "user"
	GroupChat   ChatType = "group"
	ChannelChat ChatType = "channel"
	BotChat     ChatType = "bot"
)

type Peer struct {
	ID         string   `json:"id"`
	AccessHash string   `json:"accessHash"`
	ChatType   ChatType `json:"chatType"`
}

type UserInfo struct {
	FirstName   string  `json:"firstName"`
	LastName    string  `json:"lastName,omitempty"`
	Username    string  `json:"username,omitempty"`
	IsBot       bool    `json:"isBot"`
	PeerID      string  `json:"peerId"`
	AccessHash  string  `json:"accessHash"`
	UnreadCount int     `json:"unreadCount"`
	LastSeen    *string `json:"lastSeen,omitempty"`
	IsOnline    bool    `json:"isOnline"`
	IsTyping    bool    `json:"isTyping"`
	HasStories  bool    `json:"hasStories"`
}

type ChannelInfo struct {
	ChannelTitle      string  `json:"title"`
	Username          *string `json:"username,omitempty"`
	ID                string  `json:"id"`
	AccessHash        string  `json:"accessHash"`
	IsCreator         bool    `json:"isCreator"`
	IsBroadcast       bool    `json:"isBroadcast"`
	ParticipantsCount *int    `json:"participantsCount,omitempty"`
	UnreadCount       int     `json:"unreadCount"`
	HasStories        bool    `json:"hasStories"`
}

type FormattedMessage struct {
	ID                   int               `json:"id"`
	Sender               string            `json:"sender"`
	Content              string            `json:"content"`
	IsFromMe             bool              `json:"isFromMe"`
	Media                *string           `json:"media,omitempty"`
	Date                 time.Time         `json:"date"`
	IsUnsupportedMessage bool              `json:"isUnsupportedMessage"`
	WebPage              *WebPage          `json:"webPage,omitempty"`
	Document             *Document         `json:"document,omitempty"`
	FromID               *string           `json:"fromId,omitempty"`
	SenderUserInfo       *UserInfo         `json:"senderUserInfo,omitempty"`
	ReplyTo              *FormattedMessage `json:"replyTo,omitempty"`
}

type WebPage struct {
	URL        string  `json:"url"`
	DisplayURL *string `json:"displayUrl,omitempty"`
}

type Document struct {
	Document string `json:"document"`
}

type UserStatus struct {
	IsOnline bool      `json:"isOnline"`
	LastSeen time.Time `json:"lastSeen"`
}

type Notification struct {
	NewMessage   *NewMessageNotification `json:"newMessage,omitempty"`
	UserStatus   *UserStatusNotification `json:"userStatus,omitempty"`
	UserTyping   *UserTypingNotification `json:"userTyping,omitempty"`
	Error        *ErrorNotification      `json:"error,omitempty"`
	SearchResult *SearchUsersMsg
}

type NewMessageNotification struct {
	Message        FormattedMessage `json:"message"`
	User           *UserInfo        `json:"user,omitempty"`
	ChannelOrGroup *ChannelInfo     `json:"channelOrGroup,omitempty"`
}

type UserStatusNotification struct {
	UserInfo UserInfo   `json:"userInfo"`
	Status   UserStatus `json:"status"`
}

type UserTypingNotification struct {
	User UserInfo `json:"user"`
}

type ErrorNotification struct {
	Error   error  `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code,omitempty"`
}

type FilterableItem interface {
	FilterValue() string
	Title() string
}

func (u UserInfo) Title() string {
	return u.FirstName
}

func (u UserInfo) FilterValue() string {
	return u.FirstName
}

func (c ChannelInfo) Title() string {
	return c.ChannelTitle
}

func (c ChannelInfo) FilterValue() string {
	return c.ChannelTitle
}

func (m FormattedMessage) Title() string {
	return m.Content
}

func (m FormattedMessage) FilterValue() string {
	return m.Content
}
