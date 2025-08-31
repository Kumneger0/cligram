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
	Response *[]UserInfo `json:"response,omitempty"`
	Err      error       `json:"error,omitempty"`
}

type ChannelsMsg struct {
	Response *[]ChannelInfo `json:"response,omitempty"`
	Err      error          `json:"error,omitempty"`
}
type GroupsMsg struct {
	Response *[]ChannelInfo `json:"response,omitempty"`
	Err      error          `json:"error,omitempty"`
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
