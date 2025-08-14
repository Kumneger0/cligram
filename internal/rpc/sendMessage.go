package rpc

import (
	"encoding/json"

	tea "github.com/charmbracelet/bubbletea"
)

type SendMessageRPCResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Error   *struct {
		Code    int         `json:"code"`
		Message string      `json:"message"`
		Data    interface{} `json:"data,omitempty"`
	} `json:"error,omitempty"`
	Result SendMessageResponseType `json:"result,omitempty"`
}

type SendMessageResponseType struct {
	MessageID *int `json:"messageId,omitempty"`
}

type SendMessageMsg struct {
	Response SendMessageRPCResponse
	Err      error
}

func (c *JSONRPCClient) SendMessage(pInfo PeerInfo,
	msg string,
	isReplay bool,
	replyToMessageID string,
	cType ChatType, isFile bool,
	filePath string) tea.Cmd {
	return func() tea.Msg {
		paramsFixed := make([]interface{}, 7)
		paramsFixed[0] = pInfo
		paramsFixed[1] = msg
		paramsFixed[2] = isReplay
		paramsFixed[3] = replyToMessageID
		paramsFixed[4] = cType
		paramsFixed[5] = isFile
		paramsFixed[6] = filePath
		rpcResponse, rpcError := c.Call("sendMessage", paramsFixed)

		if rpcError != nil {
			return SendMessageMsg{Err: rpcError}
		}
		var rpcResponseResult SendMessageRPCResponse

		if rpcError := json.Unmarshal(rpcResponse, &rpcResponseResult); rpcError != nil {
			return SendMessageMsg{Err: rpcError}
		}
		return SendMessageMsg{Response: rpcResponseResult}
	}
}
