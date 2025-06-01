package ui

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	dialogBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#874BFD")).
		Padding(1, 2).
		BorderTop(true).
		BorderLeft(true).
		BorderRight(true).
		BorderBottom(true)
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
	UnreadCount int      `json:"unreadCount"`
	LastSeen    LastSeen `json:"lastSeen"`
	IsOnline    bool     `json:"isOnline"`
}

type ChannelAndGroupInfo struct {
	ChannelTitle      string  `json:"title"`
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

func (c ChannelAndGroupInfo) FilterValue() string {
	return c.ChannelTitle
}

func (c ChannelAndGroupInfo) Title() string {
	return c.ChannelTitle
}

type Model struct {
	Users           list.Model
	SelectedUser    UserInfo
	Channels        list.Model
	IsModalVisible  bool
	ModalContent    string
	SelectedChannel ChannelAndGroupInfo
	Groups          list.Model
	SelectedGroup   ChannelAndGroupInfo
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

// this is just temporary just to get things working
// definetly i need to remove this
func GetModalContent(errorMessage string) string {
	var modalContent strings.Builder
	modalContent.WriteString(errorMessage + "\n")
	modalContent.WriteString("\n" + "press ctrl + c or q to close")
	modalWidth := max(40, len(errorMessage)+4)
	return dialogBoxStyle.Width(modalWidth).Render(modalContent.String())
}

func setItemStyles(m *Model) string {
	if m.IsModalVisible {
		modalView := lipgloss.Place(
			m.Width,
			m.Height,
			lipgloss.Center, lipgloss.Center, m.ModalContent,
		)
		return modalView
	}

	sidebarWidth := m.Width * 30 / 100
	mainWidth := m.Width - sidebarWidth
	contentHeight := m.Height * 90 / 100
	inputHeight := m.Height - contentHeight

	m.Users.SetHeight(contentHeight - 4)
	m.Users.SetWidth(sidebarWidth)
	m.Channels.SetWidth(sidebarWidth)
	m.Channels.SetHeight(contentHeight - 4)

	m.Groups.SetWidth(sidebarWidth)

	m.Groups.SetHeight(contentHeight - 4)

	mainStyle := getMainStyle(mainWidth, contentHeight, m)

	var userNameOrChannelName string
	if m.Mode == "users" {
		userNameOrChannelName = m.SelectedUser.Title()
	}
	if m.Mode == "channels" {
		userNameOrChannelName = m.SelectedChannel.FilterValue()
	}
	if m.Mode == "groups" {
		userNameOrChannelName = m.SelectedGroup.FilterValue()
	}

	title := titleStyle.Render(userNameOrChannelName)
	line := strings.Repeat("─", max(0, mainWidth-4-lipgloss.Width(title)))
	headerView := lipgloss.JoinVertical(lipgloss.Center, title, line)

	mainContent := lipgloss.JoinVertical(
		lipgloss.Top,
		headerView,
		m.Vp.View(),
	)

	chatView := mainStyle.Render(mainContent)
	var sideBarContent string
	if m.Mode == "users" {
		sideBarContent = m.Users.View()
	} else if m.Mode == "channels" {
		sideBarContent = m.Channels.View()
	} else {
		sideBarContent = m.Groups.View()
	}
	sidebar := getSideBarStyles(sidebarWidth, contentHeight, m).Render(sideBarContent)

	if m.FocusedOn == "input" {
		m.Input.Focus()
	}

	inputView := getInputStyle(m, inputHeight).Render(m.Input.View())
	row := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, chatView)
	ui := lipgloss.JoinVertical(lipgloss.Top, row, inputView)
	return ui
}

func Debounce(fn func(args ...interface{}) tea.Msg, delay time.Duration) func(args ...interface{}) tea.Cmd {
	var mu sync.Mutex
	var timer *time.Timer
	var lastArgs []interface{}

	return func(args ...interface{}) tea.Cmd {
		mu.Lock()
		defer mu.Unlock()

		lastArgs = args

		if timer != nil {
			timer.Stop()
		}

		return func() tea.Msg {
			msgChan := make(chan tea.Msg, 1)

			timer = time.AfterFunc(delay, func() {
				mu.Lock()
				defer mu.Unlock()

				argsToPass := make([]interface{}, len(lastArgs))
				copy(argsToPass, lastArgs)

				msgChan <- fn(argsToPass...)
				close(msgChan)
			})

			return <-msgChan
		}
	}
}
