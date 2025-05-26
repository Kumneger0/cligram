package ui

import (
	"fmt"
	"io"
	"strings"

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
	} else if entry, ok := item.(ChannelAndGroupInfo); ok {
		title = entry.Title()
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
	return tea.Batch(
		rpc.RpcClient.GetUserChats(),
		rpc.RpcClient.GetUserChannel(),
		rpc.RpcClient.GetUserGroups(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab":
			return changeFocusMode(&m, "tab")
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
			fmt.Println("opps we fucked up", msg.Err.Error())
			return m, nil
		}

		duplicatedUsers := msg.Response.Result
		var users []list.Item
		for _, du := range duplicatedUsers {
			users = append(users, UserInfo{
				unreadCount: du.UnreadCount,
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
			fmt.Println("opps we fucked up", msg.Err.Error())
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
				IsCreator:         false,
				IsBroadcast:       false,
				ParticipantsCount: nil,
				UnreadCount:       du.UnreadCount,
			})
		}
		m.Channels.SetItems(channels)
		return m, nil

	case rpc.UserGroupsMsg:
		if msg.Err != nil {
			fmt.Println("opps we fucked up", msg.Err.Error())
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
				IsCreator:         false,
				IsBroadcast:       false,
				ParticipantsCount: nil,
				UnreadCount:       du.UnreadCount,
			})
		}
		m.Groups.SetItems(groups)
		return m, nil

	case tea.WindowSizeMsg:
		m.Width = msg.Width - 4
		m.Height = msg.Height - 4
		m.updateViewport()
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
	_, err := rpc.RpcClient.SendMessage(peerInfo, userMsg, false, "idi", cType, false, nil)
	if err != nil {
		//TODO: find out a way to show the error message in ui
		// fmt.Println(strings.Repeat(err.Error(), 10))
		return *m, nil
	}
	m.Input.Reset()
	return *m, nil
}

func updateFocusedComponent(m *Model, msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	if m.FocusedOn == "input" {
		m.Input.Focus()
		m.Input, cmd = m.Input.Update(msg)
	} else if m.FocusedOn == "sideBar" {
		m.Input.Blur()
		if m.Mode == "channels" {
			m.Channels, cmd = m.Channels.Update(msg)
		} else if m.Mode == "users" {
			m.Users, cmd = m.Users.Update(msg)
		} else {
			m.Groups, cmd = m.Groups.Update(msg)
		}

	} else {
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
		fmt.Println(strings.Repeat("uff there is something off", 10), err.Error())
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
	if currentlyFoucsedOn == "sideBar" {
		m.FocusedOn = "mainView"
	} else if currentlyFoucsedOn == "mainView" {
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
		return *m, nil
	case "u":
		m.Mode = "users"
		return *m, nil
	case "g":
		m.Mode = "groups"
		return *m, nil
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

func setItemStyles(m *Model) string {
	sidebarWidth := m.Width * 30 / 100
	mainWidth := m.Width - sidebarWidth
	contentHeight := m.Height * 90 / 100
	inputHeight := m.Height - contentHeight

	//there are some extra spaces at top and bottom if we incude the becomes ugly so
	// this one is a solution i found
	// feels a bit hacky
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
