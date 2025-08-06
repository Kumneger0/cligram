package ui

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kumneger0/cligram/internal/rpc"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case rpc.SendMessageMsg:
		if msg.Err != nil || msg.Response.Error != nil {
			slog.Error("Failed to send message", "error", msg.Err.Error())
			m.IsModalVisible = true
			m.ModalContent = GetModalContent(msg.Err.Error())
			return m, nil
		}
		if m.SelectedFile == "uploading..." {
			m.SelectedFile = ""
		}
		return m, nil
	case rpc.EditMessageMsg:
		if msg.Err != nil {
			slog.Error("Failed to edit message", "error", msg.Err.Error())
			m.IsModalVisible = true
			m.ModalContent = GetModalContent(msg.Err.Error())
			return m, nil
		}
		response := msg.Response
		if response.Error != nil {
			slog.Error("Failed to edit message", "error", response.Error.Message)
			m.IsModalVisible = true
			m.ModalContent = GetModalContent(response.Error.Message)
			return m, nil
		}
		if response.Result {
			if selectedMessage, ok := m.ChatUI.SelectedItem().(rpc.FormattedMessage); ok {
				selectedMessage.Content = msg.UpdatedMessage
				items := m.ChatUI.Items()
				items[m.ChatUI.GlobalIndex()] = selectedMessage
				m.ChatUI.SetItems(items)
			}
			m.EditMessage = nil
		}

	case rpc.SetUserTypingMsg:
		if msg.Err != nil {
			slog.Error("failed to set user typing", "error", msg.Err.Error())
		}
		return m, nil

	case rpc.MarkMessagesAsReadMsg:
		model, cmd := m.handleMarkMessagesAsRead(msg)
		m = model.(Model)
		cmds = append(cmds, cmd)
	case rpc.NewMessageMsg:
		model, cmd := m.handleNewMessage(msg)
		m = model.(Model)
		cmds = append(cmds, cmd)
		return m, cmd
	case rpc.UserOnlineOffline:
		model, cmd := m.handleUserOnlineOffline(msg)
		m = model.(Model)
		cmds = append(cmds, cmd)
	case MessageDeletionConfrimResponseMsg:
		model, cmd := m.handleMessageDeletion(msg)
		m = model.(Model)
		cmds = append(cmds, cmd)
	case rpc.GetMessagesMsg:
		model, cmd := m.handleGetMessages(msg)
		m = model.(Model)
		cmds = append(cmds, cmd)
	case tea.KeyMsg:
		model, cmd := m.handleKeyPress(msg)
		m = model.(Model)
		cmds = append(cmds, cmd)
	case rpc.UserChatsMsg:
		model, cmd := m.handleUserChats(msg)
		m = model.(Model)
		cmds = append(cmds, cmd)
	case rpc.UserChannelMsg:
		model, cmd := m.handleUserChannels(msg)
		m = model.(Model)
		cmds = append(cmds, cmd)
	case rpc.UserGroupsMsg:
		model, cmd := m.handleUserGroups(msg)
		m = model.(Model)
		cmds = append(cmds, cmd)
	case ForwardMsg:
		model, cmd := m.handleForwardMessage(msg)
		m = model.(Model)
		cmds = append(cmds, cmd)
	case tea.WindowSizeMsg:
		model, cmd := m.handleWindowSize(msg)
		m = model.(Model)
		cmds = append(cmds, cmd)
		m.Filepicker, cmd = m.Filepicker.Update(msg)
		cmds = append(cmds, cmd)
	case SelectSearchedUserResult:
		model, cmd := m.handleSearchedUserResult(msg)
		m = model.(Model)
		cmds = append(cmds, cmd)
	}
	return updateFocusedComponent(&m, msg, &cmds)
}

func (m Model) handleMarkMessagesAsRead(msg rpc.MarkMessagesAsReadMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		slog.Error("Failed to mark messages as read", "error", msg.Err.Error())
		return m, nil
	}
	if msg.Response.Result {
		m.SelectedUser.UnreadCount = 0
	}
	userIndex := getUserIndex(m, m.SelectedUser)
	if userIndex != -1 {
		items := m.Users.Items()
		user := items[userIndex].(rpc.UserInfo)
		user.UnreadCount = 0
		items[userIndex] = user
		m.Users.SetItems(items)
	}
	return m, nil
}

