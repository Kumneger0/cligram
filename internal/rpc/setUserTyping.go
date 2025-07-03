package rpc

import (
	"encoding/json"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type SetUserTypingJsonRpcResponse struct {
	JsonRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Error   *struct {
		Code    int         `json:"code"`
		Message string      `json:"message"`
		Data    interface{} `json:"data,omitempty"`
	} `json:"error,omitempty"`
	Result bool `json:"result,omitempty"`
}

type SetUserTypingMsg struct {
	Response SetUserTypingJsonRpcResponse
	Err      error
}

func (c *JsonRpcClient) SetUserTyping(userPeer PeerInfo, chatType ChatType) tea.Cmd {
	rpcResponse, err := c.Call("setUserTyping", []interface{}{userPeer, chatType})
	return func() tea.Msg {
		time.Sleep(1000 * time.Millisecond)
		if err != nil {
			return SetUserTypingMsg{Err: err}
		}
		var response SetUserTypingJsonRpcResponse
		if err := json.Unmarshal(rpcResponse, &response); err != nil {
			return SetUserTypingMsg{Err: fmt.Errorf("failed to unmarshal response JSON '%s': %w", string(rpcResponse), err)}
		}
		if response.Error != nil {
			return SetUserTypingMsg{Err: fmt.Errorf("ERROR: %s", response.Error.Message)}
		}
		return SetUserTypingMsg{Response: response}
	}
}
