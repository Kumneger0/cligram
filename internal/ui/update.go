package ui

import (
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gotd/td/tg"
	"github.com/kumneger0/cligram/internal/notification"
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
		if msg.Response {
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
		l := listForUser(&m, user)
		userIndex := getUserIndex(*l, user)
		if userIndex != -1 {
			items := l.Items()
			items[userIndex] = user
			cmds = append(cmds, l.SetItems(items))
		}
		if user.IsTyping {
			cmds = append(cmds, tea.Tick(time.Second*3, func(t time.Time) tea.Msg {
				user.IsOnline = false
				user.IsTyping = false
				return types.UserTypingNotification{User: user}
			}))
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
	case types.GetAllStoriesMsg:
		model, cmd := m.updateUserStories(msg)
		m = model.(Model)
		cmds = append(cmds, cmd)
	}
	return updateFocusedComponent(&m, msg, &cmds)
}

func (m Model) updateUserStories(msg types.GetAllStoriesMsg) (tea.Model, tea.Cmd) {
	m.Stories = msg.Stories
	return m, nil
}

func (m Model) handleMarkMessagesAsRead(msg types.MarkMessagesAsReadMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		slog.Error("Failed to mark messages as read", "error", msg.Err.Error())
		return m, nil
	}
	if msg.Response {
		m.SelectedUser.UnreadCount = 0
	}
	l := listForUser(&m, m.SelectedUser)
	userIndex := getUserIndex(*l, m.SelectedUser)
	if userIndex != -1 {
		items := l.Items()
		user := items[userIndex].(types.UserInfo)
		user.UnreadCount = 0
		return m, l.SetItem(userIndex, user)
	}
	return m, nil
}

func sendNewMessageNotification[T types.UserInfo | types.ChannelInfo](item T, message *tg.Message) {
	switch v := any(item).(type) {
	case types.UserInfo:
		if v.NotifySettings != nil && (v.NotifySettings.Silent || (v.NotifySettings.MuteUntil != 0 && int64(v.NotifySettings.MuteUntil) > time.Now().Unix())) {
			return
		}

		title := v.FirstName
		if v.LastName != "" {
			title = fmt.Sprintf("%s %s", v.FirstName, v.LastName)
		}

		content := "Sent you a new message"
		if v.NotifySettings != nil && v.NotifySettings.ShowPreviews {
			content = message.Message
		}

		notification.Notify(title, content)

	case types.ChannelInfo:
		if v.NotifySettings != nil && (v.NotifySettings.Silent || (v.NotifySettings.MuteUntil != 0 && int64(v.NotifySettings.MuteUntil) > time.Now().Unix())) {
			return
		}

		title := v.ChannelTitle
		content := "New message"

		if v.NotifySettings != nil && v.NotifySettings.ShowPreviews {
			content = message.Message
		}

		notification.Notify(title, content)
	}
}

