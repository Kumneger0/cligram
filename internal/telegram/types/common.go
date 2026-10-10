package types // nolint:revive

import (
	"time"

	"github.com/gotd/td/tg"
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
	FirstName           string                 `json:"firstName"`
	LastName            string                 `json:"lastName,omitempty"`
	Username            string                 `json:"username,omitempty"`
	IsBot               bool                   `json:"isBot"`
	PeerID              string                 `json:"peerId"`
	AccessHash          string                 `json:"accessHash"`
	UnreadCount         int                    `json:"unreadCount"`
	LastSeen            *string                `json:"lastSeen,omitempty"`
	IsOnline            bool                   `json:"isOnline"`
	IsTyping            bool                   `json:"isTyping"`
	HasStories          bool                   `json:"hasStories"`
	NotifySettings      *tg.PeerNotifySettings `json:"notifySettings,omitempty"`
	Premium             bool                   `json:"premium"`
	ReadInboxMaxID      int                    `json:"readInboxMaxId"`
	ReadOutboxMaxID     int                    `json:"readOutboxMaxId"`
	PhoneCallsPrivate   bool                   `json:"phoneCallsPrivate"`
	PhoneCallsAvailable bool                   `json:"phoneCallsAvailable"`
}

type ChannelInfo struct {
	ChannelTitle      string                 `json:"title"`
	Username          *string                `json:"username,omitempty"`
	ID                string                 `json:"id"`
	AccessHash        string                 `json:"accessHash"`
	IsCreator         bool                   `json:"isCreator"`
	IsBroadcast       bool                   `json:"isBroadcast"`
	ParticipantsCount *int                   `json:"participantsCount,omitempty"`
	UnreadCount       int                    `json:"unreadCount"`
	HasStories        bool                   `json:"hasStories"`
	NotifySettings    *tg.PeerNotifySettings `json:"notifySettings,omitempty"`
	ReadInboxMaxID    int                    `json:"readInboxMaxId"`
	ReadOutboxMaxID   int                    `json:"readOutboxMaxId"`
	IsForum           bool                   `json:"isForum"`
}

type FormattedMessage struct {
	ID                   int                     `json:"id"`
	Sender               string                  `json:"sender"`
	Content              string                  `json:"content"`
	IsFromMe             bool                    `json:"isFromMe"`
	Media                *string                 `json:"media,omitempty"`
	MediaAttachment      *MediaAttachment        `json:"mediaAttachment,omitempty"`
	Date                 time.Time               `json:"date"`
	IsUnsupportedMessage bool                    `json:"isUnsupportedMessage"`
	WebPage              *WebPage                `json:"webPage,omitempty"`
	Document             *Document               `json:"document,omitempty"`
	FromID               *string                 `json:"fromId,omitempty"`
	SenderUserInfo       *UserInfo               `json:"senderUserInfo,omitempty"`
	ReplyTo              *FormattedMessage       `json:"replyTo,omitempty"`
	PeerID               *string                 `json:"peerId,omitempty"`
	Reactions            *tg.MessageReactions    `json:"reactions,omitempty"`
	Views                int                     `json:"view"`
	HasWebPagePreview    bool                    `json:"hasWebPagePreview"`
	MessageMediaWebPage  *tg.MessageMediaWebPage `json:"messageMediaWebPage"`
}

type MediaType string

const (
	MediaTypePhoto    MediaType = "photo"
	MediaTypeVideo    MediaType = "video"
	MediaTypeAudio    MediaType = "audio"
	MediaTypeVoice    MediaType = "voice"
	MediaTypeDocument MediaType = "document"
)

type MediaStatus string

const (
	MediaStatusIdle        MediaStatus = "idle"
	MediaStatusDownloading MediaStatus = "downloading"
	MediaStatusDownloaded  MediaStatus = "downloaded"
	MediaStatusFailed      MediaStatus = "failed"
)

type MediaAttachment struct {
	Type          MediaType                 `json:"type"`
	FileName      string                    `json:"fileName,omitempty"`
	Title         string                    `json:"title,omitempty"`
	Performer     string                    `json:"performer,omitempty"`
	FileSize      int64                     `json:"fileSize,omitempty"`
	MimeType      string                    `json:"mimeType,omitempty"`
	Duration      int                       `json:"duration,omitempty"`
	Width         int                       `json:"width,omitempty"`
	Height        int                       `json:"height,omitempty"`
	Status        MediaStatus               `json:"status"`
	LocalPath     string                    `json:"localPath,omitempty"`
	InputLocation tg.InputFileLocationClass `json:"-"`
}