func (m Model) handleNewMessage(msg rpc.NewMessageMsg) (tea.Model, tea.Cmd) {
	fmt.Println("new message is comming", msg.User)
	if m.SelectedUser.PeerID == msg.User.PeerID {
		if m.Mode == ModeUsers || m.Mode == ModeBots {
			m.SelectedUser.UnreadCount++
			var isCurrentConversationUsersChat bool = false
			for _, v := range m.Conversations {
				if v.Sender == m.SelectedUser.FirstName {
					isCurrentConversationUsersChat = true
					break
				}
			}
			if isCurrentConversationUsersChat {
				copy(m.Conversations[:], m.Conversations[1:])
				m.Conversations[len(m.Conversations)-1] = msg.Message
				m.updateConverstaions()
			}
		}
	} else {
		userIndex := getUserIndex(m, msg.User)
		if userIndex != -1 {
			items := m.Users.Items()
			user := items[userIndex].(rpc.UserInfo)
			user.UnreadCount++
			items[userIndex] = user
			m.Users.SetItems(items)
		}
	}
	return m, nil
}

func (m Model) handleUserOnlineOffline(msg rpc.UserOnlineOffline) (tea.Model, tea.Cmd) {
	var user rpc.UserInfo
	for _, v := range m.Users.Items() {
		u, ok := v.(rpc.UserInfo)
		if ok && u.FirstName == msg.FirstName {
			user = u
			break
		}
	}

	userIndex := getUserIndex(m, user)
	if userIndex != -1 {
		items := m.Users.Items()
		user := items[userIndex].(rpc.UserInfo)
		user.IsOnline = msg.Status == "online"
		items[userIndex] = user
		m.Users.SetItems(items)
	}
	return m, nil
}

func (m Model) handleMessageDeletion(msg MessageDeletionConfrimResponseMsg) (tea.Model, tea.Cmd) {
	if !msg.yes {
		return m, nil
	}

	peer, cType := m.getPeerInfoAndChatType()
	selectedItemInChat := m.ChatUI.SelectedItem().(rpc.FormattedMessage)

	response, err := rpc.RpcClient.DeleteMessage(peer, int(selectedItemInChat.ID), cType)
	if err != nil {
		m.IsModalVisible = true
		m.ModalContent = GetModalContent(err.Error())
		return m, nil
	}

	if response.Error != nil {
		m.IsModalVisible = true
		m.ModalContent = GetModalContent(response.Error.Message)
		return m, nil
	}

	if response.Result.Status == "success" {
		var updatedConversations [50]rpc.FormattedMessage
		for i, v := range m.Conversations {
			if v.ID != selectedItemInChat.ID {
				updatedConversations[i] = v
			}
		}
		m.Conversations = updatedConversations
		cmd := m.ChatUI.SetItems(formatMessages(updatedConversations))
		return m, cmd
	}

	return m, nil
}

func (m Model) getPeerInfoAndChatType() (rpc.PeerInfo, rpc.ChatType) {
	var peer rpc.PeerInfo
	var cType rpc.ChatType

	switch m.Mode {
	case ModeUsers:
		peer = rpc.PeerInfo{
			PeerID:     m.SelectedUser.PeerID,
			AccessHash: m.SelectedUser.AccessHash,
		}
		cType = rpc.UserChat
	case ModeChannels:
		peer = rpc.PeerInfo{
			PeerID:     m.SelectedChannel.ChannelID,
			AccessHash: m.SelectedChannel.AccessHash,
		}
		cType = rpc.ChannelChat
	case ModeGroups:
		peer = rpc.PeerInfo{
			PeerID:     m.SelectedGroup.ChannelID,
			AccessHash: m.SelectedGroup.AccessHash,
		}
		cType = rpc.GroupChat
	}

	return peer, cType
}

