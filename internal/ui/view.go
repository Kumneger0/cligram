package ui

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kumneger0/cligram/internal/config"
	"github.com/kumneger0/cligram/internal/telegram"
	"github.com/kumneger0/cligram/internal/telegram/types"
)

type CustomDelegate struct {
	list.DefaultDelegate
	*Model
}

func (d CustomDelegate) Height() int {
	return 1
}

func (d CustomDelegate) Spacing() int {
	return 0
}

func (d CustomDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}
func (d CustomDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	var title string
	var prefix string
	var unreadBadge string

	switch item := item.(type) {
	case types.UserInfo:
		title = item.Title()
		if item.IsOnline {
			prefix = "🟢 "
		} else {
			prefix = "👤 "
		}
		if item.UnreadCount > 0 {
			unreadBadge = unreadCountStyle.Render(strconv.Itoa(item.UnreadCount))
		}
	case types.ChannelInfo:
		title = item.Title()
		if item.IsBroadcast {
			prefix = "📢 "
		} else {
			prefix = "👥 "
		}
		if item.UnreadCount > 0 {
			unreadBadge = unreadCountStyle.Render(strconv.Itoa(item.UnreadCount))
		}
	default:
		return
	}

	availWidth := m.Width() - 10
	if availWidth < 0 {
		availWidth = 0
	}

	name := lipgloss.NewStyle().MaxWidth(availWidth).Render(title)

	// Join prefix, name (truncated), and badge
	var content string
	if unreadBadge != "" {
		occupied := lipgloss.Width(prefix) + lipgloss.Width(name) + lipgloss.Width(unreadBadge) + 2
		spacerWidth := m.Width() - occupied
		if spacerWidth < 0 {
			spacerWidth = 0
		}
		spacer := strings.Repeat(" ", spacerWidth)
		content = lipgloss.JoinHorizontal(lipgloss.Top, prefix, name, spacer, unreadBadge)
	} else {
		content = lipgloss.JoinHorizontal(lipgloss.Top, prefix, name)
	}

	isOnSideBar := d.Model.FocusedOn == SideBar
	style := normalStyle
	if index == m.Index() && isOnSideBar {
		style = selectedStyle
	}

	fmt.Fprint(w, style.UnsetWidth().Render(content))
}

func (m Model) Init() tea.Cmd {
	filePickerInitCMD := m.Filepicker.Init()
	storiesCMD := telegram.Cligram.GetAllStories(telegram.Cligram.Context())

	return tea.Batch(filePickerInitCMD, storiesCMD)
}

func getChannelIndex(m Model, channel types.ChannelInfo) int {
	var index int = -1
	for i, v := range m.Channels.Items() {
		if v.FilterValue() == channel.ChannelTitle {
			index = i
		}
	}
	return index
}

func getGroupIndex(m Model, group types.ChannelInfo) int {
	var index int = -1
	for i, v := range m.Groups.Items() {
		if v.FilterValue() == group.ChannelTitle {
			index = i
		}
	}
	return index
}

func getUserIndex(listToSearchFrom list.Model, user types.UserInfo) int {
	var index int = -1
	for i, v := range listToSearchFrom.Items() {
		if v.(types.UserInfo).PeerID == user.PeerID {
			index = i
			break
		}
	}
	return index
}

