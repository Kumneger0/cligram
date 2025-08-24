package rpc

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gotd/td/tg"
)

type UserChatsMsg struct {
	Response *[]UserInfo
	Err      error
}

func (c *TelegramClient) GetUserChats(isBot bool) (UserChatsMsg, error) {
	dilaogsSlice, err := getAllDialogs(c)
	if err != nil {
		return UserChatsMsg{Response: &[]UserInfo{}, Err: err}, err
	}

	if dilaogsSlice != nil {
		var users []UserInfo
		for _, userClass := range dilaogsSlice.Users {
			if user, ok := userClass.(*tg.User); ok && user.Bot == isBot {
				users = append(users, *userInfoFromTG(user, user.ID))
			}
		}

		return UserChatsMsg{
			Response: &users,
			Err:      nil,
		}, nil
	}
	return UserChatsMsg{Response: &[]UserInfo{}, Err: err}, err
}

func getAllDialogs(c *TelegramClient) (*tg.MessagesDialogsSlice, error) {
	dialogs, err := c.Client.API().MessagesGetDialogs(c.ctx, &tg.MessagesGetDialogsRequest{
		Limit:      50,
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

func (c *TelegramClient) GetUserChatsCmd(chatType Chat) tea.Cmd {
	return func() tea.Msg {
		userBotChats, err := c.GetUserChats(true)
		return UserChatsMsg{Response: userBotChats.Response, Err: err}
	}
}
