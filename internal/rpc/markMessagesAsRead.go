package rpc

import (
	"fmt"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gotd/td/tg"
)

type MarkMessagesAsReadMsg struct {
	Response bool
	Err      error
}

func (c *TelegramClient) MarkMessagesAsRead(userPeer PeerInfo, chatType ChatType) tea.Cmd {
	return func() tea.Msg {
		var peer tg.InputPeerClass
		peerID, err := strconv.ParseInt(userPeer.PeerID, 10, 64)
		if err != nil {
			return MarkMessagesAsReadMsg{Response: false, Err: err}
		}
		accessHash, err := strconv.ParseInt(userPeer.PeerID, 10, 64)
		if err != nil {
			return MarkMessagesAsReadMsg{Response: false, Err: err}
		}
		switch chatType {
		case UserChat, ChatType(Bot):
			peer = &tg.InputPeerUser{
				UserID:     peerID,
				AccessHash: accessHash,
			}
		case ChannelChat, GroupChat:
			peer = &tg.InputPeerChannel{
				ChannelID:  peerID,
				AccessHash: accessHash,
			}
		}
		markHistoryASReadReqeust := &tg.MessagesReadHistoryRequest{
			Peer: peer,
		}
		_, err = c.Client.API().MessagesReadHistory(c.ctx, markHistoryASReadReqeust)

		if err != nil {
			fmt.Println(err)
			return MarkMessagesAsReadMsg{Response: false, Err: err}
		}
		return MarkMessagesAsReadMsg{Response: true, Err: nil}
	}
}
