package ui

import (
	"fmt"
	"io"
	"math/rand"
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

	if entry, ok := item.(UserInfo); ok {
		title = entry.Title()
		title = "👤 " + title
	} else if entry, ok := item.(ChannelAndGroupInfo); ok {
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
	return rpc.RpcClient.GetChats()
}

func getChannelIndex(m Model, channel ChannelAndGroupInfo) int {
	var index int = -1
	for i, v := range m.Channels.Items() {
		if v.FilterValue() == channel.ChannelTitle {
			index = i
		}
	}
	return index
}

func getGroupIndex(m Model, group ChannelAndGroupInfo) int {
	var index int = -1
	for i, v := range m.Groups.Items() {
		if v.FilterValue() == group.ChannelTitle {
			index = i
		}
	}
	return index
}

func getUserIndex(m Model, user UserInfo) int {
	var index int = -1
	for i, v := range m.Users.Items() {
		if v.FilterValue() == user.FilterValue() {
			index = i
		}
	}
	return index
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab":
			return changeFocusMode(&m, "tab")
		case "m":
			if m.FocusedOn != "input" {
				m.IsModalVisible = true
				return m, nil
			}
			input, cmd := m.Input.Update(msg)
			m.Input = input
			return m, cmd
		case "c":
			return changeSideBarMode(&m, "c")
		case "u":
			return changeSideBarMode(&m, "u")
		case "g":
			return changeSideBarMode(&m, "g")
		case "enter":
			if m.FocusedOn == "input" {
				return sendMessage(&m)
			}
			if m.FocusedOn == "sideBar" {
				return handleUserChange(&m)
			}
			//TODO:at this time we must be in mainView
			//figure out what to do when someone clicks enter while in mainView

		}
	case rpc.UserChatsMsg:
		if msg.Err != nil {
			m.IsModalVisible = true
			m.ModalContent = GetModalContent(msg.Err.Error())
			return m, nil
		}

		duplicatedUsers := msg.Response.Result
		var users []list.Item
		for _, du := range duplicatedUsers {
			users = append(users, UserInfo{
				UnreadCount: du.UnreadCount,
				FirstName:   du.FirstName,
				IsBot:       du.IsBot,
				PeerID:      du.PeerID,
				AccessHash:  du.PeerID,
				LastSeen:    LastSeen(du.LastSeen),
				IsOnline:    du.IsOnline,
			})
		}
		m.Users.SetItems(users)
		return m, nil
	case rpc.UserChannelMsg:
		if msg.Err != nil {
			m.IsModalVisible = true
			m.ModalContent = GetModalContent(msg.Err.Error())
			return m, nil
		}

		duplicatedUsers := msg.Response.Result
		var channels []list.Item
		for _, du := range duplicatedUsers {
			channels = append(channels, ChannelAndGroupInfo{
				ChannelTitle:      du.ChannelTitle,
				Username:          nil,
				ChannelID:         du.ChannelID,
				AccessHash:        du.AccessHash,
				IsCreator:         du.IsCreator,
				IsBroadcast:       du.IsBroadcast,
				ParticipantsCount: du.ParticipantsCount,
				UnreadCount:       du.UnreadCount,
			})
		}
		m.Channels.SetItems(channels)
		return m, nil
	case rpc.UserGroupsMsg:
		if msg.Err != nil {
			m.IsModalVisible = true
			m.ModalContent = GetModalContent(msg.Err.Error())
			return m, nil
		}

		duplicatedGroups := msg.Response.Result
		var groups []list.Item
		for _, du := range duplicatedGroups {
			groups = append(groups, ChannelAndGroupInfo{
				ChannelTitle:      du.ChannelTitle,
				Username:          nil,
				ChannelID:         du.ChannelID,
				AccessHash:        du.AccessHash,
				IsCreator:         du.IsCreator,
				IsBroadcast:       du.IsBroadcast,
				ParticipantsCount: du.ParticipantsCount,
				UnreadCount:       du.UnreadCount,
			})
		}
		m.Groups.SetItems(groups)
		return m, nil

	case tea.WindowSizeMsg:
		m.Width = msg.Width - 4
		m.Height = msg.Height - 4
		m.updateViewport()
	case SelectSearchedUserResult:
		if msg.user != nil {
			m.SelectedUser = *msg.user
			index := getUserIndex(m, *msg.user)
			if index != -1 {
				m.Users.Select(index)
				m.FocusedOn = "sideBar"
				m.Mode = "users"
				return handleUserChange(&m)
			}
			newUpdatedUsers := append(m.Users.Items(), *msg.user)
			updateUserCmd := m.Users.SetItems(newUpdatedUsers)

			index = getUserIndex(m, *msg.user)

			if index != -1 {
				m.Users.Select(index)
				m.FocusedOn = "sideBar"
				m.Mode = "users"
				m, handleUserChangeCmd := handleUserChange(&m)
				return m, tea.Batch(updateUserCmd, handleUserChangeCmd)
			}

			return m, updateUserCmd
		}

		if msg.channel != nil {
			m.SelectedChannel = *msg.channel
			index := getChannelIndex(m, *msg.channel)

			if index != -1 {
				m.Channels.Select(index)
				m.FocusedOn = "sideBar"
				m.Mode = "channels"
				return handleUserChange(&m)
			}

			newUpdatedChannelsList := append(m.Channels.Items(), *msg.channel)
			setItemsCmd := m.Channels.SetItems(newUpdatedChannelsList)
			index = getChannelIndex(m, *msg.channel)

			if index != -1 {
				m.Channels.Select(index)
				m.FocusedOn = "sideBar"
				m.Mode = "channels"
				m, handleChangeUserCmd := handleUserChange(&m)
				return m, tea.Batch(setItemsCmd, handleChangeUserCmd)
			}
			return m, setItemsCmd
		}

		if msg.group != nil {
			m.SelectedGroup = *msg.group
			index := getGroupIndex(m, *msg.group)

			if index != -1 {
				m.Groups.Select(index)
				m.FocusedOn = "sideBar"
				m.Mode = "groups"
				return handleUserChange(&m)
			}

			newUpdatedGroupsList := append(m.Groups.Items(), *msg.group)
			setItemsCmd := m.Groups.SetItems(newUpdatedGroupsList)
			index = getGroupIndex(m, *msg.group)

			if index != -1 {
				m.Groups.Select(index)
				m.FocusedOn = "sideBar"
				m.Mode = "groups"
				m, handleChangeUserCmd := handleUserChange(&m)
				return m, tea.Batch(setItemsCmd, handleChangeUserCmd)
			}
			return m, setItemsCmd
		}
	}
	return updateFocusedComponent(&m, msg)
}

