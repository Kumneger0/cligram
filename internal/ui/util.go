package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	// "github.com/kumneger0/cligram/internal/config"

	"github.com/kumneger0/cligram/internal/config"
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
	viewport            viewport.Model
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
		return renderModal(m)
	}

	dimensions := calculateLayoutDimensions(m)

	updateListDimensions(m, dimensions)

	mainContent := prepareMainContent(m, dimensions)

	sidebarContent := prepareSidebarContent(m, dimensions)

	inputView := prepareInputView(m, dimensions)

	row := lipgloss.JoinHorizontal(lipgloss.Top, sidebarContent, mainContent)
	return lipgloss.JoinVertical(lipgloss.Top, row, inputView)
}

type layoutDimensions struct {
	sidebarWidth  int
	mainWidth     int
	contentHeight int
	inputHeight   int
}

func calculateLayoutDimensions(m *Model) layoutDimensions {
	sidebarWidth := m.Width * 30 / 100
	return layoutDimensions{
		sidebarWidth:  sidebarWidth,
		mainWidth:     m.Width - sidebarWidth,
		contentHeight: m.Height * 90 / 100,
		inputHeight:   m.Height - (m.Height * 90 / 100),
	}
}

func updateListDimensions(m *Model, d layoutDimensions) {
	listHeight := d.contentHeight - 4
	m.Users.SetHeight(listHeight)
	m.Users.SetWidth(d.sidebarWidth)
	m.Channels.SetWidth(d.sidebarWidth)
	m.Channels.SetHeight(listHeight)
	m.Groups.SetWidth(d.sidebarWidth)
	m.Groups.SetHeight(listHeight)
}

func renderModal(m *Model) string {
	return lipgloss.Place(
		m.Width,
		m.Height,
		lipgloss.Center,
		lipgloss.Center,
		m.ModalContent,
	)
}

func getUserOrChannelName(m *Model) string {
	switch m.Mode {
	case ModeUsers:
		return formatUserName(m.SelectedUser)
	case ModeChannels:
		return formatChannelName(m.SelectedChannel)
	case ModeGroups:
		return formatGroupName(m.SelectedGroup)
	default:
		return ""
	}
}

func formatUserName(user rpc.UserInfo) string {
	name := user.Title()
	if user.IsOnline {
		return name + " Online"
	}
	if user.LastSeen != nil {
		return name + " " + *user.LastSeen
	}
	return name
}

func formatChannelOrGroupName(name string, count *int) string {
	if count == nil {
		return name
	}
	return fmt.Sprintf("%s %d Members", name, *count)
}

func formatChannelName(channel rpc.ChannelAndGroupInfo) string {
	return formatChannelOrGroupName(channel.FilterValue(), channel.ParticipantsCount)
}

func formatGroupName(group rpc.ChannelAndGroupInfo) string {
	return formatChannelOrGroupName(group.FilterValue(), group.ParticipantsCount)
}

func prepareMainContent(m *Model, d layoutDimensions) string {
	mainStyle := getMainStyle(d.mainWidth, d.contentHeight, m)

	if m.MainViewLoading {
		return mainStyle.Render("Loading...")
	}

	// m.ChatUI.SetItems(formatMessages(m.Conversations))
	m.ChatUI.SetWidth(d.mainWidth * 70 / 100)
	m.ChatUI.SetHeight(15)

	userNameOrChannelName := getUserOrChannelName(m)
	title := titleStyle.Render(userNameOrChannelName)
	line := strings.Repeat("─", max(0, d.mainWidth-4-lipgloss.Width(title)))
	headerView := lipgloss.JoinVertical(lipgloss.Center, title, line)

	chatsView := m.ChatUI.View()
	m.viewport.SetContent(chatsView)
	mainViewContent := m.viewport.View()
	if m.IsFilepickerVisible {
		mainViewContent = prepareFilepickerView(m)
	}

	mainContent := lipgloss.JoinVertical(
		lipgloss.Top,
		headerView,
		mainViewContent,
	)

	return mainStyle.Render(mainContent)
}

func prepareFilepickerView(m *Model) string {
	var s strings.Builder
	s.WriteString("\n  ")
	if m.SelectedFile == "" {
		s.WriteString("Pick a file:")
	} else {
		s.WriteString("Selected file: click ctrl + a to close file picker\n" + m.Filepicker.Styles.Selected.Render(m.SelectedFile))
	}
	s.WriteString("\n\n" + m.Filepicker.View() + "\n")
	return s.String()
}

func prepareSidebarContent(m *Model, d layoutDimensions) string {
	var content string
	switch m.Mode {
	case ModeUsers:
		content = m.Users.View()
	case ModeChannels:
		content = m.Channels.View()
	case ModeGroups:
		content = m.Groups.View()
	}
	return getSideBarStyles(d.sidebarWidth, d.contentHeight, m).Render(content)
}

func prepareInputView(m *Model, d layoutDimensions) string {
	if m.FocusedOn == Input {
		m.Input.Focus()
	}

	inputView := getInputStyle(m, d.inputHeight).Render(m.Input.View())

	if m.IsReply && m.ReplyTo != nil {
		replyContext := fmt.Sprintf("Reply to \n%s", strings.Split(m.ReplyTo.Content, "\n")[0])
		inputView = lipgloss.JoinVertical(lipgloss.Top, replyContext, inputView)
	}

	if m.SelectedFile != "" {
		fileContext := fmt.Sprintf("File \n%s", strings.Split(m.SelectedFile, "\n")[0])
		inputView = lipgloss.JoinVertical(lipgloss.Top, fileContext, inputView)
	}

	return inputView
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
						slog.Error("Failed to marshal messages", "error", err.Error())
					}
					AddToCache(cacheKey, string(messages))
				}
				cmd = rpc.RpcClient.GetMessages(pInfo, cType, &offsetID, nil, nil)
				conversationLastIndex := len(m.Conversations) - 1
				m.ChatUI.Select(conversationLastIndex)
			}
		} else if globalIndex == totalItems-1 && !isUp {
			cacheKey := pInfo.AccessHash + pInfo.PeerID
			messages, err := GetFromCache(cacheKey)
			if err != nil {
				slog.Error("Failed to get messages from cache", "error", err.Error())
			}
			if messages == nil {
				return *m, nil
			}
			var formattedMessages []rpc.FormattedMessage
			err = json.Unmarshal([]byte(*messages), &formattedMessages)

			if err != nil {
				slog.Error("Failed to unmarshal messages", "error", err.Error())
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

func SendUserIsTyping(m *Model) tea.Cmd {
	userConfig := config.GetConfig()
	if !*userConfig.Chat.SendTypingState {
		return nil
	}

	if (m.Mode == ModeUsers || m.Mode == ModeGroups) && m.FocusedOn == Input {
		var pInfo rpc.PeerInfo
		if m.Mode == ModeUsers {
			pInfo = rpc.PeerInfo{
				PeerID:     m.SelectedUser.PeerID,
				AccessHash: m.SelectedUser.AccessHash,
			}
		}
		if m.Mode == ModeGroups {
			pInfo = rpc.PeerInfo{
				PeerID:     m.SelectedGroup.ChannelID,
				AccessHash: m.SelectedGroup.AccessHash,
			}
		}
		err := rpc.RpcClient.SetUserTyping(pInfo, "user")
		if err != nil {
			slog.Error("Failed to send user is typing event", "error", err.Error())
		}
	}
	return nil
}
