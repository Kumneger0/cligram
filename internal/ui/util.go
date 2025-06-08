package ui

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
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

type UserInfo struct {
	FirstName   string  `json:"firstName"`
	IsBot       bool    `json:"isBot"`
	PeerID      string  `json:"peerId"`
	AccessHash  string  `json:"accessHash"`
	UnreadCount int     `json:"unreadCount"`
	LastSeen    *string `json:"lastSeen"`
	IsOnline    bool    `json:"isOnline"`
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

func (m FormattedMessage) Title() string {
	return m.Content
}

func (m FormattedMessage) FilterValue() string {
	return m.Content
}

type MessagesDelegate struct {
	list.DefaultDelegate
}

func (d MessagesDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	var title string

	if entry, ok := item.(FormattedMessage); ok {
		title = entry.Title()
		if entry.IsFromMe {
			title = "You: " + title
		} else {
			title = entry.Sender + ": " + title
		}
	} else {
		return
	}
	str := lipgloss.NewStyle().Width(50).Height(2).Render(title)
	if index == m.Index() {
		fmt.Fprint(w, selectedStyle.Render(" "+str+" "))
	} else {
		fmt.Fprint(w, normalStyle.Render(" "+str+" "))
	}
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

type Mode string

const (
	ModeUsers    Mode = "users"
	ModeChannels Mode = "channels"
	ModeGroups   Mode = "groups"
)

type FocusedOn string

const (
	SideBar  FocusedOn = "sideBar"
	Mainview FocusedOn = "mainView"
	Input    FocusedOn = "input"
)

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
	Mode            Mode
	Input           textinput.Model
	FocusedOn       FocusedOn
	ChatUI          list.Model
	Conversations   []FormattedMessage
	IsReply         bool
	ReplyTo         *FormattedMessage
}

func formatMessages(msgs []FormattedMessage) []list.Item {
	var lines []list.Item
	for _, m := range msgs {
		lines = append(lines, FormattedMessage{
			ID:                   m.ID,
			Sender:               m.Sender,
			Content:              m.Content,
			IsFromMe:             m.IsFromMe,
			Media:                m.Media,
			Date:                 m.Date,
			IsUnsupportedMessage: m.IsUnsupportedMessage,
			WebPage:              m.WebPage,
			Document:             m.Document,
			FromID:               m.FromID,
		})
	}

	return lines
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
	m.ChatUI.SetItems(formatMessages(m.Conversations))

	w := mainWidth * 70 / 100
	m.ChatUI.SetWidth(w)
	m.ChatUI.SetHeight(15)
	var userNameOrChannelName string
	if m.Mode == ModeUsers {
		lastSeenTime := m.SelectedUser.LastSeen
		userNameOrChannelName = m.SelectedUser.Title()

		if m.SelectedUser.IsOnline {
			userNameOrChannelName += " " + "Online"
		} else if lastSeenTime != nil {
			userNameOrChannelName += " " + *lastSeenTime
		}
	}

	if m.Mode == ModeChannels {
		selectedChannel := m.SelectedChannel
		userNameOrChannelName = selectedChannel.FilterValue()
		if selectedChannel.ParticipantsCount != nil {
			var sb strings.Builder
			sb.WriteString(selectedChannel.FilterValue())
			sb.WriteString(" ")
			if selectedChannel.ParticipantsCount != nil {
				sb.WriteString(strconv.Itoa(*selectedChannel.ParticipantsCount))
				sb.WriteString(" ")
				sb.WriteString("Members")
			}
			userNameOrChannelName = sb.String()
		}
	}
	if m.Mode == ModeGroups {
		selectedGroup := m.SelectedGroup
		userNameOrChannelName = selectedGroup.FilterValue()
		if selectedGroup.ParticipantsCount != nil {
			var sb strings.Builder
			sb.WriteString(selectedGroup.FilterValue())
			sb.WriteString(" ")
			if selectedGroup.ParticipantsCount != nil {
				sb.WriteString(strconv.Itoa(*selectedGroup.ParticipantsCount))
				sb.WriteString(" ")
				sb.WriteString("Members")
			}
			userNameOrChannelName = sb.String()
		}
	}

	title := titleStyle.Render(userNameOrChannelName)
	line := strings.Repeat("─", max(0, mainWidth-4-lipgloss.Width(title)))
	headerView := lipgloss.JoinVertical(lipgloss.Center, title, line)

	mainContent := lipgloss.JoinVertical(
		lipgloss.Top,
		headerView,
		m.ChatUI.View(),
	)

	chatView := mainStyle.Render(mainContent)
	var sideBarContent string
	switch m.Mode {
	case ModeUsers:
		sideBarContent = m.Users.View()
	case ModeChannels:
		sideBarContent = m.Channels.View()
	case ModeGroups:
		sideBarContent = m.Groups.View()
	default:
		sideBarContent = ""
	}
	sidebar := getSideBarStyles(sidebarWidth, contentHeight, m).Render(sideBarContent)

	if m.FocusedOn == Input {
		m.Input.Focus()
	}

	inputView := getInputStyle(m, inputHeight).Render(m.Input.View())
	if m.IsReply && m.ReplyTo != nil {
		//get one line only
		str := strings.Join([]string{"Reply to \n", strings.Split(m.ReplyTo.Content, "\n")[0]}, "")
		inputView = lipgloss.JoinVertical(lipgloss.Top, str, inputView)
	}

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