func (m Model) handleNewMessage(msg types.NewMessageNotification) (tea.Model, tea.Cmd) {
	peerID := msg.FromID
	var userInfo *types.UserInfo
	for _, v := range slices.Concat(m.Users.Items(), m.Bots.Items()) {
		if user, ok := v.(types.UserInfo); ok && user.PeerID == peerID {
			userInfo = &user
			break
		}
	}

	var channelOrGroupInfo *types.ChannelInfo
	if userInfo == nil {
		for _, v := range slices.Concat(m.Channels.Items(), m.Groups.Items()) {
			if cg, ok := v.(types.ChannelInfo); ok && cg.ID == peerID {
				channelOrGroupInfo = &cg
				break
			}
		}
	}

	if userInfo != nil && !msg.Message.Out {
		sendNewMessageNotification(*userInfo, msg.Message)
	}

	if channelOrGroupInfo != nil && !msg.Message.Out {
		sendNewMessageNotification(*channelOrGroupInfo, msg.Message)
	}

	if channelOrGroupInfo != nil {
		if m.Mode != ModeGroups && m.Mode != ModeChannels {
			return m, nil
		}

		chatType := types.GroupChat
		if m.Mode == ModeChannels {
			chatType = types.ChannelChat
		}

		if m.SelectedChannel.ID == channelOrGroupInfo.ID || m.SelectedGroup.ID == channelOrGroupInfo.ID {
			formattedMessage := getFormattedMessageFunc(GetFormattedMessageArg{
				ChatType:           chatType,
				ChannelOrGroupInfo: channelOrGroupInfo,
				Message:            msg.Message,
			})

			filled := len(filterEmptyMessages(m.Conversations))
			if filled < len(m.Conversations) {
				m.Conversations[filled] = formattedMessage
			} else {
				copy(m.Conversations[:], m.Conversations[1:])
				m.Conversations[len(m.Conversations)-1] = formattedMessage
			}
			m.updateConversations()
			return m, nil
		}

		if groupIndex := getGroupIndex(m, *channelOrGroupInfo); groupIndex != -1 && !msg.Message.Out {
			group := m.Groups.Items()[groupIndex].(types.ChannelInfo)
			group.UnreadCount++
			m.Groups.SetItem(groupIndex, group)
		}
		return m, nil
	}

	if userInfo == nil || (m.Mode != ModeUsers && m.Mode != ModeBots) {
		return m, nil
	}

	if m.SelectedUser.PeerID != userInfo.PeerID && !msg.Message.Out {
		l := listForUser(&m, *userInfo)
		if userIndex := getUserIndex(*l, *userInfo); userIndex != -1 {
			user := l.Items()[userIndex].(types.UserInfo)
			user.UnreadCount++
			return m, l.SetItem(userIndex, user)
		}
		return m, nil
	}

	chatType := types.UserChat
	if userInfo.IsBot {
		chatType = types.BotChat
	}

	if !msg.Message.Out {
		m.SelectedUser.UnreadCount++
	}

	formattedMessage := getFormattedMessageFunc(GetFormattedMessageArg{
		ChatType: chatType,
		UserInfo: userInfo,
		Message:  msg.Message,
	})

	filled := len(filterEmptyMessages(m.Conversations))
	if filled < len(m.Conversations) {
		m.Conversations[filled] = formattedMessage
	} else {
		copy(m.Conversations[:], m.Conversations[1:])
		m.Conversations[len(m.Conversations)-1] = formattedMessage
	}
	m.updateConversations()
	return m, nil
}

type GetFormattedMessageArg struct {
	ChatType           types.ChatType
	ChannelOrGroupInfo *types.ChannelInfo
	UserInfo           *types.UserInfo
	Message            *tg.Message
}

func getFormattedMessageFunc(arg GetFormattedMessageArg) types.FormattedMessage {
	var sender string
	var fromID *string

	if (arg.ChatType == types.UserChat || arg.ChatType == types.BotChat) && arg.UserInfo != nil {
		sender = arg.UserInfo.FirstName
		fromID = &arg.UserInfo.PeerID
	} else if arg.ChannelOrGroupInfo != nil {
		sender = arg.ChannelOrGroupInfo.ChannelTitle
		fromID = &arg.ChannelOrGroupInfo.ID
	}

	if arg.Message.Out {
		sender = "You"
	}

	var media *string
	if arg.Message.Media != nil {
		mediaStr := fmt.Sprintf("%T", arg.Message.Media)
		media = &mediaStr
	}

	return types.FormattedMessage{
		ID:                   arg.Message.ID,
		Sender:               sender,
		Content:              arg.Message.Message,
		IsFromMe:             arg.Message.GetOut(),
		Media:                media,
		IsUnsupportedMessage: media != nil,
		Date:                 time.Unix(int64(arg.Message.Date), 0),
		FromID:               fromID,
		ReplyTo:              nil,
		SenderUserInfo:       arg.UserInfo,
	}
}

func (m Model) handleUserOnlineOffline(msg types.UserStatusNotification) (tea.Model, tea.Cmd) {
	var user types.UserInfo
	for _, v := range m.Users.Items() {
		if u, ok := v.(types.UserInfo); ok && u.PeerID == msg.UserInfo.PeerID {
			user = u
			break
		}
	}
	l := listForUser(&m, user)
	userIndex := getUserIndex(*l, user)
	if userIndex != -1 {
		items := l.Items()
		u := items[userIndex].(types.UserInfo)
		u.IsOnline = msg.Status.IsOnline
		items[userIndex] = u
		return m, l.SetItems(items)
	}
	return m, nil
}

