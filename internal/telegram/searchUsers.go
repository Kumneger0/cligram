package telegram

import (
	"strconv"

	"github.com/gotd/td/tg"
)

type UserSearchResult struct {
	Result *struct {
		Users    []UserInfo            `json:"users"`
		Channels []ChannelAndGroupInfo `json:"channels"`
	} `json:"result,omitempty"`
}

type SearchUserMsg struct {
	Response *UserSearchResult
	Err      error
}

func (c *CligramClient) SearchUsers(query string) SearchUserMsg {
	request := &tg.ContactsSearchRequest{
		Q:     query,
		Limit: 10,
	}
	contactsFound, err := c.Client.API().ContactsSearch(c.ctx, request)
	if err != nil {
		return SearchUserMsg{Response: nil, Err: err}
	}

	var users []UserInfo
	for _, user := range contactsFound.Users {
		u := user.(*tg.User)
		users = append(users, UserInfo{
			FirstName:  u.FirstName,
			IsBot:      u.Bot,
			PeerID:     strconv.FormatInt(u.ID, 10),
			IsTyping:   false,
			AccessHash: strconv.FormatInt(u.AccessHash, 10),
			LastSeen:   nil,
			IsOnline:   false,
		})
	}

	var channels []ChannelAndGroupInfo

	for _, chatClass := range contactsFound.Chats {
		chat := chatClass.(*tg.Channel)
		userName := ""
		channels = append(channels, ChannelAndGroupInfo{
			ChannelTitle:      chat.Title,
			Username:          &userName,
			ChannelID:         strconv.FormatInt(chat.ID, 10),
			AccessHash:        strconv.FormatInt(chat.AccessHash, 10),
			IsCreator:         chat.Creator,
			IsBroadcast:       chat.Broadcast,
			ParticipantsCount: &chat.ParticipantsCount,
		})
	}

	searchResult := UserSearchResult{
		Result: &struct {
			Users    []UserInfo            "json:\"users\""
			Channels []ChannelAndGroupInfo "json:\"channels\""
		}{
			Users:    users,
			Channels: channels,
		},
	}

	return SearchUserMsg{Response: &searchResult, Err: nil}
}
