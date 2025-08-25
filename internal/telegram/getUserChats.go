package telegram

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gotd/td/tg"
)

type UserChatsMsg struct {
	Response *[]UserInfo
	Err      error
}

type UserPeerIDUnreadCount struct {
	unreadCount int
	peerID      int64
}

func getUnreadCount(u []UserPeerIDUnreadCount, peerID int64) int {
	var unreadCount int
	for _, p := range u {
		if p.peerID == peerID {
			unreadCount = p.unreadCount
		}
	}
	return unreadCount
}

func (c *CligramClient) GetUserChats(isBot bool) (UserChatsMsg, error) {
	dilaogsSlice, err := getAllDialogs(c)
	if err != nil {
		return UserChatsMsg{Response: &[]UserInfo{}, Err: err}, err
	}

	if dilaogsSlice != nil {
		var userPeerIDs []UserPeerIDUnreadCount
		for _, userClass := range dilaogsSlice.Dialogs {
			if dialog, ok := userClass.(*tg.Dialog); ok {
				if peer, ok := dialog.Peer.(*tg.PeerUser); ok {
					userPeerIDs = append(userPeerIDs, UserPeerIDUnreadCount{
						unreadCount: dialog.UnreadCount,
						peerID:      peer.UserID,
					})
				}
			}
		}

		var users []UserInfo
		for _, userClass := range dilaogsSlice.Users {
			if userTG, ok := userClass.(*tg.User); ok && userTG.Bot == isBot {
				userLastSeenStatus := getUserOnlineStatus(userTG.Status)
				user := *userInfoFromTG(userTG, userTG.ID)
				unreadCount := getUnreadCount(userPeerIDs, userTG.ID)
				user.IsOnline = userLastSeenStatus.IsOnline
				user.LastSeen = userLastSeenStatus.LastSeen
				user.UnreadCount = unreadCount
				users = append(users, user)
			}
		}

		return UserChatsMsg{
			Response: &users,
			Err:      nil,
		}, nil
	}
	return UserChatsMsg{Response: &[]UserInfo{}, Err: err}, err
}

func getAllDialogs(c *CligramClient) (*tg.MessagesDialogsSlice, error) {
	dialogs, err := c.Client.API().MessagesGetDialogs(c.ctx, &tg.MessagesGetDialogsRequest{
		Limit:      1000,
		OffsetPeer: &tg.InputPeerSelf{},
	})

	if err != nil {
		return nil, err
	}

	if dilaogsSlice, ok := dialogs.(*tg.MessagesDialogsSlice); ok {
		return dilaogsSlice, nil
	}
	return nil, fmt.Errorf("opps there was an eror while getting dilos")
}

type Chat string

const (
	ModeUser Chat = "user"
	ModeBot  Chat = "bot"
)

func (c *CligramClient) GetUserChatsCmd(chatType Chat) tea.Cmd {
	return func() tea.Msg {
		userBotChats, err := c.GetUserChats(chatType == ModeBot)
		return UserChatsMsg{Response: userBotChats.Response, Err: err}
	}
}
