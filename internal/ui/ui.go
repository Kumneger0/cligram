package ui

import (
	"fmt"
	"io"
	"log/slog"
	"math/rand"
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
}

func (d CustomDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	var title string

	var hasUnreadMessages bool = false

	if entry, ok := item.(rpc.UserInfo); ok {
		if entry.UnreadCount > 0 {
			hasUnreadMessages = true
		}

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

	str := lipgloss.NewStyle().Width(50).Render(title)
	if index == m.Index() {
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

	if m.Mode == ModeUsers {
		cType = rpc.UserChat
		peerInfo = rpc.PeerInfo{
			AccessHash: m.SelectedUser.AccessHash,
			PeerID:     m.SelectedUser.PeerID,
		}
	}

	if m.Mode == ModeChannels {
		cType = rpc.ChannelChat
		peerInfo = rpc.PeerInfo{
			AccessHash: m.SelectedChannel.AccessHash,
			PeerID:     m.SelectedChannel.ChannelID,
		}
	}

	if m.Mode == ModeGroups {
		cType = rpc.GroupChat
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

	var filepath *string
	if isFile {
		filepath = &m.SelectedFile
	}

	replayToMessageId := strconv.FormatInt(messageToReply.ID, 10)
	response, err := rpc.RpcClient.SendMessage(peerInfo, userMsg, m.IsReply && m.ReplyTo != nil, replayToMessageId, cType, isFile, filepath)

	config := config.GetConfig()

	m.SelectedFile = ""
	if err != nil {
		slog.Error("Failed to send message", "error", err.Error())
		return *m, nil
	}

	if response.Error != nil {
		slog.Error("Failed to send message", "error", response.Error.Message)
		return *m, nil
	}

	newMessage := rpc.FormattedMessage{
		ID:                   int64(rand.Int()),
		Sender:               "you",
		IsFromMe:             true,
		Content:              userMsg,
		Media:                nil,
		Date:                 time.Now(),
		IsUnsupportedMessage: false,
		WebPage:              nil,
		Document:             nil,
		FromID:               nil,
	}
	firstOneRemoved := m.Conversations[1:]
	firstOneRemoved = append(firstOneRemoved, newMessage)
	copy(m.Conversations[:], firstOneRemoved)

	var cmd tea.Cmd

	if *config.Chat.ReadReceiptMode == "default" {
		cmd = rpc.RpcClient.MarkMessagesAsRead(rpc.PeerInfo{
			AccessHash: peerInfo.AccessHash,
			PeerID:     peerInfo.PeerID,
		}, cType)
	}

	m.Input.Reset()
	m.IsReply = false
	m.ReplyTo = nil

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
		m.Input, cmd = m.Input.Update(msg)
		cmds = append(cmds, cmd)
	case SideBar:
		m.Input.Blur()
		switch m.Mode {
		case ModeChannels:
			m.Channels, cmd = m.Channels.Update(msg)
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
	return *m, tea.Batch(cmds...)
}

func handleUserChange(m *Model) (Model, tea.Cmd) {
	pInfo, cType := getGetMessageParams(m)
	cmd := rpc.RpcClient.GetMessages(pInfo, cType, nil, nil, nil)
	config := config.GetConfig()

	if *config.Chat.ReadReceiptMode == "instant" {
		cmd = tea.Batch(cmd, rpc.RpcClient.MarkMessagesAsRead(rpc.PeerInfo{
			AccessHash: pInfo.AccessHash,
			PeerID:     pInfo.PeerID,
		}, cType))
	}

	m.Conversations = [50]rpc.FormattedMessage{}
	m.MainViewLoading = true
	m.ChatUI.SetItems([]list.Item{})
	return *m, cmd
}


func changeFocusMode(m *Model, msg string, shift bool) (Model, tea.Cmd) {
	var cmds []tea.Cmd
	currentlyFoucsedOn := m.FocusedOn
	canWrite := (m.Mode == ModeUsers || m.Mode == ModeGroups) || (m.Mode == ModeChannels && m.SelectedChannel.IsCreator)

	if currentlyFoucsedOn == SideBar {
		if shift {
			m.FocusedOn = Input
		} else {
			m.FocusedOn = Mainview
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
		} else {
			m.FocusedOn = SideBar
		}
	}
	return updateFocusedComponent(m, msg, &cmds)
}

func changeSideBarMode(m *Model, msg string) (Model, tea.Cmd) {
	if m.FocusedOn != SideBar {
		return *m, nil
	}
	switch msg {
	case "c":
		m.Mode = ModeChannels
		if !m.SelectedChannel.IsCreator {
			m.Input.SetValue("Not Allowed To Type")
		} else {
			m.Input.Reset()
		}
		return *m, rpc.RpcClient.GetUserChannel()
	case "u":
		m.Mode = ModeUsers
		return *m, nil
	case "g":
		m.Mode = ModeGroups
		return *m, rpc.RpcClient.GetUserGroups()
	}
	return *m, nil
}

func (m *Model) updateConverstaions() {
	sidebarWidth := m.Width * 30 / 100
	mainWidth := m.Width - sidebarWidth
	w := mainWidth * 70 / 100
	m.ChatUI.SetWidth(w)
	m.ChatUI.SetHeight(int(float64(m.Height) / 2.6666666665))
	m.ChatUI.SetItems(formatMessages(m.Conversations))
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

	ui := setItemStyles(&m)
	return ui
}
