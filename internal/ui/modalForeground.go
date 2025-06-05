package ui

import (
	"fmt"
	"io"
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

type ChannelOrUserType string

const (
	CHANNEL ChannelOrUserType = "CHANNEL"
	GROUP   ChannelOrUserType = "GROUP"
	USER    ChannelOrUserType = "USER"
)

type SearchDelegate struct {
	list.DefaultDelegate
}

func (d SearchDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	var title string

	entry, ok := item.(SearchResult)
	if ok {
		title = entry.FilterValue()
		switch entry.ChannelOrUserType {
		case USER:
			title = "👤 " + title
		case GROUP:
			title = "👥 " + title
		case CHANNEL:
			title = "📢 " + title
		}
	} else {
		return
	}

	str := lipgloss.NewStyle().Width(50).Render(title)
	if index == m.Index() {
		fmt.Fprint(w, selectedStyle.Render(" "+str+" "))
	} else {
		fmt.Fprint(w, normalStyle.Render(" "+str+" "))
	}
}

type SearchResult struct {
	Name              string
	IsBot             bool
	PeerID            string
	AccessHash        string
	UnreadCount       int
	ChannelOrUserType ChannelOrUserType
}

func (s SearchResult) Title() string {
	return s.Name
}

func (s SearchResult) FilterValue() string {
	return s.Name
}

type SelectSearchedUserResult struct {
	user    *UserInfo
	channel *ChannelAndGroupInfo
	group   *ChannelAndGroupInfo
}

type CloseOverlay struct{}

type Foreground struct {
	windowWidth          int
	windowHeight         int
	input                textinput.Model
	searchResultCombined list.Model
	focusedOn            focusState
	searchResultUsers    []UserInfo
	SearchResultChannels []ChannelAndGroupInfo
}

func (f Foreground) Init() tea.Cmd {
	f.focusedOn = SEARCH
	return nil
}

var debouncedSearch = Debounce(func(args ...interface{}) tea.Msg {
	query := args[0].(string)
	return rpc.RpcClient.Search(query)
}, 300*time.Millisecond)

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
			if m.focusedOn == LIST {
				selectedUser := m.searchResultCombined.SelectedItem()
				if selectedUser != nil {
					user, ok := selectedUser.(SearchResult)
					if ok {
						closeCommandCMD := func() tea.Msg {
							return CloseOverlay{}
						}

						var channel *ChannelAndGroupInfo = nil
						if user.ChannelOrUserType == CHANNEL {
							for _, v := range m.SearchResultChannels {
								if v.ChannelID == user.PeerID {
									channel = &v
								}
							}
						}

						var u *UserInfo = nil
						if user.ChannelOrUserType == USER {
							for _, v := range m.searchResultUsers {
								if v.PeerID == user.PeerID {
									u = &v
								}
							}
						}

						var group *ChannelAndGroupInfo = nil
						if user.ChannelOrUserType == GROUP {
							for _, v := range m.SearchResultChannels {
								if v.ChannelID == user.PeerID {
									group = &v
								}
							}
						}

						cmd := func() tea.Msg {
							if u != nil {
								return SelectSearchedUserResult{
									user:    u,
									channel: nil,
								}
							}
							if channel != nil {
								return SelectSearchedUserResult{
									channel: channel,
									user:    nil,
								}

							}
							if group != nil {
								return SelectSearchedUserResult{
									group:   group,
									user:    nil,
								}
							}
							return nil
						}
						return m, tea.Batch(cmd, closeCommandCMD)
					}
				}
			}
		}

		if m.input.Focused() {
			searchValue := m.input.Value()
			if len(searchValue) >= 3 {
				searchCmd := debouncedSearch(searchValue)
				cmds = append(cmds, searchCmd)
			}
		}
	case rpc.SearchUserMsg:
		if msg.Err != nil {
			//TODO:show error message here
		} else {
			// Wait for the message and return it
			result := msg.Response.Result
			var users []list.Item
			for _, v := range result.Users {
				users = append(users, SearchResult{
					Name:              v.FirstName,
					IsBot:             v.IsBot,
					PeerID:            v.PeerID,
					AccessHash:        v.AccessHash,
					UnreadCount:       v.UnreadCount,
					ChannelOrUserType: USER,
				})
			}

			for _, v := range result.Channels {
				var channelOrGroup ChannelOrUserType
				if v.IsBroadcast {
					channelOrGroup = CHANNEL
				} else {
					channelOrGroup = GROUP
				}
				users = append(users, SearchResult{
					Name:              v.ChannelTitle,
					IsBot:             false,
					PeerID:            v.ChannelID,
					AccessHash:        v.AccessHash,
					UnreadCount:       v.UnreadCount,
					ChannelOrUserType: channelOrGroup,
				})
			}
			setTotalSearchResultUsers(msg, m)
			m.searchResultCombined.SetItems(users)
		}
	}

	input, cmd := m.input.Update(message)
	m.input = input

	cmds = append(cmds, cmd)
	if m.focusedOn == LIST {
		users, userCmd := m.searchResultCombined.Update(message)
		m.searchResultCombined = users
		cmds = append(cmds, userCmd)
	}

	return m, tea.Batch(cmds...)
}

func setTotalSearchResultUsers(searchMsg rpc.SearchUserMsg, m *Foreground) {
	var users []UserInfo
	for _, v := range searchMsg.Response.Result.Users {
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
	var channels []ChannelAndGroupInfo
	for _, v := range searchMsg.Response.Result.Channels {
		channels = append(channels, ChannelAndGroupInfo{
			ChannelTitle:      v.ChannelTitle,
			Username:          v.Username,
			ChannelID:         v.ChannelID,
			AccessHash:        v.AccessHash,
			IsCreator:         v.IsCreator,
			IsBroadcast:       v.IsBroadcast,
			ParticipantsCount: v.ParticipantsCount,
			UnreadCount:       v.UnreadCount,
		})
	}
	m.searchResultUsers = users
	m.SearchResultChannels = channels
}

func (f Foreground) View() string {
	foreStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder(), true).
		BorderForeground(lipgloss.Color("6")).
		Padding(0, 1)

	boldStyle := lipgloss.NewStyle().Bold(true)
	title := boldStyle.Render("Search")
	content := getSearchView(f)
	var border lipgloss.Border

	if f.focusedOn == SEARCH {
		border = lipgloss.NormalBorder()
	} else {
		border = lipgloss.DoubleBorder()
	}

	searchResult := lipgloss.NewStyle().Border(border).Render(f.searchResultCombined.View())
	layout := lipgloss.JoinVertical(lipgloss.Left, title, content, searchResult)

	return foreStyle.Render(layout)
}

func getSearchView(m Foreground) string {
	var border lipgloss.Border

	if m.focusedOn == LIST {
		border = lipgloss.NormalBorder()
	} else {
		border = lipgloss.DoubleBorder()
	}

	textViewString := lipgloss.NewStyle().Width(m.windowWidth/3).Height(5).Padding(0, 1).Border(border).Render(m.input.View())
	return textViewString
}
