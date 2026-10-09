package ui

import (
	"fmt"
	"log/slog"
	"os/exec"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gotd/td/tg"
	"github.com/kumneger0/cligram/internal/notification"
	"github.com/kumneger0/cligram/internal/telegram"
	"github.com/kumneger0/cligram/internal/telegram/client"
	"github.com/kumneger0/cligram/internal/telegram/types"
	"go.dalton.dog/bubbleup"
)

func (m *Model) checkAndFetchCustomEmojis(messages []types.FormattedMessage) tea.Cmd {
	var cmds []tea.Cmd
	if m.CustomEmojis == nil {
		m.CustomEmojis = make(map[int64]*tg.Document)
	}

	for _, msg := range messages {
		if msg.Reactions != nil {
			for _, r := range msg.Reactions.Results {
				if reaction, ok := r.Reaction.(*tg.ReactionCustomEmoji); ok {
					documentID := reaction.DocumentID
					if _, found := m.CustomEmojis[documentID]; !found {
						cmds = append(cmds, FetchCustomEmojiDocumentCmd(documentID))
					}
				}
			}
		}
	}
	return tea.Batch(cmds...)
}

type CallTickMsg time.Time

func tickCall() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return CallTickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case CallTickMsg:
		if m.CallOverlay.State == CallOverlayActive {
			tTime := time.Time(msg)
			if !tTime.IsZero() && !m.CallOverlay.StartTime.IsZero() && tTime.After(m.CallOverlay.StartTime) {
				m.CallOverlay.ElapsedSeconds = int(tTime.Sub(m.CallOverlay.StartTime).Seconds())
			} else {
				m.CallOverlay.ElapsedSeconds++
			}
			return m, tickCall()
		}
		return m, nil
	case types.GetUserDHConfigRequest:
		if m.PhoneCallDhConfigs == nil || m.PhoneCallDhConfigs.Configs[msg.UserID] == nil {
			slog.Warn("GetUserDHConfigRequest: no DHConfig stored for user", "userID", msg.UserID)
			return m, nil
		}
		dhConfig := *m.PhoneCallDhConfigs.Configs[msg.UserID]
		go func() { client.DhConfigRecvChannel <- dhConfig }()
		return m, nil
	case types.CallUserResponse:
		if msg.Err != nil {
			slog.Error("call initiation failed", "error", msg.Err)
			m.CallOverlay.SetNone()
			m.ActiveCallUserID = 0
			alertCmd := m.Alert.NewAlertCmd(bubbleup.ErrorKey, "Call failed: "+msg.Err.Error())
			return m, alertCmd
		}
		slog.Info("call initiated", "userID", msg.UserID)
		if m.CallOverlay.State == CallOverlayDialing && msg.UserID != nil {
			userName := m.CallOverlay.UserName
			if userName == "" && (m.Mode == ModeUsers || m.Mode == ModeAll) {
				userName = m.SelectedUser.FirstName
			}
			m.CallOverlay.SetActive(userName, *msg.UserID)
			m.ActiveCallUserID = *msg.UserID
			return m, tickCall()
		}
		return m, nil
	case types.CallNotification:
		switch msg.State {
		case types.CallStateIncoming:
			m.CallOverlay.SetIncoming(msg.UserName, msg.UserID)
			return m, tea.Printf("\a")
		case types.CallStateActive:
			userName := msg.UserName
			if (userName == "" || strings.HasPrefix(userName, "User ")) && m.CallOverlay.UserName != "" {
				userName = m.CallOverlay.UserName
			}
			if (userName == "" || strings.HasPrefix(userName, "User ")) && (m.Mode == ModeUsers || m.Mode == ModeAll) && m.SelectedUser.FirstName != "" {
				userName = m.SelectedUser.FirstName
			}
			m.CallOverlay.SetActive(userName, msg.UserID)
			m.CallOverlay.SetRelay(msg.IsRelay)
			m.ActiveCallUserID = msg.UserID
			return m, tickCall()
		case types.CallStateMissingHelper:
			m.CallOverlay.SetMissingHelper()
			alertCmd := m.Alert.NewAlertCmd(bubbleup.ErrorKey, "⚠️ cligram-voip helper missing")
			return m, alertCmd
		case types.CallStateBusyMissed:
			alertCmd := m.Alert.NewAlertCmd(bubbleup.InfoKey, "📞 Missed call (line busy)")
			return m, alertCmd
		case types.CallStateEnded, types.CallStateDeclined:
			m.CallOverlay.SetNone()
			m.ActiveCallUserID = 0
			if msg.Err != nil {
				alertCmd := m.Alert.NewAlertCmd(bubbleup.ErrorKey, "📞 Call disconnected: "+msg.Err.Error())
				return m, alertCmd
			}
			alertCmd := m.Alert.NewAlertCmd(bubbleup.InfoKey, "📞 Call ended")
			return m, alertCmd
		}
		return m, nil
	case types.CallAcceptedMsg:
		if msg.Err != nil {
			slog.Error("failed to accept call", "error", msg.Err)
			m.CallOverlay.SetNone()
			m.ActiveCallUserID = 0
			alertCmd := m.Alert.NewAlertCmd(bubbleup.ErrorKey, "Failed to accept call: "+msg.Err.Error())
			return m, alertCmd
		}
		return m, nil
	case types.CallDeclinedMsg:
		m.CallOverlay.SetNone()
		m.ActiveCallUserID = 0
		if msg.Err != nil {
			slog.Error("failed to decline call", "error", msg.Err)
			alertCmd := m.Alert.NewAlertCmd(bubbleup.ErrorKey, "Failed to decline call: "+msg.Err.Error())
			return m, alertCmd
		}
		return m, nil
	case types.CallHungUpMsg:
		m.CallOverlay.SetNone()
		m.ActiveCallUserID = 0
		if msg.Err != nil {
			slog.Error("failed to hang up call", "error", msg.Err)
			alertCmd := m.Alert.NewAlertCmd(bubbleup.ErrorKey, "Failed to hang up call: "+msg.Err.Error())
			return m, alertCmd
		}
		return m, nil
	case types.CallMuteToggledMsg:
		if msg.Err != nil {
			slog.Error("failed to toggle mute", "error", msg.Err)
			alertCmd := m.Alert.NewAlertCmd(bubbleup.ErrorKey, "Mute toggle failed: "+msg.Err.Error())
			return m, alertCmd
		}
		m.CallOverlay.Muted = msg.Muted
		return m, nil
	case types.GetAllChatsResponseMSG:
		if msg.Err != nil {
			slog.Error("Failed to get all chats", "error", msg.Err.Error())
			m.SideBarLoading = false
			m.AllLoading = false
			m.UsersLoading = false
			m.BotsLoading = false
			m.ChannelsLoading = false
			m.GroupsLoading = false
			m.OnPagination = false
			m.Alert = m.Alert.WithAllowEscToClose().WithPosition(bubbleup.TopLeftPosition)
			alertCmd := m.Alert.NewAlertCmd(bubbleup.ErrorKey, msg.Err.Error())
			return m, alertCmd
		}
		m.OffsetDate = msg.Chats.OffsetDate
		m.OffsetID = msg.Chats.OffsetID
		m.SideBarLoading = false
		m.OnPagination = false
		m.AllLoading = false
		m.AllLoaded = true
		m.UsersLoading = false
		m.UsersLoaded = true
		m.BotsLoading = false
		m.BotsLoaded = true
		m.ChannelsLoading = false
		m.ChannelsLoaded = true
		m.GroupsLoading = false
		m.GroupsLoaded = true

		var allItems []types.FilterableItem
		if len(msg.Chats.All) > 0 {
			allItems = msg.Chats.All
		} else {
			for _, u := range msg.Chats.PrivateChats {
				allItems = append(allItems, u)
			}
			for _, b := range msg.Chats.Bots {
				allItems = append(allItems, b)
			}
			for _, g := range msg.Chats.Groups {
				allItems = append(allItems, g)
			}
			for _, c := range msg.Chats.Channels {
				allItems = append(allItems, c)
			}
		}
		cmdAll := appendListItems(&m.All, allItems)

		var usersList []types.UserInfo
		var botsList []types.UserInfo
		if len(msg.Chats.Bots) > 0 {
			usersList = msg.Chats.PrivateChats
			botsList = msg.Chats.Bots
		} else {
			for _, u := range msg.Chats.PrivateChats {
				if u.IsBot {
					botsList = append(botsList, u)
				} else {
					usersList = append(usersList, u)
				}
			}
		}

		cmdUsers := appendListItems(&m.Users, usersList)
		cmdBots := appendListItems(&m.Bots, botsList)
		cmdChannels := appendListItems(&m.Channels, msg.Chats.Channels)
		cmdGroups := appendListItems(&m.Groups, msg.Chats.Groups)

		return m, tea.Batch(cmdAll, cmdUsers, cmdBots, cmdChannels, cmdGroups)

	case types.GetChannelForumsResponseMsg:
		m.ForumTopicLoading = false
		m.MainViewLoading = false
		if msg.Err != nil {
			slog.Error("Failed to get channel forums", "error", msg.Err.Error())
			m.Alert = m.Alert.WithAllowEscToClose().WithPosition(bubbleup.TopLeftPosition)
			alertCmd := m.Alert.NewAlertCmd(bubbleup.ErrorKey, msg.Err.Error())
			return m, alertCmd
		}
		var forumTopicList []list.Item
		for _, forum := range msg.Forums {
			forumTopicList = append(forumTopicList, forum)
		}
		m.SelectedGroupForumTopics = list.New(forumTopicList, list.NewDefaultDelegate(), 0, 0)
		m.ShowForumTopics = true
		m.SelectedForumTopic = nil
		m.FocusedOn = Main
	case types.ShouldHighlightSpecificMessageMsg:
		items := m.ChatUI.Items()
		for i, item := range items {
			if formattedMessage, ok := item.(types.FormattedMessage); ok {
				if formattedMessage.ID == msg.MessageID {
					m.ChatUI.Select(i)
				}
			} else {
				slog.Error("Failed to Cast Item to types.Formatted message")
			}
		}
	case types.GetEntityInfoMsg:
		if msg.Err != nil {
			slog.Error("Failed to get entity info", "error", msg.Err.Error())
			m.Alert = m.Alert.WithAllowEscToClose().WithPosition(bubbleup.TopLeftPosition)
			alertCmd := m.Alert.NewAlertCmd(bubbleup.ErrorKey, msg.Err.Error())
			return m, alertCmd
		}
		info := msg.Response
		if info.User != nil {
			m.Mode = ModeUsers
			if info.User.IsBot {
				m.Mode = ModeBots
				if userIndex := getUserIndex(m.Bots, *info.User); userIndex != -1 {
					m.Bots.Select(userIndex)
					m.SelectedUser = m.Bots.Items()[userIndex].(types.UserInfo)
				} else {
					items := append(m.Bots.Items(), *info.User)
					m.Bots = list.New(items, list.NewDefaultDelegate(), 0, 0)
					m.SelectedUser = *info.User
				}
			} else {
				if userIndex := getUserIndex(m.Users, *info.User); userIndex != -1 {
					m.Users.Select(userIndex)
					m.SelectedUser = m.Users.Items()[userIndex].(types.UserInfo)
				} else {
					items := append(m.Users.Items(), *info.User)
					m.Users = list.New(items, list.NewDefaultDelegate(), 0, 0)
					m.SelectedUser = *info.User
				}
			}
		}
		if info.Channel != nil {
			m.Mode = ModeChannels
			if !info.Channel.IsBroadcast {
				m.Mode = ModeGroups
				if groupIndex := getGroupIndex(m, *info.Channel); groupIndex != -1 {
					m.Groups.Select(groupIndex)
					m.SelectedGroup = m.Groups.Items()[groupIndex].(types.ChannelInfo)
				} else {
					items := append(m.Groups.Items(), *info.Channel)
					m.Groups = list.New(items, list.NewDefaultDelegate(), 0, 0)
					m.Groups.Select(len(m.Groups.Items()) - 1)
					m.SelectedGroup = *info.Channel
				}
			} else {
				if channelIndex := getChannelIndex(m, *info.Channel); channelIndex != -1 {
					m.Channels.Select(channelIndex)
					m.SelectedChannel = m.Channels.Items()[channelIndex].(types.ChannelInfo)
				} else {
					items := append(m.Channels.Items(), *info.Channel)
					m.Channels = list.New(items, list.NewDefaultDelegate(), 0, 0)
					m.Channels.Select(len(m.Channels.Items()) - 1)
					m.SelectedChannel = *info.Channel
				}
			}
		}
		if info.Group != nil {
			m.Mode = ModeGroups
			if groupIndex := getGroupIndex(m, *info.Group); groupIndex != -1 {
				m.Groups.Select(groupIndex)
				m.SelectedGroup = m.Groups.Items()[groupIndex].(types.ChannelInfo)
			} else {
				items := append(m.Groups.Items(), *info.Group)
				m.Groups = list.New(items, list.NewDefaultDelegate(), 0, 0)
				m.Groups.Select(len(m.Groups.Items()) - 1)
				m.SelectedGroup = *info.Group
			}
		}
		var offsetID *int
		isForum := (info.Channel != nil && info.Channel.IsForum) || (info.Group != nil && info.Group.IsForum)
		if !isForum && len(msg.MessageIDs) > 0 {
			if id, err := strconv.ParseInt(msg.MessageIDs[0], 10, 64); err == nil {
				offsetIDInt := int(id)
				offsetID = &offsetIDInt
			}
		}

		highlightTheSelectedMessageCmd := func() tea.Msg {
			if offsetID == nil {
				return nil
			}
			return types.ShouldHighlightSpecificMessageMsg{
				MessageID: *offsetID,
			}
		}
		model, cmd := handleUserChange(&m, offsetID, highlightTheSelectedMessageCmd)
		m = model
		cmds = append(cmds, cmd)
	case types.OpenNewChatWithPeerMsg:
		return m, telegram.Cligram.GetEntityInfo(msg.Chat)
	case types.SendMessageMsg:
		if msg.Err != nil {
			slog.Error("Failed to send message", "error", msg.Err.Error())
			m.IsModalVisible = true
			m.ModalContent = GetModalContent(msg.Err.Error())
			var updatedConversations []types.FormattedMessage
			removed := false
			for _, v := range m.Conversations {
				if v.ID != msg.RandID {
					updatedConversations = append(updatedConversations, v)
				} else {
					removed = true
				}
			}
			if removed {
				m.Conversations = updatedConversations
				cmds = append(cmds, m.updateConversations())
			}

			m.Alert = m.Alert.WithAllowEscToClose().WithPosition(bubbleup.TopLeftPosition)
			alertCmd := m.Alert.NewAlertCmd(bubbleup.ErrorKey, msg.Err.Error())
			cmds = append(cmds, alertCmd)
		} else if msg.Response != nil && msg.Response.MessageID != nil {
			found := false
			for i, conv := range m.Conversations {
				if conv.ID == msg.RandID {
					m.Conversations[i].ID = *msg.Response.MessageID
					found = true
					break
				}
			}
			if found {
				cmds = append(cmds, m.updateConversations())
			}
		}
		if m.SelectedFile == "uploading..." {
			m.SelectedFile = ""
		}
		return m, tea.Batch(cmds...)
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
	case types.ReadHistoryOutboxNotification:
		if msg.MaxID <= 0 {
			return m, nil
		}
		model, cmd := m.handleReadHistoryOutbox(msg)
		m = model.(Model)
		return m, cmd
	case types.UserStatusNotification:
		model, cmd := m.handleUserOnlineOffline(msg)
		m = model.(Model)
		cmds = append(cmds, cmd)
	case MessageDeletionConfirmResponseMsg:
		model, cmd := m.handleMessageDeletion(msg)
		m = model.(Model)
		cmds = append(cmds, cmd)
	case types.GetMessagesMsg:
		model, cmd := m.handleGetMessages(msg)
		m = model.(Model)
		cmds = append(cmds, cmd)
	case spinner.TickMsg:
		m.ensureSidebarSpinner()
		var spinCmd tea.Cmd
		m.SidebarSpinner, spinCmd = m.SidebarSpinner.Update(msg)
		if m.isAnyCategoryLoading() {
			cmds = append(cmds, spinCmd)
		}
		return m, tea.Batch(cmds...)
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
	case types.SendReactionResponseMsg:
		if msg.Err != nil {
			slog.Error("Failed to send reaction", "error", msg.Err.Error())
			return m, nil
		}
		if msg.Response {
			pInfo := getMessageParams(&m)
			return m, telegram.Cligram.GetSingleMessage(telegram.Cligram.Context(), pInfo, msg.MessageID)
		}
	case types.SingleMessageMsg:
		if msg.Err != nil {
			slog.Error("Failed to fetch message", "error", msg.Err.Error())
			return m, nil
		}

		isValid := false
		if msg.Message != nil && msg.Message.PeerID != nil {
			switch m.Mode {
			case ModeUsers, ModeBots:
				isValid = *msg.Message.PeerID == m.SelectedUser.PeerID
			case ModeChannels:
				isValid = *msg.Message.PeerID == m.SelectedChannel.ID
			case ModeGroups:
				isValid = *msg.Message.PeerID == m.SelectedGroup.ID
			}
		}

		if isValid {
			for i, conv := range m.Conversations {
				if conv.ID == msg.Message.ID {
					m.Conversations[i] = *msg.Message
					cmds = append(cmds, m.updateConversations())
					break
				}
			}
		}

	case types.SendReactionMsg:
		var peer types.Peer
		if m.SelectedUser.PeerID != "" {
			peer = peerFromItem(m.SelectedUser)
			message := m.SelectedMessage()
			if message != nil {
				alreadyReacted := false
				if message.Reactions != nil {
					for _, r := range message.Reactions.Results {
						if _, ok := r.GetChosenOrder(); ok {
							if reaction, ok := r.Reaction.(*tg.ReactionEmoji); ok && reaction.Emoticon == msg.Reaction.Reaction {
								alreadyReacted = true
								break
							}
						}
					}
				}

				cmds = append(cmds, telegram.Cligram.SendReaction(telegram.Cligram.Context(), types.SendReactionRequest{
					Peer:      peer,
					MessageID: message.ID,
					Emoticon:  msg.Reaction.Reaction,
					Remove:    alreadyReacted,
				}))
			}
		}
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
	case types.CurrentUserMsg:
		if msg.Err != nil {
			slog.Error("Failed to get current user", "error", msg.Err.Error())
			return m, nil
		}
		m.CurrentUser = msg.User
	case CustomEmojiDocumentMsg:
		if msg.Err != nil {
			slog.Error("Failed to fetch custom emoji document", "error", msg.Err, "document_id", msg.DocumentID)
			return m, nil
		}
		if m.CustomEmojis == nil {
			m.CustomEmojis = make(map[int64]*tg.Document)
		}
		m.CustomEmojis[msg.DocumentID] = msg.Document
		return m, nil
	}
	outAlert, outCmd := m.Alert.Update(msg)
	m.Alert = outAlert.(bubbleup.AlertModel)

	cmds = append(cmds, outCmd)
	return updateFocusedComponent(&m, msg, &cmds)
}

