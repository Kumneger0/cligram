package rpc

import (
	"encoding/json"
	"log/slog"
	"time"
)

type SetUserTypingJSONRPCResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    any    `json:"data,omitempty"`
	} `json:"error,omitempty"`
	Result bool `json:"result,omitempty"`
}

func (c *JSONRPCClient) SetUserTyping(userPeer PeerInfo, chatType ChatType) {
	time.Sleep(1000 * time.Millisecond)
	rpcResponse, err := c.Call("setUserTyping", []any{userPeer, chatType})
	if err != nil {
		slog.Error(err.Error())
	}
	var response SetUserTypingJSONRPCResponse
	if err := json.Unmarshal(rpcResponse, &response); err != nil {
		slog.Error(err.Error())
	}
	if response.Error != nil {
		slog.Error(err.Error())
	}
}