func (m Model) handleGetMessages(msg rpc.GetMessagesMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		slog.Error("Failed to get messages", "error", msg.Err.Error())
		m.IsModalVisible = true
		m.ModalContent = GetModalContent(msg.Err.Error())
		m.MainViewLoading = false
		return m, nil
	}

	messagesWeGot := len(filterEmptyMessages(msg.Messages.Result))
	if messagesWeGot < 1 {
		if selectedChat, ok := m.Users.SelectedItem().(rpc.UserInfo); ok && selectedChat.IsBot {
			m.Input.SetValue("/start")
		}
	}

	if messagesWeGot < 1 {
		m.MainViewLoading = false
		return m, nil
	}

	m.Conversations = m.mergeConversations(msg.Messages.Result, messagesWeGot)
	conversationLastIndex := len(m.Conversations) - 1
	m.updateConverstaions()
	m.ChatUI.Select(conversationLastIndex)
	m.MainViewLoading = false
	return m, nil
}

func (m Model) mergeConversations(newMessages [50]rpc.FormattedMessage, messagesWeGot int) [50]rpc.FormattedMessage {
	var updatedConversations [50]rpc.FormattedMessage
	if messagesWeGot >= 50 {
		return newMessages
	}

	var oldMessages []rpc.FormattedMessage
	for _, v := range m.ChatUI.Items() {
		if msg, ok := v.(rpc.FormattedMessage); ok && msg.ID != 0 {
			oldMessages = append(oldMessages, msg)
		}
	}

	oldMessagesLength := len(oldMessages)
	if (messagesWeGot + oldMessagesLength) <= 50 {
		updatedConversations = newMessages
	} else {
		numberMessagesWeShouldTakeFromOldConversation := 50 - messagesWeGot
		messagesToAppend := m.Conversations[:numberMessagesWeShouldTakeFromOldConversation]
		messagesToAppend = append(messagesToAppend, newMessages[:]...)
		copy(updatedConversations[:], messagesToAppend)
	}

	return updatedConversations
}

func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	switch msg.String() {
	case "up", "down":
		/*
			         as of now we are only showing last 50 messages
					 TODO: consider implementing pagination and show all messages
		*/
		// return m, nil
	case "ctrl+a":
		m, cmd := m.handleCtrlA()
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	case "q", "ctrl+c":
		return m, tea.Quit
	case "tab":
		if !m.IsFilepickerVisible {
			m, cmd := changeFocusMode(&m, "tab", false)
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)
		}
		return m, nil
	case "shift+tab":
		m, cmd := changeFocusMode(&m, "tab", true)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	case "m":
		m, cmd := m.handleMKey(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	case "c":
		m, cmd := changeSideBarMode(&m, "c")
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	case "u":
		m, cmd := changeSideBarMode(&m, "u")
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	case "g":
		m, cmd := changeSideBarMode(&m, "g")
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	case "b":
		m, cmd := changeSideBarMode(&m, "b")
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	case "enter":
		m, cmd := m.handleEnterKey()
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	case "r":
		m, cmd := m.handleReplyKey()
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	case "d":
		m, cmd := m.handleDeleteKey()
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	case "f":
		m, cmd := m.handleForwardKey()
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	case "e":
		m, cmd := m.handleEditKey()
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}
	cmds = append(cmds, SendUserIsTyping(&m))
	return m, tea.Batch(cmds...)
}

func (m Model) handleEditKey() (tea.Model, tea.Cmd) {
	if m.FocusedOn == Mainview {
		if selectedItem, ok := m.ChatUI.SelectedItem().(rpc.FormattedMessage); ok && strings.ToLower(selectedItem.Sender) == "you" {
			m.FocusedOn = Input
			m.Input.SetValue(selectedItem.Content)
			m.EditMessage = &selectedItem
			return m, nil
		}
	}
	return m, nil
}

func (m Model) handleCtrlA() (tea.Model, tea.Cmd) {
	if m.FocusedOn == Input {
		m.IsFilepickerVisible = true
		m.FocusedOn = Mainview
		return m, nil
	}
	if m.IsFilepickerVisible {
		m.IsFilepickerVisible = false
		m.FocusedOn = Input
	}
	return m, nil
}

