package rpc

import (
	"encoding/json"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type UserChatsMsg struct {
	Response *UserChatsJsonRpcResponse
	Err      error
}

type UserChatsJsonRpcResponse struct {
	JsonRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Error   *struct {
		Code    int         `json:"code"`
		Message string      `json:"message"`
		Data    interface{} `json:"data,omitempty"`
	} `json:"error,omitempty"`
	Result []UserInfo `json:"result,omitempty"`
}

func (c *JsonRpcClient) GetUserChats() UserChatsMsg {
	// if c == nil {
	// 	return UserChatsMsg{Err: fmt.Errorf("c is nil")}
	// }
	userChatRpcResponse, err := c.Call("getUserChats", []string{"user"})

	if err != nil {
		return UserChatsMsg{Err: err}
	}
	var response UserChatsJsonRpcResponse
	if err := json.Unmarshal(userChatRpcResponse, &response); err != nil {
		return UserChatsMsg{Err: fmt.Errorf("failed to unmarshal response JSON '%s': %w", string(userChatRpcResponse), err)}
	}
	if response.Error != nil {
		return UserChatsMsg{Err: fmt.Errorf("ERROR: %s", response.Error.Message)}
	}
	return UserChatsMsg{Err: nil, Response: &response}
}

func (c *JsonRpcClient) GetChats() tea.Cmd {
	userChatRpcResponse, err := c.Call("getUserChats", []string{"user"})
	return func() tea.Msg {
		if err != nil {
			return UserChatsMsg{Err: err}
		}
		var response UserChatsJsonRpcResponse
		if err := json.Unmarshal(userChatRpcResponse, &response); err != nil {
			return UserChatsMsg{Err: fmt.Errorf("failed to unmarshal response JSON '%s': %w", string(userChatRpcResponse), err)}
		}
		if response.Error != nil {
			return UserChatsMsg{Err: fmt.Errorf("ERROR: %s", response.Error.Message)}
		}
		return UserChatsMsg{Err: nil, Response: &response}
	}

}