func (m Model) updateUserStories(msg types.GetAllStoriesMsg) (tea.Model, tea.Cmd) {
	m.Stories = msg.Stories
	m.StoriesLoading = false
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

func (m Model) handleReadHistoryOutbox(msg types.ReadHistoryOutboxNotification) (tea.Model, tea.Cmd) {
	peerID := msg.PeerID
	var userInfo *types.UserInfo
	for _, v := range slices.Concat(m.Users.Items(), m.Bots.Items()) {
		if user, ok := v.(types.UserInfo); ok && user.PeerID == peerID {
			userInfo = &user
			break
		}
	}

	if userInfo != nil {
		var listToUpdate *list.Model
		if userInfo.IsBot {
			listToUpdate = &m.Bots
		} else {
			listToUpdate = &m.Users
		}

		userIndex := getUserIndex(*listToUpdate, *userInfo)
		if userIndex != -1 {
			updatedUser := *userInfo
			updatedUser.ReadOutboxMaxID = msg.MaxID
			cmd := listToUpdate.SetItem(userIndex, updatedUser)
			if m.SelectedUser.PeerID == updatedUser.PeerID {
				m.SelectedUser = updatedUser
			}
			return m, tea.Batch(cmd, m.updateConversations())
		}
	}

	var groupInfo *types.ChannelInfo
	for _, v := range m.Groups.Items() {
		if cg, ok := v.(types.ChannelInfo); ok && cg.ID == peerID {
			groupInfo = &cg
			break
		}
	}

	if groupInfo != nil {
		groupIndex := getGroupIndex(m, *groupInfo)
		if groupIndex != -1 {
			updatedGroup := *groupInfo
			updatedGroup.ReadOutboxMaxID = msg.MaxID
			cmd := m.Groups.SetItem(groupIndex, updatedGroup)
			if m.SelectedGroup.ID == updatedGroup.ID {
				m.SelectedGroup = updatedGroup
			}
			return m, tea.Batch(cmd, m.updateConversations())
		}
	}
	return m, nil
}

func (m Model) handleNewMessage(msg types.NewMessageNotification) (tea.Model, tea.Cmd) {
	peerID := msg.FromID
	if msg.PeerID != "" {
		peerID = msg.PeerID
	}

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

	currentPeerID := m.CurrentPeerID()
	// If message does not belong to currently open chat:
	if currentPeerID == "" || peerID != currentPeerID {
		if !msg.Message.Out {
			var unreadCmds []tea.Cmd
			if userInfo != nil {
				l := listForUser(&m, *userInfo)
				if userIndex := getUserIndex(*l, *userInfo); userIndex != -1 {
					user := l.Items()[userIndex].(types.UserInfo)
					user.UnreadCount++
					unreadCmds = append(unreadCmds, l.SetItem(userIndex, user))
				}
				for allIdx, item := range m.All.Items() {
					if u, ok := item.(types.UserInfo); ok && u.PeerID == userInfo.PeerID {
						u.UnreadCount++
						unreadCmds = append(unreadCmds, m.All.SetItem(allIdx, u))
						break
					}
				}
			}
			if channelOrGroupInfo != nil {
				if groupIndex := getGroupIndex(m, *channelOrGroupInfo); groupIndex != -1 {
					group := m.Groups.Items()[groupIndex].(types.ChannelInfo)
					group.UnreadCount++
					unreadCmds = append(unreadCmds, m.Groups.SetItem(groupIndex, group))
				}
				if channelIndex := getChannelIndex(m, *channelOrGroupInfo); channelIndex != -1 {
					channel := m.Channels.Items()[channelIndex].(types.ChannelInfo)
					channel.UnreadCount++
					unreadCmds = append(unreadCmds, m.Channels.SetItem(channelIndex, channel))
				}
				for allIdx, item := range m.All.Items() {
					if cg, ok := item.(types.ChannelInfo); ok && cg.ID == channelOrGroupInfo.ID {
						cg.UnreadCount++
						unreadCmds = append(unreadCmds, m.All.SetItem(allIdx, cg))
						break
					}
				}
			}
			return m, tea.Batch(unreadCmds...)
		}
		return m, nil
	}

	// Message belongs to currently open chat!
	var chatType types.ChatType
	if channelOrGroupInfo != nil {
		if channelOrGroupInfo.IsBroadcast {
			chatType = types.ChannelChat
		} else {
			chatType = types.GroupChat
		}
	} else if userInfo != nil {
		if userInfo.IsBot {
			chatType = types.BotChat
		} else {
			chatType = types.UserChat
		}
	} else {
		chatType = types.UserChat
	}

	if !msg.Message.Out && userInfo != nil {
		m.SelectedUser.UnreadCount++
	}

	// Check if this message was already added (e.g. optimistic send or duplicate update)
	alreadyPresent := false
	for i, c := range m.Conversations {
		if c.ID == msg.Message.ID || (msg.Message.Out && c.IsFromMe && c.Content == msg.Message.Message) {
			m.Conversations[i].ID = msg.Message.ID
			alreadyPresent = true
			break
		}
	}

	if !alreadyPresent {
		formattedMessage := getFormattedMessageFunc(GetFormattedMessageArg{
			ChatType:           chatType,
			ChannelOrGroupInfo: channelOrGroupInfo,
			UserInfo:           userInfo,
			Message:            msg.Message,
		})
		formattedMessage.PeerID = &peerID

		wasAtBottom := m.viewport.AtBottom()
		m.Conversations = append(m.Conversations, formattedMessage)
		if len(m.Conversations) > 250 {
			m.Conversations = m.Conversations[len(m.Conversations)-250:]
		}
		if wasAtBottom {
			m.SelectedMessageIndex = len(m.Conversations) - 1
			m.viewport.GotoBottom()
		} else if m.SelectedMessageIndex >= len(m.Conversations) {
			m.SelectedMessageIndex = len(m.Conversations) - 1
		}
	}

	cmd := m.updateConversations()
	fetchCmd := m.checkAndFetchCustomEmojis(m.Conversations)
	return m, tea.Batch(fetchCmd, cmd)
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

func (m Model) handleMessageDeletion(msg MessageDeletionConfirmResponseMsg) (tea.Model, tea.Cmd) {
	if !msg.yes {
		return m, nil
	}
	peer := getMessageParams(&m)
	var selectedID int
	if msgPtr := m.SelectedMessage(); msgPtr != nil {
		selectedID = msgPtr.ID
	}
	if selectedID == 0 {
		return m, nil
	}
	response, err := telegram.Cligram.DeleteMessage(telegram.Cligram.Context(), types.DeleteMessageRequest{
		Peer:      peer,
		MessageID: selectedID,
	})
	if err != nil {
		m.IsModalVisible = true
		m.ModalContent = GetModalContent(err.Error())
		return m, nil
	}
	if response.Status == "success" {
		var updatedConversations []types.FormattedMessage
		for _, v := range m.Conversations {
			if v.ID != selectedID {
				updatedConversations = append(updatedConversations, v)
			}
		}
		m.Conversations = updatedConversations
		if m.SelectedMessageIndex >= len(m.Conversations) {
			m.SelectedMessageIndex = len(m.Conversations) - 1
		}
		cmd := m.updateConversations()
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

	currentPeer := m.CurrentPeerID()
	if msg.PeerID != "" && currentPeer != "" && msg.PeerID != currentPeer {
		slog.Debug("discarding stale GetMessagesMsg for different peer", "msgPeer", msg.PeerID, "currentPeer", currentPeer)
		return m, nil
	}

	nonEmpty := filterEmptyMessages(msg.Messages[:])
	if len(nonEmpty) < 1 {
		if selectedChat, ok := m.Users.SelectedItem().(types.UserInfo); ok && selectedChat.IsBot {
			m.Input.SetValue("/start")
		}
		return m, nil
	}

	wasAtBottom := m.viewport.AtBottom() || len(m.Conversations) == 0
	m.Conversations = m.mergeConversations(nonEmpty)
	cmd := m.updateConversations()

	if wasAtBottom {
		m.SelectedMessageIndex = len(m.Conversations) - 1
		m.viewport.GotoBottom()
	} else if m.SelectedMessageIndex >= len(m.Conversations) {
		m.SelectedMessageIndex = len(m.Conversations) - 1
	}

	fetchCmd := m.checkAndFetchCustomEmojis(filterEmptyMessages(m.Conversations))
	return m, tea.Batch(fetchCmd, cmd)
}

func (m Model) mergeConversations(newMessages []types.FormattedMessage) []types.FormattedMessage {
	seen := make(map[int]bool, len(m.Conversations)+len(newMessages))
	var combined []types.FormattedMessage

	for _, msg := range m.Conversations {
		if msg.ID != 0 && !seen[msg.ID] {
			seen[msg.ID] = true
			combined = append(combined, msg)
		}
	}
	for _, msg := range newMessages {
		if msg.ID != 0 && !seen[msg.ID] {
			seen[msg.ID] = true
			combined = append(combined, msg)
		}
	}

	sort.SliceStable(combined, func(i, j int) bool {
		if combined[i].Date.Equal(combined[j].Date) {
			return combined[i].ID < combined[j].ID
		}
		return combined[i].Date.Before(combined[j].Date)
	})

	if len(combined) > 250 {
		combined = combined[len(combined)-250:]
	}
	return combined
}

func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle call overlay keybindings first
	if m.CallOverlay.State == CallOverlayMissingHelper {
		switch msg.String() {
		case "enter", "esc":
			m.CallOverlay.SetNone()
			return m, nil
		}
	}
	if m.CallOverlay.State == CallOverlayDialing {
		switch msg.String() {
		case "alt+h", "esc":
			if telegram.Cligram != nil {
				return m, telegram.Cligram.HangupCall(telegram.Cligram.Context(), m.CallOverlay.UserID)
			}
			m.CallOverlay.SetNone()
			m.ActiveCallUserID = 0
			return m, nil
		}
	}
	if m.CallOverlay.State == CallOverlayIncoming {
		switch msg.String() {
		case "a":
			if telegram.Cligram != nil {
				return m, telegram.Cligram.AcceptCall(telegram.Cligram.Context(), m.CallOverlay.UserID)
			}
			return m, nil
		case "d", "esc":
			userID := m.CallOverlay.UserID
			m.CallOverlay.SetNone()
			m.ActiveCallUserID = 0
			if telegram.Cligram != nil {
				return m, telegram.Cligram.DeclineCall(telegram.Cligram.Context(), userID)
			}
			return m, nil
		default:
			// Intercept and absorb all other keystrokes so typing doesn't leak into chat inputs
			return m, nil
		}
	}
	if m.CallOverlay.State == CallOverlayActive {
		switch msg.String() {
		case "alt+m":
			if telegram.Cligram != nil {
				return m, telegram.Cligram.ToggleMute(m.ActiveCallUserID)
			}
			return m, nil
		case "alt+h":
			if telegram.Cligram != nil {
				return m, telegram.Cligram.HangupCall(telegram.Cligram.Context(), m.ActiveCallUserID)
			}
			m.CallOverlay.SetNone()
			m.ActiveCallUserID = 0
			return m, nil
		}
	}

	switch msg.String() {
	case "ctrl+p":
		canCall := false
		var userToCall types.UserInfo
		if m.Mode == ModeUsers && m.SelectedUser.PeerID != "" {
			canCall = true
			userToCall = m.SelectedUser
		} else if m.Mode == ModeAll {
			if selected := m.All.SelectedItem(); selected != nil {
				if u, ok := selected.(types.UserInfo); ok && !u.IsBot && u.PeerID != "" {
					canCall = true
					userToCall = u
				}
			}
		}
		if canCall && m.CallOverlay.State == CallOverlayNone {
			pInfo := peerFromItem(userToCall)
			uID, err := strconv.ParseInt(pInfo.ID, 10, 64)
			if err != nil {
				slog.Error("failed to parse peer ID for call", "error", err)
				alertCmd := m.Alert.NewAlertCmd(bubbleup.ErrorKey, "Invalid user ID: "+err.Error())
				return m, alertCmd
			}
			m.CallOverlay.SetDialing(userToCall.FirstName, uID)
			return m, telegram.Cligram.CallUser(telegram.Cligram.Context(), pInfo)
		}
		return m, nil
	case "shift+down":
		if m.FocusedOn == Main {
			if len(m.Conversations) > 0 {
				m.SelectedMessageIndex = len(m.Conversations) - 1
				m.scrollSelectedMessageIntoView()
			}
			m.viewport.GotoBottom()
		}
	case "up", "k":
		if m.FocusedOn == Main {
			if m.ShowForumTopics && m.SelectedForumTopic == nil {
				return m, nil
			}
			if len(m.Conversations) > 0 {
				if m.SelectedMessageIndex < 0 {
					m.SelectedMessageIndex = len(m.Conversations) - 1
				} else if m.SelectedMessageIndex > 0 {
					m.SelectedMessageIndex--
				}
				m.scrollSelectedMessageIntoView()
			}
			return m, nil
		}
	case "down", "j":
		if m.FocusedOn == Main {
			if m.ShowForumTopics && m.SelectedForumTopic == nil {
				return m, nil
			}
			if len(m.Conversations) > 0 {
				if m.SelectedMessageIndex < 0 {
					m.SelectedMessageIndex = 0
				} else if m.SelectedMessageIndex < len(m.Conversations)-1 {
					m.SelectedMessageIndex++
				}
				m.scrollSelectedMessageIntoView()
			}
			return m, nil
		}
	case "pgup":
		if m.FocusedOn == Main && len(m.Conversations) > 0 {
			m.SelectedMessageIndex -= 10
			if m.SelectedMessageIndex < 0 {
				m.SelectedMessageIndex = 0
			}
			m.scrollSelectedMessageIntoView()
			return m, nil
		}
	case "pgdown":
		if m.FocusedOn == Main && len(m.Conversations) > 0 {
			m.SelectedMessageIndex += 10
			if m.SelectedMessageIndex >= len(m.Conversations) {
				m.SelectedMessageIndex = len(m.Conversations) - 1
			}
			m.scrollSelectedMessageIntoView()
			return m, nil
		}
	case "home":
		if m.FocusedOn == Main && len(m.Conversations) > 0 {
			m.SelectedMessageIndex = 0
			m.scrollSelectedMessageIntoView()
			return m, nil
		}
	case "end":
		if m.FocusedOn == Main && len(m.Conversations) > 0 {
			m.SelectedMessageIndex = len(m.Conversations) - 1
			m.scrollSelectedMessageIntoView()
			return m, nil
		}
	case "i":
		if m.FocusedOn == Main && !isReadOnlyBroadcast(&m) {
			m.FocusedOn = Input
			m.Input.Focus()
			return m, nil
		}
	case "esc":
		if m.FocusedOn == Input {
			m.FocusedOn = Main
			m.Input.Blur()
			if len(m.Conversations) > 0 {
				if m.SelectedMessageIndex < 0 {
					m.SelectedMessageIndex = len(m.Conversations) - 1
				} else if m.SelectedMessageIndex >= len(m.Conversations) {
					m.SelectedMessageIndex = len(m.Conversations) - 1
				}
				m.scrollSelectedMessageIntoView()
			}
			return m, nil
		}
		if m.FocusedOn == Main {
			m.FocusedOn = SideBar
			return m, nil
		}
	case "ctrl+a":
		m, cmd := m.handleCtrlA()
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	case "q", "ctrl+c":
		return m, tea.Quit
	case "backspace":
		if m.FocusedOn == Main && m.ShowForumTopics && m.SelectedForumTopic != nil {
			m.SelectedForumTopic = nil
			m.MainViewLoading = false
			m.Conversations = nil
			m.SelectedMessageIndex = -1
			m.viewport.SetContent("")
			m.ChatUI.SetItems([]list.Item{})
			m.ChatUI.ResetSelected()
			return m, nil
		}
		if m.FocusedOn == Main && m.ShowForumTopics && m.SelectedForumTopic == nil {
			m.ShowForumTopics = false
			m.FocusedOn = SideBar
			return m, nil
		}
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
	case "a":
		m, cmd := changeSideBarMode(&m, "a")
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
	case "ctrl+r":
		if m.FocusedOn == Main && m.SelectedMessage() != nil {
			cmds := []tea.Cmd{telegram.Cligram.GetAvailableReactions(telegram.Cligram.Context())}
			if m.CurrentUser == nil {
				cmds = append(cmds, func() tea.Msg {
					user, err := telegram.Cligram.GetMe(telegram.Cligram.Context())
					return types.CurrentUserMsg{User: user, Err: err}
				})
			}
			return m, tea.Batch(cmds...)
		}
	case "alt+s":
		return m, func() tea.Msg { return m.Stories }
	}
	cmds = append(cmds, SendUserIsTyping(&m))
	return m, tea.Batch(cmds...)
}

func (m Model) handleListPagination() (Model, tea.Cmd) {
	if telegram.Cligram == nil {
		return m, nil
	}

	var activeList *list.Model

	switch m.Mode {
	case ModeAll:
		activeList = &m.All
	case ModeUsers:
		activeList = &m.Users
	case ModeBots:
		activeList = &m.Bots
	case ModeChannels:
		activeList = &m.Channels
	case ModeGroups:
		activeList = &m.Groups
	default:
		return m, nil
	}

	visibleItemsCount := len(activeList.VisibleItems())
	if visibleItemsCount == 0 || activeList.Index() < visibleItemsCount-6 {
		return m, nil
	}

	// Do not paginate if already loading/paginating, or end of list reached (-1)
	if m.isCurrentCategoryLoading() || m.SideBarLoading || m.OnPagination || m.OffsetDate == -1 || m.OffsetID == -1 {
		return m, nil
	}

	m.ensureSidebarSpinner()
	m.OnPagination = true
	m.setCategoryLoading(m.Mode, true)
	return m, tea.Batch(
		m.SidebarSpinner.Tick,
		telegram.Cligram.GetAllChats(telegram.Cligram.Context(), m.OffsetDate, m.OffsetID, 50),
	)
}

func (m Model) handleEditKey() (tea.Model, tea.Cmd) {
	if m.FocusedOn == Main {
		selectedItem := m.SelectedMessage()
		if selectedItem != nil && (selectedItem.IsFromMe || strings.ToLower(selectedItem.Sender) == "you") {
			sentDate := selectedItem.Date
			now := time.Now()
			diff := now.Sub(sentDate)
			if !sentDate.IsZero() && diff.Hours() >= 48 {
				m.Alert = m.Alert.WithAllowEscToClose().WithPosition(bubbleup.TopLeftPosition)
				alertCmd := m.Alert.NewAlertCmd(bubbleup.ErrorKey, "sorry u can't edit a message older than 48 hours")
				return m, alertCmd
			}
			m.FocusedOn = Input
			m.Input.SetValue(selectedItem.Content)
			m.EditMessage = selectedItem
			m.SkipNextInput = true
		}
	}
	return m, nil
}

func (m Model) handleCtrlA() (tea.Model, tea.Cmd) {
	if m.FocusedOn == Input {
		m.IsFilepickerVisible = true
		m.FocusedOn = Main
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
		return handleUserChange(&m, nil, nil)
	}
	if m.FocusedOn == Main && m.ShowForumTopics && m.SelectedForumTopic == nil {
		selectedItem := m.SelectedGroupForumTopics.SelectedItem()
		if selectedItem == nil {
			return m, nil
		}
		forumTopic, ok := selectedItem.(types.ForumTopicInfo)
		if !ok {
			return m, nil
		}
		m.SelectedForumTopic = &forumTopic
		m.MainViewLoading = true
		m.Conversations = nil
		m.SelectedMessageIndex = -1
		m.viewport.SetContent("")
		m.ChatUI.SetItems([]list.Item{})
		m.ChatUI.ResetSelected()

		pInfo := getMessageParams(&m)
		topicID := forumTopic.ID
		cmd := telegram.Cligram.GetMessages(telegram.Cligram.Context(), types.GetMessagesRequest{
			Peer:     pInfo,
			Limit:    50,
			TopMsgID: &topicID,
		})
		return m, cmd
	}

	if m.FocusedOn == Main {
		selectedMessage := m.SelectedMessage()
		if selectedMessage != nil && selectedMessage.MessageMediaWebPage != nil {
			if webPage, ok := selectedMessage.MessageMediaWebPage.Webpage.(*tg.WebPage); ok {
				if entity := getEntityName(webPage.URL); entity != nil {
					return m, func() tea.Msg {
						return types.OpenNewChatWithPeerMsg{
							Chat: entity,
						}
					}
				}

				url := webPage.URL
				var cmd tea.Cmd
				switch runtime.GOOS {
				case "darwin":
					cmd = func() tea.Msg {
						err := exec.Command("open", url).Start()
						if err != nil {
							slog.Error("Failed to open URL", "error", err.Error(), "url", url)
						}
						return nil
					}
				case "linux":
					cmd = func() tea.Msg {
						err := exec.Command("xdg-open", url).Start()
						if err != nil {
							slog.Error("Failed to open URL", "error", err.Error(), "url", url)
						}
						return nil
					}
				}
				return m, cmd
			}
		}
		return m, nil
	}
	return m, nil
}

func getEntityName(link string) *types.EntityPreviewInfo {
	var entityPreview *types.EntityPreviewInfo = nil
	for _, prefix := range []string{"@", "http://t.me/", "https://t.me/", "t.me/", "telegram.me/", "www.t.me/", "www.telegram.me/"} {
		if after, ok := strings.CutPrefix(link, prefix); ok {
			entityPreview = &types.EntityPreviewInfo{}
			if parts := strings.SplitN(after, "/", 3); len(parts) > 0 {
				entityPreview.Entity = parts[0]
				if len(parts) >= 2 {
					entityPreview.MessageID = append(entityPreview.MessageID, parts[1])
				}
				if len(parts) >= 3 {
					entityPreview.MessageID = append(entityPreview.MessageID, parts[2])
				}
				break
			}
			entityPreview.Entity = after
			break
		}
	}
	return entityPreview
}

func (m Model) handleReplyKey() (tea.Model, tea.Cmd) {
	if m.FocusedOn == Main {
		canWrite := (m.Mode == ModeUsers || m.Mode == ModeGroups) || (m.Mode == ModeChannels && m.SelectedChannel.IsCreator)
		if canWrite {
			selectedMessage := m.SelectedMessage()
			if selectedMessage != nil {
				m.IsReply = true
				m.FocusedOn = Input
				m.SkipNextInput = true
				m.ReplyTo = selectedMessage
			}
		}
	}
	return m, nil
}

func (m Model) handleDeleteKey() (tea.Model, tea.Cmd) {
	if m.FocusedOn == Main {
		selectedItem := m.SelectedMessage()
		if selectedItem != nil {
			itemCopy := *selectedItem
			return m, func() tea.Msg {
				return OpenModalMsg{ModalMode: ModalModeDeleteMessage, Message: &itemCopy}
			}
		}
	}
	return m, nil
}

func (m Model) handleForwardKey() (tea.Model, tea.Cmd) {
	if m.FocusedOn != Main {
		return m, nil
	}
	selectedMessage := m.SelectedMessage()
	if selectedMessage == nil {
		return m, nil
	}
	msgCopy := *selectedMessage

	var from list.Item
	switch m.Mode {
	case ModeAll:
		from = m.All.SelectedItem()
	case ModeUsers, ModeBots:
		from = m.SelectedUser
	case ModeChannels:
		from = m.SelectedChannel
	case ModeGroups:
		from = &m.SelectedGroup
	}

	return m, func() tea.Msg {
		return OpenModalMsg{
			ModalMode:    ModalModeForwardMessage,
			Message:      &msgCopy,
			UsersList:    &m.Users,
			ChannelsList: &m.Channels,
			GroupsList:   &m.Groups,
			BotsList:     &m.Bots,
			FromPeer:     &from,
		}
	}
}

func itemUniqueKey(item list.Item) string {
	switch v := item.(type) {
	case types.UserInfo:
		if v.IsBot {
			return "bot:" + v.PeerID
		}
		return "user:" + v.PeerID
	case *types.UserInfo:
		if v != nil {
			if v.IsBot {
				return "bot:" + v.PeerID
			}
			return "user:" + v.PeerID
		}
	case types.ChannelInfo:
		if v.IsBroadcast {
			return "channel:" + v.ID
		}
		return "group:" + v.ID
	case *types.ChannelInfo:
		if v != nil {
			if v.IsBroadcast {
				return "channel:" + v.ID
			}
			return "group:" + v.ID
		}
	}
	if item != nil {
		return item.FilterValue()
	}
	return ""
}

func appendListItems[T types.FilterableItem](l *list.Model, items []T) tea.Cmd {
	current := l.Items()
	existing := make(map[string]bool, len(current)+len(items))
	for _, it := range current {
		key := itemUniqueKey(it)
		if key != "" {
			existing[key] = true
		}
	}
	for _, it := range items {
		key := itemUniqueKey(it)
		if key != "" && !existing[key] {
			current = append(current, it)
			existing[key] = true
		}
	}
	return l.SetItems(current)
}

func (m Model) handleUserChats(msg types.UserChatsMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		m.OnPagination = false
		m.UsersLoading = false
		m.BotsLoading = false
		m.SideBarLoading = false
		m.IsModalVisible = true
		m.ModalContent = GetModalContent(msg.Err.Error())
		return m, nil
	}
	if msg.Response == nil {
		m.OnPagination = false
		m.UsersLoading = false
		m.BotsLoading = false
		m.SideBarLoading = false
		return m, nil
	}
	m.SideBarLoading = false
	m.OnPagination = false

	if msg.IsBot {
		m.BotsLoading = false
		m.BotsLoaded = true
		m.BotsOffsetDate = msg.Response.OffsetDate
		m.BotsOffsetID = msg.Response.OffsetID
		cmd := appendListItems(&m.Bots, msg.Response.Data)
		return m, cmd
	}

	m.UsersLoading = false
	m.UsersLoaded = true
	m.UsersOffsetDate = msg.Response.OffsetDate
	m.UsersOffsetID = msg.Response.OffsetID
	m.OffsetDate = msg.Response.OffsetDate
	m.OffsetID = msg.Response.OffsetID
	cmd := appendListItems(&m.Users, msg.Response.Data)
	return m, cmd
}

func (m Model) handleUserChannels(msg types.ChannelsMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		m.OnPagination = false
		m.ChannelsLoading = false
		m.SideBarLoading = false
		m.IsModalVisible = true
		m.ModalContent = GetModalContent(msg.Err.Error())
		return m, nil
	}
	if msg.Response == nil {
		m.OnPagination = false
		m.ChannelsLoading = false
		m.SideBarLoading = false
		return m, nil
	}
	m.SideBarLoading = false
	m.OnPagination = false
	m.ChannelsLoading = false
	m.ChannelsLoaded = true
	m.ChannelsOffsetDate = msg.Response.OffsetDate
	m.ChannelsOffsetID = msg.Response.OffsetID
	m.OffsetDate = msg.Response.OffsetDate
	m.OffsetID = msg.Response.OffsetID
	cmd := appendListItems(&m.Channels, msg.Response.Data)
	return m, cmd
}

