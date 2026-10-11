package ui

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kumneger0/cligram/internal/config"
	"github.com/kumneger0/cligram/internal/telegram"
	"github.com/kumneger0/cligram/internal/telegram/types"
	overlay "github.com/rmhubbert/bubbletea-overlay"
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
	var icon string
	var unreadCount int

	switch it := item.(type) {
	case types.UserInfo:
		title = it.Title()
		unreadCount = it.UnreadCount
		if it.IsBot {
			icon = "🤖 "
		} else if it.IsOnline {
			icon = "🟢 "
		} else {
			icon = "👤 "
		}
	case types.ChannelInfo:
		title = it.Title()
		unreadCount = it.UnreadCount
		if it.IsBroadcast {
			icon = "📢 "
		} else if it.IsForum {
			icon = "💬 "
		} else {
			icon = "👥 "
		}
	default:
		return
	}

	isOnSideBar := d.Model.FocusedOn == SideBar
	isSelected := index == m.Index()

	var badge string
	if unreadCount > 0 {
		badge = unreadCountStyle.Render(strconv.Itoa(unreadCount))
	}

	var indicator string
	if isSelected && isOnSideBar {
		indicator = selectedIndicatorStyle.Render("▎") + " "
	} else if isSelected {
		indicator = "· "
	} else {
		indicator = "  "
	}

	totalWidth := m.Width()
	if totalWidth <= 0 {
		totalWidth = 30
	}

	badgeWidth := lipgloss.Width(badge)
	indicatorWidth := lipgloss.Width(indicator)
	iconWidth := lipgloss.Width(icon)

	availNameWidth := totalWidth - indicatorWidth - iconWidth - badgeWidth - 1
	if availNameWidth < 3 {
		availNameWidth = 3
	}

	name := lipgloss.NewStyle().MaxWidth(availNameWidth).Render(title)
	nameWidth := lipgloss.Width(name)

	spacerWidth := totalWidth - indicatorWidth - iconWidth - nameWidth - badgeWidth
	if spacerWidth < 0 {
		spacerWidth = 0
	}
	spacer := strings.Repeat(" ", spacerWidth)

	var rowContent string
	if badge != "" {
		rowContent = indicator + icon + name + spacer + badge
	} else {
		rowContent = indicator + icon + name + spacer
	}

	rowStyle := listItemNormalStyle
	if isSelected && isOnSideBar {
		rowStyle = listItemSelectedStyle
	} else if isSelected {
		rowStyle = listItemUnfocusedStyle
	}

	fmt.Fprint(w, rowStyle.Width(totalWidth).Render(rowContent))
}

func (m Model) Init() tea.Cmd {
	m.ensureSidebarSpinner()
	if telegram.Cligram == nil {
		return nil
	}
	storiesCMD := telegram.Cligram.GetAllStories(telegram.Cligram.Context())
	return tea.Batch(storiesCMD, telegram.Cligram.GetAllChats(telegram.Cligram.Context(), 0, 0, 50), m.SidebarSpinner.Tick)
}

func getChannelIndex(m Model, channel types.ChannelInfo) int {
	var index int = -1
	for i, v := range m.Channels.Items() {
		if v.FilterValue() == channel.ChannelTitle {
			index = i
		}
	}
	return index
}

func getGroupIndex(m Model, group types.ChannelInfo) int {
	var index int = -1
	for i, v := range m.Groups.Items() {
		if v.FilterValue() == group.ChannelTitle {
			index = i
		}
	}
	return index
}

func getUserIndex(listToSearchFrom list.Model, user types.UserInfo) int {
	var index int = -1
	for i, v := range listToSearchFrom.Items() {
		if v.(types.UserInfo).PeerID == user.PeerID {
			index = i
			break
		}
	}
	return index
}

