package rpc

import (
	"encoding/json"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type UserChatsMsg struct {
	Response *UserChatsJsonRpcResponse
	Err      error
}

type DuplicatedLastSeen struct {
	Type   string
	Time   *time.Time
	Status *string
}

type DuplicatedUserInfo struct {
	FirstName   string             `json:"firstName"`
	IsBot       bool               `json:"isBot"`
	PeerID      string             `json:"peerId"`
	AccessHash  string             `json:"accessHash"`
	UnreadCount int                `json:"unreadCount"`
	LastSeen    DuplicatedLastSeen `json:"lastSeen"`
	IsOnline    bool               `json:"isOnline"`
}

// there is a lot of room here for refacoring
// the the first one is i redefined userInfo here b/c of import cycle
// figure out and eliminate
func (du DuplicatedUserInfo) Title() string {
	return du.FirstName
}

func (du DuplicatedUserInfo) FilterValue() string {
	return du.FirstName
}

type UserChatsJsonRpcResponse struct {
	JsonRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Error   *struct {
		Code    int         `json:"code"`
		Message string      `json:"message"`
		Data    interface{} `json:"data,omitempty"`
	} `json:"error,omitempty"`
	Result []DuplicatedUserInfo `json:"result,omitempty"`
}

func (c *JsonRpcClient) GetUserChats() tea.Cmd {
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
			return UserChatsMsg{Err: fmt.Errorf(response.Error.Message)}
		}
		return UserChatsMsg{Err: nil, Response: &response}
	}
}




// this needs to be removed this is redifinded
// ideally we should use the LastSeen struct it has UnmarshalJSON function
// i just redefined b/c of import cyle issue
// will remove this after refactoring the the code base
// this was just quick fix
// TODO:remove this function it is duplicated

func (ls *DuplicatedLastSeen) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		ls.Type, ls.Time, ls.Status = "", nil, nil
		return nil
	}

	var aux struct {
		Type  string          `json:"type"`
		Value json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return fmt.Errorf("LastSeen: cannot unmarshal wrapper: %w", err)
	}

	ls.Type = aux.Type

	switch aux.Type {
	case "time":
		var t time.Time
		if err := json.Unmarshal(aux.Value, &t); err != nil {
			return fmt.Errorf("LastSeen: invalid time value: %w", err)
		}
		ls.Time = &t
		ls.Status = nil

	case "status":
		var s string
		if err := json.Unmarshal(aux.Value, &s); err != nil {
			return fmt.Errorf("LastSeen: invalid status value: %w", err)
		}
		ls.Status = &s
		ls.Time = nil

	default:
		return fmt.Errorf("LastSeen: unknown type %q", aux.Type)
	}
	return nil
}