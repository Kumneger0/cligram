package rpc

import (
	"encoding/json"
	"time"
)

type UserConversationResponse struct {
	JsonRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Error   *struct {
		Code    int         `json:"code"`
		Message string      `json:"message"`
		Data    interface{} `json:"data,omitempty"`
	} `json:"error,omitempty"`
	Result []FormattedMessage `json:"result,omitempty"`
}

type FormattedMessage struct {
	ID                   int64     `json:"id"`
	Sender               string    `json:"sender"`
	Content              string    `json:"content"`
	IsFromMe             bool      `json:"isFromMe"`
	Media                *string   `json:"media,omitempty"`
	Date                 time.Time `json:"date"`
	IsUnsupportedMessage bool      `json:"isUnsupportedMessage"`
	WebPage              *struct {
		URL        string  `json:"url"`
		DisplayURL *string `json:"displayUrl,omitempty"`
	} `json:"webPage,omitempty"`
	Document *struct {
		Document string `json:"document"`
	} `json:"document,omitempty"`
	FromID *string `json:"fromId"`
}

type PeerInfoParams struct {
	AccessHash                  string `json:"accessHash"`
	PeerID                      string `json:"peerId"`
	UserFirstNameOrChannelTitle string `json:"userFirtNameOrChannelTitle"`
}

type ChatType string

const (
	UserChat    ChatType = "user"
	GroupChat   ChatType = "group"
	ChannelChat ChatType = "channel"
)

type IterParams map[string]interface{}

func (c *JsonRpcClient) GetMessages(
	pInfo PeerInfoParams,
	cType ChatType,
	offsetID *int,
	chatAreaWidth *int,
	itParams IterParams,
) (UserConversationResponse, error) {
	paramsFixed := make([]interface{}, 5)
	paramsFixed[0] = pInfo
	paramsFixed[1] = cType

	if offsetID != nil {
		paramsFixed[2] = *offsetID
	} else {
		paramsFixed[2] = nil
	}

	if chatAreaWidth != nil {
		paramsFixed[3] = *chatAreaWidth
	} else {
		paramsFixed[3] = nil
	}

	if itParams != nil {
		paramsFixed[4] = itParams
	} else {
		paramsFixed[4] = nil
	}

	allMesssages, err := c.Call("getAllMessages", paramsFixed)
	if err != nil {
		//TODO: show toast message
		// loging the error will make the ui verry ugly
	}

	var formatedMessage UserConversationResponse

	if err := json.Unmarshal(allMesssages, &formatedMessage); err != nil {
		return UserConversationResponse{}, err
	}

	return formatedMessage, nil
}
