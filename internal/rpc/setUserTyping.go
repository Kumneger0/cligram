package rpc

import (
	"encoding/json"
	"fmt"
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

func (c *JsonRpcClient) SetUserTyping(userPeer PeerInfo, chatType ChatType) error {
	rpcResponse, err := c.Call("setUserTyping", []interface{}{userPeer, chatType})
	if err != nil {
		return err
	}
	var response SetUserTypingJsonRpcResponse
	if err := json.Unmarshal(rpcResponse, &response); err != nil {
		return fmt.Errorf("failed to unmarshal response JSON '%s': %w", string(rpcResponse), err)
	}
	if response.Error != nil {
		return fmt.Errorf("ERROR: %s", response.Error.Message)
	}
	return nil
}
