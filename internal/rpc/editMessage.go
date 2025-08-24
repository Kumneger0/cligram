package rpc

import (
	"log/slog"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gotd/td/tg"
)

type EditMessageJSONRPCResponse struct {
	Result bool `json:"result,omitempty"`
}

type EditMessageMsg struct {
	Response       EditMessageJSONRPCResponse
	Err            error
	UpdatedMessage string
}

func (c *TelegramClient) EditMessage(userPeer PeerInfo, chatType ChatType, messageID int, newMessage string) tea.Cmd {
	return func() tea.Msg {
		var peer tg.InputPeerClass

		accessHash, err := strconv.ParseInt(userPeer.AccessHash, 10, 64)

		if err != nil {
			slog.Error(err.Error())
			return nil
		}

		peerID, err := strconv.ParseInt(userPeer.PeerID, 10, 64)

		if chatType == UserChat || chatType == ChatType(Bot) {
			peer = &tg.InputPeerUser{
				UserID:     peerID,
				AccessHash: accessHash,
			}
		}

		if chatType == ChannelChat || chatType == GroupChat {
			peer = &tg.InputPeerChannel{
				ChannelID:  peerID,
				AccessHash: accessHash,
			}
		}

		if err != nil {
			slog.Error(err.Error())
			return nil
		}

		editMessageRequest := &tg.MessagesEditMessageRequest{
			Peer:    peer,
			Message: newMessage,
			ID:      messageID,
		}

		_, err = c.Client.API().MessagesEditMessage(c.ctx, editMessageRequest)

		if err != nil {
			slog.Error(err.Error())
			return EditMessageMsg{
				Response:       EditMessageJSONRPCResponse{Result: false},
				Err:            err,
				UpdatedMessage: newMessage,
			}
		}

		return EditMessageMsg{
			Response: EditMessageJSONRPCResponse{
				Result: true,
			},
			Err:            nil,
			UpdatedMessage: newMessage,
		}
	}
}
