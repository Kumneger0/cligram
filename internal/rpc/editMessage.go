package rpc

import (
	"encoding/json"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type EditMessageJSONRPCResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Error   *struct {
		Code    int         `json:"code"`
		Message string      `json:"message"`
		Data    interface{} `json:"data,omitempty"`
	} `json:"error,omitempty"`
	Result bool `json:"result,omitempty"`
}

type EditMessageMsg struct {
	Response       EditMessageJSONRPCResponse
	Err            error
	UpdatedMessage string
}

func (c *JSONRPCClient) EditMessage(userPeer PeerInfo, chatType ChatType, messageID int, newMessage string) tea.Cmd {
	return func() tea.Msg {
		rpcResponse, err := c.Call("editMessage", []interface{}{userPeer, messageID, newMessage, chatType})
		if err != nil {
			return EditMessageMsg{Err: err}
		}
		var response EditMessageJSONRPCResponse
		if err := json.Unmarshal(rpcResponse, &response); err != nil {
			return EditMessageMsg{Err: fmt.Errorf("failed to unmarshal response JSON '%s': %w", string(rpcResponse), err)}
		}
		if response.Error != nil {
			return EditMessageMsg{Err: fmt.Errorf("ERROR: %s", response.Error.Message)}
		}
		return EditMessageMsg{Response: response, UpdatedMessage: newMessage}
	}
}