func (m Model) handleUserGroups(msg types.GroupsMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		m.OnPagination = false
		m.GroupsLoading = false
		m.SideBarLoading = false
		m.IsModalVisible = true
		m.ModalContent = GetModalContent(msg.Err.Error())
		return m, nil
	}
	if msg.Response == nil {
		m.OnPagination = false
		m.GroupsLoading = false
		m.SideBarLoading = false
		return m, nil
	}
	m.SideBarLoading = false
	m.OnPagination = false
	m.GroupsLoading = false
	m.GroupsLoaded = true
	m.GroupsOffsetDate = msg.Response.OffsetDate
	m.GroupsOffsetID = msg.Response.OffsetID
	m.OffsetDate = msg.Response.OffsetDate
	m.OffsetID = msg.Response.OffsetID
	cmd := appendListItems(&m.Groups, msg.Response.Data)
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

func peerFromItem(item list.Item) types.Peer {
	switch p := item.(type) {
	case ForwardDestinationItem:
		return peerFromItem(p.OriginalItem)
	case *ForwardDestinationItem:
		if p != nil {
			return peerFromItem(p.OriginalItem)
		}
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
	d := calculateLayoutDimensions(&m)
	updateListDimensions(&m, d)
	availWidth := max(10, d.mainWidth-6)
	availHeight := max(4, d.contentHeight-8)
	oldOffset := m.viewport.YOffset
	if m.viewport.Width == 0 && m.viewport.Height == 0 {
		m.viewport = viewport.New(availWidth, availHeight)
		m.viewport.KeyMap = viewport.KeyMap{}
	} else {
		m.viewport.Width = availWidth
		m.viewport.Height = availHeight
	}
	cmd := m.updateConversations()
	if oldOffset > 0 {
		m.viewport.SetYOffset(oldOffset)
	}
	return m, cmd
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
		return handleUserChange(&m, nil, nil)
	}

	updateUserCmd := l.SetItems(append(l.Items(), user))
	index = getUserIndex(*l, user)
	if index != -1 {
		l.Select(index)
		m.FocusedOn = SideBar
		m.Mode = targetMode
		m, handleUserChangeCmd := handleUserChange(&m, nil, nil)
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
		return handleUserChange(&m, nil, nil)
	}

	setItemsCmd := m.Channels.SetItems(append(m.Channels.Items(), channel))
	index = getChannelIndex(m, channel)
	if index != -1 {
		m.Channels.Select(index)
		m.FocusedOn = SideBar
		m.Mode = ModeChannels
		m, handleChangeUserCmd := handleUserChange(&m, nil, nil)
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
		return handleUserChange(&m, nil, nil)
	}

	setItemsCmd := m.Groups.SetItems(append(m.Groups.Items(), group))
	index = getGroupIndex(m, group)
	if index != -1 {
		m.Groups.Select(index)
		m.FocusedOn = SideBar
		m.Mode = ModeGroups
		m, handleChangeUserCmd := handleUserChange(&m, nil, nil)
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
