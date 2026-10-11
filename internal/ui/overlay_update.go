package ui

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kumneger0/cligram/internal/telegram"
	"github.com/kumneger0/cligram/internal/telegram/types"
)

type MemberSearchTickMsg struct {
	Peer  types.Peer
	Query string
	Seq   int
}

func (m *Foreground) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	m.ensureSpinner()
	var cmds []tea.Cmd
	switch msg := message.(type) {
	case spinner.TickMsg:
		if m.storiesLoading || m.isDownloadingStory || m.membersLoading || m.isPaginatingMembers || m.isSearchingMembers || m.isJoining || (m.ModalMode == ModalModeComments && (m.commentsLoading || m.isPostingComment)) {
			var spinCmd tea.Cmd
			m.spinner, spinCmd = m.spinner.Update(msg)
			return m, spinCmd
		}
		return m, nil

	case types.StoriesDownloadStatusMsg:
		m.isDownloadingStory = false
		m.downloadingStoryPeer = ""
		if m.stories != nil {
			for idx, value := range m.stories.Items() {
				if story, ok := value.(types.Stories); ok {
					if story.IsSelected || (msg.Peer.ID != "" && story.UserInfo.PeerID == msg.Peer.ID) {
						story.IsSelected = false
						_ = m.stories.SetItem(idx, story)
					}
				}
			}
		}
		if msg.Err != nil {
			m.Error = msg.Err
			return m, nil
		}
		return m, nil

	case types.GetAllStoriesMsg:
		m.storiesLoading = false
		if msg.Err != nil {
			m.Error = msg.Err
			return m, nil
		}
		m.initStories(msg.Stories)
	case []types.Stories:
		m.storiesLoading = false
		m.initStories(msg)
	case types.AvailableReactions:
		if msg.Err != nil {
			m.Error = msg.Err
			return m, nil
		}
		m.allReactions = msg.Reactions
		m.selectedReactionIndex = 0

	case MemberSearchTickMsg:
		if m.ModalMode == ModalModeGroupMembers && msg.Seq == m.membersSearchSeq && strings.TrimSpace(m.membersInput.Value()) == msg.Query {
			m.isSearchingMembers = true
			if telegram.Cligram != nil && telegram.Cligram.Context() != nil {
				searchCmd := telegram.Cligram.SearchGroupMembers(telegram.Cligram.Context(), msg.Peer, msg.Query)
				return m, tea.Batch(searchCmd, m.spinner.Tick)
			}
		}
		return m, nil

	case types.SearchGroupMembersMsg:
		if m.ModalMode != ModalModeGroupMembers || msg.Query != strings.TrimSpace(m.membersInput.Value()) {
			return m, nil
		}
		m.isSearchingMembers = false
		if msg.Err != nil {
			slog.Error("failed to search group members", "peerID", msg.PeerID, "query", msg.Query, "error", msg.Err.Error())
			return m, nil
		}
		seen := make(map[string]bool)
		var combined []list.Item
		for _, member := range msg.Members {
			if !seen[member.User.PeerID] {
				seen[member.User.PeerID] = true
				combined = append(combined, member)
			}
		}
		for _, member := range m.allMembers {
			val := strings.ToLower(member.FilterValue())
			qLower := strings.ToLower(msg.Query)
			if strings.Contains(val, qLower) && !seen[member.User.PeerID] {
				seen[member.User.PeerID] = true
				combined = append(combined, member)
			}
		}

		if m.membersList != nil {
			m.membersList.SetItems(combined)
		} else {
			mList := list.New(combined, GroupMembersDelegate{Foreground: m}, 10, 10)
			mList.SetShowFilter(false)
			mList.SetShowPagination(false)
			mList.SetShowTitle(false)
			mList.SetShowHelp(false)
			mList.SetShowStatusBar(false)
			m.membersList = &mList
		}
		return m, nil

	case types.GetGroupMembersMsg:
		m.membersLoading = false
		m.isPaginatingMembers = false
		if msg.Err != nil {
			slog.Error("failed to get group members", "peerID", msg.PeerID, "offset", msg.Offset, "error", msg.Err.Error())
			if msg.Offset == 0 {
				m.groupMembersError = msg.Err
			} else {
				m.groupMembersError = fmt.Errorf("failed to load more members: %w", msg.Err)
			}
			return m, nil
		}
		if m.totalMembers == 0 && msg.TotalCount > 0 {
			m.totalMembers = msg.TotalCount
		} else if msg.TotalCount > m.totalMembers {
			m.totalMembers = msg.TotalCount
		}
		if msg.Offset == 0 {
			m.allMembers = msg.Members
			m.membersOffset = len(msg.Members)
			m.hasMoreMembers = len(msg.Members) == 200
			items := m.filterMembers(m.membersInput.Value())
			mList := list.New(items, GroupMembersDelegate{Foreground: m}, 10, 10)
			mList.SetShowFilter(false)
			mList.SetShowPagination(false)
			mList.SetShowTitle(false)
			mList.SetShowHelp(false)
			mList.SetShowStatusBar(false)
			m.membersList = &mList
			return m, nil
		}

		// Subsequent page pagination
		m.allMembers = append(m.allMembers, msg.Members...)
		m.membersOffset += len(msg.Members)
		m.hasMoreMembers = len(msg.Members) == 200
		if m.membersList != nil {
			items := m.filterMembers(m.membersInput.Value())
			m.membersList.SetItems(items)
		}
		return m, nil
	case tea.WindowSizeMsg:
		m.searchResultCombined = list.New([]list.Item{}, SearchDelegate{Foreground: m}, 10, 10)
		m.searchResultCombined.Title = "Search User Result"
		m.searchResultCombined.SetShowStatusBar(false)
		m.searchResultCombined.SetShowFilter(false)
		m.searchResultCombined.SetShowHelp(false)
		m.searchResultCombined.SetShowTitle(false)
		m.searchResultCombined.SetShowPagination(false)
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height
		m.focusedOn = SEARCH
		input := textinput.New()
		input.Placeholder = "Search..."
		input.Prompt = "🔍 "
		input.CharLimit = 256
		m.input = input
		m.input.Focus()
	case tea.KeyMsg:
		model, cmd := m.handleKeyPress(msg, &cmds)
		m = model.(*Foreground)
		cmds = append(cmds, cmd)
	case types.SearchUsersMsg:
		model, cmd := m.handleSearch(msg, &cmds)
		m = model.(*Foreground)
		return m, cmd
	case OpenModalMsg:
		m.ModalMode = msg.ModalMode
		m.AlertTitle = msg.AlertTitle
		m.AlertMessage = msg.AlertMessage
		m.Message = msg.Message
		m.fromPeer = msg.FromPeer
		m.UsersList = msg.UsersList
		m.ChannelsList = msg.ChannelsList
		m.GroupsList = msg.GroupsList
		m.BotsList = msg.BotsList
		if msg.ModalMode == ModalModeSearch {
			m.focusedOn = SEARCH
			m.input.Focus()
		} else if msg.ModalMode == ModalModeForwardMessage {
			m.initForwardPicker(msg)
		} else if msg.ModalMode == ModalModeShowStories {
			m.ensureSpinner()
			if len(msg.Stories) > 0 {
				m.initStories(msg.Stories)
				m.storiesLoading = false
			} else {
				m.storiesLoading = true
				m.stories = nil
				cmds = append(cmds, m.spinner.Tick)
			}
		} else if msg.ModalMode == ModalModeGroupMembers {
			m.ensureSpinner()
			m.groupTitle = msg.GroupTitle
			m.groupPeer = msg.GroupPeer
			m.totalMembers = msg.TotalMembers
			m.membersOffset = 0
			m.hasMoreMembers = true
			m.isPaginatingMembers = false
			m.isSearchingMembers = false
			m.membersSearchSeq = 0
			m.membersLoading = true
			m.groupMembersError = nil
			m.allMembers = nil
			m.membersList = nil
			m.focusedOn = SEARCH
			membersInput := textinput.New()
			membersInput.Placeholder = "Filter members..."
			membersInput.Prompt = "🔍 "
			membersInput.CharLimit = 128
			membersInput.Focus()
			m.membersInput = membersInput
			cmds = append(cmds, m.spinner.Tick)
		} else if msg.ModalMode == ModalModeJoinPreview {
			m.ensureSpinner()
			m.ModalMode = ModalModeJoinPreview
			m.JoinPreview = msg.JoinPreview
			m.isJoining = false
			m.joinError = nil
		} else if msg.ModalMode == ModalModeComments {
			m.ensureSpinner()
			m.ModalMode = ModalModeComments
			m.commentsPost = msg.CommentsPost
			if m.commentsPost == nil {
				m.commentsPost = msg.Message
			}
			m.comments = nil
			m.commentsDiscussionChat = nil
			m.commentsTopMsgID = 0
			m.commentsLoading = true
			m.commentsError = nil
			m.isPostingComment = false
			m.isJoining = false
			m.commentsScrollOffset = 0
			m.commentsCursor = 0
			m.focusedOn = SEARCH

			ta := textarea.New()
			ta.Placeholder = "Write a comment... (Enter to send)"
			ta.Prompt = ""
			ta.ShowLineNumbers = false
			ta.CharLimit = 4096
			ta.SetHeight(4)
			ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
			ta.BlurredStyle.CursorLine = lipgloss.NewStyle()
			ta.Focus()
			m.commentInput = ta
			cmds = append(cmds, m.spinner.Tick)
		}
	case types.DiscussionMessageMsg:
		if m.ModalMode == ModalModeComments {
			m.commentsLoading = false
			if msg.Err != nil {
				m.commentsError = msg.Err
				return m, nil
			}
			m.commentsDiscussionChat = msg.Chat
			m.commentsTopMsgID = msg.TopMsgID
			m.comments = msg.Comments
			m.commentsScrollOffset = 0
			m.commentsCursor = 0
			if m.commentsDiscussionChat == nil {
				m.commentsError = fmt.Errorf("no linked discussion group found")
				return m, nil
			}
			if len(m.comments) > 0 {
				m.focusedOn = LIST
				m.commentInput.Blur()
			} else if !m.commentsDiscussionChat.IsLeft && !m.commentsDiscussionChat.IsMuted {
				m.focusedOn = SEARCH
				m.commentInput.Focus()
			} else {
				m.focusedOn = LIST
				m.commentInput.Blur()
			}
			return m, nil
		}
	case types.JoinChannelMsg:
		if m.ModalMode == ModalModeComments {
			m.isJoining = false
			if msg.Err != nil && !strings.Contains(msg.Err.Error(), "USER_ALREADY_PARTICIPANT") {
				m.commentsError = msg.Err
				return m, nil
			}
			if m.commentsDiscussionChat != nil {
				m.commentsDiscussionChat.IsLeft = false
				if !m.commentsDiscussionChat.IsMuted {
					m.focusedOn = SEARCH
					m.commentInput.Focus()
				}
			}
			return m, nil
		}
	case types.SendMessageMsg:
		if m.ModalMode == ModalModeComments {
			m.isPostingComment = false
			if msg.Err != nil {
				m.commentsError = msg.Err
			}
			return m, nil
		}
	case types.CurrentUserMsg:
		if msg.Err != nil {
			return m, nil
		}
		m.isMePremium = msg.User.Premium
	}

	if m.ModalMode == ModalModeForwardMessage {
		prevVal := m.forwardInput.Value()
		fInput, fCmd := m.forwardInput.Update(message)
		m.forwardInput = fInput
		cmds = append(cmds, fCmd)

		if m.forwardInput.Value() != prevVal {
			query := strings.ToLower(strings.TrimSpace(m.forwardInput.Value()))
			var filtered []list.Item
			for _, item := range m.allForwardDestinations {
				dest, ok := item.(ForwardDestinationItem)
				if !ok {
					continue
				}
				if query == "" || strings.Contains(strings.ToLower(dest.Name), query) {
					filtered = append(filtered, dest)
				}
			}
			m.forwardDestinations.SetItems(filtered)
		}
	} else if m.ModalMode == ModalModeGroupMembers {
		if m.focusedOn == SEARCH {
			prevVal := m.membersInput.Value()
			mInput, mCmd := m.membersInput.Update(message)
			m.membersInput = mInput
			cmds = append(cmds, mCmd)

			if m.membersInput.Value() != prevVal {
				query := m.membersInput.Value()
				filtered := m.filterMembers(query)
				if m.membersList != nil {
					m.membersList.SetItems(filtered)
				} else {
					mList := list.New(filtered, GroupMembersDelegate{Foreground: m}, 10, 10)
					mList.SetShowFilter(false)
					mList.SetShowPagination(false)
					mList.SetShowTitle(false)
					mList.SetShowHelp(false)
					mList.SetShowStatusBar(false)
					m.membersList = &mList
				}

				trimmed := strings.TrimSpace(query)
				if trimmed == "" {
					m.isSearchingMembers = false
				} else if m.groupPeer != nil {
					m.membersSearchSeq++
					seq := m.membersSearchSeq
					peer := *m.groupPeer
					cmds = append(cmds, tea.Tick(300*time.Millisecond, func(t time.Time) tea.Msg {
						return MemberSearchTickMsg{
							Peer:  peer,
							Query: trimmed,
							Seq:   seq,
						}
					}))
				}
			}
		}
	} else if m.ModalMode == ModalModeComments {
		if m.focusedOn == SEARCH && (m.commentsDiscussionChat == nil || (!m.commentsDiscussionChat.IsLeft && !m.commentsDiscussionChat.IsMuted)) {
			isEnter := false
			if kMsg, ok := message.(tea.KeyMsg); ok && kMsg.String() == "enter" {
				isEnter = true
			}
			if !isEnter {
				cInput, cCmd := m.commentInput.Update(message)
				m.commentInput = cInput
				cmds = append(cmds, cCmd)
			}
		}
	} else {
		input, cmd := m.input.Update(message)
		m.input = input
		cmds = append(cmds, cmd)
	}

	if m.UsersList != nil {
		userList, userListCmd := m.UsersList.Update(message)
		m.UsersList = &userList
		cmds = append(cmds, userListCmd)
	}

	if m.ModalMode == ModalModeSearch && m.focusedOn == LIST {
		users, userCmd := m.searchResultCombined.Update(message)
		m.searchResultCombined = users
		cmds = append(cmds, userCmd)
	}

	if m.availableReactions != nil {
		reactions, reactionCmd := m.availableReactions.Update(message)
		m.availableReactions = &reactions
		cmds = append(cmds, reactionCmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Foreground) handleSearch(msg types.SearchUsersMsg, cmds *[]tea.Cmd) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		slog.Error("Failed to search user", "error", msg.Err.Error())
	} else {
		result := msg.Response
		var users []list.Item
		for _, v := range *result {
			var channelOrUserType ChannelOrUserType = USER
			if v.IsBot {
				channelOrUserType = BOT
			}
			users = append(users, SearchResult{
				Name:              v.FirstName,
				IsBot:             v.IsBot,
				PeerID:            v.PeerID,
				AccessHash:        v.AccessHash,
				UnreadCount:       v.UnreadCount,
				ChannelOrUserType: channelOrUserType,
			})
			//TODO:list out channels too
		}
		setTotalSearchResultUsers(msg, m)
		cmd := m.searchResultCombined.SetItems(users)
		*cmds = append(*cmds, cmd)
	}
	return m, nil
}

