package rpc

import (
	"encoding/json"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type UserChatsMsg struct {
	Response *UserChatsJSONRPCResponse
	Err      error
}

type UserChatsJSONRPCResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Error   *struct {
		Code    int         `json:"code"`
		Message string      `json:"message"`
		Data    interface{} `json:"data,omitempty"`
	} `json:"error,omitempty"`
	Result []UserInfo `json:"result,omitempty"`
}

func (c *JSONRPCClient) GetUserChats() UserChatsMsg {
	if c == nil {
		return UserChatsMsg{Err: fmt.Errorf("js backend is not running try restarting the app, please open an issue on github if the problem persists")}
	}
	userChatRPCResponse, err := c.Call("getUserChats", []string{"user"})

	if err != nil {
		return UserChatsMsg{Err: err}
	}
	var response UserChatsJSONRPCResponse
	if err := json.Unmarshal(userChatRPCResponse, &response); err != nil {
		return UserChatsMsg{Err: fmt.Errorf("failed to unmarshal response JSON '%s': %w", string(userChatRPCResponse), err)}
	}
	if response.Error != nil {
		return UserChatsMsg{Err: fmt.Errorf("ERROR: %s", response.Error.Message)}
	}
	return UserChatsMsg{Err: nil, Response: &response}
}

type Chat string

const (
	ModeUser Chat = "user"
	ModeBot  Chat = "bot"
)

func (c *JSONRPCClient) GetUserChatsCmd(chatType Chat) tea.Cmd {
	return func() tea.Msg {
		userChatRPCResponse, err := c.Call("getUserChats", []string{string(chatType)})
		if err != nil {
			return UserChatsMsg{Err: err}
		}
		var response UserChatsJSONRPCResponse
		if err := json.Unmarshal(userChatRPCResponse, &response); err != nil {
			return UserChatsMsg{Err: fmt.Errorf("failed to unmarshal response JSON '%s': %w", string(userChatRPCResponse), err)}
		}
		if response.Error != nil {
			return UserChatsMsg{Err: fmt.Errorf("ERROR: %s", response.Error.Message)}
		}
		return UserChatsMsg{Err: nil, Response: &response}
	}
}
