package types // nolint:revive

// SendMessageRequest represents a message sending request
type SendMessageRequest struct {
	Peer             Peer   `json:"peer"`
	Message          string `json:"message"`
	IsReply          bool   `json:"isReply"`
	ReplyToMessageID string `json:"replyToMessageId,omitempty"`
	IsFile           bool   `json:"isFile"`
	FilePath         string `json:"filePath,omitempty"`
}

// GetMessagesRequest represents a request to get message history
type GetMessagesRequest struct {
	Peer          Peer `json:"peer"`
	Limit         int  `json:"limit"`
	OffsetID      *int `json:"offsetId,omitempty"`
	ChatAreaWidth *int `json:"chatAreaWidth,omitempty"`
}

// DeleteMessageRequest represents a message deletion request
type DeleteMessageRequest struct {
	Peer      Peer `json:"peer"`
	MessageID int  `json:"messageId"`
}

// EditMessageRequest represents a message edit request
type EditMessageRequest struct {
	Peer       Peer   `json:"peer"`
	MessageID  int    `json:"messageId"`
	NewMessage string `json:"newMessage"`
}

// ForwardMessagesRequest represents a message forwarding request
type ForwardMessagesRequest struct {
	FromPeer   Peer  `json:"fromPeer"`
	ToPeer     Peer  `json:"toPeer"`
	MessageIDs []int `json:"messageIds"`
}

// MarkAsReadRequest represents a mark as read request
type MarkAsReadRequest struct {
	Peer Peer `json:"peer"`
	// MessageID int   `json:"messageId"`
}

// SetTypingRequest represents a typing status request
type SetTypingRequest struct {
	Peer Peer `json:"peer"`
}
