package ui

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kumneger0/cligram/internal/config"
	"github.com/kumneger0/cligram/internal/rpc"
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
	if entry, ok := item.(rpc.UserInfo); ok {
		hasUnreadMessages := entry.UnreadCount > 0
		title = entry.Title()

		if entry.IsOnline {
			title = "🟢 " + title
		}
		title = "👤 " + title
		if hasUnreadMessages {
			title = title + " 🔴" + "(" + strconv.Itoa(entry.UnreadCount) + ")"
		}
	} else if entry, ok := item.(rpc.ChannelAndGroupInfo); ok {
		title = entry.Title()
		if entry.IsBroadcast {
			title = "📢 " + title
		} else {
			title = "👥 " + title
		}
	} else {
		return
	}
	isOnSideBar := d.Model.FocusedOn == SideBar
	str := lipgloss.NewStyle().Width(50).Render(title)
	if index == m.Index() && isOnSideBar {
		fmt.Fprint(w, selectedStyle.Render(" "+str+" "))
	} else {
		fmt.Fprint(w, normalStyle.Render(" "+str+" "))
	}
}

func (m Model) Init() tea.Cmd {
	return m.Filepicker.Init()
}

func getChannelIndex(m Model, channel rpc.ChannelAndGroupInfo) int {
	var index int = -1
	for i, v := range m.Channels.Items() {
		if v.FilterValue() == channel.ChannelTitle {
			index = i
		}
	}
	return index
}

func getGroupIndex(m Model, group rpc.ChannelAndGroupInfo) int {
	var index int = -1
	for i, v := range m.Groups.Items() {
		if v.FilterValue() == group.ChannelTitle {
			index = i
		}
	}
	return index
}

func getUserIndex(m Model, user rpc.UserInfo) int {
	var index int = -1
	for i, v := range m.Users.Items() {
		if v.FilterValue() == user.FilterValue() {
			index = i
		}
	}
	return index
}

