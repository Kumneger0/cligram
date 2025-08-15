package rpc

import (
	"encoding/json"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type UserChannelResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    any    `json:"data,omitempty"`
	} `json:"error,omitempty"`
	Result []ChannelAndGroupInfo `json:"result,omitempty"`
}

type UserChannelMsg struct {
	Err      error
	Response *UserChannelResponse
}

func (c *JSONRPCClient) GetUserChannel() tea.Cmd {
	return func() tea.Msg {
		userChannelRPCResponse, err := c.Call("getUserChats", []string{"channel"})
		if err != nil {
			return UserChannelMsg{Err: err}
		}
		var response UserChannelResponse
		if err := json.Unmarshal(userChannelRPCResponse, &response); err != nil {
			return UserChannelMsg{Err: fmt.Errorf("failed to unmarshal response JSON '%s': %w", string(userChannelRPCResponse), err)}
		}
		if response.Error != nil {
			return UserChannelMsg{Err: fmt.Errorf("ERROR: %s", response.Error.Message)}
		}
		return UserChannelMsg{Err: nil, Response: &response}
	}
}
