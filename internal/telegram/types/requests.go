package types // nolint:revive

type SendMessageRequest struct {
	Peer             Peer   `json:"peer"`
	Message          string `json:"message"`
	IsReply          bool   `json:"isReply"`
	ReplyToMessageID string `json:"replyToMessageId,omitempty"`
	IsFile           bool   `json:"isFile"`
	FilePath         string `json:"filePath,omitempty"`
}

type GetMessagesRequest struct {
	Peer          Peer `json:"peer"`
	Limit         int  `json:"limit"`
	OffsetID      *int `json:"offsetId,omitempty"`
	ChatAreaWidth *int `json:"chatAreaWidth,omitempty"`
}

type DeleteMessageRequest struct {
	Peer      Peer `json:"peer"`
	MessageID int  `json:"messageId"`
}

type EditMessageRequest struct {
	Peer       Peer   `json:"peer"`
	MessageID  int    `json:"messageId"`
	NewMessage string `json:"newMessage"`
}

type ForwardMessagesRequest struct {
	FromPeer   Peer  `json:"fromPeer"`
	ToPeer     Peer  `json:"toPeer"`
	MessageIDs []int `json:"messageIds"`
}

type MarkAsReadRequest struct {
	Peer Peer `json:"peer"`
	// MessageID int   `json:"messageId"`
}

type SetTypingRequest struct {
	Peer Peer `json:"peer"`
}
