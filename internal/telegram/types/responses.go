// nolint:revive
package types

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

type SearchUsersResponse struct {
	Users []UserInfo `json:"users"`
}

type SendMessageMsg struct {
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
	//the id of story -1 on error
	ID int
	// whether that the download is finished or not
	// finished means the the story is written to fileSystem hidden folder
	Done bool
	//there was some error during downloading for saving to fileSystem
	// so we should notify bubbletea to handle this accordingly
	Err error
}

type GetUserChatsResult struct {
	Data                 []UserInfo
	OffsetDate, OffsetID int
}

type GetChannelsResult struct {
	Data                 []ChannelInfo
	OffsetDate, OffsetID int
}
