package telegram

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gotd/td/tg"
)

type UserChannelResponse struct {
	Result []ChannelAndGroupInfo `json:"result,omitempty"`
}

type UserGroupsMsg struct {
	Err error
	//groupds and channels share same propertiy so we don't have to redefine its types use this instead
	Response *UserChannelResponse
}

type UserChannelMsg struct {
	Err      error
	Response *UserChannelResponse
}

func (c *CligramClient) GetUserChannel(broadcast bool) tea.Cmd {
	return func() tea.Msg {
		dilaogsSlice, err := getAllDialogs(c)
		if err != nil {
			if broadcast {
				return UserChatsMsg{Err: err}
			}
			return UserGroupsMsg{Err: err}
		}

		var userPeerIDs []UserPeerIDUnreadCount
		if dilaogsSlice != nil {
			for _, userClass := range dilaogsSlice.Dialogs {
				if dialog, ok := userClass.(*tg.Dialog); ok {
					if peer, ok := dialog.Peer.(*tg.PeerChannel); ok {
						userPeerIDs = append(userPeerIDs, UserPeerIDUnreadCount{
							unreadCount: dialog.UnreadCount,
							peerID:      peer.ChannelID,
						})
					}
				}
			}
		}

		var channels []ChannelAndGroupInfo
		for _, dialogClass := range dilaogsSlice.Chats {
			if peer, ok := dialogClass.(*tg.Channel); ok && peer.Broadcast == broadcast {
				channel := channelAndGroupInfoFromTG(peer)
				unreadCount := getUnreadCount(userPeerIDs, peer.ID)
				channel.UnreadCount = unreadCount
				channels = append(channels, *channel)
			}
		}

		if broadcast {
			return UserChannelMsg{Err: nil, Response: &UserChannelResponse{
				Result: channels,
			}}
		}

		return UserGroupsMsg{Err: nil, Response: &UserChannelResponse{
			Result: channels,
		}}
	}
}
