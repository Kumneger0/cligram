package telegram

import (
	"log/slog"
	"strconv"

	"github.com/gotd/td/tg"
)

type SetUserTypingJSONRPCResponse struct {
	Result bool `json:"result,omitempty"`
}

func (c *CligramClient) SetUserTyping(userPeer PeerInfo, chatType ChatType) {
	var peer tg.InputPeerClass
	switch chatType {
	case UserChat, ChatType(Bot):
		userID, err := strconv.ParseInt(userPeer.PeerID, 10, 64)
		if err != nil {
			slog.Error(err.Error())
		}
		peer = &tg.InputPeerUser{
			UserID: userID,
		}
	case ChannelChat, GroupChat:
		channelID, err := strconv.ParseInt(userPeer.PeerID, 10, 64)
		if err != nil {
			slog.Error(err.Error())
		}
		peer = &tg.InputPeerChannel{
			ChannelID: channelID,
		}
	}
	request := &tg.MessagesSetTypingRequest{
		Peer:   peer,
		Action: &tg.SendMessageTypingAction{},
	}
	_, err := c.Client.API().MessagesSetTyping(c.ctx, request)
	if err != nil {
		slog.Error(err.Error())
	}
}
