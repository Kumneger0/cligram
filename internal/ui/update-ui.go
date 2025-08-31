package ui

import (
	"log/slog"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kumneger0/cligram/internal/telegram"
	"github.com/kumneger0/cligram/internal/telegram/types"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case types.SendMessageMsg:
		if msg.Err != nil {
			slog.Error("Failed to send message", "error", msg.Err.Error())
			m.IsModalVisible = true
			m.ModalContent = GetModalContent(msg.Err.Error())
			return m, nil
		}
		if m.SelectedFile == "uploading..." {
			m.SelectedFile = ""
		}
		return m, nil
	case types.EditMessageMsg:
		if msg.Err != nil {
			slog.Error("Failed to edit message", "error", msg.Err.Error())
			m.IsModalVisible = true
			m.ModalContent = GetModalContent(msg.Err.Error())
			return m, nil
		}
		response := msg.Response
		if response {
			if selectedMessage, ok := m.ChatUI.SelectedItem().(types.FormattedMessage); ok {
				selectedMessage.Content = msg.UpdatedMessage
				items := m.ChatUI.Items()
				items[m.ChatUI.GlobalIndex()] = selectedMessage
				m.ChatUI.SetItems(items)
			}
			m.EditMessage = nil
		}

	case types.UserTypingNotification:
		user := msg.User
		if m.SelectedUser.PeerID == user.PeerID {
			m.SelectedUser = user
		}
		userIndex := getUserIndex(m, user)
		if userIndex != -1 {
			items := m.Users.Items()
			items[userIndex] = user
			cmd := m.Users.SetItems(items)
			cmds = append(cmds, cmd)
		}
		if user.IsTyping {
			rcmd := tea.Tick(time.Second*3, func(t time.Time) tea.Msg {
				user.IsOnline = false
				user.IsTyping = false
				return types.UserTypingNotification{User: user}
			})
			cmds = append(cmds, rcmd)
		}
	case types.ErrorNotification:
		m.ModalContent = GetModalContent(msg.Error.Error())
		m.IsModalVisible = true
	case types.MarkMessagesAsReadMsg:
		model, cmd := m.handleMarkMessagesAsRead(msg)
		m = model.(Model)
		cmds = append(cmds, cmd)
	case types.NewMessageNotification:
		model, cmd := m.handleNewMessage(msg)
		m = model.(Model)
		cmds = append(cmds, cmd)
		return m, cmd
	case types.UserStatusNotification:
		model, cmd := m.handleUserOnlineOffline(msg)
		m = model.(Model)
		cmds = append(cmds, cmd)
	case MessageDeletionConfrimResponseMsg:
		model, cmd := m.handleMessageDeletion(msg)
		m = model.(Model)
		cmds = append(cmds, cmd)
	case types.GetMessagesMsg:
		model, cmd := m.handleGetMessages(msg)
		m = model.(Model)
		cmds = append(cmds, cmd)
	case tea.KeyMsg:
		model, cmd := m.handleKeyPress(msg)
		m = model.(Model)
		cmds = append(cmds, cmd)
	case types.UserChatsMsg:
		model, cmd := m.handleUserChats(msg)
		m = model.(Model)
		cmds = append(cmds, cmd)
	case types.ChannelsMsg:
		model, cmd := m.handleUserChannels(msg)
		m = model.(Model)
		cmds = append(cmds, cmd)
	case types.GroupsMsg:
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

func (m Model) handleMarkMessagesAsRead(msg types.MarkMessagesAsReadMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		slog.Error("Failed to mark messages as read", "error", msg.Err.Error())
		return m, nil
	}
	if msg.Response {
		m.SelectedUser.UnreadCount = 0
	}
	userIndex := getUserIndex(m, m.SelectedUser)
	if userIndex != -1 {
		items := m.Users.Items()
		user := items[userIndex].(types.UserInfo)
		user.UnreadCount = 0
		m.Users.SetItem(userIndex, user)
	}
	return m, nil
}