func sendMessage(m *Model) (Model, tea.Cmd) {
	userMsg := m.Input.Value()
	m.Input.Reset()
	peerInfo := getMessageParams(m)
	if m.Drafts != nil {
		if peerInfo.ID != "" {
			delete(m.Drafts, peerInfo.ID)
		}
		if m.ActivePeerID != "" {
			delete(m.Drafts, m.ActivePeerID)
		}
	}
	var messageToReply types.FormattedMessage
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
		m, cmd := m.editMessage(peerInfo, userMsg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}

	replyToMessageID := ""
	if m.IsReply && m.ReplyTo != nil {
		replyToMessageID = strconv.Itoa(messageToReply.ID)
	}

	randID := rand.Int()

	var topMsgID *int
	if m.SelectedForumTopic != nil {
		id := m.SelectedForumTopic.ID
		topMsgID = &id
	}

	cmds = append(cmds, telegram.Cligram.SendMessage(telegram.Cligram.Context(),
		types.SendMessageRequest{
			RandID:           randID,
			Peer:             peerInfo,
			Message:          userMsg,
			IsReply:          m.IsReply && m.ReplyTo != nil,
			ReplyToMessageID: replyToMessageID,
			IsFile:           isFile,
			FilePath:         filepath,
			TopMsgID:         topMsgID,
		}))
	if isFile {
		m.SelectedFile = "uploading..."
	}
	content := userMsg
	if isFile {
		content = "This Message is not supported by this Telegram client."
	}
	cligramConfig := config.GetConfig()
	newMessage := types.FormattedMessage{
		ID:                   randID,
		Sender:               "you",
		IsFromMe:             true,
		Content:              content,
		Media:                nil,
		Date:                 time.Now(),
		IsUnsupportedMessage: isFile,
		WebPage:              nil,
		Document:             nil,
		FromID:               nil,
		PeerID:               &peerInfo.ID,
	}
	if m.IsReply && m.ReplyTo != nil {
		replyCopy := *m.ReplyTo
		newMessage.ReplyTo = &replyCopy
	}
	m.ActivePeerID = peerInfo.ID
	m.Conversations = append(m.Conversations, newMessage)
	if len(m.Conversations) > 250 {
		m.Conversations = m.Conversations[len(m.Conversations)-250:]
	}
	m.SelectedMessageIndex = len(m.Conversations) - 1
	m.viewport.GotoBottom()
	if *cligramConfig.Chat.ReadReceiptMode == "default" {
		cmd := telegram.Cligram.MarkMessagesAsRead(telegram.Cligram.Context(), types.MarkAsReadRequest{
			Peer: peerInfo,
		})
		cmds = append(cmds, cmd)
	}
	m.Input.Reset()
	m.IsReply = false
	m.ReplyTo = nil

	cmds = append(cmds, m.updateConversations())
	return *m, tea.Batch(cmds...)
}

func (m *Model) editMessage(peerInfo types.Peer, userMsg string) (Model, tea.Cmd) {
	if m.EditMessage == nil {
		return *m, nil
	}
	editMsgID := m.EditMessage.ID
	now := time.Now()
	for i, c := range m.Conversations {
		if c.ID == editMsgID {
			m.Conversations[i].Content = userMsg
			m.Conversations[i].IsEdited = true
			m.Conversations[i].EditDate = &now
			break
		}
	}
	for idx, item := range m.ChatUI.Items() {
		if fm, ok := item.(types.FormattedMessage); ok && fm.ID == editMsgID {
			fm.Content = userMsg
			fm.IsEdited = true
			fm.EditDate = &now
			_ = m.ChatUI.SetItem(idx, fm)
			break
		}
	}
	m.EditMessage = nil
	m.Input.Reset()
	updateCmd := m.updateConversations()

	cmd := func() tea.Msg {
		res := telegram.Cligram.EditMessage(telegram.Cligram.Context(), types.EditMessageRequest{
			Peer:       peerInfo,
			MessageID:  int(editMsgID),
			NewMessage: userMsg,
		})
		res.UpdatedMessage = userMsg
		return res
	}

	return *m, tea.Batch(cmd, updateCmd)
}

