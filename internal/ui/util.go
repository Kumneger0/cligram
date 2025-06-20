package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kumneger0/cligram/internal/rpc"

	"github.com/hashicorp/golang-lru/v2/expirable"
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

type MessagesDelegate struct {
	list.DefaultDelegate
}

func (d MessagesDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	var title string

	if entry, ok := item.(rpc.FormattedMessage); ok {
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
	Filepicker          filepicker.Model
	IsFilepickerVisible bool
	SelectedFile        string
	Users               list.Model
	SelectedUser        rpc.UserInfo
	Channels            list.Model
	IsModalVisible      bool
	ModalContent        string
	SelectedChannel     rpc.ChannelAndGroupInfo
	Groups              list.Model
	SelectedGroup       rpc.ChannelAndGroupInfo
	Height              int
	Width               int
	MainViewLoading     bool
	SideBarLoading      bool
	Mode                Mode
	Input               textinput.Model
	FocusedOn           FocusedOn
	ChatUI              list.Model
	Conversations       [50]rpc.FormattedMessage
	IsReply             bool
	ReplyTo             *rpc.FormattedMessage
}

func filterEmptyMessages(msgs [50]rpc.FormattedMessage) []rpc.FormattedMessage {
	var filteredMsgs []rpc.FormattedMessage
	for _, m := range msgs {
		if m.ID != 0 {
			filteredMsgs = append(filteredMsgs, m)
		}
	}
	return filteredMsgs
}

func formatMessages(msgs [50]rpc.FormattedMessage) []list.Item {
	filteredMsgs := filterEmptyMessages(msgs)
	var lines []list.Item
	for _, m := range filteredMsgs {
		lines = append(lines, m)
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

	var s strings.Builder
	s.WriteString("\n  ")
	if m.SelectedFile == "" {
		s.WriteString("Pick a file:")
	} else {
		s.WriteString("Selected file: click ctrl + a to close file picker\n" + m.Filepicker.Styles.Selected.Render(m.SelectedFile))
	}

	mainViewContent := m.ChatUI.View()

	if m.IsFilepickerVisible {
		s.WriteString("\n\n" + m.Filepicker.View() + "\n")
		mainViewContent = s.String()
	}

	mainContent := lipgloss.JoinVertical(
		lipgloss.Top,
		headerView,
		mainViewContent,
	)
	var chatView string
	if m.MainViewLoading {
		chatView = mainStyle.Render("Loading...")
	} else {
		chatView = mainStyle.Render(mainContent)
	}
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
		str := strings.Join([]string{"Reply to \n", strings.Split(m.ReplyTo.Content, "\n")[0]}, "")
		inputView = lipgloss.JoinVertical(lipgloss.Top, str, inputView)
	}

	if m.SelectedFile != "" {
		str := strings.Join([]string{"File \n", strings.Split(m.SelectedFile, "\n")[0]}, "")
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

func handleUpDownArrowKeys(m *Model, isUp bool) (Model, tea.Cmd) {
	var cmd tea.Cmd
	if m.FocusedOn == Mainview {
		totalItems := len(m.ChatUI.Items())
		globalIndex := m.ChatUI.GlobalIndex()
		pInfo, cType := getGetMessageParams(m)
		if isUp && globalIndex == 0 {
			if selectedConversation, ok := m.ChatUI.SelectedItem().(rpc.FormattedMessage); ok {
				offsetID := int(selectedConversation.ID)
				cacheKey := pInfo.AccessHash + pInfo.PeerID
				if len(m.Conversations) > 1 {
					messages, err := json.Marshal(m.Conversations[:])
					if err != nil {
						fmt.Println("uff wht")
					}
					isAdded := AddToCache(cacheKey, string(messages))
					if !isAdded {
						// fmt.Println("we have failed to add items to cache")
					} else {
						// fmt.Println("Added to cache")
					}
				}
				cmd = rpc.RpcClient.GetMessages(pInfo, cType, &offsetID, nil, nil)
				conversationLastIndex := len(m.Conversations) - 1
				m.ChatUI.Select(conversationLastIndex)
			}
		} else if globalIndex == totalItems-1 && !isUp {
			cacheKey := pInfo.AccessHash + pInfo.PeerID
			messages, err := GetFromCache(cacheKey)
			if err != nil {
				fmt.Printf("we have failed to get messages from cache %s", err.Error())
			}
			if messages == nil {
				return *m, nil
			}
			var formattedMessages []rpc.FormattedMessage
			err = json.Unmarshal([]byte(*messages), &formattedMessages)

			if err != nil {
				fmt.Println("oops what the fuck is happening", err.Error())
			}

			if len(formattedMessages) == 0 {
				return *m, cmd
			}
			userConversation := rpc.UserConversationResponse{
				JsonRPC: "2.0",
				ID:      rand.Int(),
				Error:   nil,
				Result:  [50]rpc.FormattedMessage(formattedMessages),
			}
			messagesMsg := rpc.GetMessagesMsg{
				Messages: userConversation,
				Err:      nil,
			}
			cmd = func() tea.Msg {
				return messagesMsg
			}
			//get those chats from lru cache insted when we go down basically we are gonna see chat we already saw so making new rpc request is kinda unnessary
		}

	}
	return *m, cmd
}

func getGetMessageParams(m *Model) (rpc.PeerInfoParams, rpc.ChatType) {
	var cType rpc.ChatType
	var pInfo rpc.PeerInfoParams
	if m.Mode == ModeUsers {
		m.SelectedUser = m.Users.SelectedItem().(rpc.UserInfo)
		cType = "user"
		pInfo = rpc.PeerInfoParams{
			AccessHash:                  m.SelectedUser.AccessHash,
			PeerID:                      m.SelectedUser.PeerID,
			UserFirstNameOrChannelTitle: m.SelectedUser.FirstName,
		}
	}
	if m.Mode == ModeChannels {
		m.SelectedChannel = m.Channels.SelectedItem().(rpc.ChannelAndGroupInfo)
		cType = "channel"
		pInfo = rpc.PeerInfoParams{
			AccessHash:                  m.SelectedChannel.AccessHash,
			PeerID:                      m.SelectedChannel.ChannelID,
			UserFirstNameOrChannelTitle: m.SelectedChannel.ChannelTitle,
		}
		if m.SelectedChannel.IsCreator {
			m.Input.Reset()
		}
	}
	if m.Mode == ModeGroups {
		m.SelectedGroup = m.Groups.SelectedItem().(rpc.ChannelAndGroupInfo)
		cType = "group"
		pInfo = rpc.PeerInfoParams{
			AccessHash:                  m.SelectedGroup.AccessHash,
			PeerID:                      m.SelectedGroup.ChannelID,
			UserFirstNameOrChannelTitle: m.SelectedGroup.ChannelTitle,
		}
	}
	return pInfo, cType
}

var oldMessagesCache *expirable.LRU[string, string]

func AddToCache(key string, value string) bool {
	if oldMessagesCache == nil {
		oldMessagesCache = expirable.NewLRU[string, string](5, nil, time.Minute*10)
	}
	added := oldMessagesCache.Add(key, value)
	return added
}

func GetFromCache(key string) (*string, error) {
	if oldMessagesCache == nil {
		return nil, nil
	}
	oldMessages, ok := oldMessagesCache.Get(key)
	if !ok {
		return nil, fmt.Errorf("failed to find value for key %s", key)
	}
	return &oldMessages, nil
}
