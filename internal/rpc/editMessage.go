package rpc

import (
	"encoding/json"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type EditMessageJsonRpcResponse struct {
	JsonRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Error   *struct {
		Code    int         `json:"code"`
		Message string      `json:"message"`
		Data    interface{} `json:"data,omitempty"`
	} `json:"error,omitempty"`
	Result bool `json:"result,omitempty"`
}

type EditMessageMsg struct {
	Response       EditMessageJsonRpcResponse
	Err            error
	UpdatedMessage string
}

func (c *JsonRpcClient) EditMessage(userPeer PeerInfo, chatType ChatType, messageId int, newMessage string) tea.Cmd {
	return func() tea.Msg {
		rpcResponse, err := c.Call("editMessage", []interface{}{userPeer, messageId, newMessage, chatType})
		if err != nil {
			return EditMessageMsg{Err: err}
		}
		var response EditMessageJsonRpcResponse
		if err := json.Unmarshal(rpcResponse, &response); err != nil {
			return EditMessageMsg{Err: fmt.Errorf("failed to unmarshal response JSON '%s': %w", string(rpcResponse), err)}
		}
		if response.Error != nil {
			return EditMessageMsg{Err: fmt.Errorf("ERROR: %s", response.Error.Message)}
		}
		return EditMessageMsg{Response: response, UpdatedMessage: newMessage}
	}
}