func updateFocusedComponent(m *Model, msg tea.Msg, cmdsFromParent *[]tea.Cmd) (Model, tea.Cmd) {
	var cmd tea.Cmd
	cmds := *cmdsFromParent

	switch m.FocusedOn {
	case Input:
		if !m.CanFocusInput() || m.isUnjoinedChat() {
			m.Input.Blur()
			return *m, tea.Batch(cmds...)
		}
		m.Input.Focus()
		if !m.SkipNextInput {
			m.Input, cmd = m.Input.Update(msg)
			cmds = append(cmds, cmd)
			if suggestCmd := m.updateBotCommandSuggestions(); suggestCmd != nil {
				cmds = append(cmds, suggestCmd)
			}
		}
	case SideBar:
		m.Input.Blur()
		switch m.Mode {
		case ModeAll:
			m.All, cmd = m.All.Update(msg)
			cmds = append(cmds, cmd)
		case ModeChannels:
			m.Channels, cmd = m.Channels.Update(msg)
			cmds = append(cmds, cmd)
		case ModeBots:
			m.Bots, cmd = m.Bots.Update(msg)
			cmds = append(cmds, cmd)
		case ModeUsers:
			m.Users, cmd = m.Users.Update(msg)
			cmds = append(cmds, cmd)
		default:
			m.Groups, cmd = m.Groups.Update(msg)
			cmds = append(cmds, cmd)
		}
		if _, isKey := msg.(tea.KeyMsg); isKey {
			pagModel, pagCmd := m.handleListPagination()
			*m = pagModel
			if pagCmd != nil {
				cmds = append(cmds, pagCmd)
			}
		}
	default:
		m.Input.Blur()
		if m.ShowForumTopics && m.SelectedForumTopic == nil {
			m.SelectedGroupForumTopics, cmd = m.SelectedGroupForumTopics.Update(msg)
			cmds = append(cmds, cmd)
		} else {
			if _, isKey := msg.(tea.KeyMsg); !isKey {
				m.viewport, cmd = m.viewport.Update(msg)
				cmds = append(cmds, cmd)
			}
		}
	}
	m.SkipNextInput = false
	return *m, tea.Batch(cmds...)
}