type DownloadMediaCompleteMsg struct {
	MessageID int
	PeerID    string
	LocalPath string
	Err       error
}

type DownloadMediaProgressMsg struct {
	MessageID int
	PeerID    string
	Status    MediaStatus
	Progress  float64
}

type ShouldHighlightSpecificMessageMsg struct {
	MessageID int
}

type OpenNewChatWithPeerMsg struct {
	Chat *EntityPreviewInfo
}

type EntityPreviewInfo struct {
	Entity    string
	MessageID []string
}

type ResolvedPeerInfo struct {
	User    *UserInfo
	Channel *ChannelInfo
	Group   *ChannelInfo
}

type GetEntityInfoMsg struct {
	Response   *ResolvedPeerInfo `json:"response,omitempty"`
	MessageIDs []string          `json:"messageId,omitempty"`
	Err        error             `json:"error,omitempty"`
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
	NewMessage             *NewMessageNotification        `json:"newMessage,omitempty"`
	UserStatus             *UserStatusNotification        `json:"userStatus,omitempty"`
	UserTyping             *UserTypingNotification        `json:"userTyping,omitempty"`
	Error                  *ErrorNotification             `json:"error,omitempty"`
	SearchResult           *SearchUsersMsg                `json:"searchResult,omitempty"`
	ReadHistoryOutbox      *ReadHistoryOutboxNotification `json:"readHistoryOutbox,omitempty"`
	GetUserDHConfigRequest *GetUserDHConfigRequest        `json:"getUserDHConfigRequest,omitempty"`
	CallEvent              *CallNotification              `json:"callEvent,omitempty"`
}

type CallState string

const (
	CallStateIncoming      CallState = "incoming"
	CallStateOutgoing      CallState = "outgoing"
	CallStateActive        CallState = "active"
	CallStateEnded         CallState = "ended"
	CallStateDeclined      CallState = "declined"
	CallStateMissingHelper CallState = "missing_helper"
	CallStateBusyMissed    CallState = "busy_missed"
)

type CallNotification struct {
	UserID   int64
	UserName string
	State    CallState
	Duration int
	IsRelay  bool
	Err      error
}

type GetUserDHConfigRequest struct {
	UserID int64 `json:"userId"`
}

type ForumTopicInfo struct {
	ID          int    `json:"id"`
	TopicTitle  string `json:"title"`
	UnreadCount int    `json:"unreadCount"`
}

type NewMessageNotification struct {
	ID      int         `json:"id"`
	FromID  string      `json:"fromId"`
	PeerID  string      `json:"peerId,omitempty"`
	Message *tg.Message `json:"message"`
}

type ReadHistoryOutboxNotification struct {
	PeerID   string   `json:"peerId"`
	MaxID    int      `json:"maxId"`
	PeerType ChatType `json:"peerType"`
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

func (u UserInfo) DisplayName() string {
	name := u.FirstName
	if u.LastName != "" {
		name += " " + u.LastName
	}
	if name == "" {
		return "Deleted Account"
	}
	return name
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

func (f ForumTopicInfo) Title() string {
	return f.TopicTitle
}

func (f ForumTopicInfo) FilterValue() string {
	return f.TopicTitle
}

type AccountsOnDeviceInfo struct {
	AccountName string
	PhoneNumber string
	Path        string
	ModTime     int64
}

type GroupMemberRole string

const (
	MemberRoleOwner  GroupMemberRole = "Owner"
	MemberRoleAdmin  GroupMemberRole = "Admin"
	MemberRoleMember GroupMemberRole = "Member"
)

type GroupMemberInfo struct {
	User        UserInfo        `json:"user"`
	Role        GroupMemberRole `json:"role"`
	CustomTitle string          `json:"customTitle,omitempty"`
}

func (g GroupMemberInfo) Title() string {
	return g.User.Title()
}

func (g GroupMemberInfo) FilterValue() string {
	val := g.User.FirstName
	if g.User.LastName != "" {
		val += " " + g.User.LastName
	}
	if g.User.Username != "" {
		val += " @" + g.User.Username
	}
	return val
}

type GetGroupMembersMsg struct {
	PeerID     string            `json:"peerId"`
	Offset     int               `json:"offset"`
	Members    []GroupMemberInfo `json:"members"`
	TotalCount int               `json:"totalCount"`
	Err        error             `json:"err,omitempty"`
}

type SearchGroupMembersMsg struct {
	PeerID     string            `json:"peerId"`
	Query      string            `json:"query"`
	Members    []GroupMemberInfo `json:"members"`
	TotalCount int               `json:"totalCount"`
	Err        error             `json:"err,omitempty"`
}
