package rpc

import (
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gotd/td/tg"
)

type UserChannelResponse struct {
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    any    `json:"data,omitempty"`
	} `json:"error,omitempty"`
	Result []ChannelAndGroupInfo `json:"result,omitempty"`
}

type UserChannelMsg struct {
	Err      error
	Response *UserChannelResponse
}

func (c *TelegramClient) GetUserChannel() tea.Cmd {
	return func() tea.Msg {
		dilaogsSlice, err := getAllDialogs(c)

		if err != nil {
			return UserChatsMsg{Err: err}
		}

		var channel []ChannelAndGroupInfo

		for _, dialogClass := range dilaogsSlice.Chats {
			if peer, ok := dialogClass.(*tg.Channel); ok && peer.Broadcast {
				channel = append(channel, ChannelAndGroupInfo{
					ChannelTitle:      peer.Title,
					Username:          &peer.Username,
					ChannelID:         strconv.FormatInt(peer.ID, 10),
					AccessHash:        strconv.FormatInt(peer.AccessHash, 10),
					IsCreator:         peer.Creator,
					IsBroadcast:       peer.Broadcast,
					ParticipantsCount: &peer.ParticipantsCount,
					UnreadCount:       0,
				})
			}
		}
		return UserChannelMsg{Err: nil, Response: &UserChannelResponse{
			Error:  nil,
			Result: channel,
		}}
	}
}
