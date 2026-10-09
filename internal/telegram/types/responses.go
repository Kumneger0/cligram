package types // nolint:revive

import (
	"math/big"
	"time"

	"github.com/gotd/td/tg"
)

type GetAllChatsResponseMSG struct {
	Chats AllChats
	Err   error
}
type AllChats struct {
	All                  []FilterableItem `json:"all,omitempty"`
	PrivateChats         []UserInfo       `json:"chats"`
	Bots                 []UserInfo       `json:"bots,omitempty"`
	Channels             []ChannelInfo    `json:"channels"`
	Groups               []ChannelInfo    `json:"groups"`
	OffsetDate, OffsetID int
}

type GetChannelForumsResponseMsg struct {
	Forums []ForumTopicInfo
	Err    error
}

type DHConfig struct {
	G      int32  // generator from Telegram's MessagesGetDhConfig
	P      []byte // safe prime
	A      *big.Int
	GA     *big.Int
	GAHash []byte // SHA-256(g^a mod p) — sent in PhoneRequestCall and needed to confirm
	Random []byte // server-provided random bytes from MessagesGetDhConfig
}

type CallUserResponse struct {
	Call     *tg.PhonePhoneCall
	DHConfig *DHConfig
	UserID   *int64
	Err      error
}

type SendMessageResponse struct {
	MessageID *int `json:"messageId,omitempty"`
}

type DeleteMessageResponse struct {
	Status string `json:"status"`
}

type MarkMessagesAsReadMsg struct {
	Response bool  `json:"response,omitempty"`
	Err      error `json:"error,omitempty"`
}
type GetMessagesResponse struct {
	Messages [50]FormattedMessage `json:"messages"`
}

type UserChatsResponse struct {
	Users []UserInfo `json:"users"`
}

type ChannelsResponse struct {
	Channels []ChannelInfo `json:"channels"`
}

type Reaction struct {
	tg.AvailableReaction
}

type AvailableReactions struct {
	Reactions []Reaction `json:"reactions"`
	Err       error
}

func (s Reaction) FilterValue() string {
	return s.Reaction
}

type SendReactionMsg struct {
	Reaction Reaction
}

type SendReactionResponseMsg struct {
	Err       error
	Response  bool
	MessageID int
	Emoticon  string
}

type SearchUsersResponse struct {
	Users []UserInfo `json:"users"`
}

type SendMessageMsg struct {
	RandID   int                  `json:"randId"`
	Response *SendMessageResponse `json:"response,omitempty"`
	Err      error                `json:"error,omitempty"`
}

type GetMessagesMsg struct {
	Messages [50]FormattedMessage `json:"messages"`
	Err      error                `json:"error,omitempty"`
}

type UserChatsMsg struct {
	Response *GetUserChatsResult `json:"response,omitempty"`
	Err      error               `json:"error,omitempty"`
	IsBot    bool                `json:"isBot,omitempty"`
}

type ChannelsMsg struct {
	Response *GetChannelsResult `json:"response,omitempty"`
	Err      error              `json:"error,omitempty"`
}
type GroupsMsg struct {
	Response *GetChannelsResult `json:"response,omitempty"`
	Err      error              `json:"error,omitempty"`
}

type SearchUsersMsg struct {
	Response *[]UserInfo `json:"response,omitempty"`
	Err      error       `json:"error,omitempty"`
}

type EditMessageMsg struct {
	Response       bool
	Err            error
	UpdatedMessage string
}

type Stories struct {
	UserInfo   UserInfo
	ID         int
	Data       []byte
	IsSelected bool
	Date       time.Time
	ExpireDate time.Time
	Caption    string
	MediaType  string
	IsPinned   bool
}

// will move it to better place in the future let's keep it here for now
func (s Stories) FilterValue() string {
	return s.UserInfo.FirstName
}

type GetAllStoriesMsg struct {
	Stories []Stories
	Err     error
}

type StoriesDownloadStatusMsg struct {
	//the id stories empty array on error
	IDs []int
	// whether that the download is finished or not
	// finished means the the story is written to fileSystem hidden folder
	Done bool
	//there was some error during downloading for saving to fileSystem
	// so we should notify bubbletea to handle this accordingly
	Err error
	// a peer who posted the story
	Peer Peer
}

type GetUserChatsResult struct {
	Data                 []UserInfo
	OffsetDate, OffsetID int
}

type GetChannelsResult struct {
	Data                 []ChannelInfo
	OffsetDate, OffsetID int
}

type CurrentUserMsg struct {
	User *UserInfo
	Err  error
}

type SingleMessageMsg struct {
	Message *FormattedMessage
	Err     error
}

type CallAcceptedMsg struct {
	UserID int64
	Err    error
}

type CallDeclinedMsg struct {
	UserID int64
	Err    error
}

type CallHungUpMsg struct {
	UserID int64
	Err    error
}

type CallMuteToggledMsg struct {
	UserID int64
	Muted  bool
	Err    error
}
