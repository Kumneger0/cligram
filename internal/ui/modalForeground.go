package ui

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kumneger0/cligram/internal/telegram"
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
	BOT     ChannelOrUserType = "BOT"
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
		case BOT:
			title = "🤖 " + title
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
	user    *telegram.UserInfo
	channel *telegram.ChannelAndGroupInfo
	group   *telegram.ChannelAndGroupInfo
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
	Message   *telegram.FormattedMessage
	UsersList *list.Model
}

type ForwardMsg struct {
	msg      *telegram.FormattedMessage
	receiver *list.Item
	fromPeer *list.Item
}

type Foreground struct {
	windowWidth          int
	windowHeight         int
	input                textinput.Model
	searchResultCombined list.Model
	focusedOn            focusState
	searchResultUsers    []telegram.UserInfo
	SearchResultChannels []telegram.ChannelAndGroupInfo
	ModalMode            ModalMode
	UsersList            *list.Model
	Message              *telegram.FormattedMessage
	fromPeer             *list.Item
}

func (f Foreground) Init() tea.Cmd {
	return nil
}

type MessageDeletionConfrimResponseMsg struct {
	yes bool
}

var debouncedSearch = Debounce(func(args ...any) tea.Msg {
	query := args[0].(string)
	result := telegram.Cligram.SearchUsers(query)
	return result
}, 300*time.Millisecond)

func setTotalSearchResultUsers(searchMsg telegram.SearchUserMsg, m *Foreground) {
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

func (m Model) GetUserAccessHashFromModel(userID int64) (telegram.UserInfo, error) {
	var userInfo telegram.UserInfo
	for _, value := range m.Users.Items() {
		if user, ok := value.(telegram.UserInfo); ok && user.PeerID == strconv.FormatInt(userID, 10) {
			userInfo = user
		}
	}
	return userInfo, nil
}
