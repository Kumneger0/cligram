package rpc

import (
	"encoding/json"
)

type chatType string

const (
	ChatTypeUser    chatType = "user"
	ChatTypeGroup   chatType = "group"
	ChatTypeChannel chatType = "channel"
)

type forwardMessagesMethodParams struct {
	FromPeer PeerInfo `json:"fromPeer"`
	IDs      []int    `json:"id"`
	ToPeer   PeerInfo `json:"toPeer"`
	Type     ChatType `json:"type"`
}

type ForwardMessagesRpcResponse struct {
	JsonRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Error   *struct {
		Code    int         `json:"code"`
		Message string      `json:"message"`
		Data    interface{} `json:"data,omitempty"`
	} `json:"error,omitempty"`
	Result *json.RawMessage `json:"result,omitempty"`
}

type forwardMessageMsg struct {
	result ForwardMessagesRpcResponse
	err    error
}

func (c *JsonRpcClient) ForwardMessages(
	fromPeer PeerInfo,
	messageIDs []int,
	toPeer PeerInfo,
	chatType ChatType,
) (ForwardMessagesRpcResponse, error) {
	methodParams := forwardMessagesMethodParams{
		FromPeer: fromPeer,
		IDs:      messageIDs,
		ToPeer:   toPeer,
		Type:     chatType,
	}

	rpcCallParams := []interface{}{methodParams}

	responseBytes, err := c.Call("forwardMessage", rpcCallParams)
	if err != nil {
		return ForwardMessagesRpcResponse{}, err
	}

	var rpcResponse ForwardMessagesRpcResponse
	if err := json.Unmarshal(responseBytes, &rpcResponse); err != nil {
		return ForwardMessagesRpcResponse{}, err
	}

	return rpcResponse, nil
}
