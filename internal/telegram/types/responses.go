package types // nolint:revive

import "github.com/gotd/td/tg"

type GetAllChatsResponse struct {
	PrivateChats         []UserInfo    `json:"chats"`
	Channels             []ChannelInfo `json:"channels"`
	Groups               []ChannelInfo `json:"groups"`
	OffsetDate, OffsetID int
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