func sendMessage(m *Model) (Model, tea.Cmd) {
	userMsg := m.Input.Value()
	m.Input.Reset()
	peerInfo := getMessageParams(m)
	var messageToReply types.FormattedMessage
	if m.ReplyTo != nil {
		messageToReply = *m.ReplyTo
	}
	var isFile bool
	if m.SelectedFile != "" {
		isFile = true
	}
	var filepath string
	if isFile {
		filepath = m.SelectedFile
	}
	if _, err := os.Stat(m.SelectedFile); os.IsNotExist(err) && isFile {
		m.Input.SetValue("Invalid file path")
		return *m, nil
	}
	var cmds []tea.Cmd
	if m.EditMessage != nil {
		m, cmd := m.editMessage(peerInfo, userMsg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}

	replyToMessageID := ""
	if m.IsReply && m.ReplyTo != nil {
		replyToMessageID = strconv.Itoa(messageToReply.ID)
	}

	randID := rand.Int()

	cmds = append(cmds, telegram.Cligram.SendMessage(telegram.Cligram.Context(),
		types.SendMessageRequest{
			RandID:           randID,
			Peer:             peerInfo,
			Message:          userMsg,
			IsReply:          m.IsReply && m.ReplyTo != nil,
			ReplyToMessageID: replyToMessageID,
			IsFile:           isFile,
			FilePath:         filepath,
		}))
	if isFile {
		m.SelectedFile = "uploading..."
	}
	content := userMsg
	if isFile {
		content = "This Message is not supported by this Telegram client."
	}
	cligramConfig := config.GetConfig()
	newMessage := types.FormattedMessage{
		ID:                   randID,
		Sender:               "you",
		IsFromMe:             true,
		Content:              content,
		Media:                nil,
		Date:                 time.Now(),
		IsUnsupportedMessage: isFile,
		WebPage:              nil,
		Document:             nil,
		FromID:               nil,
	}
	firstOneRemoved := m.Conversations[1:]
	firstOneRemoved = append(firstOneRemoved, newMessage)
	copy(m.Conversations[:], firstOneRemoved)
	if *cligramConfig.Chat.ReadReceiptMode == "default" {
		cmd := telegram.Cligram.MarkMessagesAsRead(telegram.Cligram.Context(), types.MarkAsReadRequest{
			Peer: peerInfo,
		})
		cmds = append(cmds, cmd)
	}
	m.Input.Reset()
	m.IsReply = false
	m.ReplyTo = nil

	m.updateConversations()
	return *m, tea.Batch(cmds...)
}

func (m *Model) editMessage(peerInfo types.Peer, userMsg string) (Model, tea.Cmd) {
	cmd := func() tea.Msg {
		telegram.Cligram.EditMessage(telegram.Cligram.Context(), types.EditMessageRequest{
			Peer:       peerInfo,
			MessageID:  int(m.EditMessage.ID),
			NewMessage: userMsg,
		})
		return nil
	}

	return *m, cmd
}

func updateFocusedComponent(m *Model, msg tea.Msg, cmdsFromParent *[]tea.Cmd) (Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds = *cmdsFromParent
	m.Filepicker, cmd = m.Filepicker.Update(msg)
	m.Filepicker.SetHeight(m.Height - 13)
	cmds = append(cmds, cmd)
	if didSelect, path := m.Filepicker.DidSelectFile(msg); didSelect && m.IsFilepickerVisible {
		m.SelectedFile = path
		m.IsFilepickerVisible = false
	}

	switch m.FocusedOn {
	case Input:
		m.Input.Focus()
		if !m.SkipNextInput {
			m.Input, cmd = m.Input.Update(msg)
			cmds = append(cmds, cmd)
		}
	case SideBar:
		m.Input.Blur()
		switch m.Mode {
		case ModeChannels:
			m.Channels, cmd = m.Channels.Update(msg)
			cmds = append(cmds, cmd)
		case ModeBots:
			m.Bots, cmd = m.Bots.Update(msg)
			cmds = append(cmds, cmd)
		case ModeUsers:
			m.Users, cmd = m.Users.Update(msg)
			cmds = append(cmds, cmd)
		default:
			m.Groups, cmd = m.Groups.Update(msg)
			cmds = append(cmds, cmd)
		}
	default:
		m.ChatUI, cmd = m.ChatUI.Update(msg)
		cmds = append(cmds, cmd)
	}
	m.SkipNextInput = false
	return *m, tea.Batch(cmds...)
}

func handleUserChange(m *Model) (Model, tea.Cmd) {
	pInfo := getMessageParams(m)
	cmd := telegram.Cligram.GetMessages(telegram.Cligram.Context(), types.GetMessagesRequest{
		Peer:          pInfo,
		Limit:         50,
		OffsetID:      nil,
		ChatAreaWidth: nil,
	})
	cligramConfig := config.GetConfig()
	if *cligramConfig.Chat.ReadReceiptMode == "instant" {
		markAsReadCmd := telegram.Cligram.MarkMessagesAsRead(telegram.Cligram.Context(), types.MarkAsReadRequest{
			Peer: pInfo,
		})
		return *m, tea.Batch(cmd, markAsReadCmd)
	}
	m.Conversations = [50]types.FormattedMessage{}
	m.MainViewLoading = true
	m.ChatUI.ResetSelected()
	m.ChatUI.SetItems([]list.Item{})
	return *m, cmd
}

func changeFocusMode(m *Model, msg string, shift bool) (Model, tea.Cmd) {
	var cmds []tea.Cmd
	currentlyFocusedOn := m.FocusedOn
	canWrite := (m.Mode == ModeUsers || m.Mode == ModeGroups || m.Mode == ModeBots) || (m.Mode == ModeChannels && m.SelectedChannel.IsCreator)
	if currentlyFocusedOn == SideBar {
		if shift {
			m.FocusedOn = Input
		} else {
			m.FocusedOn = Main
			chatListLastIndex := len(m.ChatUI.Items()) - 1
			m.ChatUI.Select(chatListLastIndex)
		}
	} else if currentlyFocusedOn == Main && canWrite {
		if shift {
			m.FocusedOn = SideBar
		} else {
			m.FocusedOn = Input
		}
	} else {
		if shift {
			m.FocusedOn = Main
			chatListLastIndex := len(m.ChatUI.Items()) - 1
			m.ChatUI.Select(chatListLastIndex)
		} else {
			m.FocusedOn = SideBar
		}
	}
	return updateFocusedComponent(m, msg, &cmds)
}

func changeSideBarMode(m *Model, msg string) (Model, tea.Cmd) {
	if m.FocusedOn == SideBar {
		m.ChatUI.ResetSelected()
		m.OnPagination = false
	}
	areWeInGroupMode := m.Mode == ModeGroups && m.FocusedOn == Main
	if m.FocusedOn == SideBar || areWeInGroupMode {
		switch msg {
		case "c":
			m.Mode = ModeChannels
			if !m.SelectedChannel.IsCreator {
				m.Input.SetValue("Not Allowed To Type")
			} else {
				m.Input.Reset()
			}
			return *m, nil
		case "u":
			m.Mode = ModeUsers
			if areWeInGroupMode {
				selectedUser := m.getMessageSenderUserInfo()
				if selectedUser != nil {
					m.SelectedUser = *selectedUser
					userItems := m.Users.Items()

					var foundIndex int = -1
					for i, v := range userItems {
						if v.FilterValue() == m.SelectedUser.FilterValue() {
							foundIndex = i
							break
						}
					}
					if foundIndex == -1 {
						userItems = append(userItems, list.Item(m.SelectedUser))
						m.Users.SetItems(userItems)
						m.Users.Select(len(userItems) - 1)
					} else {
						m.Users.Select(foundIndex)
					}
					m.ChatUI.SetItems(nil)
					return *m, telegram.Cligram.GetMessages(telegram.Cligram.Context(), types.GetMessagesRequest{
						Peer: getMessageParams(m),
						//TODO:  i might need to revisit this one
						Limit:         50,
						OffsetID:      nil,
						ChatAreaWidth: nil,
					})
				}
			}
			return *m, nil

		case "g":
			m.Mode = ModeGroups
			return *m, nil
		case "b":
			m.Mode = ModeBots
			m.Bots.Select(0)
			m.Bots.ResetSelected()
			return *m, nil
		}
		return *m, nil
	}
	return *m, nil
}

func (m *Model) getMessageSenderUserInfo() *types.UserInfo {
	if selectedItem, ok := m.ChatUI.SelectedItem().(types.FormattedMessage); ok {
		return selectedItem.SenderUserInfo
	}
	return nil
}

func (m *Model) updateConversations() {
	m.ChatUI.SetItems(formatMessages(m.Conversations))
	m.viewport.SetContent(m.ChatUI.View())
	m.viewport.GotoBottom()
}

func (m Model) View() string {
	m.Users.Title = "Chats"
	m.Channels.Title = "Channels"
	m.Channels.SetShowStatusBar(false)
	m.Users.SetShowStatusBar(false)
	m.Groups.Title = "Groups"
	m.Groups.SetShowStatusBar(false)
	m.updateDelegates()

	ui := setItemStyles(&m)
	return ui
}

func (m *Model) updateDelegates() {
	usersDelegate := CustomDelegate{Model: m}
	channelsDelegate := CustomDelegate{Model: m}
	groupsDelegate := CustomDelegate{Model: m}
	mainViewDelegate := MessagesDelegate{Model: m}
	// storiesDelegate := StoriesDelegate{Model: m}
	// m.Stories.SetDelegate(storiesDelegate)
	m.Users.SetDelegate(usersDelegate)
	m.Channels.SetDelegate(channelsDelegate)
	m.Groups.SetDelegate(groupsDelegate)
	m.ChatUI.SetDelegate(mainViewDelegate)
}
