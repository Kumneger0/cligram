package rpc

import (
	"encoding/json"
	"fmt"
)

type UserSearchRPCResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    any    `json:"data,omitempty"`
	} `json:"error,omitempty"`
	Result *struct {
		Users    []UserInfo            `json:"users"`
		Channels []ChannelAndGroupInfo `json:"channels"`
	} `json:"result,omitempty"`
}

type SearchUserMsg struct {
	Response *UserSearchRPCResponse
	Err      error
}

func (c *JSONRPCClient) SearchUsers(query string) SearchUserMsg {
	userChatRPCResponse, err := c.Call("searchUsers", []string{query})
	if err != nil {
		return SearchUserMsg{Err: err}
	}
	var response UserSearchRPCResponse
	if err := json.Unmarshal(userChatRPCResponse, &response); err != nil {
		return SearchUserMsg{Err: fmt.Errorf("failed to unmarshal response JSON '%s': %w", string(userChatRPCResponse), err)}
	}
	if response.Error != nil {
		return SearchUserMsg{Err: fmt.Errorf("ERROR: %s", response.Error.Message)}
	}
	return SearchUserMsg{Err: nil, Response: &response}
}