func sendMessage(m *Model) (Model, tea.Cmd) {
	userMsg := m.Input.Value()
	m.Input.Reset()
	var cType rpc.ChatType
	var peerInfo rpc.PeerInfo
	if m.Mode == ModeUsers || m.Mode == ModeBots {
		if m.Mode == ModeUsers {
			cType = rpc.ChatType(rpc.UserChat)
		}
		if m.Mode == ModeBots {
			cType = rpc.ChatType(rpc.Bot)
		}
		peerInfo = rpc.PeerInfo{
			AccessHash: m.SelectedUser.AccessHash,
			PeerID:     m.SelectedUser.PeerID,
		}
	}
	if m.Mode == ModeChannels {
		cType = rpc.ChatType(rpc.ChannelChat)
		peerInfo = rpc.PeerInfo{
			AccessHash: m.SelectedChannel.AccessHash,
			PeerID:     m.SelectedChannel.ChannelID,
		}
	}
	if m.Mode == ModeGroups {
		cType = rpc.ChatType(rpc.GroupChat)
		peerInfo = rpc.PeerInfo{
			AccessHash: m.SelectedGroup.AccessHash,
			PeerID:     m.SelectedGroup.ChannelID,
		}
	}
	var messageToReply rpc.FormattedMessage
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
		m, cmd := m.editMessage(peerInfo, cType, userMsg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}
	replayToMessageID := strconv.FormatInt(messageToReply.ID, 10)
	cmds = append(cmds, rpc.RPCClient.SendMessage(peerInfo, userMsg, m.IsReply && m.ReplyTo != nil, replayToMessageID, cType, isFile, filepath))
	if isFile {
		m.SelectedFile = "uploading..."
	}
	content := userMsg
	if isFile {
		content = "This Message is not supported by this Telegram client."
	}
	cligramConfig := config.GetConfig()
	newMessage := rpc.FormattedMessage{
		ID:                   int64(rand.Int()),
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
	var cmd tea.Cmd
	if *cligramConfig.Chat.ReadReceiptMode == "default" {
		cmd = rpc.RPCClient.MarkMessagesAsRead(rpc.PeerInfo{
			AccessHash: peerInfo.AccessHash,
			PeerID:     peerInfo.PeerID,
		}, cType)
		cmds = append(cmds, cmd)
	}
	m.Input.Reset()
	m.IsReply = false
	m.ReplyTo = nil
	m.updateConverstaions()
	return *m, tea.Batch(cmds...)
}

func (m *Model) editMessage(peerInfo rpc.PeerInfo, cType rpc.ChatType, userMsg string) (Model, tea.Cmd) {
	cmd := rpc.RPCClient.EditMessage(rpc.PeerInfo{
		AccessHash: peerInfo.AccessHash,
		PeerID:     peerInfo.PeerID,
	}, cType, int(m.EditMessage.ID), userMsg)

	return *m, tea.Batch(cmd)
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
		m.Input, cmd = m.Input.Update(msg)
		cmds = append(cmds, cmd)
	case SideBar:
		m.Input.Blur()
		switch m.Mode {
		case ModeChannels:
			m.Channels, cmd = m.Channels.Update(msg)
			cmds = append(cmds, cmd)
		case ModeUsers, ModeBots:
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
	return *m, tea.Batch(cmds...)
}

func handleUserChange(m *Model) (Model, tea.Cmd) {
	pInfo, cType := getMessageParams(m)
	cmd := rpc.RPCClient.GetAllMessages(pInfo, cType, 50, nil, nil, nil)
	cligramConfig := config.GetConfig()
	if *cligramConfig.Chat.ReadReceiptMode == "instant" {
		cmd = tea.Batch(cmd, rpc.RPCClient.MarkMessagesAsRead(rpc.PeerInfo{
			AccessHash: pInfo.AccessHash,
			PeerID:     pInfo.PeerID,
		}, cType))
	}
	m.Conversations = [50]rpc.FormattedMessage{}
	m.MainViewLoading = true
	m.ChatUI.ResetSelected()
	m.ChatUI.SetItems([]list.Item{})
	return *m, cmd
}

func changeFocusMode(m *Model, msg string, shift bool) (Model, tea.Cmd) {
	var cmds []tea.Cmd
	currentlyFoucsedOn := m.FocusedOn
	canWrite := (m.Mode == ModeUsers || m.Mode == ModeGroups || m.Mode == ModeBots) || (m.Mode == ModeChannels && m.SelectedChannel.IsCreator)
	if currentlyFoucsedOn == SideBar {
		if shift {
			m.FocusedOn = Input
		} else {
			m.FocusedOn = Mainview
			chatListLastIndex := len(m.ChatUI.Items()) - 1
			m.ChatUI.Select(chatListLastIndex)
		}
	} else if currentlyFoucsedOn == Mainview && canWrite {
		if shift {
			m.FocusedOn = SideBar
		} else {
			m.FocusedOn = Input
		}
	} else {
		if shift {
			m.FocusedOn = Mainview
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
		m.AreWeSwitchingModes = true
	}
	areWeInGroupMode := m.Mode == ModeGroups && m.FocusedOn == Mainview
	//i don't think 🤔 we should keep the list in memory
	// if we keep this is in memory this will cause high memory usage especially for users
	// who has many chats so for now le't clear this up
	// TODO:🤔 can we do better
	clearSidebarLists := func(clearUsers, clearChannels, clearGroups bool) {
		if clearUsers {
			m.Users.SetItems(nil)
		}
		if clearChannels {
			m.Channels.SetItems(nil)
		}
		if clearGroups {
			m.Groups.SetItems(nil)
		}
	}

	if m.FocusedOn == SideBar || areWeInGroupMode {
		switch msg {
		case "c":
			clearSidebarLists(true, false, true)
			m.Mode = ModeChannels
			if !m.SelectedChannel.IsCreator {
				m.Input.SetValue("Not Allowed To Type")
			} else {
				m.Input.Reset()
			}
			return *m, rpc.RPCClient.GetUserChannel()
		case "u":
			m.Mode = ModeUsers
			clearSidebarLists(false, true, true)
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
					return *m, rpc.RPCClient.GetAllMessages(rpc.PeerInfoParams{
						AccessHash: m.SelectedUser.AccessHash,
						PeerID:     m.SelectedUser.PeerID,
					}, rpc.UserChat, 50, nil, nil, nil)
				}
			}
			return *m, rpc.RPCClient.GetUserChatsCmd(rpc.ModeUser)
		case "g":
			m.Mode = ModeGroups
			clearSidebarLists(true, true, false)
			return *m, rpc.RPCClient.GetUserGroups()
		case "b":
			m.Mode = ModeBots
			clearSidebarLists(false, true, true)
			m.Users.ResetSelected()
			return *m, rpc.RPCClient.GetUserChatsCmd(rpc.ModeBot)
		}
		return *m, nil
	}
	return *m, nil
}

func (m *Model) getMessageSenderUserInfo() *rpc.UserInfo {
	if selectedItem, ok := m.ChatUI.SelectedItem().(rpc.FormattedMessage); ok {
		return selectedItem.SenderUserInfo
	}
	return nil
}

func (m *Model) updateConverstaions() {
	m.ChatUI.SetItems(formatMessages(m.Conversations))
	m.viewport.SetContent(m.ChatUI.View())
	m.viewport.GotoBottom()
}

func getItemBorder(isSelected bool) lipgloss.Border {
	if isSelected {
		return lipgloss.DoubleBorder()
	}
	return lipgloss.NormalBorder()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
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
	m.Users.SetDelegate(usersDelegate)
	m.Channels.SetDelegate(channelsDelegate)
	m.Groups.SetDelegate(groupsDelegate)
	m.ChatUI.SetDelegate(mainViewDelegate)
}
