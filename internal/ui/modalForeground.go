package ui

import (
	"fmt"
	"io"
	"strings"
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
	user    *rpc.UserInfo
	channel *rpc.ChannelAndGroupInfo
	group   *rpc.ChannelAndGroupInfo
}

type CloseOverlay struct{}

type ModalMode string

const (
	ModalModeSearch         ModalMode = "SEARCH"
	ModalModeForwardMessage ModalMode = "FORWARD_MESSAGE"
	ModalModeDeleteMessage  ModalMode = "DELETE_MESSAGE"
)

type OpenModalMsg struct {
	ModalMode ModalMode
	FromPeer  *list.Item
	Message   *rpc.FormattedMessage
	UsersList *list.Model
}

type ForwardMsg struct {
	msg      *rpc.FormattedMessage
	reciever *list.Item
	fromPeer *list.Item
}

type Foreground struct {
	windowWidth          int
	windowHeight         int
	input                textinput.Model
	searchResultCombined list.Model
	focusedOn            focusState
	searchResultUsers    []rpc.UserInfo
	SearchResultChannels []rpc.ChannelAndGroupInfo
	ModalMode            ModalMode
	UsersList            *list.Model
	Message              *rpc.FormattedMessage
	fromPeer             *list.Item
}

func (f Foreground) Init() tea.Cmd {
	f.focusedOn = SEARCH
	return nil
}

type MessageDeletionConfrimResponseMsg struct {
	yes bool
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
			if m.ModalMode == ModalModeForwardMessage {
				if m.UsersList != nil {
					selectedUser := m.UsersList.SelectedItem()

					closeCommandCMD := func() tea.Msg {
						return CloseOverlay{}
					}

					var from list.Item
					if m.fromPeer != nil {
						from = *m.fromPeer
					}

					if selectedUser != nil {
						return m, tea.Batch(closeCommandCMD, func() tea.Msg {
							return ForwardMsg{
								msg:      m.Message,
								reciever: &selectedUser,
								fromPeer: &from,
							}
						})
					}
				}
			}
			if m.focusedOn == LIST {
				selectedUser := m.searchResultCombined.SelectedItem()
				if selectedUser != nil {
					user, ok := selectedUser.(SearchResult)
					if ok {
						closeCommandCMD := func() tea.Msg {
							return CloseOverlay{}
						}

						var channel *rpc.ChannelAndGroupInfo = nil
						if user.ChannelOrUserType == CHANNEL {
							for _, v := range m.SearchResultChannels {
								if v.ChannelID == user.PeerID {
									channel = &v
								}
							}
						}

						var u *rpc.UserInfo = nil
						if user.ChannelOrUserType == USER {
							for _, v := range m.searchResultUsers {
								if v.PeerID == user.PeerID {
									u = &v
								}
							}
						}

						var group *rpc.ChannelAndGroupInfo = nil
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
									group: group,
									user:  nil,
								}
							}
							return nil
						}
						return m, tea.Batch(cmd, closeCommandCMD)
					}
				}
			}
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

func setTotalSearchResultUsers(searchMsg rpc.SearchUserMsg, m *Foreground) {
	m.searchResultUsers = searchMsg.Response.Result.Users
	m.SearchResultChannels = searchMsg.Response.Result.Channels
}

func (f Foreground) View() string {

	foreStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder(), true).
		BorderForeground(lipgloss.Color("6")).
		Padding(0, 1)

	if f.ModalMode == ModalModeForwardMessage {
		title := "Forward Message"
		boldStyle := lipgloss.NewStyle().Bold(true)
		title = boldStyle.Render(title)
		f.UsersList.SetShowFilter(false)
		f.UsersList.SetShowStatusBar(false)
		f.UsersList.SetShowTitle(false)
		f.UsersList.SetShowHelp(false)
		f.UsersList.SetWidth(f.windowWidth / 3)
		f.UsersList.SetHeight(f.windowHeight / 2)
		content := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Render(f.UsersList.View())
		layout := lipgloss.JoinVertical(lipgloss.Left, title, content)
		return foreStyle.Render(layout)
	}
	if f.ModalMode == ModalModeDeleteMessage {
		title := "Delete Message"

		yes := lipgloss.NewStyle().
			Foreground(lipgloss.Color("fff")).
			Background(lipgloss.Color("#25D366")).
			Render("Y")

		no := lipgloss.NewStyle().
			Foreground(lipgloss.Color("fff")).
			Background(lipgloss.Color("#dc2626")).
			Render("N")

		contentStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("161b22")).
			Background(lipgloss.Color("fff")).
			Padding(1, 2)

		var content strings.Builder
		content.WriteString("Are You Sure You want to delete this message \n")
		content.WriteString("Press")
		content.WriteString(" ")
		content.WriteString(yes)
		content.WriteString(" to confirm")
		content.WriteString(" or ")
		content.WriteString(no)
		content.WriteString(" to cancel")

		contentString := contentStyle.Render(content.String())

		boldStyle := lipgloss.NewStyle().Bold(true)
		title = boldStyle.Render(title)
		layout := lipgloss.JoinVertical(lipgloss.Left, title, contentString)
		return foreStyle.Render(layout)
	}

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
