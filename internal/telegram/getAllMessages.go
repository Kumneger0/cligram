package telegram

import (
	"fmt"
	"slices"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gotd/td/tg"
)

type FormattedMessage struct {
	ID                   int64     `json:"id"`
	Sender               string    `json:"sender"`
	Content              string    `json:"content"`
	IsFromMe             bool      `json:"isFromMe"`
	Media                *string   `json:"media,omitempty"`
	Date                 time.Time `json:"date"`
	IsUnsupportedMessage bool      `json:"isUnsupportedMessage"`
	WebPage              *struct {
		URL        string  `json:"url"`
		DisplayURL *string `json:"displayUrl,omitempty"`
	} `json:"webPage,omitempty"`
	Document *struct {
		Document string `json:"document"`
	} `json:"document,omitempty"`
	FromID         *string   `json:"fromId"`
	SenderUserInfo *UserInfo `json:"senderUserInfo,omitempty"`
}

type GetMessagesMsg struct {
	Messages [50]FormattedMessage
	Err      error
}

func (m FormattedMessage) Title() string {
	return m.Content
}

func (m FormattedMessage) FilterValue() string {
	return m.Content
}

type PeerInfoParams struct {
	AccessHash                  string `json:"accessHash"`
	PeerID                      string `json:"peerId"`
	UserFirstNameOrChannelTitle string `json:"userFirtNameOrChannelTitle"`
}

type ChatType string

const (
	UserChat    ChatType = "user"
	GroupChat   ChatType = "group"
	ChannelChat ChatType = "channel"
	Bot         ChatType = "bot"
)

type IterParams map[string]any

func (c *CligramClient) GetAllMessages(
	peerInfo PeerInfoParams,
	chatType ChatType,
	limit int,
	offsetID *int,
	chatAreaWidth *int,
	itParams IterParams,
) tea.Cmd {
	return func() tea.Msg {
		parseInt64 := func(s string) (int64, error) {
			return strconv.ParseInt(s, 10, 64)
		}

		userID, err := parseInt64(peerInfo.PeerID)
		if err != nil {
			return GetMessagesMsg{Messages: [50]FormattedMessage{}, Err: err}
		}
		accessHash, err := parseInt64(peerInfo.AccessHash)
		if err != nil {
			return GetMessagesMsg{Messages: [50]FormattedMessage{}, Err: err}
		}

		var inputPeer tg.InputPeerClass
		switch string(chatType) {
		case "user":
			inputPeer = &tg.InputPeerUser{UserID: userID, AccessHash: accessHash}
		case "channel":
			inputPeer = &tg.InputPeerChannel{ChannelID: userID, AccessHash: accessHash}
		case "group":
			inputPeer = &tg.InputPeerChat{ChatID: userID}
		default:
			inputPeer = &tg.InputPeerUser{UserID: userID, AccessHash: accessHash}
		}

		req := &tg.MessagesGetHistoryRequest{
			Peer:  inputPeer,
			Limit: limit,
		}
		if offsetID != nil {
			req.OffsetID = *offsetID
		}

		history, err := c.Client.API().MessagesGetHistory(c.ctx, req)
		if err != nil {
			return GetMessagesMsg{Messages: [50]FormattedMessage{}, Err: err}
		}

		var msgs []tg.MessageClass
		switch h := history.(type) {
		case *tg.MessagesMessagesSlice:
			msgs = h.Messages
		case *tg.MessagesChannelMessages:
			msgs = h.Messages
		case *tg.MessagesMessages:
			msgs = h.Messages
		default:
			return GetMessagesMsg{Messages: [50]FormattedMessage{}, Err: fmt.Errorf("unsupported history type: %T", history)}
		}

		var out [50]FormattedMessage
		writeIndex := 0
		limitIter := min(len(msgs), 50)

		for _, m := range msgs[:limitIter] {
			if writeIndex >= 50 {
				break
			}
			msg, ok := m.(*tg.Message)
			if !ok {
				continue
			}

			sender := ""
			if msg.Out {
				sender = "you"
			} else {
				peerUser := getPeerID(msg.PeerID)
				if peerUser != nil {
					if string(peerUser.ChatType) == "user" {
						inputUser := &tg.InputUser{UserID: peerUser.peerID}
						if users, err := c.Client.API().UsersGetUsers(c.ctx, []tg.InputUserClass{inputUser}); err == nil && len(users) > 0 {
							if u, ok := users[0].(*tg.User); ok {
								sender = u.FirstName
							}
						}
					}
				}
			}

			var fromID *string
			if !msg.Out {
				peerUser := getPeerID(msg.PeerID)
				if peerUser != nil {
					id := strconv.FormatInt(peerUser.peerID, 10)
					fromID = &id
				}
			}

			out[writeIndex] = FormattedMessage{
				ID:                   int64(msg.ID),
				Sender:               sender,
				Content:              msg.Message,
				IsFromMe:             msg.Out,
				Media:                nil,
				Date:                 time.Unix(int64(msg.Date), 0),
				IsUnsupportedMessage: false,
				WebPage:              nil,
				Document:             nil,
				FromID:               fromID,
			}
			writeIndex++
		}

		if writeIndex > 1 {
			sliceToReverse := out[:writeIndex]
			slices.Reverse(sliceToReverse)
			copy(out[:writeIndex], sliceToReverse)
		}

		return GetMessagesMsg{Messages: out, Err: nil}
	}
}

type PeerType struct {
	ChatType
	peerID int64
}

func getPeerID(peerClass tg.PeerClass) *PeerType {
	switch peer := peerClass.(type) {
	case *tg.PeerUser:
		return &PeerType{ChatType: UserChat, peerID: peer.UserID}
	case *tg.PeerChannel:
		return &PeerType{ChatType: ChannelChat, peerID: peer.ChannelID}
	case *tg.PeerChat:
		return &PeerType{ChatType: GroupChat, peerID: peer.ChatID}
	default:
		break
	}
	return nil
}
