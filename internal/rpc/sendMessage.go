package rpc

import (
	"encoding/json"

	tea "github.com/charmbracelet/bubbletea"
)

type SendMessageRpcResponse struct {
	JsonRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Error   *struct {
		Code    int         `json:"code"`
		Message string      `json:"message"`
		Data    interface{} `json:"data,omitempty"`
	} `json:"error,omitempty"`
	Result SendMessageResponseType `json:"result,omitempty"`
}

type SendMessageResponseType struct {
	MessageId *int `json:"messageId,omitempty"`
}

type SendMessageMsg struct {
	Response SendMessageRpcResponse
	Err      error
}

func (c *JsonRpcClient) SendMessage(pInfo PeerInfo,
	msg string,
	isReplay bool,
	replyToMessageId string,
	cType ChatType, isFile bool,
	filePath string) tea.Cmd {
	return func() tea.Msg {
		paramsFixed := make([]interface{}, 7)
		paramsFixed[0] = pInfo
		paramsFixed[1] = msg
		paramsFixed[2] = isReplay
		paramsFixed[3] = replyToMessageId
		paramsFixed[4] = cType
		paramsFixed[5] = isFile
		paramsFixed[6] = filePath
		sendMessageRpcResponse, err := c.Call("sendMessage", paramsFixed)

		if err != nil {
			return SendMessageMsg{Err: err}
		}
		var sendMessageRpcResponseResult SendMessageRpcResponse

		if err := json.Unmarshal(sendMessageRpcResponse, &sendMessageRpcResponseResult); err != nil {
			return SendMessageMsg{Err: err}
		}
		return SendMessageMsg{Response: sendMessageRpcResponseResult}
	}
}