func sendMessage(m *Model) (Model, tea.Cmd) {
	userMsg := m.Input.Value()
	m.Input.Reset()
	var cType rpc.ChatType
	var peerInfo rpc.PeerInfo

	if m.Mode == "users" {
		cType = rpc.UserChat
		peerInfo = rpc.PeerInfo{
			AccessHash: m.SelectedUser.AccessHash,
			PeerID:     m.SelectedUser.PeerID,
		}
	}

	if m.Mode == "channels" {
		cType = rpc.ChannelChat
		peerInfo = rpc.PeerInfo{
			AccessHash: m.SelectedChannel.AccessHash,
			PeerID:     m.SelectedChannel.ChannelID,
		}
	}

	if m.Mode == "groups" {
		cType = rpc.GroupChat
		peerInfo = rpc.PeerInfo{
			AccessHash: m.SelectedGroup.AccessHash,
			PeerID:     m.SelectedGroup.ChannelID,
		}
	}

	//TODO:we dont't need this
	//will fix it when i start working in replay feature
	replayToMessageId := "88"

	_, err := rpc.RpcClient.SendMessage(peerInfo, userMsg, false, replayToMessageId, cType, false, nil)
	if err != nil {
		//TODO: find out a way to show the error message in ui
		// fmt.Println(strings.Repeat(err.Error(), 10))
		return *m, nil
	}

	m.Conversations = append(m.Conversations, FormattedMessage{
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
	return *m, nil
}

func updateFocusedComponent(m *Model, msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.FocusedOn {
	case "input":
		m.Input.Focus()
		m.Input, cmd = m.Input.Update(msg)
	case "sideBar":
		m.Input.Blur()
		switch m.Mode {
		case "channels":
			m.Channels, cmd = m.Channels.Update(msg)
		case "users":
			m.Users, cmd = m.Users.Update(msg)
		default:
			m.Groups, cmd = m.Groups.Update(msg)
		}

	default:
		m.Vp, cmd = m.Vp.Update(msg)
	}
	return *m, cmd
}

func handleUserChange(m *Model) (Model, tea.Cmd) {
	var cType rpc.ChatType
	var pInfo rpc.PeerInfoParams
	if m.Mode == "users" {
		m.SelectedUser = m.Users.SelectedItem().(UserInfo)
		cType = "user"
		pInfo = rpc.PeerInfoParams{
			AccessHash:                  m.SelectedUser.AccessHash,
			PeerID:                      m.SelectedUser.PeerID,
			UserFirstNameOrChannelTitle: m.SelectedUser.FirstName,
		}
	}
	if m.Mode == "channels" {
		m.SelectedChannel = m.Channels.SelectedItem().(ChannelAndGroupInfo)
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
	if m.Mode == "groups" {
		m.SelectedGroup = m.Groups.SelectedItem().(ChannelAndGroupInfo)
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

	var formatedMessages []FormattedMessage

	for _, v := range result.Result {
		formatedMessages = append(formatedMessages, FormattedMessage{
			ID:                   v.ID,
			Sender:               v.Sender,
			Content:              v.Content,
			IsFromMe:             v.IsFromMe,
			Media:                v.Media,
			Date:                 v.Date,
			IsUnsupportedMessage: v.IsUnsupportedMessage,
			WebPage:              v.WebPage,
			Document:             v.Document,
			FromID:               v.FromID,
		})

	}
	m.Conversations = formatedMessages
	m.updateViewport()
	return *m, nil
}

func changeFocusMode(m *Model, msg string) (Model, tea.Cmd) {
	currentlyFoucsedOn := m.FocusedOn
	canWrite := (m.Mode == "users" || m.Mode == "groups") || (m.Mode == "channels" && m.SelectedChannel.IsCreator)

	if currentlyFoucsedOn == "sideBar" {
		m.FocusedOn = "mainView"
	} else if currentlyFoucsedOn == "mainView" && canWrite {
		m.FocusedOn = "input"
	} else {
		m.FocusedOn = "sideBar"
	}
	return updateFocusedComponent(m, msg)
}

func changeSideBarMode(m *Model, msg string) (Model, tea.Cmd) {
	if m.FocusedOn != "sideBar" {
		return updateFocusedComponent(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(msg)})
	}
	switch msg {
	case "c":
		m.Mode = "channels"
		if !m.SelectedChannel.IsCreator {
			m.Input.SetValue("Not Allowed To Type")
		} else {
			m.Input.Reset()
		}
		return *m, rpc.RpcClient.GetUserChannel()
	case "u":
		m.Mode = "users"
		return *m, nil
	case "g":
		m.Mode = "groups"
		return *m, rpc.RpcClient.GetUserGroups()
	}
	return *m, nil
}

func (m *Model) updateViewport() {
	sidebarWidth := m.Width * 30 / 100
	mainWidth := m.Width - sidebarWidth
	contentHeight := m.Height * 90 / 100

	headerHeight := 2

	w, h := mainWidth*70/100, contentHeight*90/100-headerHeight
	m.Vp.Width = w
	m.Vp.Height = h

	m.Vp.YPosition = headerHeight

	m.Vp.SetContent(formatMessages(m.Conversations))
	m.Vp.GotoBottom()
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
