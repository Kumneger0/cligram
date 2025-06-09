package ui

import (
	"fmt"
	"io"
	"math/rand"
	"strconv"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kumneger0/cligram/internal/rpc"
)

type CustomDelegate struct {
	list.DefaultDelegate
}

func (d CustomDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	var title string

	if entry, ok := item.(rpc.UserInfo); ok {
		title = entry.Title()
		title = "👤 " + title
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
	return nil
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

	messageToReply := *m.ReplyTo
	replayToMessageId := strconv.FormatInt(messageToReply.ID, 10)

	response, err := rpc.RpcClient.SendMessage(peerInfo, userMsg, m.IsReply && m.ReplyTo != nil, replayToMessageId, cType, false, nil)
	if err != nil {
		//TODO: trigger to show toast message
		fmt.Println(err.Error())
		return *m, nil
	}

	if response.Error != nil {
		return *m, nil
	}

	m.Conversations = append(m.Conversations, rpc.FormattedMessage{
		//This is just to show the message immediatly after sending
		// so we don't have to refetch the whole message since we are the one who is sending
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
	})

	m.Input.Reset()
	m.IsReply = false
	m.ReplyTo = nil
	return *m, nil
}

func updateFocusedComponent(m *Model, msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.FocusedOn {
	case Input:
		m.Input.Focus()
		m.Input, cmd = m.Input.Update(msg)
	case SideBar:
		m.Input.Blur()
		switch m.Mode {
		case ModeChannels:
			m.Channels, cmd = m.Channels.Update(msg)
		case ModeUsers:
			m.Users, cmd = m.Users.Update(msg)
		default:
			m.Groups, cmd = m.Groups.Update(msg)
		}

	default:
		m.ChatUI, cmd = m.ChatUI.Update(msg)
	}
	return *m, cmd
}

func handleUserChange(m *Model) (Model, tea.Cmd) {
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
	result, err := rpc.RpcClient.GetMessages(pInfo, cType, nil, nil, nil)
	if err != nil {
		//TODO: SHOW toast message here
		// fmt.Println(strings.Repeat("uff there is something off", 10), err.Error())
		return *m, nil
	}

	formatedMessages := result.Result
	m.Conversations = formatedMessages
	m.updateConverstaions()
	return *m, nil
}

func changeFocusMode(m *Model, msg string) (Model, tea.Cmd) {
	currentlyFoucsedOn := m.FocusedOn
	canWrite := (m.Mode == ModeUsers || m.Mode == ModeGroups) || (m.Mode == ModeChannels && m.SelectedChannel.IsCreator)

	if currentlyFoucsedOn == SideBar {
		m.FocusedOn = Mainview
	} else if currentlyFoucsedOn == Mainview && canWrite {
		m.FocusedOn = Input
	} else {
		m.FocusedOn = SideBar
	}
	return updateFocusedComponent(m, msg)
}

func changeSideBarMode(m *Model, msg string) (Model, tea.Cmd) {
	if m.FocusedOn != SideBar {
		return updateFocusedComponent(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(msg)})
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
	// contentHeight := m.Height * 90 / 100

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