func (m Model) handleNewMessage(msg types.NewMessageNotification) (tea.Model, tea.Cmd) {
	areInGroupOrChannelMode := m.Mode == ModeGroups || m.Mode == ModeChannels
	if msg.ChannelOrGroup != nil {
		if !areInGroupOrChannelMode {
			return m, nil
		}
		isGroupOrChannelSelected := isChannelOrGroupSelected(&m, &msg)
		if isGroupOrChannelSelected {
			copy(m.Conversations[:], m.Conversations[1:])
			m.Conversations[len(m.Conversations)-1] = msg.Message
			m.updateConverstaions()
			return m, nil
		}
		groupIndex := getGroupIndex(m, *msg.ChannelOrGroup)
		if groupIndex != -1 {
			items := m.Groups.Items()
			group := items[groupIndex].(types.ChannelInfo)
			group.UnreadCount++
			m.Groups.SetItem(groupIndex, group)
		}
	}

	if msg.User == nil {
		return m, nil
	}
	isSelected := isUserSelected(&m, &msg)
	areWeInBotOrUserMode := m.Mode == ModeUsers || m.Mode == ModeBots

	if !areWeInBotOrUserMode {
		return m, nil
	}

	if !isSelected {
		userIndex := getUserIndex(m, *msg.User)
		if userIndex != -1 {
			items := m.Users.Items()
			user := items[userIndex].(types.UserInfo)
			user.UnreadCount++
			m.Users.SetItem(userIndex, user)
		}
		return m, nil
	}

	m.SelectedUser.UnreadCount++
	var shouldWeUpdateCurrentActiveConversation bool = false
	for _, v := range m.Conversations {
		if v.FromID != nil && *v.FromID == m.SelectedUser.PeerID {
			shouldWeUpdateCurrentActiveConversation = true
			break
		}
	}
	if shouldWeUpdateCurrentActiveConversation {
		copy(m.Conversations[:], m.Conversations[1:])
		var formattedMessage = msg.Message
		formattedMessage.Sender = m.SelectedUser.FirstName
		m.Conversations[len(m.Conversations)-1] = formattedMessage
		m.updateConverstaions()
	}
	return m, nil
}

func isChannelOrGroupSelected(m *Model, msg *types.NewMessageNotification) bool {
	return m.SelectedChannel.ID == msg.ChannelOrGroup.ID ||
		m.SelectedGroup.ID == msg.ChannelOrGroup.ID
}

func isUserSelected(m *Model, msg *types.NewMessageNotification) bool {
	return m.SelectedUser.PeerID == msg.User.PeerID
}

func (m Model) handleUserOnlineOffline(msg types.UserStatusNotification) (tea.Model, tea.Cmd) {
	var user types.UserInfo
	for _, v := range m.Users.Items() {
		u, ok := v.(types.UserInfo)
		if ok && u.PeerID == msg.UserInfo.PeerID {
			user = u
			break
		}
	}

	userIndex := getUserIndex(m, user)
	if userIndex != -1 {
		items := m.Users.Items()
		user := items[userIndex].(types.UserInfo)
		user.IsOnline = msg.Status.IsOnline
		items[userIndex] = user
		m.Users.SetItems(items)
	}
	return m, nil
}

