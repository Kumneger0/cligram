package rpc

import (
	"encoding/json"
	"fmt"
)

type UserSearchRpcResponse struct {
	JsonRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Error   *struct {
		Code    int         `json:"code"`
		Message string      `json:"message"`
		Data    interface{} `json:"data,omitempty"`
	} `json:"error,omitempty"`
	Result *struct {
		Users    []UserInfo            `json:"users"`
		Channels []ChannelAndGroupInfo `json:"channels"`
	} `json:"result,omitempty"`
}

type SearchUserMsg struct {
	Response *UserSearchRpcResponse
	Err      error
}

func (c *JsonRpcClient) Search(query string) SearchUserMsg {
	userChatRpcResponse, err := c.Call("searchUsers", []string{query})
	if err != nil {
		return SearchUserMsg{Err: err}
	}
	var response UserSearchRpcResponse
	if err := json.Unmarshal(userChatRpcResponse, &response); err != nil {
		return SearchUserMsg{Err: fmt.Errorf("failed to unmarshal response JSON '%s': %w", string(userChatRpcResponse), err)}
	}
	if response.Error != nil {
		return SearchUserMsg{Err: fmt.Errorf("ERROR: %s", response.Error.Message)}
	}
	return SearchUserMsg{Err: nil, Response: &response}
}