func (m Model) handleMessageDeletion(msg MessageDeletionConfrimResponseMsg) (tea.Model, tea.Cmd) {
	if !msg.yes {
		return m, nil
	}
	peer := getMessageParams(&m)
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

func (m Model) handleGetMessages(msg types.GetMessagesMsg) (tea.Model, tea.Cmd) {
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
		return m, nil
	}

	m.Conversations = m.mergeConversations(msg.Messages, messagesWeGot)
	m.updateConversations()
	m.ChatUI.Select(len(m.Conversations) - 1)
	return m, nil
}

func (m Model) mergeConversations(newMessages [50]types.FormattedMessage, messagesWeGot int) [50]types.FormattedMessage {
	if messagesWeGot >= 50 {
		return newMessages
	}
	var oldMessages []types.FormattedMessage
	for _, v := range m.ChatUI.Items() {
		if msg, ok := v.(types.FormattedMessage); ok && msg.ID != 0 {
			oldMessages = append(oldMessages, msg)
		}
	}
	var updatedConversations [50]types.FormattedMessage
	if (messagesWeGot + len(oldMessages)) <= 50 {
		updatedConversations = newMessages
	} else {
		take := 50 - messagesWeGot
		combined := append(m.Conversations[:take], newMessages[:]...)
		copy(updatedConversations[:], combined)
	}
	return updatedConversations
}

func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg.String() {
	case "shift+down":
		if m.FocusedOn == Mainview {
			m.ChatUI.Select(len(m.ChatUI.Items()) - 1)
		}
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
		if m.FocusedOn != Input {
			m.IsModalVisible = true
		}
		return m, nil
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
	case "alt+s":
		return m, func() tea.Msg { return m.Stories }
	}
	cmds = append(cmds, SendUserIsTyping(&m))
	return m, tea.Batch(cmds...)
}

func (m Model) handleListPagination() (Model, tea.Cmd) {
	if m.Users.Index() < len(m.Users.VisibleItems())-6 {
		return m, nil
	}
	if m.OffsetDate == -1 || m.OffsetID == -1 || m.OnPagination {
		return m, nil
	}
	m.OnPagination = true
	if m.Mode == ModeUsers || m.Mode == ModeBots {
		return m, telegram.Cligram.GetUserChats(telegram.Cligram.Context(), types.ChatType(m.Mode), m.OffsetDate, m.OffsetID)
	}
	return m, telegram.Cligram.GetUserChannels(telegram.Cligram.Context(), m.Mode == ModeChannels, m.OffsetDate, m.OffsetID)
}

func (m Model) handleEditKey() (tea.Model, tea.Cmd) {
	if m.FocusedOn == Mainview {
		if selectedItem, ok := m.ChatUI.SelectedItem().(types.FormattedMessage); ok && strings.ToLower(selectedItem.Sender) == "you" {
			m.FocusedOn = Input
			m.Input.SetValue(selectedItem.Content)
			m.EditMessage = &selectedItem
			m.SkipNextInput = true
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
		}
	}
	return m, nil
}

func (m Model) handleDeleteKey() (tea.Model, tea.Cmd) {
	if m.FocusedOn == Mainview {
		selectedItem := m.ChatUI.SelectedItem().(types.FormattedMessage)
		return m, func() tea.Msg {
			return OpenModalMsg{ModalMode: ModalModeDeleteMessage, Message: &selectedItem}
		}
	}
	return m, nil
}

func (m Model) handleForwardKey() (tea.Model, tea.Cmd) {
	if m.FocusedOn != Mainview {
		return m, nil
	}
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

// appendListItems appends items with a non-empty title to the list, returning the SetItems cmd.
func appendListItems[T types.FilterableItem](l *list.Model, items []T) tea.Cmd {
	current := l.Items()
	for _, it := range items {
		if it.FilterValue() != "" {
			current = append(current, it)
		}
	}
	return l.SetItems(current)
}

func (m Model) handleUserChats(msg types.UserChatsMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		m.IsModalVisible = true
		m.ModalContent = GetModalContent(msg.Err.Error())
		return m, nil
	}
	if msg.Response == nil {
		return m, nil
	}
	cmd := appendListItems(&m.Users, msg.Response.Data)
	m.OffsetDate = msg.Response.OffsetDate
	m.OffsetID = msg.Response.OffsetID
	m.OnPagination = false
	return m, cmd
}

