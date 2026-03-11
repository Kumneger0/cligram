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
	"github.com/kumneger0/cligram/internal/telegram/types"
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
	*Foreground
}

func (d SearchDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
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

	width := 20
	if d.Foreground != nil {
		width = max(20, d.Foreground.windowWidth/3)
	}
	str := lipgloss.NewStyle().Width(width).Render(title)
	if index == m.Index() {
		fmt.Fprint(w, selectedStyle.Render(" "+str+" "))
	} else {
		fmt.Fprint(w, normalStyle.Render(" "+str+" "))
	}
}

type StoriesDelegate struct {
	list.DefaultDelegate
	*Foreground
}

func (s StoriesDelegate) Height() int {
	return 1
}

func (s StoriesDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (s StoriesDelegate) Spacing() int {
	return 0
}

func (s StoriesDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	allStoryItems := m.Items()
	if len(allStoryItems) == 0 {
		return
	}

	currentItem, ok := allStoryItems[index].(types.Stories)
	if !ok {
		return
	}

	title := item.FilterValue()
	loading := currentItem.IsSelected

	var style lipgloss.Style
	if index == m.Index() {
		style = selectedStyle
	} else {
		style = normalStyle
	}

	if loading {
		loadingText := lipgloss.NewStyle().Foreground(DefaultTheme.AccentColor).Render(" loading...")
		fmt.Fprint(w, style.Render(" "+title+loadingText))
	} else {
		fmt.Fprint(w, style.Render(" "+title+""))
	}
}

type ReactionsDelegate struct {
	list.DefaultDelegate
	*Foreground
}

func (r ReactionsDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

func (r ReactionsDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	var title string

	entry, ok := item.(types.Reaction)
	if ok {
		title = entry.FilterValue()
	} else {
		return
	}

	width := 20
	if r.Foreground != nil {
		width = max(20, r.Foreground.windowWidth/3)
	}
	str := lipgloss.NewStyle().Width(width).Render(title)
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
	Bot     *types.UserInfo
	user    *types.UserInfo
	channel *types.ChannelInfo
	group   *types.ChannelInfo
}

type CloseOverlay struct{}

type ModalMode string

const (
	ModalModeSearch         ModalMode = "SEARCH"
	ModalModeForwardMessage ModalMode = "FORWARD_MESSAGE"
	ModalModeDeleteMessage  ModalMode = "DELETE_MESSAGE"
	ModalModeShowStories    ModalMode = "SHOW_STORIES"
	ModalModeSendReaction   ModalMode = "SEND_REACTION"
)

type OpenModalMsg struct {
	ModalMode ModalMode
	FromPeer  *list.Item
	Message   *types.FormattedMessage
	UsersList *list.Model
}

type ForwardMsg struct {
	msg      *types.FormattedMessage
	receiver *list.Item
	fromPeer *list.Item
}

type Foreground struct {
	Error                 error
	windowWidth           int
	windowHeight          int
	input                 textinput.Model
	searchResultCombined  list.Model
	focusedOn             focusState
	searchResultUsers     []types.UserInfo
	SearchResultChannels  []types.ChannelInfo
	ModalMode             ModalMode
	UsersList             *list.Model
	Message               *types.FormattedMessage
	fromPeer              *list.Item
	stories               *list.Model
	availableReactions    *list.Model
	allReactions          []types.Reaction
	selectedReactionIndex int
	isMePremium           bool
}

func (f Foreground) Init() tea.Cmd {
	return nil
}

type MessageDeletionConfrimResponseMsg struct {
	yes bool
}

var debouncedSearch = Debounce(func(args ...any) tea.Msg {
	query := args[0].(string)
	go telegram.Cligram.SearchUsers(telegram.Cligram.Context(), query)
	return nil
}, 300*time.Millisecond)

func setTotalSearchResultUsers(searchMsg types.SearchUsersMsg, m *Foreground) {
	if searchMsg.Response == nil {
		m.searchResultUsers = nil
		return
	}
	m.searchResultUsers = *searchMsg.Response
}

func (f Foreground) View() string {
	if f.Error != nil {
		errorTitle := lipgloss.NewStyle().
			Foreground(DefaultTheme.ErrorColor).
			Bold(true).
			Render("Error Occurred")

		errorMessage := lipgloss.NewStyle().
			Foreground(DefaultTheme.PrimaryText).
			Width(max(20, f.windowWidth/3)).
			Align(lipgloss.Center).
			Render(f.Error.Error())

		closeHint := lipgloss.NewStyle().
			Foreground(DefaultTheme.SecondaryText).
			Italic(true).
			Render("Press ESC or Q to close")

		errorBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(DefaultTheme.ErrorColor).
			Padding(1, 2).
			Align(lipgloss.Center).
			Render(lipgloss.JoinVertical(lipgloss.Center, errorTitle, "", errorMessage, "", closeHint))

		return lipgloss.Place(f.windowWidth, f.windowHeight, lipgloss.Center, lipgloss.Center, errorBox)
	}
	foreStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder(), true).
		BorderForeground(DefaultTheme.AccentColor). // Use theme accent color
		Padding(0, 1)

	if f.ModalMode == ModalModeShowStories && f.stories != nil {
		title := lipgloss.NewStyle().Foreground(DefaultTheme.PrimaryText).Bold(true).Render("Stories")
		var content string
		content = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(DefaultTheme.BorderColor).Render(f.stories.View())
		layout := lipgloss.JoinVertical(lipgloss.Left, title, content)
		return foreStyle.Render(layout)
	}

	if f.ModalMode == ModalModeForwardMessage {
		title := lipgloss.NewStyle().Foreground(DefaultTheme.PrimaryText).Bold(true).Render("Forward Message")
		f.UsersList.SetShowFilter(false)
		f.UsersList.SetShowStatusBar(false)
		f.UsersList.SetShowTitle(false)
		f.UsersList.SetShowHelp(false)
		f.UsersList.SetWidth(max(20, f.windowWidth/3))
		f.UsersList.SetHeight(max(10, f.windowHeight/2))
		content := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(DefaultTheme.BorderColor).Render(f.UsersList.View())
		layout := lipgloss.JoinVertical(lipgloss.Left, title, content)
		return foreStyle.Render(layout)
	}
	if f.ModalMode == ModalModeDeleteMessage {
		title := lipgloss.NewStyle().Foreground(DefaultTheme.PrimaryText).Bold(true).Render("Delete Message")
		yes := lipgloss.NewStyle().
			Foreground(DefaultTheme.SelectedFg).
			Background(DefaultTheme.AccentColor).
			Padding(0, 1).
			Render("Y")
		no := lipgloss.NewStyle().
			Foreground(DefaultTheme.SelectedFg).
			Background(DefaultTheme.ErrorColor).
			Padding(0, 1).
			Render("N")
		contentStyle := lipgloss.NewStyle().
			Foreground(DefaultTheme.PrimaryText).
			Background(DefaultTheme.SubtleBg).
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
		layout := lipgloss.JoinVertical(lipgloss.Left, title, contentString)
		return foreStyle.Render(layout)
	}
	if f.ModalMode == ModalModeSendReaction {
		title := lipgloss.NewStyle().Foreground(DefaultTheme.PrimaryText).Bold(true).Render("Send Reaction")
		var content string
		if len(f.allReactions) > 0 {
			content = renderReactionsGrid(f)
		} else {
			content = lipgloss.NewStyle().Foreground(DefaultTheme.AccentColor).Render("Loading reactions...")
		}
		layout := lipgloss.JoinVertical(lipgloss.Left, title, content)
		return foreStyle.Render(layout)
	}
	title := lipgloss.NewStyle().Foreground(DefaultTheme.PrimaryText).Bold(true).Render("Search")
	content := getSearchView(f)
	var searchResultBorderStyle lipgloss.Style
	if f.focusedOn == SEARCH {
		searchResultBorderStyle = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(DefaultTheme.BorderColor)
	} else {
		searchResultBorderStyle = lipgloss.NewStyle().Border(lipgloss.DoubleBorder()).BorderForeground(DefaultTheme.AccentColor)
	}

	searchResult := searchResultBorderStyle.Render(f.searchResultCombined.View())
	layout := lipgloss.JoinVertical(lipgloss.Left, title, content, searchResult)
	return foreStyle.Render(layout)
}

func getSearchView(m Foreground) string {
	var inputBorderStyle lipgloss.Style
	if m.focusedOn == LIST {
		inputBorderStyle = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(DefaultTheme.BorderColor)
	} else {
		inputBorderStyle = lipgloss.NewStyle().Border(lipgloss.DoubleBorder()).BorderForeground(DefaultTheme.AccentColor)
	}
	textViewString := lipgloss.NewStyle().Width(max(20, m.windowWidth/3)).Height(5).Padding(0, 1).Inherit(inputBorderStyle).Background(DefaultTheme.InputBg).Render(m.input.View())
	return textViewString
}

func renderReactionsGrid(f Foreground) string {
	const columns = 8
	var rows []string
	var currentRow []string

	for i, reaction := range f.allReactions {
		style := lipgloss.NewStyle().Padding(0, 1)
		reactionText := reaction.Reaction

		if i == f.selectedReactionIndex {
			style = style.
				Foreground(DefaultTheme.SelectedFg).
				Background(DefaultTheme.AccentColor)
		}

		if reaction.AvailableReaction.Premium && !f.isMePremium {
			reactionText = reactionText + "🔒"
			if i != f.selectedReactionIndex {
				style = style.Foreground(DefaultTheme.SecondaryText)
			}
		} else if reaction.AvailableReaction.Premium {
			reactionText = reactionText + "⭐"
		}

		currentRow = append(currentRow, style.Render(reactionText))

		if len(currentRow) == columns || i == len(f.allReactions)-1 {
			rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, currentRow...))
			currentRow = nil
		}
	}

	grid := lipgloss.JoinVertical(lipgloss.Left, rows...)
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(DefaultTheme.BorderColor).
		Padding(1).
		Render(grid)
}

func (m Model) GetUserAccessHashFromModel(userID int64) (types.UserInfo, error) {
	var userInfo types.UserInfo
	for _, value := range m.Users.Items() {
		if user, ok := value.(types.UserInfo); ok && user.PeerID == strconv.FormatInt(userID, 10) {
			userInfo = user
		}
	}
	return userInfo, nil
}
