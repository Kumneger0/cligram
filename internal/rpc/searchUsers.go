package rpc

import (
	"encoding/json"
	"fmt"
)

type SearchUserMsg struct {
	Response *UserChatsJsonRpcResponse
	Err      error
}


func (c *JsonRpcClient) Search(query string) SearchUserMsg {
	userChatRpcResponse, err := c.Call("searchUsers", []string{query})
		if err != nil {
			return SearchUserMsg{Err: err}
		}
		var response UserChatsJsonRpcResponse
		if err := json.Unmarshal(userChatRpcResponse, &response); err != nil {
			return SearchUserMsg{Err: fmt.Errorf("failed to unmarshal response JSON '%s': %w", string(userChatRpcResponse), err)}
		}
		if response.Error != nil {
			return SearchUserMsg{Err: fmt.Errorf(response.Error.Message)}
		}
		return SearchUserMsg{Err: nil, Response: &response}
}
