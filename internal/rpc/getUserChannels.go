package rpc

import (
	"encoding/json"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type UserChannelResponse struct {
	JsonRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Error   *struct {
		Code    int         `json:"code"`
		Message string      `json:"message"`
		Data    interface{} `json:"data,omitempty"`
	} `json:"error,omitempty"`
	Result []ChannelAndGroupInfo `json:"result,omitempty"`
}

type UserChannelMsg struct {
	Err      error
	Response *UserChannelResponse
}

func (c *JsonRpcClient) GetUserChannel() tea.Cmd {
	userChannelRpcResponse, err := c.Call("getUserChats", []string{"channel"})
	return func() tea.Msg {
		if err != nil {
			return UserChannelMsg{Err: err}
		}
		var response UserChannelResponse
		if err := json.Unmarshal(userChannelRpcResponse, &response); err != nil {
			return UserChannelMsg{Err: fmt.Errorf("failed to unmarshal response JSON '%s': %w", string(userChannelRpcResponse), err)}
		}
		if response.Error != nil {
			return UserChannelMsg{Err: fmt.Errorf("ERROR: %s", response.Error.Message)}
		}
		return UserChannelMsg{Err: nil, Response: &response}
	}
}
