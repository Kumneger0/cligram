package ui

import (
	"log/slog"
	"os"
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
	"github.com/kumneger0/cligram/internal/config"
	"github.com/kumneger0/cligram/internal/notification"
	"github.com/kumneger0/cligram/internal/telegram"
	"github.com/kumneger0/cligram/internal/telegram/client"
	"github.com/kumneger0/cligram/internal/telegram/shared"
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
			targetName := m.CallOverlay.UserName
			if targetName == "" && (m.Mode == ModeUsers || m.Mode == ModeAll) {
				targetName = m.SelectedUser.FirstName
			}
			if targetName == "" {
				targetName = "this user"
			}
			m.CallOverlay.SetNone()
			m.ActiveCallUserID = 0
			var alertCmd tea.Cmd
			if types.IsPrivacyRestricted(msg.Err) {
				alertCmd = m.Alert.NewAlertCmd(bubbleup.ErrorKey, "Cannot call "+targetName+" due to their privacy settings.")
			} else {
				alertCmd = m.Alert.NewAlertCmd(bubbleup.ErrorKey, "Call failed: "+msg.Err.Error())
			}
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
	case types.NotifySettingsUpdateNotification:
		return m.handleNotifySettingsUpdate(msg)
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
		if msg.Chats.AccountNotifySettings != nil {
			m.AccountNotifySettings = msg.Chats.AccountNotifySettings
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
	case OpenModalMsg:
		m.IsModalVisible = false
		m.ModalContent = ""
		return m, nil
	case CloseOverlay:
		m.IsModalVisible = false
		m.ModalContent = ""
		m.CommentsModalOpen = false
		return m, nil
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
		m.EditMessage = nil
		return m, nil

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
	case types.EditMessageNotification:
		model, cmd := m.handleEditMessage(msg)
		m = model.(Model)
		return m, cmd
	case types.DeleteMessagesNotification:
		model, cmd := m.handleDeleteMessages(msg)
		m = model.(Model)
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
	case types.DownloadMediaCompleteMsg:
		for i := range m.Conversations {
			if m.Conversations[i].ID == msg.MessageID && m.Conversations[i].MediaAttachment != nil {
				if msg.Err != nil {
					m.Conversations[i].MediaAttachment.Status = types.MediaStatusFailed
					m.Alert = m.Alert.WithAllowEscToClose().WithPosition(bubbleup.TopLeftPosition)
					alertCmd := m.Alert.NewAlertCmd(bubbleup.ErrorKey, "Media download failed: "+msg.Err.Error())
					return m, alertCmd
				}
				m.Conversations[i].MediaAttachment.Status = types.MediaStatusDownloaded
				m.Conversations[i].MediaAttachment.LocalPath = msg.LocalPath
				m.Alert = m.Alert.WithAllowEscToClose().WithPosition(bubbleup.TopLeftPosition)
				alertCmd := m.Alert.NewAlertCmd(bubbleup.InfoKey, "Opened media file")
				return m, alertCmd
			}
		}
		return m, nil
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
	case FilePickedMsg:
		if !msg.Canceled && msg.Path != "" {
			if _, err := os.Stat(msg.Path); err == nil {
				m.SelectedFile = msg.Path
				m.FocusedOn = Input
				m.Input.Focus()
			}
		}
		return m, nil
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
	case types.AccountSwitchedMsg:
		if msg.Err != nil {
			slog.Error("Failed to switch account", "account", msg.Account, "error", msg.Err)
			return m, nil
		}
		m.CurrentUser = msg.User
		m.ActivePeerID = ""
		m.SelectedMessageIndex = -1
		m.Conversations = []types.FormattedMessage{}
		m.SelectedUser = types.UserInfo{}
		m.SelectedChannel = types.ChannelInfo{}
		m.SelectedGroup = types.ChannelInfo{}
		m.SelectedForumTopic = nil
		m.Drafts = make(map[string]string)
		m.OffsetDate = 0
		m.OffsetID = 0
		m.UsersOffsetDate = 0
		m.UsersOffsetID = 0
		m.ChannelsOffsetDate = 0
		m.ChannelsOffsetID = 0
		m.GroupsOffsetDate = 0
		m.GroupsOffsetID = 0
		m.BotsOffsetDate = 0
		m.BotsOffsetID = 0

		allList := list.New([]list.Item{}, CustomDelegate{Model: &m}, 10, 20)
		allList.SetShowPagination(false)
		allList.SetShowHelp(false)
		allList.SetShowTitle(false)
		allList.SetShowStatusBar(false)

		userList := list.New([]list.Item{}, CustomDelegate{Model: &m}, 10, 20)
		userList.SetShowPagination(false)
		userList.SetShowHelp(false)
		userList.SetShowTitle(false)

		channels := list.New([]list.Item{}, CustomDelegate{Model: &m}, 10, 20)
		channels.SetShowPagination(false)
		channels.SetShowHelp(false)
		channels.SetShowTitle(false)

		groups := list.New([]list.Item{}, CustomDelegate{Model: &m}, 10, 20)
		groups.SetShowPagination(false)
		groups.SetShowHelp(false)
		groups.SetShowTitle(false)

		botsList := list.New([]list.Item{}, CustomDelegate{Model: &m}, 10, 20)
		botsList.SetShowPagination(false)
		botsList.SetShowHelp(false)
		botsList.SetShowTitle(false)

		chatList := list.New([]list.Item{}, MessagesDelegate{Model: &m}, 5, 2)
		chatList.SetShowPagination(false)
		chatList.SetShowHelp(false)
		chatList.SetShowFilter(false)
		chatList.SetShowTitle(false)
		chatList.SetShowStatusBar(false)

		forumTopics := list.New([]list.Item{}, ForumTopicsDelegate{Model: &m}, 10, 20)
		forumTopics.SetShowPagination(false)
		forumTopics.SetShowHelp(false)
		forumTopics.SetShowFilter(false)
		forumTopics.SetShowTitle(false)
		forumTopics.SetShowStatusBar(false)

		m.All = allList
		m.Users = userList
		m.Channels = channels
		m.Groups = groups
		m.Bots = botsList
		m.ChatUI = chatList
		m.SelectedGroupForumTopics = forumTopics

		m.SideBarLoading = true
		m.StoriesLoading = true
		m.AllLoading = true
		m.AllLoaded = false
		m.UsersLoaded = false
		m.ChannelsLoaded = false
		m.GroupsLoaded = false
		m.BotsLoaded = false

		var switchCmds []tea.Cmd
		switchCmds = append(switchCmds, m.SidebarSpinner.Tick)
		if telegram.Cligram != nil && telegram.Cligram.Context() != nil {
			switchCmds = append(switchCmds,
				telegram.Cligram.GetAllChats(telegram.Cligram.Context(), 0, 0, 50),
				telegram.Cligram.GetAllStories(telegram.Cligram.Context()),
			)
			if m.CurrentUser == nil {
				switchCmds = append(switchCmds, func() tea.Msg {
					user, err := telegram.Cligram.GetMe(telegram.Cligram.Context())
					return types.CurrentUserMsg{User: user, Err: err}
				})
			}
		}
		return m, tea.Batch(switchCmds...)
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
	case types.PeerBotCommandsMsg:
		if m.BotCommandsCache == nil {
			m.BotCommandsCache = make(map[string][]types.BotCommand)
		}
		if msg.Err == nil {
			m.BotCommandsCache[msg.PeerID] = msg.Commands
		}
		if m.FetchingBotCommandsPeer == msg.PeerID {
			m.FetchingBotCommandsPeer = ""
		}
		if msg.PeerID == m.CurrentPeerID() {
			m.updateBotCommandSuggestions()
		}
		return m, nil
	case RouteParsedLinkMsg:
		if msg.Link != nil {
			return m, routeTelegramLink(&m, msg.Link)
		}
		return m, nil
	case types.CheckChatInviteMsg:
		if msg.Err != nil {
			slog.Error("Failed to check chat invite", "hash", msg.Hash, "error", msg.Err.Error())
			m.Alert = m.Alert.WithAllowEscToClose().WithPosition(bubbleup.TopLeftPosition)
			alertCmd := m.Alert.NewAlertCmd(bubbleup.ErrorKey, "Invite error: "+msg.Err.Error())
			return m, alertCmd
		}
		if msg.AlreadyJoined && msg.Chat != nil {
			return m, selectAndNavigateToChannel(&m, *msg.Chat, nil, "Already a member of "+msg.Chat.Title())
		}
		if msg.Preview != nil {
			return m, func() tea.Msg {
				return OpenModalMsg{
					ModalMode:   ModalModeJoinPreview,
					JoinPreview: msg.Preview,
				}
			}
		}
		return m, nil
	case types.ImportChatInviteMsg:
		if msg.Err != nil {
			slog.Error("Failed to import chat invite", "hash", msg.Hash, "error", msg.Err.Error())
			if msg.Err.Error() == "USER_ALREADY_PARTICIPANT" {
				m.Alert = m.Alert.WithAllowEscToClose().WithPosition(bubbleup.TopLeftPosition)
				alertCmd := m.Alert.NewAlertCmd(bubbleup.InfoKey, "You are already a member of this chat")
				return m, tea.Batch(alertCmd, func() tea.Msg { return CloseOverlay{} })
			}
			if msg.Err.Error() == "INVITE_REQUEST_SENT" {
				m.Alert = m.Alert.WithAllowEscToClose().WithPosition(bubbleup.TopLeftPosition)
				alertCmd := m.Alert.NewAlertCmd(bubbleup.InfoKey, "Join request sent to administrators")
				return m, tea.Batch(alertCmd, func() tea.Msg { return CloseOverlay{} })
			}
			m.Alert = m.Alert.WithAllowEscToClose().WithPosition(bubbleup.TopLeftPosition)
			alertCmd := m.Alert.NewAlertCmd(bubbleup.ErrorKey, "Join failed: "+msg.Err.Error())
			return m, tea.Batch(alertCmd, func() tea.Msg { return CloseOverlay{} })
		}
		if msg.Chat != nil {
			return m, tea.Batch(
				func() tea.Msg { return CloseOverlay{} },
				selectAndNavigateToChannel(&m, *msg.Chat, nil, "Joined "+msg.Chat.Title()+"!"),
			)
		}
		return m, func() tea.Msg { return CloseOverlay{} }
	case types.JoinChannelMsg:
		m.IsJoiningChannel = false
		if m.CommentsModalOpen {
			return m, nil
		}
		var closeOverlayCmd tea.Cmd = func() tea.Msg { return CloseOverlay{} }
		if msg.Err != nil {
			slog.Error("Failed to join channel", "channelID", msg.ChannelID, "error", msg.Err.Error())
			if strings.Contains(msg.Err.Error(), "INVITE_REQUEST_SENT") {
				m.JoinRequestSent = true
				m.Alert = m.Alert.WithAllowEscToClose().WithPosition(bubbleup.TopLeftPosition)
				alertCmd := m.Alert.NewAlertCmd(bubbleup.InfoKey, "Join request sent to administrators")
				return m, tea.Batch(alertCmd, closeOverlayCmd)
			}
			if strings.Contains(msg.Err.Error(), "USER_ALREADY_PARTICIPANT") {
				m.markCurrentChannelJoined()
				m.Alert = m.Alert.WithAllowEscToClose().WithPosition(bubbleup.TopLeftPosition)
				alertCmd := m.Alert.NewAlertCmd(bubbleup.InfoKey, "You are already a member of this channel")
				return m, tea.Batch(alertCmd, closeOverlayCmd)
			}
			m.Alert = m.Alert.WithAllowEscToClose().WithPosition(bubbleup.TopLeftPosition)
			alertCmd := m.Alert.NewAlertCmd(bubbleup.ErrorKey, "Join failed: "+msg.Err.Error())
			return m, tea.Batch(alertCmd, closeOverlayCmd)
		}
		m.JoinRequestSent = false
		m.markCurrentChannelJoined()
		if msg.Chat != nil {
			return m, tea.Batch(
				closeOverlayCmd,
				selectAndNavigateToChannel(&m, *msg.Chat, nil, "Joined "+msg.Chat.Title()+"!"),
			)
		}
		title := ""
		if ch := m.currentUnjoinedChannelInfo(); ch != nil {
			title = ch.ChannelTitle
		}
		if title == "" {
			title = m.SelectedChannel.ChannelTitle
		}
		m.Alert = m.Alert.WithAllowEscToClose().WithPosition(bubbleup.TopLeftPosition)
		alertCmd := m.Alert.NewAlertCmd(bubbleup.InfoKey, "Joined "+title+" successfully!")
		return m, tea.Batch(alertCmd, closeOverlayCmd)
	case types.DiscussionMessageMsg:
		// ModalModeComments in overlay handles displaying comments and write access.
		// Never navigate background away from the active channel.
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

func (m *Model) sendNewMessageNotification(item any, message *tg.Message) {
	currentPeerID := m.CurrentPeerID()
	decision := EvaluateNotificationCascade(item, message, currentPeerID, m.AccountNotifySettings)
	if !decision.ShouldNotify {
		return
	}

	title, content := FormatNotificationContent(item, message, decision.ShowPreview)
	notification.Notify(title, content)
}

func (m Model) handleNotifySettingsUpdate(msg types.NotifySettingsUpdateNotification) (tea.Model, tea.Cmd) {
	if msg.NotifySettings == nil {
		return m, nil
	}

	if msg.SourceType != "peer" {
		if m.AccountNotifySettings == nil {
			m.AccountNotifySettings = &types.AccountNotifySettings{}
		}
		switch msg.SourceType {
		case "users":
			m.AccountNotifySettings.Users = msg.NotifySettings
		case "chats":
			m.AccountNotifySettings.Chats = msg.NotifySettings
		case "broadcasts":
			m.AccountNotifySettings.Broadcasts = msg.NotifySettings
		}
		return m, nil
	}

	peerID := msg.PeerID
	if peerID == "" {
		return m, nil
	}

	var cmds []tea.Cmd

	if m.SelectedUser.PeerID == peerID {
		m.SelectedUser.NotifySettings = msg.NotifySettings
	}
	if m.SelectedChannel.ID == peerID {
		m.SelectedChannel.NotifySettings = msg.NotifySettings
	}
	if m.SelectedGroup.ID == peerID {
		m.SelectedGroup.NotifySettings = msg.NotifySettings
	}

	for _, l := range []*list.Model{&m.Users, &m.Bots} {
		for i, item := range l.Items() {
			if u, ok := item.(types.UserInfo); ok && u.PeerID == peerID {
				u.NotifySettings = msg.NotifySettings
				cmds = append(cmds, l.SetItem(i, u))
				break
			}
		}
	}

	for _, l := range []*list.Model{&m.Channels, &m.Groups} {
		for i, item := range l.Items() {
			if c, ok := item.(types.ChannelInfo); ok && c.ID == peerID {
				c.NotifySettings = msg.NotifySettings
				cmds = append(cmds, l.SetItem(i, c))
				break
			}
		}
	}

	for i, item := range m.All.Items() {
		switch it := item.(type) {
		case types.UserInfo:
			if it.PeerID == peerID {
				it.NotifySettings = msg.NotifySettings
				cmds = append(cmds, m.All.SetItem(i, it))
			}
		case types.ChannelInfo:
			if it.ID == peerID {
				it.NotifySettings = msg.NotifySettings
				cmds = append(cmds, m.All.SetItem(i, it))
			}
		}
	}

	return m, tea.Batch(cmds...)
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

	if userInfo == nil && m.SelectedUser.PeerID == peerID {
		userInfo = &m.SelectedUser
	}
	if channelOrGroupInfo == nil {
		if m.SelectedChannel.ID == peerID {
			channelOrGroupInfo = &m.SelectedChannel
		} else if m.SelectedGroup.ID == peerID {
			channelOrGroupInfo = &m.SelectedGroup
		}
	}

	if userInfo != nil && msg.Message != nil {
		m.sendNewMessageNotification(*userInfo, msg.Message)
	}

	if channelOrGroupInfo != nil && msg.Message != nil {
		m.sendNewMessageNotification(*channelOrGroupInfo, msg.Message)
	}

	isOut := false
	if msg.Message != nil {
		isOut = msg.Message.Out
	} else if msg.Formatted != nil {
		isOut = msg.Formatted.IsFromMe
	}

	currentPeerID := m.CurrentPeerID()
	// If message does not belong to currently open chat:
	if currentPeerID == "" || peerID != currentPeerID {
		if !isOut {
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

	var markReadCmd tea.Cmd
	cligramConfig := config.GetConfig()
	if !isOut && cligramConfig.Chat.ReadReceiptMode != nil && *cligramConfig.Chat.ReadReceiptMode == "instant" {
		peer := types.Peer{
			ID:       peerID,
			ChatType: chatType,
		}
		if channelOrGroupInfo != nil {
			peer.AccessHash = channelOrGroupInfo.AccessHash
		} else if userInfo != nil {
			peer.AccessHash = userInfo.AccessHash
		}
		markReadCmd = telegram.Cligram.MarkMessagesAsRead(telegram.Cligram.Context(), types.MarkAsReadRequest{
			Peer: peer,
		})
	} else if !isOut && userInfo != nil {
		m.SelectedUser.UnreadCount++
	}

	// Check if this message was already added (e.g. optimistic send or duplicate update)
	alreadyPresent := false
	for i, c := range m.Conversations {
		if c.ID == msg.ID || (msg.Message != nil && msg.Message.Out && c.IsFromMe && c.Content == msg.Message.Message) {
			m.Conversations[i].ID = msg.ID
			alreadyPresent = true
			break
		}
	}

	if !alreadyPresent {
		var formattedMessage types.FormattedMessage
		if msg.Formatted != nil {
			formattedMessage = *msg.Formatted
		} else if msg.Message != nil {
			formattedMessage = getFormattedMessageFunc(GetFormattedMessageArg{
				ChatType:           chatType,
				ChannelOrGroupInfo: channelOrGroupInfo,
				UserInfo:           userInfo,
				Message:            msg.Message,
			})
		}
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
	return m, tea.Batch(fetchCmd, cmd, markReadCmd)
}

func (m Model) handleEditMessage(msg types.EditMessageNotification) (tea.Model, tea.Cmd) {
	currentPeerID := m.CurrentPeerID()
	if msg.PeerID != "" && currentPeerID != "" && msg.PeerID != currentPeerID {
		return m, nil
	}

	foundIdx := -1
	for i, c := range m.Conversations {
		if c.ID == msg.ID {
			foundIdx = i
			break
		}
	}
	if foundIdx == -1 {
		return m, nil
	}

	if msg.Message != nil {
		var userInfo *types.UserInfo
		for _, v := range slices.Concat(m.Users.Items(), m.Bots.Items()) {
			if user, ok := v.(types.UserInfo); ok && user.PeerID == currentPeerID {
				userInfo = &user
				break
			}
		}

		var channelOrGroupInfo *types.ChannelInfo
		if userInfo == nil {
			for _, v := range slices.Concat(m.Channels.Items(), m.Groups.Items()) {
				if cg, ok := v.(types.ChannelInfo); ok && cg.ID == currentPeerID {
					channelOrGroupInfo = &cg
					break
				}
			}
		}

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

		formatted := getFormattedMessageFunc(GetFormattedMessageArg{
			ChatType:           chatType,
			ChannelOrGroupInfo: channelOrGroupInfo,
			UserInfo:           userInfo,
			Message:            msg.Message,
		})
		formatted.IsEdited = true
		editDate := msg.EditDate
		formatted.EditDate = &editDate
		formatted.PeerID = m.Conversations[foundIdx].PeerID
		m.Conversations[foundIdx] = formatted
	} else {
		m.Conversations[foundIdx].IsEdited = true
		editDate := msg.EditDate
		m.Conversations[foundIdx].EditDate = &editDate
	}

	cmd := m.updateConversations()
	fetchCmd := m.checkAndFetchCustomEmojis(m.Conversations)
	return m, tea.Batch(fetchCmd, cmd)
}

func (m Model) handleDeleteMessages(msg types.DeleteMessagesNotification) (tea.Model, tea.Cmd) {
	if len(msg.Messages) == 0 || len(m.Conversations) == 0 {
		return m, nil
	}

	if msg.ChannelID != 0 {
		currentPeerID := m.CurrentPeerID()
		channelIDStr := strconv.FormatInt(msg.ChannelID, 10)
		if currentPeerID != "" && currentPeerID != channelIDStr {
			return m, nil
		}
	}

	deletedSet := make(map[int]struct{}, len(msg.Messages))
	for _, id := range msg.Messages {
		deletedSet[id] = struct{}{}
	}

	newConversations := make([]types.FormattedMessage, 0, len(m.Conversations))
	for _, c := range m.Conversations {
		if _, deleted := deletedSet[c.ID]; !deleted {
			newConversations = append(newConversations, c)
		}
	}

	if len(newConversations) == len(m.Conversations) {
		return m, nil
	}

	m.Conversations = newConversations
	if m.SelectedMessageIndex >= len(m.Conversations) {
		m.SelectedMessageIndex = len(m.Conversations) - 1
	}

	cmd := m.updateConversations()
	return m, cmd
}

type GetFormattedMessageArg struct {
	ChatType           types.ChatType
	ChannelOrGroupInfo *types.ChannelInfo
	UserInfo           *types.UserInfo
	Message            *tg.Message
}

func getFormattedMessageFunc(arg GetFormattedMessageArg) types.FormattedMessage {
	var formatted *types.FormattedMessage
	if (arg.ChatType == types.UserChat || arg.ChatType == types.BotChat) && arg.UserInfo != nil {
		formatted = shared.FormatMessage(arg.Message, arg.UserInfo, nil)
	} else if arg.ChatType == types.GroupChat {
		var groupID string
		if arg.ChannelOrGroupInfo != nil {
			groupID = arg.ChannelOrGroupInfo.ID
		}
		var users []tg.UserClass
		if arg.UserInfo != nil {
			if uid, err := strconv.ParseInt(arg.UserInfo.PeerID, 10, 64); err == nil {
				users = []tg.UserClass{&tg.User{
					ID:        uid,
					FirstName: arg.UserInfo.FirstName,
					LastName:  arg.UserInfo.LastName,
					Username:  arg.UserInfo.Username,
				}}
			}
		}
		formatted = shared.FormatGroupMessage(arg.Message, groupID, users, nil, nil)
	} else if arg.ChannelOrGroupInfo != nil {
		formatted = shared.FormatMessage(arg.Message, arg.ChannelOrGroupInfo, nil)
	} else if arg.UserInfo != nil {
		formatted = shared.FormatMessage(arg.Message, arg.UserInfo, nil)
	} else {
		var nilUser *types.UserInfo
		formatted = shared.FormatMessage(arg.Message, nilUser, nil)
	}

	if formatted != nil {
		return *formatted
	}

	return types.FormattedMessage{
		ID:       arg.Message.ID,
		Content:  arg.Message.Message,
		IsFromMe: arg.Message.GetOut(),
		Date:     time.Unix(int64(arg.Message.Date), 0),
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
			if userToCall.PhoneCallsPrivate {
				alertCmd := m.Alert.NewAlertCmd(bubbleup.ErrorKey, "Cannot call "+userToCall.FirstName+" due to their privacy settings.")
				return m, alertCmd
			}
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
		if m.FocusedOn == Input && m.ShowBotCommands && len(m.FilteredBotCommands) > 0 {
			if msg.String() == "up" {
				m.SelectedBotCommandIndex--
				if m.SelectedBotCommandIndex < 0 {
					m.SelectedBotCommandIndex = len(m.FilteredBotCommands) - 1
				}
				m.SkipNextInput = true
				return m, nil
			}
		}
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
		if m.FocusedOn == Input && m.ShowBotCommands && len(m.FilteredBotCommands) > 0 {
			if msg.String() == "down" {
				m.SelectedBotCommandIndex++
				if m.SelectedBotCommandIndex >= len(m.FilteredBotCommands) {
					m.SelectedBotCommandIndex = 0
				}
				m.SkipNextInput = true
				return m, nil
			}
		}
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
		if m.FocusedOn == Main && m.CanSendMessages() {
			m.FocusedOn = Input
			m.Input.Focus()
			return m, nil
		}
	case "esc":
		if m.IsModalVisible {
			m.IsModalVisible = false
			m.ModalContent = ""
			return m, nil
		}
		if m.ShowBotCommands {
			m.ShowBotCommands = false
			m.FilteredBotCommands = nil
			m.SkipNextInput = true
			return m, nil
		}
		if m.SelectedFile != "" && m.SelectedFile != "uploading..." {
			m.SelectedFile = ""
			return m, nil
		}
		if m.FocusedOn == Input {
			if m.IsReply {
				m.IsReply = false
				m.ReplyTo = nil
			}
			if m.EditMessage != nil {
				m.EditMessage = nil
				m.Input.Reset()
			}
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
			if m.IsReply {
				m.IsReply = false
				m.ReplyTo = nil
				return m, nil
			}
			m.FocusedOn = SideBar
			return m, nil
		}
	case "ctrl+o":
		m, cmd := m.handleFileAttachKey()
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	case "ctrl+c":
		return m, tea.Quit
	case "q":
		if m.IsModalVisible {
			m.IsModalVisible = false
			m.ModalContent = ""
			return m, nil
		}
		if m.FocusedOn != Input {
			return m, tea.Quit
		}
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
		if m.FocusedOn == Input && m.ShowBotCommands && len(m.FilteredBotCommands) > 0 {
			m.completeBotCommand()
			m.SkipNextInput = true
			return m, nil
		}
		m, cmd := changeFocusMode(&m, "tab", false)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	case "shift+tab":
		m, cmd := changeFocusMode(&m, "tab", true)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	case "a":
		m, cmd := changeSideBarMode(&m, "a")
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	case "c":
		if m.FocusedOn == Main && m.CurrentInputState() == InputStateBroadcastReadOnly {
			if hasComments, _, _ := m.SelectedMessageHasComments(); hasComments {
				return m.openCommentsForSelectedMessage()
			}
		}
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
	case "m":
		m, cmd := m.handleMemberRosterKey()
		if cmd != nil {
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)
		}
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
	case "f2":
		return m, func() tea.Msg {
			return OpenModalMsg{
				ModalMode: ModalModeGhostMode,
			}
		}
	case ",":
		if m.FocusedOn != Input {
			return m, func() tea.Msg {
				return OpenModalMsg{
					ModalMode: ModalModeGhostMode,
				}
			}
		}
	case "ctrl+a":
		if m.FocusedOn == Input {
			m, cmd := m.handleFileAttachKey()
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)
		}
		return m, func() tea.Msg {
			return OpenModalMsg{
				ModalMode: ModalModeAccountSwitcher,
			}
		}
	}
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

func (m Model) handleFileAttachKey() (tea.Model, tea.Cmd) {
	return TriggerFilePicker(&m)
}

func (m Model) handleEnterKey() (tea.Model, tea.Cmd) {
	if m.IsModalVisible {
		m.IsModalVisible = false
		m.ModalContent = ""
		return m, nil
	}
	if m.FocusedOn == Input {
		if m.isUnjoinedChat() {
			return m.joinCurrentChannel()
		}
		if m.ShowBotCommands && len(m.FilteredBotCommands) > 0 {
			m.completeBotCommand()
			m.SkipNextInput = true
			return m, nil
		}
		val := strings.TrimSpace(m.Input.Value())
		if strings.HasPrefix(val, "/join ") || strings.HasPrefix(val, "/open ") {
			linkStr := strings.TrimSpace(val[6:])
			if pl := ParseTelegramLink(linkStr); pl != nil {
				m.Input.Reset()
				return m, routeTelegramLink(&m, pl)
			}
		} else if pl := ParseTelegramLink(val); pl != nil {
			m.Input.Reset()
			return m, routeTelegramLink(&m, pl)
		}
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
		if selectedMessage != nil && selectedMessage.MediaAttachment != nil {
			if selectedMessage.MediaAttachment.Status == types.MediaStatusDownloaded && selectedMessage.MediaAttachment.LocalPath != "" {
				err := shared.OpenFileInDefaultApp(selectedMessage.MediaAttachment.LocalPath)
				if err != nil {
					m.Alert = m.Alert.WithAllowEscToClose().WithPosition(bubbleup.TopLeftPosition)
					alertCmd := m.Alert.NewAlertCmd(bubbleup.ErrorKey, "Failed to open file: "+err.Error())
					return m, alertCmd
				}
				m.Alert = m.Alert.WithAllowEscToClose().WithPosition(bubbleup.TopLeftPosition)
				alertCmd := m.Alert.NewAlertCmd(bubbleup.InfoKey, "Opened media file")
				return m, alertCmd
			}
			if selectedMessage.MediaAttachment.Status != types.MediaStatusDownloading {
				for i := range m.Conversations {
					if m.Conversations[i].ID == selectedMessage.ID && m.Conversations[i].MediaAttachment != nil {
						m.Conversations[i].MediaAttachment.Status = types.MediaStatusDownloading
						break
					}
				}
				peerID := m.CurrentPeerID()
				att := *selectedMessage.MediaAttachment
				msgID := selectedMessage.ID
				var cmd tea.Cmd
				if telegram.Cligram != nil {
					cmd = telegram.Cligram.DownloadMedia(telegram.Cligram.Context(), peerID, msgID, att)
				} else {
					cmd = func() tea.Msg {
						return types.DownloadMediaCompleteMsg{
							MessageID: msgID,
							PeerID:    peerID,
							Err:       nil,
						}
					}
				}
				return m, tea.Batch(cmd, m.SidebarSpinner.Tick)
			}
			return m, nil
		}
		if selectedMessage != nil {
			var linkToRoute *types.ParsedLink
			var externalURL string

			// 1. Check WebPage preview URL
			if selectedMessage.MessageMediaWebPage != nil {
				if webPage, ok := selectedMessage.MessageMediaWebPage.Webpage.(*tg.WebPage); ok {
					if pl := ParseTelegramLink(webPage.URL); pl != nil {
						linkToRoute = pl
					} else {
						externalURL = webPage.URL
					}
				}
			}

			// 2. If no WebPage link, extract from message Content
			if linkToRoute == nil && selectedMessage.Content != "" {
				links := ExtractTelegramLinks(selectedMessage.Content)
				if len(links) > 0 {
					linkToRoute = links[0]
				}
			}

			// 3. Route telegram link if found
			if linkToRoute != nil {
				return m, routeTelegramLink(&m, linkToRoute)
			}

			// 4. Open external URL in browser if present
			if externalURL != "" {
				var cmd tea.Cmd
				switch runtime.GOOS {
				case "darwin":
					cmd = func() tea.Msg {
						err := exec.Command("open", externalURL).Start()
						if err != nil {
							slog.Error("Failed to open URL", "error", err.Error(), "url", externalURL)
						}
						return nil
					}
				case "linux":
					cmd = func() tea.Msg {
						err := exec.Command("xdg-open", externalURL).Start()
						if err != nil {
							slog.Error("Failed to open URL", "error", err.Error(), "url", externalURL)
						}
						return nil
					}
				}
				return m, cmd
			}

			// 5. If broadcast channel and selected post has comments, open comments
			if m.CurrentInputState() == InputStateBroadcastReadOnly {
				if hasComments, _, _ := m.SelectedMessageHasComments(); hasComments {
					return m.openCommentsForSelectedMessage()
				}
			}
		}
		if m.isUnjoinedChat() {
			return m.joinCurrentChannel()
		}
		return m, nil
	}
	return m, nil
}

func (m *Model) openCommentsForSelectedMessage() (Model, tea.Cmd) {
	if m.SelectedMessageIndex < 0 || m.SelectedMessageIndex >= len(m.Conversations) {
		return *m, nil
	}
	msg := m.Conversations[m.SelectedMessageIndex]
	if !msg.HasComments && msg.RepliesCount == 0 && msg.RepliesChannelID == 0 {
		return *m, nil
	}

	ch := m.SelectedChannel
	channelID, err := strconv.ParseInt(ch.ID, 10, 64)
	if err != nil || channelID == 0 {
		return *m, nil
	}
	accessHash, _ := strconv.ParseInt(ch.AccessHash, 10, 64)

	m.CommentsModalOpen = true

	openModalCmd := func() tea.Msg {
		return OpenModalMsg{
			ModalMode:    ModalModeComments,
			Message:      &msg,
			CommentsPost: &msg,
		}
	}

	var fetchCmd tea.Cmd
	if telegram.Cligram != nil && telegram.Cligram.Context() != nil {
		fetchCmd = telegram.Cligram.GetDiscussionMessage(telegram.Cligram.Context(), channelID, accessHash, msg.ID)
	}

	return *m, tea.Batch(openModalCmd, fetchCmd)
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

func selectAndNavigateToChannel(m *Model, ch types.ChannelInfo, offsetID *int, successAlert string) tea.Cmd {
	var cmds []tea.Cmd
	if !ch.IsBroadcast {
		m.Mode = ModeGroups
		if groupIndex := getGroupIndex(*m, ch); groupIndex != -1 {
			m.Groups.Select(groupIndex)
			m.SelectedGroup = m.Groups.Items()[groupIndex].(types.ChannelInfo)
		} else {
			items := append(m.Groups.Items(), ch)
			w, h := m.Groups.Width(), m.Groups.Height()
			m.Groups = list.New(items, list.NewDefaultDelegate(), w, h)
			if h > 0 {
				m.Groups.Select(len(m.Groups.Items()) - 1)
			}
			m.SelectedGroup = ch
		}
	} else {
		m.Mode = ModeChannels
		if channelIndex := getChannelIndex(*m, ch); channelIndex != -1 {
			m.Channels.Select(channelIndex)
			m.SelectedChannel = m.Channels.Items()[channelIndex].(types.ChannelInfo)
		} else {
			items := append(m.Channels.Items(), ch)
			w, h := m.Channels.Width(), m.Channels.Height()
			m.Channels = list.New(items, list.NewDefaultDelegate(), w, h)
			if h > 0 {
				m.Channels.Select(len(m.Channels.Items()) - 1)
			}
			m.SelectedChannel = ch
		}
	}

	highlightCmd := func() tea.Msg {
		if offsetID == nil {
			return nil
		}
		return types.ShouldHighlightSpecificMessageMsg{
			MessageID: *offsetID,
		}
	}

	newM, navCmd := handleUserChange(m, offsetID, highlightCmd)
	*m = newM
	if navCmd != nil {
		cmds = append(cmds, navCmd)
	}

	if successAlert != "" {
		m.Alert = m.Alert.WithAllowEscToClose().WithPosition(bubbleup.TopLeftPosition)
		cmds = append(cmds, m.Alert.NewAlertCmd(bubbleup.InfoKey, successAlert))
	}

	return tea.Batch(cmds...)
}

func (m Model) handleReplyKey() (tea.Model, tea.Cmd) {
	if m.FocusedOn != Main && m.FocusedOn != SideBar {
		return m, nil
	}
	if !m.CanSendMessages() {
		return m, nil
	}
	if len(m.Conversations) == 0 {
		return m, nil
	}
	if m.FocusedOn == SideBar {
		pInfo := getMessageParams(&m)
		if pInfo.ID != "" && pInfo.ID != m.ActivePeerID {
			return m, nil
		}
	}
	if m.SelectedMessageIndex < 0 || m.SelectedMessageIndex >= len(m.Conversations) {
		m.SelectedMessageIndex = len(m.Conversations) - 1
		m.scrollSelectedMessageIntoView()
	}
	selectedMessage := m.SelectedMessage()
	if selectedMessage != nil {
		m.IsReply = true
		m.FocusedOn = Input
		m.SkipNextInput = true
		m.ReplyTo = selectedMessage
		m.Input.Focus()
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

func (m Model) handleMemberRosterKey() (tea.Model, tea.Cmd) {
	if m.FocusedOn != Main {
		return m, nil
	}
	pInfo := getMessageParams(&m)
	if pInfo.ID == "" {
		return m, nil
	}
	isGroup := pInfo.ChatType == types.GroupChat || (pInfo.ChatType == types.ChannelChat && !m.SelectedChannel.IsBroadcast)
	if !isGroup {
		return m, nil
	}

	title := m.SelectedGroup.ChannelTitle
	if title == "" {
		title = m.SelectedChannel.ChannelTitle
	}

	totalMembers := 0
	if m.SelectedGroup.ParticipantsCount != nil {
		totalMembers = *m.SelectedGroup.ParticipantsCount
	} else if m.SelectedChannel.ParticipantsCount != nil {
		totalMembers = *m.SelectedChannel.ParticipantsCount
	}

	openModalMsg := func() tea.Msg {
		return OpenModalMsg{
			ModalMode:    ModalModeGroupMembers,
			GroupPeer:    &pInfo,
			GroupTitle:   title,
			TotalMembers: totalMembers,
		}
	}

	var fetchCmd tea.Cmd
	if telegram.Cligram != nil && telegram.Cligram.Context() != nil {
		fetchCmd = telegram.Cligram.GetGroupMembers(telegram.Cligram.Context(), pInfo, 0)
	}

	if fetchCmd != nil {
		return m, tea.Batch(openModalMsg, fetchCmd)
	}
	return m, openModalMsg
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
