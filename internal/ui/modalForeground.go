package ui

import (
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kumneger0/cligram/internal/rpc"
)

type focusState string

const (
	SEARCH focusState = "INPUT"
	LIST   focusState = "LIST"
)

type Foreground struct {
	windowWidth      int
	windowHeight     int
	input            textinput.Model
	searchUserResult list.Model
	focusedOn        focusState
}

func (f Foreground) Init() tea.Cmd {
	f.focusedOn = SEARCH
	return nil
}

func (m *Foreground) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.searchUserResult = list.New([]list.Item{}, CustomDelegate{}, 10, 10)

		m.searchUserResult.Title = "Search User Result"
		m.searchUserResult.SetShowStatusBar(false)
		m.searchUserResult.SetShowFilter(false)
		m.searchUserResult.SetShowStatusBar(false)
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height
		input := textinput.New()
		input.Placeholder = "Search..."
		input.Prompt = "> "
		input.CharLimit = 256
		m.input = input
		m.input.Focus()
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			if m.focusedOn == SEARCH {
				m.focusedOn = LIST
				m.input.Blur()
			} else {
				m.focusedOn = SEARCH
			}
		}
		if m.input.Focused() {
			searchValue := m.input.Value()
			if len(searchValue) >= 3 {
				searchUsersCmd := tea.Tick(time.Millisecond*300, func(t time.Time) tea.Msg {
					return rpc.RpcClient.Search(searchValue)
				})
				cmds = append(cmds, searchUsersCmd)
			}
		}
	case rpc.SearchUserMsg:
		if msg.Err != nil {
			//TODO:show error message here
		} else {
			result := msg.Response.Result
			var users []list.Item
			for _, v := range result {
				users = append(users, UserInfo{
					FirstName:   v.FirstName,
					IsBot:       v.IsBot,
					PeerID:      v.PeerID,
					AccessHash:  v.AccessHash,
					UnreadCount: v.UnreadCount,
					LastSeen:    LastSeen(v.LastSeen),
					IsOnline:    v.IsOnline,
				})
			}
			m.searchUserResult.SetItems(users)
		}
	}

	input, cmd := m.input.Update(message)
	m.input = input

	users, userCmd := m.searchUserResult.Update(message)
	m.searchUserResult = users

	cmds = append(cmds, cmd, userCmd)
	return m, tea.Batch(cmds...)
}

func (f Foreground) View() string {
	foreStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder(), true).
		BorderForeground(lipgloss.Color("6")).
		Padding(0, 1)

	boldStyle := lipgloss.NewStyle().Bold(true)
	title := boldStyle.Render("Search")
	content := getSearchView(f)
	searchResult := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Render(f.searchUserResult.View())
	layout := lipgloss.JoinVertical(lipgloss.Left, title, content, searchResult)

	return foreStyle.Render(layout)
}

func getSearchView(m Foreground) string {

	var border lipgloss.Border

	if m.focusedOn == SEARCH {
		border = lipgloss.NormalBorder()
	} else {
		border = lipgloss.DoubleBorder()
	}

	textViewString := lipgloss.NewStyle().Width(m.windowWidth/3).Height(5).Padding(0, 1).Border(border).Render(m.input.View())
	return textViewString
}