func (m Model) handleMKey(_ tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.FocusedOn != Input {
		m.IsModalVisible = true
		return m, nil
	}
	return m, nil
}

func (m Model) handleEnterKey() (tea.Model, tea.Cmd) {
	if m.FocusedOn == Input {
		return sendMessage(&m)
	}
	if m.FocusedOn == SideBar {
		return handleUserChange(&m)
	}
	return m, nil
}

func (m Model) handleReplyKey() (tea.Model, tea.Cmd) {
	if m.FocusedOn == Mainview {
		canWrite := (m.Mode == ModeUsers || m.Mode == ModeGroups) || (m.Mode == ModeChannels && m.SelectedChannel.IsCreator)
		if canWrite {
			m.IsReply = true
			if selectedMessage, ok := m.ChatUI.SelectedItem().(rpc.FormattedMessage); ok {
				m.FocusedOn = Input
				m.ReplyTo = &selectedMessage
			}
			return m, nil
		}
	}
	return m, nil
}

func (m Model) handleDeleteKey() (tea.Model, tea.Cmd) {
	if m.FocusedOn == Mainview {
		selectedItem := m.ChatUI.SelectedItem().(rpc.FormattedMessage)
		return m, func() tea.Msg {
			return OpenModalMsg{
				ModalMode: ModalModeDeleteMessage,
				Message:   &selectedItem,
			}
		}
	}
	return m, nil
}

func (m Model) handleForwardKey() (tea.Model, tea.Cmd) {
	if m.FocusedOn == Mainview {
		selectedMessage, ok := m.ChatUI.SelectedItem().(rpc.FormattedMessage)
		if !ok {
			return m, nil
		}

		var from list.Item
		switch m.Mode {
		case ModeUsers:
			from = m.SelectedUser
		case ModeChannels:
			from = m.SelectedChannel
		case ModeGroups:
			from = &m.SelectedGroup
		}

		return m, func() tea.Msg {
			return OpenModalMsg{
				ModalMode: ModalModeForwardMessage,
				Message:   &selectedMessage,
				UsersList: &m.Users,
				FromPeer:  &from,
			}
		}
	}
	return m, nil
}

func (m Model) handleUserChats(msg rpc.UserChatsMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		m.IsModalVisible = true
		m.ModalContent = GetModalContent(msg.Err.Error())
		return m, nil
	}

	var users []list.Item
	for _, du := range msg.Response.Result {
		users = append(users, du)
	}
	m.Users.SetItems(users)
	return m, nil
}

func (m Model) handleUserChannels(msg rpc.UserChannelMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		m.IsModalVisible = true
		m.ModalContent = GetModalContent(msg.Err.Error())
		return m, nil
	}

	var channels []list.Item
	for _, du := range msg.Response.Result {
		channels = append(channels, du)
	}
	m.Channels.SetItems(channels)
	return m, nil
}

func (m Model) handleUserGroups(msg rpc.UserGroupsMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		m.IsModalVisible = true
		m.ModalContent = GetModalContent(msg.Err.Error())
		return m, nil
	}

	var groups []list.Item
	for _, du := range msg.Response.Result {
		groups = append(groups, du)
	}
	m.Groups.SetItems(groups)
	return m, nil
}

func (m Model) handleForwardMessage(msg ForwardMsg) (tea.Model, tea.Cmd) {
	messageToBeForwarded := msg.msg
	reciever := *msg.reciever
	fromPeer := *msg.fromPeer

	from, toPeer, cType := m.extractPeerInfo(fromPeer, reciever)
	messageIDs := []int{int(messageToBeForwarded.ID)}

	response, err := rpc.RpcClient.ForwardMessages(from, messageIDs, toPeer, cType)
	if err != nil {
		slog.Error("Failed to forward message", "error", err.Error())
		return m, nil
	}

	if response.Error != nil {
		slog.Error("Failed to forward message", "error", response.Error.Message)
	}

	return m, nil
}

