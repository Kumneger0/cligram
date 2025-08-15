package rpc

import (
	"encoding/json"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type MarkMessagesAsReadJSONRPCResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    any    `json:"data,omitempty"`
	} `json:"error,omitempty"`
	Result bool `json:"result,omitempty"`
}

type MarkMessagesAsReadMsg struct {
	Response MarkMessagesAsReadJSONRPCResponse
	Err      error
}

func (c *JSONRPCClient) MarkMessagesAsRead(userPeer PeerInfo, chatType ChatType) tea.Cmd {
	return func() tea.Msg {
		rpcResponse, err := c.Call("markUnRead", []any{userPeer, chatType})
		if err != nil {
			return MarkMessagesAsReadMsg{Err: err}
		}
		var response MarkMessagesAsReadJSONRPCResponse
		if err := json.Unmarshal(rpcResponse, &response); err != nil {
			return MarkMessagesAsReadMsg{Err: fmt.Errorf("failed to unmarshal response JSON '%s': %w", string(rpcResponse), err)}
		}
		if response.Error != nil {
			return MarkMessagesAsReadMsg{Err: fmt.Errorf("ERROR: %s", response.Error.Message)}
		}
		return MarkMessagesAsReadMsg{Response: response}
	}
}
