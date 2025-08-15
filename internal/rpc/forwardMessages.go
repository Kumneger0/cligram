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

type ForwardMessagesRPCResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    any    `json:"data,omitempty"`
	} `json:"error,omitempty"`
	Result *json.RawMessage `json:"result,omitempty"`
}

func (c *JSONRPCClient) ForwardMessages(
	fromPeer PeerInfo,
	messageIDs []int,
	toPeer PeerInfo,
	chatType ChatType,
) (ForwardMessagesRPCResponse, error) {
	methodParams := forwardMessagesMethodParams{
		FromPeer: fromPeer,
		IDs:      messageIDs,
		ToPeer:   toPeer,
		Type:     chatType,
	}

	rpcCallParams := []any{methodParams}

	responseBytes, err := c.Call("forwardMessage", rpcCallParams)
	if err != nil {
		return ForwardMessagesRPCResponse{}, err
	}

	var rpcResponse ForwardMessagesRPCResponse
	if err := json.Unmarshal(responseBytes, &rpcResponse); err != nil {
		return ForwardMessagesRPCResponse{}, err
	}

	return rpcResponse, nil
}