func (m Model) extractPeerInfo(fromPeer, reciever interface{}) (from, toPeer rpc.PeerInfo, cType rpc.ChatType) {
	if fromUser, ok := fromPeer.(rpc.UserInfo); ok {
		from.PeerID = fromUser.PeerID
		from.AccessHash = fromUser.AccessHash
		cType = "user"
	}

	if fromChannelOrGroup, ok := fromPeer.(rpc.ChannelAndGroupInfo); ok {
		from.PeerID = fromChannelOrGroup.ChannelID
		from.AccessHash = fromChannelOrGroup.AccessHash
		cType = "channel"
	}

	if userOrChannel, ok := reciever.(rpc.UserInfo); ok {
		toPeer.PeerID = userOrChannel.PeerID
		toPeer.AccessHash = userOrChannel.AccessHash
	}

	if channelOrGroup, ok := reciever.(rpc.ChannelAndGroupInfo); ok {
		toPeer.PeerID = channelOrGroup.ChannelID
		toPeer.AccessHash = channelOrGroup.AccessHash
	}

	return
}

func (m Model) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.Width = msg.Width - 4
	m.Height = msg.Height - 4
	headerHeight := 7
	footerHeight := 7
	verticalMarginHeight := headerHeight + footerHeight
	m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
	m.viewport.YPosition = headerHeight
	m.updateConverstaions()
	return m, nil
}

func (m Model) handleSearchedUserResult(msg SelectSearchedUserResult) (tea.Model, tea.Cmd) {
	if msg.user != nil {
		return m.handleSearchedUser(*msg.user)
	}
	if msg.channel != nil {
		return m.handleSearchedChannel(*msg.channel)
	}
	if msg.group != nil {
		return m.handleSearchedGroup(*msg.group)
	}
	return m, nil
}

func (m Model) handleSearchedUser(user rpc.UserInfo) (tea.Model, tea.Cmd) {
	m.SelectedUser = user
	index := getUserIndex(m, user)
	if index != -1 {
		m.Users.Select(index)
		m.FocusedOn = SideBar
		m.Mode = ModeUsers
		return handleUserChange(&m)
	}

	newUpdatedUsers := append(m.Users.Items(), user)
	updateUserCmd := m.Users.SetItems(newUpdatedUsers)

	index = getUserIndex(m, user)
	if index != -1 {
		m.Users.Select(index)
		m.FocusedOn = SideBar
		m.Mode = ModeUsers
		m, handleUserChangeCmd := handleUserChange(&m)
		return m, tea.Batch(updateUserCmd, handleUserChangeCmd)
	}

	return m, updateUserCmd
}

func (m Model) handleSearchedChannel(channel rpc.ChannelAndGroupInfo) (tea.Model, tea.Cmd) {
	m.SelectedChannel = channel
	index := getChannelIndex(m, channel)

	if index != -1 {
		m.Channels.Select(index)
		m.FocusedOn = SideBar
		m.Mode = ModeChannels
		return handleUserChange(&m)
	}

	newUpdatedChannelsList := append(m.Channels.Items(), channel)
	setItemsCmd := m.Channels.SetItems(newUpdatedChannelsList)
	index = getChannelIndex(m, channel)

	if index != -1 {
		m.Channels.Select(index)
		m.FocusedOn = SideBar
		m.Mode = ModeChannels
		m, handleChangeUserCmd := handleUserChange(&m)
		return m, tea.Batch(setItemsCmd, handleChangeUserCmd)
	}
	return m, setItemsCmd
}

func (m Model) handleSearchedGroup(group rpc.ChannelAndGroupInfo) (tea.Model, tea.Cmd) {
	m.SelectedGroup = group
	index := getGroupIndex(m, group)

	if index != -1 {
		m.Groups.Select(index)
		m.FocusedOn = SideBar
		m.Mode = ModeGroups
		return handleUserChange(&m)
	}

	newUpdatedGroupsList := append(m.Groups.Items(), group)
	setItemsCmd := m.Groups.SetItems(newUpdatedGroupsList)
	index = getGroupIndex(m, group)

	if index != -1 {
		m.Groups.Select(index)
		m.FocusedOn = SideBar
		m.Mode = ModeGroups
		m, handleChangeUserCmd := handleUserChange(&m)
		return m, tea.Batch(setItemsCmd, handleChangeUserCmd)
	}
	return m, setItemsCmd
}