func (m *Foreground) handleKeyPress(msg tea.KeyMsg, cmdsFromParent *[]tea.Cmd) (tea.Model, tea.Cmd) {
	cmds := *cmdsFromParent
	switch msg.String() {
	case "up", "ctrl+p":
		if m.ModalMode == ModalModeSendReaction {
			if m.selectedReactionIndex >= 8 {
				m.selectedReactionIndex -= 8
			}
		} else if m.ModalMode == ModalModeSearch {
			m.searchResultCombined.CursorUp()
			return m, nil
		} else if m.ModalMode == ModalModeForwardMessage {
			m.forwardDestinations.CursorUp()
			return m, nil
		} else if m.ModalMode == ModalModeShowStories && m.stories != nil {
			m.stories.CursorUp()
			return m, nil
		} else if m.ModalMode == ModalModeGroupMembers && m.membersList != nil {
			m.membersList.CursorUp()
			return m, nil
		} else if m.ModalMode == ModalModeComments {
			if m.focusedOn == LIST {
				if m.commentsCursor > 0 {
					m.commentsCursor--
				}
				return m, nil
			}
			if m.focusedOn == SEARCH && len(m.comments) > 0 && m.commentInput.Line() == 0 {
				m.focusedOn = LIST
				m.commentInput.Blur()
				m.commentsCursor = len(m.comments) - 1
				return m, nil
			}
		}
	case "down", "ctrl+n":
		if m.ModalMode == ModalModeSendReaction {
			if m.selectedReactionIndex+8 < len(m.allReactions) {
				m.selectedReactionIndex += 8
			}
		} else if m.ModalMode == ModalModeSearch {
			m.searchResultCombined.CursorDown()
			return m, nil
		} else if m.ModalMode == ModalModeForwardMessage {
			m.forwardDestinations.CursorDown()
			return m, nil
		} else if m.ModalMode == ModalModeShowStories && m.stories != nil {
			m.stories.CursorDown()
			return m, nil
		} else if m.ModalMode == ModalModeGroupMembers && m.membersList != nil {
			m.membersList.CursorDown()
			return m.checkMembersPagination()
		} else if m.ModalMode == ModalModeComments && m.focusedOn == LIST {
			if m.commentsCursor < len(m.comments)-1 {
				m.commentsCursor++
			} else if m.commentsDiscussionChat != nil && !m.commentsDiscussionChat.IsLeft && !m.commentsDiscussionChat.IsMuted {
				m.focusedOn = SEARCH
				m.commentInput.Focus()
			}
			return m, nil
		}
	case "left":
		if m.ModalMode == ModalModeSendReaction {
			if m.selectedReactionIndex > 0 {
				m.selectedReactionIndex--
			}
		}
	case "right":
		if m.ModalMode == ModalModeSendReaction {
			if m.selectedReactionIndex < len(m.allReactions)-1 {
				m.selectedReactionIndex++
			}
		}
	case "tab":
		if m.ModalMode == ModalModeComments {
			if m.commentsDiscussionChat != nil && !m.commentsDiscussionChat.IsLeft && !m.commentsDiscussionChat.IsMuted {
				if m.focusedOn == SEARCH {
					m.focusedOn = LIST
					m.commentInput.Blur()
				} else {
					m.focusedOn = SEARCH
					m.commentInput.Focus()
				}
			}
			return m, nil
		}
		if m.ModalMode == ModalModeGroupMembers {
			if m.focusedOn == SEARCH {
				m.focusedOn = LIST
				m.membersInput.Blur()
			} else {
				m.focusedOn = SEARCH
				m.membersInput.Focus()
			}
			return m, nil
		}
		if m.focusedOn == SEARCH {
			m.focusedOn = LIST
			m.input.Blur()
		} else {
			m.focusedOn = SEARCH
			m.input.Focus()
		}
	case "j":
		if m.ModalMode == ModalModeComments && m.focusedOn == LIST {
			if m.commentsCursor < len(m.comments)-1 {
				m.commentsCursor++
			} else if m.commentsDiscussionChat != nil && !m.commentsDiscussionChat.IsLeft && !m.commentsDiscussionChat.IsMuted {
				m.focusedOn = SEARCH
				m.commentInput.Focus()
			}
			return m, nil
		}
		if m.ModalMode == ModalModeGroupMembers && m.focusedOn == LIST && m.membersList != nil {
			m.membersList.CursorDown()
			return m.checkMembersPagination()
		}
	case "k":
		if m.ModalMode == ModalModeComments && m.focusedOn == LIST {
			if m.commentsCursor > 0 {
				m.commentsCursor--
			}
			return m, nil
		}
		if m.ModalMode == ModalModeGroupMembers && m.focusedOn == LIST && m.membersList != nil {
			m.membersList.CursorUp()
			return m, nil
		}
	case "i":
		if m.ModalMode == ModalModeComments && m.focusedOn == LIST {
			if m.commentsDiscussionChat != nil && !m.commentsDiscussionChat.IsLeft && !m.commentsDiscussionChat.IsMuted {
				m.focusedOn = SEARCH
				m.commentInput.Focus()
				return m, nil
			}
		}
	case "esc":
		if m.ModalMode == ModalModeComments {
			m.commentsLoading = false
			m.isPostingComment = false
			m.isJoining = false
			return m, func() tea.Msg { return CloseOverlay{} }
		}
		if m.ModalMode == ModalModeAlert {
			m.AlertTitle = ""
			m.AlertMessage = ""
		}
		if m.ModalMode == ModalModeJoinPreview {
			m.isJoining = false
			m.JoinPreview = nil
			m.joinError = nil
		}
		if m.ModalMode == ModalModeGroupMembers {
			m.membersLoading = false
			m.isPaginatingMembers = false
			m.isSearchingMembers = false
		}
		if m.ModalMode == ModalModeShowStories {
			m.storiesLoading = false
			m.isDownloadingStory = false
			if m.stories != nil {
				for idx, value := range m.stories.Items() {
					if story, ok := value.(types.Stories); ok && story.IsSelected {
						story.IsSelected = false
						_ = m.stories.SetItem(idx, story)
					}
				}
			}
		}
		return m, func() tea.Msg { return CloseOverlay{} }
	case "q", "Q":
		if m.ModalMode == ModalModeAlert || m.ModalMode == ModalModeJoinPreview || m.Error != nil {
			m.Error = nil
			m.AlertTitle = ""
			m.AlertMessage = ""
			m.isJoining = false
			m.JoinPreview = nil
			m.joinError = nil
			return m, func() tea.Msg { return CloseOverlay{} }
		}
	case "enter":
		return handleEnterKey(m)
	case "y", "Y":
		if m.ModalMode == ModalModeDeleteMessage {
			closeCommandCMD := func() tea.Msg {
				return CloseOverlay{}
			}
			return m, tea.Batch(closeCommandCMD, func() tea.Msg {
				return MessageDeletionConfirmResponseMsg{yes: true}
			})
		}
	case "n", "N":
		if m.ModalMode == ModalModeDeleteMessage {
			closeCommandCMD := func() tea.Msg {
				return CloseOverlay{}
			}
			return m, tea.Batch(closeCommandCMD, func() tea.Msg {
				return MessageDeletionConfirmResponseMsg{yes: false}
			})
		}
	}

	if m.input.Focused() {
		searchValue := m.input.Value()
		if len(searchValue) >= 3 {
			searchCmd := debouncedSearch(searchValue)
			cmds = append(cmds, searchCmd)
		}
	}
	if m.stories != nil {
		model, cmd := m.stories.Update(msg)
		m.stories = &model
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func handleEnterKey(m *Foreground) (tea.Model, tea.Cmd) {
	if m.ModalMode == ModalModeAlert || m.Error != nil {
		m.Error = nil
		m.AlertTitle = ""
		m.AlertMessage = ""
		return m, func() tea.Msg { return CloseOverlay{} }
	}
	if m.ModalMode == ModalModeJoinPreview {
		if m.isJoining || m.JoinPreview == nil {
			return m, nil
		}
		m.ensureSpinner()
		m.isJoining = true
		m.joinError = nil
		var joinCmd tea.Cmd
		if telegram.Cligram != nil && telegram.Cligram.Context() != nil {
			if m.JoinPreview.Hash != "" {
				joinCmd = telegram.Cligram.ImportChatInvite(telegram.Cligram.Context(), m.JoinPreview.Hash)
			} else if m.JoinPreview.ChannelID != 0 {
				joinCmd = telegram.Cligram.JoinChannel(telegram.Cligram.Context(), m.JoinPreview.ChannelID, m.JoinPreview.AccessHash)
			}
		}
		return m, tea.Batch(joinCmd, m.spinner.Tick)
	}
	if m.ModalMode == ModalModeShowStories {
		if m.stories == nil || len(m.stories.Items()) == 0 {
			return m, nil
		}
		story, ok := m.stories.SelectedItem().(types.Stories)
		if !ok || story.IsSelected {
			return m, nil
		}
		m.ensureSpinner()
		story.IsSelected = true
		m.isDownloadingStory = true
		m.downloadingStoryPeer = story.UserInfo.PeerID

		var batchCmds []tea.Cmd
		batchCmds = append(batchCmds, m.stories.SetItem(m.stories.GlobalIndex(), story))
		batchCmds = append(batchCmds, m.spinner.Tick)
		if telegram.Cligram != nil && telegram.Cligram.Context() != nil {
			batchCmds = append(batchCmds, telegram.Cligram.GetPeerStories(telegram.Cligram.Context(), types.Peer{
				ID:         story.UserInfo.PeerID,
				AccessHash: story.UserInfo.AccessHash,
				ChatType:   types.UserChat,
			}))
		}
		return m, tea.Batch(batchCmds...)
	}
	if m.ModalMode == ModalModeForwardMessage {
		return handleForwardMessage(m)
	}
	if m.ModalMode == ModalModeSendReaction {
		return handleSendReaction(m)
	}
	if m.ModalMode == ModalModeGroupMembers {
		return handleGroupMemberSelection(m)
	}
	if m.ModalMode == ModalModeComments {
		return handleCommentsEnterKey(m)
	}
	if m.ModalMode == ModalModeSearch || m.focusedOn == LIST {
		return handleListSelection(m)
	}
	return m, nil
}

func handleCommentsEnterKey(m *Foreground) (tea.Model, tea.Cmd) {
	if m.commentsDiscussionChat != nil && m.commentsDiscussionChat.IsLeft {
		if m.isJoining {
			return m, nil
		}
		m.ensureSpinner()
		m.isJoining = true
		m.commentsError = nil
		channelID, _ := strconv.ParseInt(m.commentsDiscussionChat.ID, 10, 64)
		accessHash, _ := strconv.ParseInt(m.commentsDiscussionChat.AccessHash, 10, 64)
		var joinCmd tea.Cmd
		if telegram.Cligram != nil && telegram.Cligram.Context() != nil {
			joinCmd = telegram.Cligram.JoinChannel(telegram.Cligram.Context(), channelID, accessHash)
		}
		return m, tea.Batch(joinCmd, m.spinner.Tick)
	}

	if m.focusedOn == LIST {
		if m.commentsDiscussionChat != nil && !m.commentsDiscussionChat.IsLeft && !m.commentsDiscussionChat.IsMuted {
			m.focusedOn = SEARCH
			m.commentInput.Focus()
			return m, nil
		}
	}

	if m.commentsDiscussionChat != nil && !m.commentsDiscussionChat.IsLeft && !m.commentsDiscussionChat.IsMuted {
		text := strings.TrimSpace(m.commentInput.Value())
		if text == "" || m.isPostingComment {
			return m, nil
		}
		m.ensureSpinner()
		m.isPostingComment = true
		m.commentsError = nil

		discChat := m.commentsDiscussionChat
		peer := types.Peer{
			ID:         discChat.ID,
			AccessHash: discChat.AccessHash,
			ChatType:   types.GroupChat,
		}
		if discChat.IsBroadcast {
			peer.ChatType = types.ChannelChat
		}
		topMsgID := m.commentsTopMsgID
		randID := int(time.Now().UnixNano() & 0x7FFFFFFF)

		req := types.SendMessageRequest{
			Peer:             peer,
			Message:          text,
			IsReply:          topMsgID != 0,
			ReplyToMessageID: strconv.Itoa(topMsgID),
			TopMsgID:         &topMsgID,
			RandID:           randID,
		}

		m.comments = append(m.comments, types.FormattedMessage{
			ID:       randID,
			Sender:   "You",
			Content:  text,
			Date:     time.Now(),
			IsFromMe: true,
		})
		m.commentsCursor = len(m.comments) - 1
		m.commentInput.Reset()

		var sendCmd tea.Cmd
		if telegram.Cligram != nil && telegram.Cligram.Context() != nil {
			sendCmd = telegram.Cligram.SendMessage(telegram.Cligram.Context(), req)
		}
		return m, tea.Batch(sendCmd, m.spinner.Tick)
	}

	return m, nil
}

func handleGroupMemberSelection(m *Foreground) (tea.Model, tea.Cmd) {
	if m.membersList == nil || len(m.membersList.Items()) == 0 {
		return m, nil
	}
	selectedItem := m.membersList.SelectedItem()
	if selectedItem == nil {
		return m, nil
	}
	member, ok := selectedItem.(types.GroupMemberInfo)
	if !ok {
		return m, nil
	}
	userCopy := member.User
	return m, tea.Batch(
		func() tea.Msg { return CloseOverlay{} },
		func() tea.Msg { return SelectSearchedUserResult{user: &userCopy} },
	)
}

func (m *Foreground) filterMembers(query string) []list.Item {
	trimmed := strings.ToLower(strings.TrimSpace(query))
	var filtered []list.Item
	for _, item := range m.allMembers {
		val := strings.ToLower(item.FilterValue())
		if trimmed == "" || strings.Contains(val, trimmed) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func (m *Foreground) checkMembersPagination() (tea.Model, tea.Cmd) {
	if !m.hasMoreMembers || m.isPaginatingMembers || m.groupPeer == nil || m.membersInput.Value() != "" {
		return m, nil
	}
	if m.membersList != nil && m.membersList.Index() >= len(m.membersList.Items())-5 {
		m.isPaginatingMembers = true
		m.groupMembersError = nil
		if telegram.Cligram != nil && telegram.Cligram.Context() != nil {
			fetchCmd := telegram.Cligram.GetGroupMembers(telegram.Cligram.Context(), *m.groupPeer, m.membersOffset)
			return m, tea.Batch(fetchCmd, m.spinner.Tick)
		}
	}
	return m, nil
}

func handleSendReaction(m *Foreground) (tea.Model, tea.Cmd) {
	if len(m.allReactions) == 0 {
		return m, nil
	}

	if m.selectedReactionIndex < 0 || m.selectedReactionIndex >= len(m.allReactions) {
		return m, nil
	}

	selectedReaction := m.allReactions[m.selectedReactionIndex]

	if selectedReaction.AvailableReaction.Premium && !m.isMePremium {
		m.Error = fmt.Errorf("this is a premium reaction, upgrade to use it")
		return m, nil
	}

	return m, tea.Batch(
		func() tea.Msg { return CloseOverlay{} },
		func() tea.Msg { return types.SendReactionMsg{Reaction: selectedReaction} },
	)
}

func handleForwardMessage(m *Foreground) (tea.Model, tea.Cmd) {
	var selectedDest list.Item
	if len(m.forwardDestinations.Items()) > 0 {
		selectedDest = m.forwardDestinations.SelectedItem()
	} else if m.UsersList != nil {
		selectedDest = m.UsersList.SelectedItem()
	}
	if selectedDest == nil {
		return m, nil
	}

	var from list.Item
	if m.fromPeer != nil {
		from = *m.fromPeer
	}

	return m, tea.Batch(
		func() tea.Msg { return CloseOverlay{} },
		func() tea.Msg {
			return ForwardMsg{
				msg:      m.Message,
				receiver: &selectedDest,
				fromPeer: &from,
			}
		},
	)
}

func (m *Foreground) initForwardPicker(msg OpenModalMsg) {
	m.focusedOn = SEARCH
	forwardInput := textinput.New()
	forwardInput.Placeholder = "Forward to..."
	forwardInput.Prompt = "✈️  "
	forwardInput.CharLimit = 256
	forwardInput.Focus()
	m.forwardInput = forwardInput

	var destinations []list.Item
	seen := make(map[string]bool)

	addDest := func(it list.Item, name, peerID, accessHash string, t ChannelOrUserType) {
		if peerID != "" && seen[peerID] {
			return
		}
		if peerID != "" {
			seen[peerID] = true
		}
		destinations = append(destinations, ForwardDestinationItem{
			OriginalItem:      it,
			Name:              name,
			PeerID:            peerID,
			AccessHash:        accessHash,
			ChannelOrUserType: t,
		})
	}

	if msg.UsersList != nil {
		for _, it := range msg.UsersList.Items() {
			if u, ok := it.(types.UserInfo); ok {
				destType := USER
				if u.IsBot {
					destType = BOT
				}
				name := u.FirstName
				if u.LastName != "" {
					name += " " + u.LastName
				}
				addDest(it, name, u.PeerID, u.AccessHash, destType)
			}
		}
	}
	if msg.BotsList != nil {
		for _, it := range msg.BotsList.Items() {
			if u, ok := it.(types.UserInfo); ok {
				name := u.FirstName
				if u.LastName != "" {
					name += " " + u.LastName
				}
				addDest(it, name, u.PeerID, u.AccessHash, BOT)
			}
		}
	}
	if msg.ChannelsList != nil {
		for _, it := range msg.ChannelsList.Items() {
			if c, ok := it.(types.ChannelInfo); ok {
				addDest(it, c.ChannelTitle, c.ID, c.AccessHash, CHANNEL)
			}
		}
	}
	if msg.GroupsList != nil {
		for _, it := range msg.GroupsList.Items() {
			if g, ok := it.(types.ChannelInfo); ok {
				addDest(it, g.ChannelTitle, g.ID, g.AccessHash, GROUP)
			}
		}
	}

	m.allForwardDestinations = destinations
	destList := list.New(destinations, SearchDelegate{Foreground: m}, 10, 10)
	destList.SetShowTitle(false)
	destList.SetShowFilter(false)
	destList.SetShowStatusBar(false)
	destList.SetShowPagination(false)
	destList.SetShowHelp(false)
	m.forwardDestinations = destList
}

func handleListSelection(m *Foreground) (tea.Model, tea.Cmd) {
	if m.ModalMode == ModalModeSearch {
		query := strings.TrimSpace(m.input.Value())
		if parsed := ParseTelegramLink(query); parsed != nil {
			return m, tea.Batch(
				func() tea.Msg { return CloseOverlay{} },
				func() tea.Msg { return RouteParsedLinkMsg{Link: parsed} },
			)
		}
	}

	selectedUser := m.searchResultCombined.SelectedItem()
	if selectedUser == nil {
		return m, nil
	}

	user, ok := selectedUser.(SearchResult)
	if !ok {
		return m, nil
	}

	result := SelectSearchedUserResult{}

	switch user.ChannelOrUserType {
	case CHANNEL:
		result.channel = findChannel(user.PeerID, m.SearchResultChannels)
	case BOT:
		result.Bot = findUser(user.PeerID, m.searchResultUsers)
	case USER:
		result.user = findUser(user.PeerID, m.searchResultUsers)
	case GROUP:
		result.group = findChannel(user.PeerID, m.SearchResultChannels)
	}

	return m, tea.Batch(
		func() tea.Msg { return result },
		func() tea.Msg { return CloseOverlay{} },
	)
}

func findChannel(peerID string, channels []types.ChannelInfo) *types.ChannelInfo {
	for _, v := range channels {
		if v.ID == peerID {
			c := v
			return &c
		}
	}
	return nil
}

func findUser(peerID string, users []types.UserInfo) *types.UserInfo {
	for _, v := range users {
		if v.PeerID == peerID {
			u := v
			return &u
		}
	}
	return nil
}

func (m *Foreground) initStories(stories []types.Stories) {
	if len(stories) == 0 {
		m.stories = nil
		return
	}
	var storiesToDisplay []list.Item
	for _, v := range stories {
		storiesToDisplay = append(storiesToDisplay, v)
	}
	stList := list.New(storiesToDisplay, StoriesDelegate{Foreground: m}, 10, 10)
	stList.SetShowFilter(false)
	stList.SetShowPagination(false)
	stList.SetShowTitle(false)
	stList.SetShowHelp(false)
	stList.SetShowStatusBar(false)
	m.stories = &stList
}