func handleUserChange(m *Model, offsetID *int, afterMessagesCmd tea.Cmd) (Model, tea.Cmd) {
	m.ShowForumTopics = false
	m.SelectedForumTopic = nil
	m.ShowBotCommands = false
	m.FilteredBotCommands = nil
	m.SelectedBotCommandIndex = 0

	pInfo := getMessageParams(m)
	if m.Drafts == nil {
		m.Drafts = make(map[string]string)
	}
	if m.ActivePeerID != "" && m.ActivePeerID != pInfo.ID {
		m.Drafts[m.ActivePeerID] = m.Input.Value()
	}
	m.ActivePeerID = pInfo.ID
	m.Input.SetValue(m.Drafts[pInfo.ID])
	m.Input.CursorEnd()

	var prefetchBotCmds tea.Cmd
	if m.BotCommandsCache == nil {
		m.BotCommandsCache = make(map[string][]types.BotCommand)
	}
	if _, cached := m.BotCommandsCache[pInfo.ID]; !cached && m.FetchingBotCommandsPeer != pInfo.ID && telegram.Cligram != nil && telegram.Cligram.Context() != nil {
		if pInfo.ChatType == types.BotChat || pInfo.ChatType == types.GroupChat || pInfo.ChatType == types.ChannelChat || (pInfo.ChatType == types.UserChat && m.SelectedUser.IsBot) {
			m.FetchingBotCommandsPeer = pInfo.ID
			prefetchBotCmds = telegram.Cligram.GetPeerBotCommands(telegram.Cligram.Context(), pInfo)
		}
	}

	isGroupForum := (m.Mode == ModeGroups && m.SelectedGroup.IsForum) ||
		(m.Mode == ModeAll && m.SelectedGroup.IsForum && pInfo.ChatType == types.GroupChat)
	if isGroupForum {
		m.ForumTopicLoading = true
		m.Conversations = nil
		m.SelectedMessageIndex = -1
		m.viewport.SetContent("")
		m.ChatUI.SetItems([]list.Item{})
		return *m, tea.Batch(telegram.Cligram.GetChannelForums(pInfo), prefetchBotCmds)
	}
	cmd := telegram.Cligram.GetMessages(telegram.Cligram.Context(), types.GetMessagesRequest{
		Peer:          pInfo,
		Limit:         50,
		OffsetID:      offsetID,
		ChatAreaWidth: nil,
	})
	cligramConfig := config.GetConfig()
	if *cligramConfig.Chat.ReadReceiptMode == "instant" {
		markAsReadCmd := telegram.Cligram.MarkMessagesAsRead(telegram.Cligram.Context(), types.MarkAsReadRequest{
			Peer: pInfo,
		})
		return *m, tea.Batch(cmd, markAsReadCmd, prefetchBotCmds)
	}
	m.Conversations = nil
	m.SelectedMessageIndex = -1
	m.viewport.SetContent("")
	m.MainViewLoading = true
	m.JoinRequestSent = false
	m.IsJoiningChannel = false
	if m.ChatUI.Height() > 0 {
		m.ChatUI.ResetSelected()
		m.ChatUI.SetItems([]list.Item{})
	}
	return *m, tea.Batch(cmd, afterMessagesCmd, prefetchBotCmds)
}

func isReadOnlyBroadcast(m *Model) bool {
	if m.isUnjoinedChat() {
		return false
	}
	if m.Mode == ModeChannels && !m.SelectedChannel.IsCreator {
		return true
	}
	if m.Mode == ModeAll {
		if selected := m.All.SelectedItem(); selected != nil {
			if ch, ok := selected.(types.ChannelInfo); ok && ch.IsBroadcast && !ch.IsCreator {
				return true
			}
		} else if m.SelectedChannel.IsBroadcast && !m.SelectedChannel.IsCreator {
			return true
		}
	}
	return false
}

func changeFocusMode(m *Model, msg string, shift bool) (Model, tea.Cmd) {
	var cmds []tea.Cmd
	currentlyFocusedOn := m.FocusedOn
	canFocus := m.CanFocusInput()
	if currentlyFocusedOn == SideBar {
		pInfo := getMessageParams(m)
		if pInfo.ID != "" && pInfo.ID != m.ActivePeerID {
			mUpdated, changeCmd := handleUserChange(m, nil, nil)
			*m = mUpdated
			if changeCmd != nil {
				cmds = append(cmds, changeCmd)
			}
		}
		if shift {
			if canFocus {
				m.FocusedOn = Input
			} else {
				m.FocusedOn = Main
			}
		} else {
			m.FocusedOn = Main
		}
	} else if currentlyFocusedOn == Main {
		if shift {
			m.FocusedOn = SideBar
		} else {
			if canFocus {
				m.FocusedOn = Input
			} else {
				m.FocusedOn = SideBar
			}
		}
	} else { // currentlyFocusedOn == Input
		if shift {
			m.FocusedOn = Main
		} else {
			m.FocusedOn = SideBar
		}
	}
	if m.FocusedOn == Main && len(m.Conversations) > 0 {
		if m.SelectedMessageIndex < 0 {
			m.SelectedMessageIndex = len(m.Conversations) - 1
		} else if m.SelectedMessageIndex >= len(m.Conversations) {
			m.SelectedMessageIndex = len(m.Conversations) - 1
		}
		m.scrollSelectedMessageIntoView()
	}
	return updateFocusedComponent(m, msg, &cmds)
}

