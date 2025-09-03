package ui

import (
	"log/slog"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kumneger0/cligram/internal/telegram/types"
)

func (m *Foreground) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.searchResultCombined = list.New([]list.Item{}, SearchDelegate{}, 10, 10)
		m.searchResultCombined.Title = "Search User Result"
		m.searchResultCombined.SetShowStatusBar(false)
		m.searchResultCombined.SetShowFilter(false)
		m.searchResultCombined.SetShowStatusBar(false)
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
		if msg.ModalMode == ModalModeSearch {
			m.focusedOn = SEARCH
		}
	}

	input, cmd := m.input.Update(message)
	m.input = input

	if m.UsersList != nil {
		userList, userListCmd := m.UsersList.Update(message)
		m.UsersList = &userList
		cmds = append(cmds, userListCmd)
	}

	cmds = append(cmds, cmd)
	if m.focusedOn == LIST {
		users, userCmd := m.searchResultCombined.Update(message)
		m.searchResultCombined = users
		cmds = append(cmds, userCmd)
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
	case "tab":
		if m.focusedOn == SEARCH {
			m.focusedOn = LIST
			m.input.Blur()
		} else {
			m.focusedOn = SEARCH
			m.input.Focus()
		}
	case "enter":
		return handleEnterKey(m)
	case "y", "Y":
		if m.ModalMode == ModalModeDeleteMessage {
			closeCommandCMD := func() tea.Msg {
				return CloseOverlay{}
			}
			return m, tea.Batch(closeCommandCMD, func() tea.Msg {
				return MessageDeletionConfrimResponseMsg{yes: true}
			})
		}
	case "n", "N":
		if m.ModalMode == ModalModeDeleteMessage {
			closeCommandCMD := func() tea.Msg {
				return CloseOverlay{}
			}
			return m, tea.Batch(closeCommandCMD, func() tea.Msg {
				return MessageDeletionConfrimResponseMsg{yes: false}
			})
		}
	}

	if m.input.Focused() {
		searchValue := m.input.Value()
		if len(searchValue) >= 3 {
			searchCmd := debouncedSearch(searchValue)
			*cmdsFromParent = append(*cmdsFromParent, searchCmd)
		}
	}
	return m, tea.Batch(cmds...)
}

func handleEnterKey(m *Foreground) (tea.Model, tea.Cmd) {
	if m.ModalMode == ModalModeForwardMessage {
		return handleForwardMessage(m)
	}

	if m.focusedOn == LIST {
		return handleListSelection(m)
	}

	return m, nil
}

func handleForwardMessage(m *Foreground) (tea.Model, tea.Cmd) {
	if m.UsersList == nil {
		return m, nil
	}

	selectedUser := m.UsersList.SelectedItem()
	if selectedUser == nil {
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
				receiver: &selectedUser,
				fromPeer: &from,
			}
		},
	)
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
	case USER, BOT:
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
			return &v
		}
	}
	return nil
}

func findUser(peerID string, users []types.UserInfo) *types.UserInfo {
	for _, v := range users {
		if v.PeerID == peerID {
			return &v
		}
	}
	return nil
}
