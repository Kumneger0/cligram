package telegram

import (
	"encoding/json"
	"math/rand"
	"strconv"

	"github.com/gotd/td/tg"
)

type ForwardMessagesRPCResponse struct {
	Result *json.RawMessage `json:"result,omitempty"`
}

func (c *CligramClient) ForwardMessages(
	fromPeer PeerInfo,
	messageIDs []int,
	toPeer PeerInfo,
	toPeerType ChatType,
	fromPeerType ChatType,
) (ForwardMessagesRPCResponse, error) {
	var from tg.InputPeerClass
	var to tg.InputPeerClass

	peerID, err := strconv.ParseInt(fromPeer.PeerID, 10, 64)

	if err != nil {
		return ForwardMessagesRPCResponse{Result: nil}, err
	}

	toPeerID, err := strconv.ParseInt(toPeer.PeerID, 10, 64)

	if err != nil {
		return ForwardMessagesRPCResponse{Result: nil}, err
	}

	switch fromPeerType {
	case UserChat, ChatType(Bot):
		from = &tg.InputPeerUser{
			UserID: peerID,
		}
	case ChannelChat, GroupChat:
		from = &tg.InputPeerChannel{
			ChannelID: peerID,
		}
	}

	switch toPeerType {
	case UserChat, ChatType(Bot):
		to = &tg.InputPeerUser{
			UserID: toPeerID,
		}
	case ChannelChat, GroupChat:
		to = &tg.InputPeerChannel{
			ChannelID: toPeerID,
		}
	}

	forwardMessages := &tg.MessagesForwardMessagesRequest{
		FromPeer: from,
		ToPeer:   to,
		RandomID: []int64{rand.Int63()},
		ID:       messageIDs,
	}
	_, err = c.Client.API().MessagesForwardMessages(c.ctx, forwardMessages)

	if err != nil {
		return ForwardMessagesRPCResponse{}, err
	}
	return ForwardMessagesRPCResponse{}, nil
}
