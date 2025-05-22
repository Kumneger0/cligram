package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

type LastSeen struct {
	Type   string
	Time   *time.Time
	Status *string
}



 type UserInfo struct {
	FirstName   string   `json:"firstName"`
	IsBot       bool     `json:"isBot"`
	PeerID      string   `json:"peerId"`
	AccessHash  string   `json:"accessHash"`
	unreadCount int      `json:"unreadCount"`
	LastSeen    LastSeen `json:"lastSeen"`
	IsOnline    bool     `json:"isOnline"`
}

type ChannelInfo struct {
	ChannelTitle            string  `json:"title"`
	Username          *string `json:"username"`
	ChannelID         string  `json:"channelId"`
	AccessHash        string  `json:"accessHash"`
	IsCreator         bool    `json:"isCreator"`
	IsBroadcast       bool    `json:"isBroadcast"`
	ParticipantsCount *int    `json:"participantsCount"`
	UnreadCount       int     `json:"unreadCount"`
}

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
	FromID *string `json:"fromId"`
}

func (u UserInfo) Title() string {
	return u.FirstName
}

func (u UserInfo) FilterValue() string {
	return u.FirstName
}

func (c ChannelInfo) FilterValue() string {
	return c.ChannelTitle
}

func (c ChannelInfo) Title() string {
	return c.ChannelTitle
}

type Model struct {
	Users           list.Model
	SelectedUser    UserInfo
	Channels        list.Model
	SelectedChannel ChannelInfo
	Height          int
	Width           int
	// mode = "users" | "channels" | "groups"
	// ideally i wanted to create like union type in typescript but i have no idea how can i do this in golang
	//figure out this later
	Mode  string
	Input textinput.Model
	// this one has also 3 possible values
	// sideBar | "mainView" | "input"
	FocusedOn     string
	Vp            viewport.Model
	Conversations []FormattedMessage
}


func (ls *LastSeen) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		ls.Type, ls.Time, ls.Status = "", nil, nil
		return nil
	}

	var aux struct {
		Type  string          `json:"type"`
		Value json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return fmt.Errorf("LastSeen: cannot unmarshal wrapper: %w", err)
	}

	ls.Type = aux.Type

	switch aux.Type {
	case "time":
		var t time.Time
		if err := json.Unmarshal(aux.Value, &t); err != nil {
			return fmt.Errorf("LastSeen: invalid time value: %w", err)
		}
		ls.Time = &t
		ls.Status = nil

	case "status":
		var s string
		if err := json.Unmarshal(aux.Value, &s); err != nil {
			return fmt.Errorf("LastSeen: invalid status value: %w", err)
		}
		ls.Status = &s
		ls.Time = nil

	default:
		return fmt.Errorf("LastSeen: unknown type %q", aux.Type)
	}
	return nil
}

func formatMessages(msgs []FormattedMessage) string {
	var lines []string
	for _, m := range msgs {
		timestamp := timestampStyle.Render(m.Date.Format("15:04"))

		var senderText string
		if m.IsFromMe {
			senderText = myMessageStyle.Render("You:")
		} else {
			senderText = senderStyle.Render(m.Sender + ":")
		}

		content := contentStyle.Render(m.Content)

		dateLine := lipgloss.JoinHorizontal(lipgloss.Top, timestamp)
		messageLine := lipgloss.JoinHorizontal(lipgloss.Top, senderText, content)

		fullMessage := lipgloss.JoinVertical(lipgloss.Left, dateLine, messageLine)

		lines = append(lines, messageStyle.Render(fullMessage))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}



func FakeConversations() []FormattedMessage {
	cwd, _ := os.Getwd()
	fakeConversationsPath := filepath.Join(cwd, "internal", "ui", "fakeConversation.txt")
	conversations, err := os.ReadFile(fakeConversationsPath)
	if err != nil {
		fmt.Println(err.Error())
	}

	var formatedMessage []FormattedMessage
	err = json.Unmarshal(conversations, &formatedMessage)
	if err != nil {
		fmt.Println(err.Error())
	}

	return formatedMessage
}