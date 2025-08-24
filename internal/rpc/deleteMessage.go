package rpc

import "github.com/gotd/td/tg"

type SuccessDeleteMessageResult struct {
	Status string `json:"status"`
}

type DeleteMessageResultResponse struct {
	Result *SuccessDeleteMessageResult `json:"result,omitempty"`
}

func (c *TelegramClient) DeleteMessage(peerInfo PeerInfo, messageID int, chatType ChatType) (DeleteMessageResultResponse, error) {
	deleteMessageRequest := &tg.MessagesDeleteMessagesRequest{
		Revoke: true,
		ID:     []int{messageID},
	}
	_, err := c.Client.API().MessagesDeleteMessages(c.ctx, deleteMessageRequest)
	if err != nil {
		return DeleteMessageResultResponse{Result: &SuccessDeleteMessageResult{
			Status: "failed",
		}}, err
	}
	return DeleteMessageResultResponse{Result: &SuccessDeleteMessageResult{
		Status: "success",
	}}, nil
}
