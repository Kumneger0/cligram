package rpc

import "encoding/json"

type SuccessDeleteMessageResult struct {
	Status string `json:"status"`
}

type DeleteMessageResultResponse struct {
	JsonRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Error   *struct {
		Code    int         `json:"code"`
		Message string      `json:"message"`
		Data    interface{} `json:"data,omitempty"`
	} `json:"error,omitempty"`
	Result *SuccessDeleteMessageResult `json:"result,omitempty"`
}

func (c *JsonRpcClient) DeleteMessage(peerInfo PeerInfo, messageId int, chatType ChatType) (DeleteMessageResultResponse, error) {
	rpcCallParams := []interface{}{}
	rpcCallParams = append(rpcCallParams, peerInfo)
	rpcCallParams = append(rpcCallParams, messageId)
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
