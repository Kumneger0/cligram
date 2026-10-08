package ui

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kumneger0/cligram/internal/telegram"
	"github.com/kumneger0/cligram/internal/telegram/types"
)

func (m *Foreground) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := message.(type) {
	case types.StoriesDownloadStatusMsg:
		if msg.Err != nil {
			//TODO: display error messages
			return m, nil
		}

		var storyToUpdate types.Stories
		var index = -1
		if !msg.Done {
			return m, nil
		}
		for idx, value := range m.stories.Items() {
			if story, ok := value.(types.Stories); ok && story.UserInfo.PeerID == msg.Peer.ID {
				storyToUpdate = story
				index = idx
				break
			}
		}

		if index != -1 {
			storyToUpdate.IsSelected = false
			return m, m.stories.SetItem(index, storyToUpdate)
		}

	case []types.Stories:
		if len(msg) == 0 {
			return m, nil
		}
		var storiesToDisplay []list.Item
		for _, v := range msg {
			storiesToDisplay = append(storiesToDisplay, v)
		}
		stories := list.New(storiesToDisplay, StoriesDelegate{Foreground: m}, 10, 10)
		m.stories = &stories
		m.stories.SetShowFilter(false)
		m.stories.SetShowPagination(false)
		m.stories.SetShowTitle(false)
		m.stories.SetShowHelp(false)
		m.stories.SetShowStatusBar(false)
	case types.AvailableReactions:
		if msg.Err != nil {
			m.Error = msg.Err
			return m, nil
		}
		m.allReactions = msg.Reactions
		m.selectedReactionIndex = 0
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

	if m.focusedOn == LIST {
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
		if m.focusedOn == SEARCH {
			m.focusedOn = LIST
			m.input.Blur()
		} else {
			m.focusedOn = SEARCH
			m.input.Focus()
		}
	case "esc":
		return m, func() tea.Msg { return CloseOverlay{} }
	case "q", "Q":
		if m.Error != nil {
			m.Error = nil
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
	if m.ModalMode == ModalModeShowStories {
		story, ok := m.stories.SelectedItem().(types.Stories)
		if !ok {
			return m, nil
		}
		story.IsSelected = true
		return m, tea.Batch(m.stories.SetItem(m.stories.GlobalIndex(), story), telegram.Cligram.GetPeerStories(telegram.Cligram.Context(), types.Peer{
			ID:         story.UserInfo.PeerID,
			AccessHash: story.UserInfo.AccessHash,
			ChatType:   types.UserChat,
		}),
		)
	}
	if m.ModalMode == ModalModeForwardMessage {
		return handleForwardMessage(m)
	}
	if m.ModalMode == ModalModeSendReaction {
		return handleSendReaction(m)
	}
	if m.ModalMode == ModalModeSearch || m.focusedOn == LIST {
		return handleListSelection(m)
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