func changeSideBarMode(m *Model, msg string) (Model, tea.Cmd) {
	if m.FocusedOn == SideBar {
		if m.ChatUI.Height() > 0 {
			m.ChatUI.ResetSelected()
		}
		m.OnPagination = false
	}
	areWeInGroupMode := m.Mode == ModeGroups && m.FocusedOn == Main
	if m.FocusedOn == SideBar || areWeInGroupMode {
		switch msg {
		case "a":
			m.Mode = ModeAll
			m.Input.Reset()
			return m.checkThresholdAndBackfill(ModeAll)
		case "c":
			m.Mode = ModeChannels
			m.Input.Reset()
			return m.checkThresholdAndBackfill(ModeChannels)
		case "u":
			m.Mode = ModeUsers
			m.Input.Reset()
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
					if telegram.Cligram != nil {
						return *m, telegram.Cligram.GetMessages(telegram.Cligram.Context(), types.GetMessagesRequest{
							Peer:          getMessageParams(m),
							Limit:         50,
							OffsetID:      nil,
							ChatAreaWidth: nil,
						})
					}
					return *m, nil
				}
			}
			return m.checkThresholdAndBackfill(ModeUsers)

		case "g":
			m.Mode = ModeGroups
			m.Input.Reset()
			return m.checkThresholdAndBackfill(ModeGroups)
		case "b":
			m.Mode = ModeBots
			m.Input.Reset()
			m.Bots.Select(0)
			m.Bots.ResetSelected()
			return m.checkThresholdAndBackfill(ModeBots)
		}
		return *m, nil
	}
	return *m, nil
}

func (m *Model) getMessageSenderUserInfo() *types.UserInfo {
	if msg := m.SelectedMessage(); msg != nil {
		return msg.SenderUserInfo
	}
	return nil
}

func (m *Model) updateConversations() tea.Cmd {
	cmd := m.ChatUI.SetItems(formatMessages(m.Conversations))
	if m.viewport.Width > 0 && m.viewport.Height > 0 {
		m.renderMessagesViewport(m.viewport.Width, m.viewport.Height)
	}
	return cmd
}

func (m Model) View() string {
	m.All.Title = "All"
	m.All.SetShowStatusBar(false)
	m.Users.Title = "Chats"
	m.Channels.Title = "Channels"
	m.Channels.SetShowStatusBar(false)
	m.Users.SetShowStatusBar(false)
	m.Groups.Title = "Groups"
	m.Groups.SetShowStatusBar(false)
	m.Bots.Title = "Bots"
	m.Bots.SetShowStatusBar(false)
	m.updateDelegates()

	ui := setItemStyles(&m)

	// Render modal call overlays centered on top of the UI without altering line count
	if m.CallOverlay.IsModal() {
		ui = overlay.Composite(m.CallOverlay.View(), ui, overlay.Center, overlay.Center, 0, 0)
	}

	return m.Alert.Render(ui)
}

func (m *Model) updateDelegates() {
	usersDelegate := CustomDelegate{Model: m}
	channelsDelegate := CustomDelegate{Model: m}
	groupsDelegate := CustomDelegate{Model: m}
	forumTopicsDelegate := ForumTopicsDelegate{Model: m}
	mainViewDelegate := MessagesDelegate{Model: m}
	// storiesDelegate := StoriesDelegate{Model: m}
	// m.Stories.SetDelegate(storiesDelegate)
	m.All.SetDelegate(usersDelegate)
	m.Users.SetDelegate(usersDelegate)
	m.Bots.SetDelegate(usersDelegate)
	m.Channels.SetDelegate(channelsDelegate)
	m.Groups.SetDelegate(groupsDelegate)
	m.SelectedGroupForumTopics.SetDelegate(forumTopicsDelegate)
	m.ChatUI.SetDelegate(mainViewDelegate)
}