func (m Model) handleMessageDeletion(msg MessageDeletionConfrimResponseMsg) (tea.Model, tea.Cmd) {
	if !msg.yes {
		return m, nil
	}
	peer := m.getPeerInfo()
	selectedItemInChat := m.ChatUI.SelectedItem().(types.FormattedMessage)
	response, err := telegram.Cligram.DeleteMessage(telegram.Cligram.Context(), types.DeleteMessageRequest{
		Peer:      peer,
		MessageID: int(selectedItemInChat.ID),
	})
	if err != nil {
		m.IsModalVisible = true
		m.ModalContent = GetModalContent(err.Error())
		return m, nil
	}
	if response.Status == "success" {
		var updatedConversations [50]types.FormattedMessage
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

func (m Model) getPeerInfo() types.Peer {
	var peer types.Peer

	switch m.Mode {
	case ModeUsers:
		peer = types.Peer{
			ID:         m.SelectedUser.PeerID,
			AccessHash: m.SelectedUser.AccessHash,
			ChatType:   types.UserChat,
		}
	case ModeBots:
		peer = types.Peer{
			ID:         m.SelectedUser.PeerID,
			AccessHash: m.SelectedUser.AccessHash,
			ChatType:   types.BotChat,
		}
	case ModeChannels:
		peer = types.Peer{
			ID:         m.SelectedUser.PeerID,
			AccessHash: m.SelectedUser.AccessHash,
			ChatType:   types.ChannelChat,
		}
	case ModeGroups:
		peer = types.Peer{
			ID:         m.SelectedUser.PeerID,
			AccessHash: m.SelectedUser.AccessHash,
			ChatType:   types.GroupChat,
		}
	}

	return peer
}

func (m Model) handleGetMessages(msg types.GetMessagesMsg) (tea.Model, tea.Cmd) {
	m.AreWeSwitchingModes = false
	m.MainViewLoading = false
	if msg.Err != nil {
		slog.Error("Failed to get messages", "error", msg.Err.Error())
		m.IsModalVisible = true
		m.ModalContent = GetModalContent(msg.Err.Error())
		return m, nil
	}

	messagesWeGot := len(filterEmptyMessages(msg.Messages))
	if messagesWeGot < 1 {
		if selectedChat, ok := m.Users.SelectedItem().(types.UserInfo); ok && selectedChat.IsBot {
			m.Input.SetValue("/start")
		}
	}

	if messagesWeGot < 1 {
		return m, nil
	}

	m.Conversations = m.mergeConversations(msg.Messages, messagesWeGot)
	conversationLastIndex := len(m.Conversations) - 1
	m.updateConverstaions()
	m.ChatUI.Select(conversationLastIndex)
	return m, nil
}

func (m Model) mergeConversations(newMessages [50]types.FormattedMessage, messagesWeGot int) [50]types.FormattedMessage {
	var updatedConversations [50]types.FormattedMessage
	if messagesWeGot >= 50 {
		return newMessages
	}

	var oldMessages []types.FormattedMessage
	for _, v := range m.ChatUI.Items() {
		if msg, ok := v.(types.FormattedMessage); ok && msg.ID != 0 {
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
	case "shift+down":
		if m.FocusedOn == Mainview {
			items := m.ChatUI.Items()
			lastIndex := len(items) - 1
			m.ChatUI.Select(lastIndex)
		}
	// case "up", "down":
	/*
	 TODO: consider implementing pagination and show all messages
	*/
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
		if selectedItem, ok := m.ChatUI.SelectedItem().(types.FormattedMessage); ok && strings.ToLower(selectedItem.Sender) == "you" {
			m.FocusedOn = Input
			m.Input.SetValue(selectedItem.Content)
			m.EditMessage = &selectedItem
			m.SkipNextInput = true
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
			if selectedMessage, ok := m.ChatUI.SelectedItem().(types.FormattedMessage); ok {
				m.FocusedOn = Input
				m.SkipNextInput = true
				m.ReplyTo = &selectedMessage
			}
			return m, nil
		}
	}
	return m, nil
}

func (m Model) handleDeleteKey() (tea.Model, tea.Cmd) {
	if m.FocusedOn == Mainview {
		selectedItem := m.ChatUI.SelectedItem().(types.FormattedMessage)
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
		selectedMessage, ok := m.ChatUI.SelectedItem().(types.FormattedMessage)
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

func (m Model) handleUserChats(msg types.UserChatsMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		m.IsModalVisible = true
		m.ModalContent = GetModalContent(msg.Err.Error())
		return m, nil
	}

	var users []list.Item
	for _, du := range *msg.Response {
		users = append(users, du)
	}
	m.Users.SetItems(users)
	m.AreWeSwitchingModes = false
	return m, nil
}

func (m Model) handleUserChannels(msg types.ChannelsMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		m.IsModalVisible = true
		m.ModalContent = GetModalContent(msg.Err.Error())
		return m, nil
	}

	var channels []list.Item
	for _, du := range *msg.Response {
		channels = append(channels, du)
	}
	m.Channels.SetItems(channels)
	m.AreWeSwitchingModes = false
	return m, nil
}

func (m Model) handleUserGroups(msg types.GroupsMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		m.IsModalVisible = true
		m.ModalContent = GetModalContent(msg.Err.Error())
		return m, nil
	}

	var groups []list.Item
	for _, du := range *msg.Response {
		groups = append(groups, du)
	}
	m.Groups.SetItems(groups)
	m.AreWeSwitchingModes = false
	return m, nil
}

func (m Model) handleForwardMessage(msg ForwardMsg) (tea.Model, tea.Cmd) {
	messageToBeForwarded := msg.msg
	receiver := *msg.receiver
	fromPeer := *msg.fromPeer

	from, toPeer := m.extractPeerInfo(fromPeer, receiver)
	messageIDs := []int{int(messageToBeForwarded.ID)}

	err := telegram.Cligram.ForwardMessages(telegram.Cligram.Context(), types.ForwardMessagesRequest{
		FromPeer:   from,
		ToPeer:     toPeer,
		MessageIDs: messageIDs,
	})
	if err != nil {
		slog.Error("Failed to forward message", "error", err.Error())
		return m, nil
	}

	return m, nil
}

func (m Model) extractPeerInfo(fromPeer, receiver list.Item) (from, toPeer types.Peer) {
	if fromUser, ok := fromPeer.(types.UserInfo); ok {
		from.ID = fromUser.PeerID
		from.AccessHash = fromUser.AccessHash
		from.ChatType = types.UserChat
	}

	if fromChannelOrGroup, ok := fromPeer.(types.ChannelInfo); ok {
		from.ID = fromChannelOrGroup.ID
		from.AccessHash = fromChannelOrGroup.AccessHash
		from.ChatType = types.ChannelChat
	}

	if userOrChannel, ok := receiver.(types.UserInfo); ok {
		toPeer.ID = userOrChannel.PeerID
		toPeer.AccessHash = userOrChannel.AccessHash
		toPeer.ChatType = types.UserChat
	}

	if channelOrGroup, ok := receiver.(types.ChannelInfo); ok {
		toPeer.ID = channelOrGroup.ID
		toPeer.AccessHash = channelOrGroup.AccessHash
		toPeer.ChatType = types.ChannelChat
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

func (m Model) handleSearchedUser(user types.UserInfo) (tea.Model, tea.Cmd) {
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

func (m Model) handleSearchedChannel(channel types.ChannelInfo) (tea.Model, tea.Cmd) {
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

func (m Model) handleSearchedGroup(group types.ChannelInfo) (tea.Model, tea.Cmd) {
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