func (m Model) handleUserChannels(msg types.ChannelsMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		m.IsModalVisible = true
		m.ModalContent = GetModalContent(msg.Err.Error())
		return m, nil
	}
	if msg.Response == nil {
		return m, nil
	}
	cmd := appendListItems(&m.Channels, msg.Response.Data)
	m.OffsetDate = msg.Response.OffsetDate
	m.OffsetID = msg.Response.OffsetID
	m.OnPagination = false
	return m, cmd
}

func (m Model) handleUserGroups(msg types.GroupsMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		m.IsModalVisible = true
		m.ModalContent = GetModalContent(msg.Err.Error())
		return m, nil
	}
	if msg.Response == nil {
		return m, nil
	}
	cmd := appendListItems(&m.Groups, msg.Response.Data)
	m.OffsetDate = msg.Response.OffsetDate
	m.OffsetID = msg.Response.OffsetID
	m.OnPagination = false
	return m, cmd
}

func (m Model) handleForwardMessage(msg ForwardMsg) (tea.Model, tea.Cmd) {
	from, toPeer := extractPeerInfo(*msg.fromPeer, *msg.receiver)
	err := telegram.Cligram.ForwardMessages(telegram.Cligram.Context(), types.ForwardMessagesRequest{
		FromPeer:   from,
		ToPeer:     toPeer,
		MessageIDs: []int{int(msg.msg.ID)},
	})
	if err != nil {
		slog.Error("Failed to forward message", "error", err.Error())
	}
	return m, nil
}

// peerFromItem converts a list.Item (UserInfo or ChannelInfo, pointer or value) into a types.Peer.
func peerFromItem(item list.Item) types.Peer {
	switch p := item.(type) {
	case types.UserInfo:
		return types.Peer{ID: p.PeerID, AccessHash: p.AccessHash, ChatType: types.UserChat}
	case *types.UserInfo:
		if p != nil {
			return types.Peer{ID: p.PeerID, AccessHash: p.AccessHash, ChatType: types.UserChat}
		}
	case types.ChannelInfo:
		chatType := types.GroupChat
		if p.IsBroadcast {
			chatType = types.ChannelChat
		}
		return types.Peer{ID: p.ID, AccessHash: p.AccessHash, ChatType: chatType}
	case *types.ChannelInfo:
		if p != nil {
			chatType := types.GroupChat
			if p.IsBroadcast {
				chatType = types.ChannelChat
			}
			return types.Peer{ID: p.ID, AccessHash: p.AccessHash, ChatType: chatType}
		}
	}
	return types.Peer{}
}

func extractPeerInfo(fromPeer, receiver list.Item) (from, toPeer types.Peer) {
	return peerFromItem(fromPeer), peerFromItem(receiver)
}

func (m Model) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.Width = msg.Width
	m.Height = msg.Height
	headerHeight := 7
	footerHeight := 7
	m.viewport = viewport.New(msg.Width, msg.Height-(headerHeight+footerHeight))
	m.viewport.YPosition = headerHeight
	m.updateConversations()
	return m, nil
}

func (m Model) handleSearchedUserResult(msg SelectSearchedUserResult) (tea.Model, tea.Cmd) {
	if msg.user != nil || msg.Bot != nil {
		userInfo := msg.user
		if userInfo == nil {
			userInfo = msg.Bot
		}
		return m.handleSearchedUser(*userInfo)
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
	l := listForUser(&m, user)
	targetMode := ModeUsers
	if user.IsBot {
		targetMode = ModeBots
	}

	index := getUserIndex(*l, user)
	if index != -1 {
		l.Select(index)
		m.FocusedOn = SideBar
		m.Mode = targetMode
		return handleUserChange(&m)
	}

	updateUserCmd := l.SetItems(append(l.Items(), user))
	index = getUserIndex(*l, user)
	if index != -1 {
		l.Select(index)
		m.FocusedOn = SideBar
		m.Mode = targetMode
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

	setItemsCmd := m.Channels.SetItems(append(m.Channels.Items(), channel))
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

	setItemsCmd := m.Groups.SetItems(append(m.Groups.Items(), group))
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

func listForUser(m *Model, user types.UserInfo) *list.Model {
	if user.IsBot {
		return &m.Bots
	}
	return &m.Users
}
