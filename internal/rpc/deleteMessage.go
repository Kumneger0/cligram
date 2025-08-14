package rpc

import "encoding/json"

type SuccessDeleteMessageResult struct {
	Status string `json:"status"`
}

type DeleteMessageResultResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    any    `json:"data,omitempty"`
	} `json:"error,omitempty"`
	Result *SuccessDeleteMessageResult `json:"result,omitempty"`
}

func (c *JSONRPCClient) DeleteMessage(peerInfo PeerInfo, messageID int, chatType ChatType) (DeleteMessageResultResponse, error) {
	rpcCallParams := []any{}
	rpcCallParams = append(rpcCallParams, peerInfo)
	rpcCallParams = append(rpcCallParams, messageID)
	rpcCallParams = append(rpcCallParams, chatType)

	responseBytes, err := c.Call("deleteMessage", rpcCallParams)
	if err != nil {
		return DeleteMessageResultResponse{}, err
	}

	var rpcResponse DeleteMessageResultResponse
	if err := json.Unmarshal(responseBytes, &rpcResponse); err != nil {
		return DeleteMessageResultResponse{}, err
	}

	return rpcResponse, nil
}
