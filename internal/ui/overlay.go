package ui

import (
	"fmt"
	"io"
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
	"github.com/muesli/reflow/wordwrap"
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

type ForwardDestinationItem struct {
	OriginalItem      list.Item
	Name              string
	PeerID            string
	AccessHash        string
	ChannelOrUserType ChannelOrUserType
}

func (f ForwardDestinationItem) FilterValue() string {
	return f.Name
}

func (f ForwardDestinationItem) Title() string {
	return f.Name
}

type SearchDelegate struct {
	list.DefaultDelegate
	*Foreground
}

func (d SearchDelegate) Height() int  { return 1 }
func (d SearchDelegate) Spacing() int { return 0 }

func (d SearchDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

func entityBadge(t ChannelOrUserType, title string) string {
	switch t {
	case USER:
		return "👤 " + title
	case GROUP:
		return "👥 " + title
	case CHANNEL:
		return "📢 " + title
	case BOT:
		return "🤖 " + title
	default:
		return title
	}
}

func (d SearchDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	var title string

	if entry, ok := item.(SearchResult); ok {
		title = entityBadge(entry.ChannelOrUserType, entry.FilterValue())
	} else if dest, ok := item.(ForwardDestinationItem); ok {
		title = entityBadge(dest.ChannelOrUserType, dest.FilterValue())
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

func formatStoryMeta(story types.Stories) string {
	var parts []string
	if !story.Date.IsZero() {
		diff := time.Since(story.Date)
		if diff < time.Minute {
			parts = append(parts, "just now")
		} else if diff < time.Hour {
			parts = append(parts, fmt.Sprintf("%dm ago", int(diff.Minutes())))
		} else if diff < 24*time.Hour {
			parts = append(parts, fmt.Sprintf("%dh ago", int(diff.Hours())))
		} else {
			parts = append(parts, story.Date.Format("02 Jan"))
		}
	}
	if !story.ExpireDate.IsZero() && time.Until(story.ExpireDate) > 0 {
		timeLeft := time.Until(story.ExpireDate)
		if timeLeft < time.Hour {
			parts = append(parts, fmt.Sprintf("%dm left", int(timeLeft.Minutes())))
		} else {
			parts = append(parts, fmt.Sprintf("%dh left", int(timeLeft.Hours())))
		}
	}
	return strings.Join(parts, " • ")
}

func (s StoriesDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	allStoryItems := m.Items()
	if len(allStoryItems) == 0 {
		return
	}

	story, ok := allStoryItems[index].(types.Stories)
	if !ok {
		return
	}

	mediaIcon := "📸"
	if story.MediaType == "video" {
		mediaIcon = "🎥"
	}
	if story.IsPinned {
		mediaIcon = "📌 " + mediaIcon
	}

	name := story.UserInfo.FirstName
	if story.UserInfo.LastName != "" {
		name += " " + story.UserInfo.LastName
	}
	if name == "" {
		name = "Unknown"
	}
	var badge string
	if story.UserInfo.Premium {
		badge = "⭐ "
	}
	var statusBadge string
	if story.IsSelected {
		spinView := "⠋"
		if s.Foreground != nil {
			s.Foreground.ensureSpinner()
			if v := s.Foreground.spinner.View(); v != "" {
				spinView = v
			}
		}
		statusBadge = lipgloss.NewStyle().Foreground(DefaultTheme.AccentColor).Bold(true).Render(spinView+" Downloading...") + " "
	}

	authorStr := fmt.Sprintf("%s%s %s%s", statusBadge, mediaIcon, badge, name)
	if story.UserInfo.Username != "" {
		authorStr += " " + lipgloss.NewStyle().Foreground(DefaultTheme.SecondaryText).Render("@"+story.UserInfo.Username)
	}

	metaStr := formatStoryMeta(story)
	if metaStr != "" {
		authorStr += " " + lipgloss.NewStyle().Foreground(DefaultTheme.SecondaryText).Render("• "+metaStr)
	}
	if story.Caption != "" {
		caption := strings.ReplaceAll(story.Caption, "\n", " ")
		if len(caption) > 24 {
			caption = caption[:24] + "…"
		}
		authorStr += " " + lipgloss.NewStyle().Foreground(DefaultTheme.SecondaryText).Italic(true).Render(`— "`+caption+`"`)
	}

	var style lipgloss.Style
	if index == m.Index() {
		style = selectedStyle
	} else {
		style = normalStyle
	}

	width := 50
	if s.Foreground != nil {
		width = max(50, min(75, s.Foreground.windowWidth-6))
	}
	str := lipgloss.NewStyle().MaxWidth(width).Width(width).Render(authorStr)
	fmt.Fprint(w, style.Render(" "+str+" "))
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

type GroupMembersDelegate struct {
	*Foreground
}

func (d GroupMembersDelegate) Height() int                               { return 1 }
func (d GroupMembersDelegate) Spacing() int                              { return 0 }
func (d GroupMembersDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d GroupMembersDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	member, ok := item.(types.GroupMemberInfo)
	if !ok {
		return
	}

	var roleBadge string
	switch member.Role {
	case types.MemberRoleOwner:
		roleBadge = lipgloss.NewStyle().Foreground(lipgloss.Color("#EAB308")).Bold(true).Render("[Owner]") + " "
		if member.CustomTitle != "" {
			roleBadge += lipgloss.NewStyle().Foreground(lipgloss.Color("#EAB308")).Italic(true).Render("("+member.CustomTitle+")") + " "
		}
	case types.MemberRoleAdmin:
		roleBadge = lipgloss.NewStyle().Foreground(lipgloss.Color("#06B6D4")).Bold(true).Render("[Admin]") + " "
		if member.CustomTitle != "" {
			roleBadge += lipgloss.NewStyle().Foreground(lipgloss.Color("#06B6D4")).Italic(true).Render("("+member.CustomTitle+")") + " "
		}
	default:
		if member.CustomTitle != "" {
			roleBadge = lipgloss.NewStyle().Foreground(DefaultTheme.SecondaryText).Italic(true).Render("("+member.CustomTitle+")") + " "
		}
	}

	name := member.User.DisplayName()

	var usernameStr string
	if member.User.Username != "" {
		usernameStr = lipgloss.NewStyle().Foreground(DefaultTheme.SecondaryText).Render(" @" + member.User.Username)
	}

	var presenceStr string
	if member.User.IsOnline {
		presenceStr = " " + lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E")).Render("Online")
	} else if member.User.LastSeen != nil && *member.User.LastSeen != "" {
		presenceStr = " " + lipgloss.NewStyle().Foreground(DefaultTheme.SecondaryText).Render(*member.User.LastSeen)
	}

	fullLine := roleBadge + name + usernameStr + presenceStr

	width := 60
	if d.Foreground != nil {
		width = max(50, min(80, d.Foreground.windowWidth-6))
	}

	str := lipgloss.NewStyle().MaxWidth(width).Width(width).Render(fullLine)
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
	ModalModeGroupMembers   ModalMode = "GROUP_MEMBERS"
	ModalModeAlert          ModalMode = "ALERT"
	ModalModeJoinPreview    ModalMode = "JOIN_PREVIEW"
	ModalModeComments       ModalMode = "COMMENTS"
)

type OpenModalMsg struct {
	ModalMode    ModalMode
	AlertTitle   string
	AlertMessage string
	FromPeer     *list.Item
	Message      *types.FormattedMessage
	CommentsPost *types.FormattedMessage
	UsersList    *list.Model
	ChannelsList *list.Model
	GroupsList   *list.Model
	BotsList     *list.Model
	Stories      []types.Stories
	Entity       *types.EntityPreviewInfo
	GroupPeer    *types.Peer
	GroupTitle   string
	TotalMembers int
	JoinPreview  *types.ChatInvitePreview
}

type ForwardMsg struct {
	msg      *types.FormattedMessage
	receiver *list.Item
	fromPeer *list.Item
}

type Foreground struct {
	Error                  error
	AlertTitle             string
	AlertMessage           string
	windowWidth            int
	windowHeight           int
	input                  textinput.Model
	searchResultCombined   list.Model
	focusedOn              focusState
	searchResultUsers      []types.UserInfo
	SearchResultChannels   []types.ChannelInfo
	ModalMode              ModalMode
	UsersList              *list.Model
	ChannelsList           *list.Model
	GroupsList             *list.Model
	BotsList               *list.Model
	forwardInput           textinput.Model
	forwardDestinations    list.Model
	allForwardDestinations []list.Item
	Entity                 *types.ResolvedPeerInfo
	Message                *types.FormattedMessage
	fromPeer               *list.Item
	stories                *list.Model
	availableReactions     *list.Model
	allReactions           []types.Reaction
	selectedReactionIndex  int
	isMePremium            bool
	spinner                spinner.Model
	storiesLoading         bool
	isDownloadingStory     bool
	downloadingStoryPeer   string
	membersList            *list.Model
	allMembers             []types.GroupMemberInfo
	membersInput           textinput.Model
	membersLoading         bool
	groupTitle             string
	groupMembersError      error
	groupPeer              *types.Peer
	membersOffset          int
	hasMoreMembers         bool
	isPaginatingMembers    bool
	totalMembers           int
	isSearchingMembers     bool
	membersSearchSeq       int
	JoinPreview            *types.ChatInvitePreview
	isJoining              bool
	joinError              error

	// Comments modal fields
	commentsPost           *types.FormattedMessage
	commentsDiscussionChat *types.ChannelInfo
	commentsTopMsgID       int
	comments               []types.FormattedMessage
	commentsLoading        bool
	commentsError          error
	commentInput           textarea.Model
	isPostingComment       bool
	commentsScrollOffset   int
	commentsCursor         int
}

func (f *Foreground) ensureSpinner() {
	if len(f.spinner.Spinner.Frames) == 0 {
		s := spinner.New()
		s.Spinner = spinner.Dot
		s.Style = lipgloss.NewStyle().Foreground(DefaultTheme.AccentColor)
		f.spinner = s
	}
}

func (f *Foreground) Init() tea.Cmd {
	return nil
}

type MessageDeletionConfirmResponseMsg struct {
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

func (f *Foreground) View() string {
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
			Render("[Enter] or [Esc] to dismiss")

		errorBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(DefaultTheme.ErrorColor).
			Padding(1, 2).
			Align(lipgloss.Center).
			Render(lipgloss.JoinVertical(lipgloss.Center, errorTitle, "", errorMessage, "", closeHint))

		return errorBox
	}
	foreStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder(), true).
		BorderForeground(DefaultTheme.AccentColor).
		Padding(0, 1)

	if f.ModalMode == ModalModeAlert {
		titleText := f.AlertTitle
		if titleText == "" {
			titleText = "Notice"
		}
		title := lipgloss.NewStyle().
			Foreground(DefaultTheme.WarningColor).
			Bold(true).
			Render("⚠️  " + titleText)

		contentWidth := max(44, min(65, f.windowWidth-6))
		if f.windowWidth <= 0 {
			contentWidth = 50
		}

		divider := lipgloss.NewStyle().
			Foreground(DefaultTheme.BorderColor).
			Render(strings.Repeat("─", contentWidth))

		message := lipgloss.NewStyle().
			Foreground(DefaultTheme.PrimaryText).
			Width(contentWidth).
			Render(f.AlertMessage)

		footerHints := lipgloss.NewStyle().
			Foreground(DefaultTheme.SecondaryText).
			Padding(0, 1).
			Render("[Enter] or [Esc] to dismiss")

		layout := lipgloss.JoinVertical(lipgloss.Left, title, divider, message, "", footerHints)
		return foreStyle.Width(contentWidth + 2).Render(layout)
	}

	if f.ModalMode == ModalModeJoinPreview {
		contentWidth := max(44, min(65, f.windowWidth-6))
		if f.windowWidth <= 0 {
			contentWidth = 50
		}

		icon := "👥"
		typeLabel := "Group Preview"
		countLabel := "members"
		titleName := "Chat"
		var memberCount int
		var aboutText string
		var reqNeeded bool

		if f.JoinPreview != nil {
			titleName = f.JoinPreview.Title
			memberCount = f.JoinPreview.ParticipantsCount
			aboutText = f.JoinPreview.About
			reqNeeded = f.JoinPreview.RequestNeeded
			if f.JoinPreview.IsChannel {
				icon = "📢"
				typeLabel = "Channel Preview"
				countLabel = "subscribers"
			}
		}

		header := lipgloss.NewStyle().
			Foreground(DefaultTheme.AccentColor).
			Bold(true).
			Render(fmt.Sprintf("%s %s", icon, typeLabel))

		divider := lipgloss.NewStyle().
			Foreground(DefaultTheme.BorderColor).
			Render(strings.Repeat("─", contentWidth))

		title := lipgloss.NewStyle().
			Foreground(DefaultTheme.PrimaryText).
			Bold(true).
			Render(titleName)

		countStr := formatNumberWithCommas(memberCount)
		memberInfo := lipgloss.NewStyle().
			Foreground(DefaultTheme.SecondaryText).
			Render(fmt.Sprintf("%s %s %s", icon, countStr, countLabel))

		var elements []string
		elements = append(elements, header, divider, title, memberInfo)

		if aboutText != "" {
			wrappedAbout := wordwrap.String(aboutText, contentWidth)
			aboutStyled := lipgloss.NewStyle().
				Foreground(DefaultTheme.PrimaryText).
				Render(wrappedAbout)
			elements = append(elements, "", aboutStyled)
		}

		if reqNeeded {
			approvalNotice := lipgloss.NewStyle().
				Foreground(DefaultTheme.WarningColor).
				Render("🔒 Admin approval is required to join this chat")
			elements = append(elements, "", approvalNotice)
		}

		if f.isJoining {
			spinView := f.spinner.View()
			if spinView == "" {
				spinView = "⠋"
			}
			joiningStatus := lipgloss.NewStyle().
				Foreground(DefaultTheme.AccentColor).
				Bold(true).
				Render(fmt.Sprintf("%s Joining...", spinView))
			elements = append(elements, "", joiningStatus)
		} else if f.joinError != nil {
			errNotice := lipgloss.NewStyle().
				Foreground(DefaultTheme.ErrorColor).
				Render(fmt.Sprintf("⚠️ %v", f.joinError))
			elements = append(elements, "", errNotice)
		}

		footerText := "[Enter] Join    [Esc] Cancel"
		footer := lipgloss.NewStyle().
			Foreground(DefaultTheme.SecondaryText).
			Padding(0, 1).
			Render(footerText)

		elements = append(elements, "", footer)

		layout := lipgloss.JoinVertical(lipgloss.Left, elements...)
		return foreStyle.Width(contentWidth + 2).Render(layout)
	}

	if f.ModalMode == ModalModeShowStories {
		contentWidth := max(50, min(75, f.windowWidth-6))
		count := 0
		if f.stories != nil {
			count = len(f.stories.Items())
		}
		spinView := f.spinner.View()
		if spinView == "" {
			spinView = "⠋"
		}
		titleText := fmt.Sprintf("📖 Stories (%d)", count)
		if f.storiesLoading && count == 0 {
			titleText = fmt.Sprintf("📖 Stories — %s Fetching stories...", spinView)
		} else if f.isDownloadingStory {
			titleText = fmt.Sprintf("📖 Stories (%d) — %s Downloading media...", count, spinView)
		}
		title := lipgloss.NewStyle().
			Foreground(DefaultTheme.PrimaryText).
			Bold(true).
			Render(titleText)

		divider := lipgloss.NewStyle().
			Foreground(DefaultTheme.BorderColor).
			Render(strings.Repeat("─", contentWidth))

		var listContent string
		if f.stories == nil || count == 0 {
			if f.storiesLoading {
				listContent = lipgloss.NewStyle().
					Foreground(DefaultTheme.SecondaryText).
					Padding(2, 2).
					Render(fmt.Sprintf("%s Fetching stories from Telegram...", spinView))
			} else {
				listContent = lipgloss.NewStyle().
					Foreground(DefaultTheme.SecondaryText).
					Italic(true).
					Padding(2, 2).
					Render("No stories available right now")
			}
		} else {
			f.stories.SetWidth(contentWidth)
			f.stories.SetHeight(min(10, max(5, f.windowHeight/2)))
			listContent = f.stories.View()
		}

		footerText := "[↑/↓] Navigate  [Enter] View Story  [Esc] Close"
		footerStyle := lipgloss.NewStyle().Foreground(DefaultTheme.SecondaryText).Padding(0, 1)
		if f.isDownloadingStory {
			footerText = fmt.Sprintf("%s Downloading story media... Please wait  [Esc] Cancel", spinView)
			footerStyle = lipgloss.NewStyle().Foreground(DefaultTheme.AccentColor).Bold(true).Padding(0, 1)
		} else if f.storiesLoading && count == 0 {
			footerText = fmt.Sprintf("%s Connecting to Telegram...  [Esc] Close", spinView)
			footerStyle = lipgloss.NewStyle().Foreground(DefaultTheme.SecondaryText).Padding(0, 1)
		}
		footerHints := footerStyle.Render(footerText)

		layout := lipgloss.JoinVertical(lipgloss.Left, title, divider, listContent, "", footerHints)
		return foreStyle.Width(contentWidth + 2).Render(layout)
	}

	if f.ModalMode == ModalModeGroupMembers {
		contentWidth := max(50, min(80, f.windowWidth-6))
		count := 0
		if f.membersList != nil {
			count = len(f.membersList.Items())
		}
		spinView := f.spinner.View()
		if spinView == "" {
			spinView = "⠋"
		}

		titleName := f.groupTitle
		if titleName == "" {
			titleName = "Group"
		}

		var titleText string
		if f.membersInput.Value() != "" {
			if f.isSearchingMembers {
				titleText = fmt.Sprintf("👥 %s Members (%d found) — %s Loading...", titleName, count, spinView)
			} else if f.totalMembers > 0 {
				titleText = fmt.Sprintf("👥 %s Members (%d found of %d)", titleName, count, f.totalMembers)
			} else {
				titleText = fmt.Sprintf("👥 %s Members (%d found)", titleName, count)
			}
		} else {
			displayCount := f.totalMembers
			if displayCount == 0 {
				displayCount = count
			}
			if (f.membersLoading && count == 0) || f.isPaginatingMembers {
				if displayCount > 0 {
					titleText = fmt.Sprintf("👥 %s Members (%d) — %s Loading...", titleName, displayCount, spinView)
				} else {
					titleText = fmt.Sprintf("👥 %s Members — %s Loading...", titleName, spinView)
				}
			} else {
				titleText = fmt.Sprintf("👥 %s Members (%d)", titleName, displayCount)
			}
		}
		title := lipgloss.NewStyle().
			Foreground(DefaultTheme.PrimaryText).
			Bold(true).
			Render(titleText)

		f.membersInput.Width = contentWidth - 4
		inputLine := lipgloss.NewStyle().
			Width(contentWidth).
			Background(DefaultTheme.InputBg).
			Padding(0, 1).
			Render(f.membersInput.View())

		divider := lipgloss.NewStyle().
			Foreground(DefaultTheme.BorderColor).
			Render(strings.Repeat("─", contentWidth))

		var listContent string
		if f.groupMembersError != nil && count == 0 {
			listContent = lipgloss.NewStyle().
				Foreground(DefaultTheme.ErrorColor).
				Padding(1, 2).
				Render(fmt.Sprintf("Failed to fetch members: %v", f.groupMembersError))
		} else if f.membersList == nil || count == 0 {
			if f.membersLoading || f.isSearchingMembers {
				listContent = lipgloss.NewStyle().
					Foreground(DefaultTheme.SecondaryText).
					Padding(2, 2).
					Render(fmt.Sprintf("%s Loading...", spinView))
			} else {
				listContent = lipgloss.NewStyle().
					Foreground(DefaultTheme.SecondaryText).
					Italic(true).
					Padding(2, 2).
					Render("No members found")
			}
		} else {
			f.membersList.SetWidth(contentWidth)
			f.membersList.SetHeight(min(12, max(5, f.windowHeight/2)))
			listContent = f.membersList.View()
		}

		footerText := "[Tab] Search/List  [↑/↓/j/k] Navigate  [Enter] Direct Message  [Esc] Close"
		footerStyle := lipgloss.NewStyle().Foreground(DefaultTheme.SecondaryText).Padding(0, 1)
		if f.groupMembersError != nil && count > 0 {
			footerStyle = lipgloss.NewStyle().Foreground(DefaultTheme.ErrorColor).Padding(0, 1)
			footerText = fmt.Sprintf("⚠️ %v  [Esc] Close", f.groupMembersError)
		} else if f.isSearchingMembers || (f.membersLoading && count == 0) || f.isPaginatingMembers {
			footerText = fmt.Sprintf("%s Loading...  [Esc] Close", spinView)
		}
		footerHints := footerStyle.Render(footerText)

		layout := lipgloss.JoinVertical(lipgloss.Left, title, "", inputLine, divider, listContent, "", footerHints)
		return foreStyle.Width(contentWidth + 2).Render(layout)
	}

	if f.ModalMode == ModalModeForwardMessage {
		title := lipgloss.NewStyle().Foreground(DefaultTheme.PrimaryText).Bold(true).Render("Forward Message")
		contentWidth := max(40, min(70, f.windowWidth*2/3))
		f.forwardInput.Width = contentWidth - 4
		inputLine := lipgloss.NewStyle().
			Width(contentWidth).
			Background(DefaultTheme.InputBg).
			Padding(0, 1).
			Render(f.forwardInput.View())
		divider := lipgloss.NewStyle().
			Foreground(DefaultTheme.BorderColor).
			Render(strings.Repeat("─", contentWidth))
		var listContent string
		if len(f.forwardDestinations.Items()) == 0 {
			listContent = lipgloss.NewStyle().
				Foreground(DefaultTheme.SecondaryText).
				Italic(true).
				Padding(1, 2).
				Render("No destinations found")
		} else {
			f.forwardDestinations.SetWidth(contentWidth)
			f.forwardDestinations.SetHeight(min(10, max(5, f.windowHeight/2)))
			listContent = f.forwardDestinations.View()
		}
		footerHints := lipgloss.NewStyle().
			Foreground(DefaultTheme.SecondaryText).
			Padding(0, 1).
			Render("[↑/↓] Navigate  [Enter] Forward  [Esc] Cancel")
		layout := lipgloss.JoinVertical(lipgloss.Left, title, "", inputLine, divider, listContent, "", footerHints)
		return foreStyle.Width(contentWidth + 2).Render(layout)
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

	if f.ModalMode == ModalModeComments {
		contentWidth := 80
		if f.windowWidth > 0 {
			contentWidth = min(120, max(50, f.windowWidth-4))
		}

		spinView := f.spinner.View()
		if spinView == "" {
			spinView = "⠋"
		}

		titleText := "💬 Comments"
		if f.commentsPost != nil {
			snippet := strings.ReplaceAll(f.commentsPost.Content, "\n", " ")
			snippet = strings.TrimSpace(snippet)
			if len(snippet) > 50 {
				snippet = snippet[:50] + "…"
			}
			if snippet != "" {
				titleText = fmt.Sprintf("💬 Comments • %s", snippet)
			}
		}
		if f.commentsLoading && len(f.comments) == 0 {
			titleText += fmt.Sprintf(" — %s Loading...", spinView)
		} else if len(f.comments) > 0 {
			titleText += fmt.Sprintf(" (%d)", len(f.comments))
		}

		title := lipgloss.NewStyle().
			Foreground(DefaultTheme.PrimaryText).
			Bold(true).
			Render(titleText)

		divider := lipgloss.NewStyle().
			Foreground(DefaultTheme.BorderColor).
			Render(strings.Repeat("─", contentWidth))

		var listContent string
		if f.commentsError != nil && len(f.comments) == 0 {
			listContent = lipgloss.NewStyle().
				Width(contentWidth-2).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(DefaultTheme.ErrorColor).
				Background(DefaultTheme.SubtleBg).
				Padding(1, 2).
				Align(lipgloss.Center).
				Foreground(DefaultTheme.ErrorColor).
				Render(fmt.Sprintf("⚠️ Failed to load comments: %v", f.commentsError))
		} else if len(f.comments) == 0 {
			if f.commentsLoading {
				listContent = lipgloss.NewStyle().
					Width(contentWidth-2).
					Border(lipgloss.RoundedBorder()).
					BorderForeground(DefaultTheme.BorderColor).
					Background(DefaultTheme.SubtleBg).
					Padding(2, 2).
					Align(lipgloss.Center).
					Foreground(DefaultTheme.SecondaryText).
					Render(fmt.Sprintf("%s Fetching comments...", spinView))
			} else {
				listContent = lipgloss.NewStyle().
					Width(contentWidth-2).
					Border(lipgloss.RoundedBorder()).
					BorderForeground(DefaultTheme.BorderColor).
					Background(DefaultTheme.SubtleBg).
					Padding(2, 2).
					Align(lipgloss.Center).
					Foreground(DefaultTheme.SecondaryText).
					Italic(true).
					Render("💭 No comments yet\n\nBe the first to leave a comment below!")
			}
		} else {
			if f.commentsCursor < 0 {
				f.commentsCursor = 0
			}
			if f.commentsCursor >= len(f.comments) {
				f.commentsCursor = len(f.comments) - 1
			}

			// Pre-render and measure all comment cards
			type renderedCardInfo struct {
				rendered string
				height   int
			}
			allCards := make([]renderedCardInfo, len(f.comments))

			for i, c := range f.comments {
				isSelected := (f.focusedOn == LIST && i == f.commentsCursor)

				senderName := c.Sender
				if senderName == "" {
					senderName = "Unknown"
				}
				if c.IsFromMe {
					senderName = "You"
				}

				timeStr := ""
				if !c.Date.IsZero() {
					timeStr = c.Date.Format("15:04 • Jan 02")
				}

				wrappedBody := wordwrap.String(c.Content, contentWidth-8)

				// Header: pointer + avatar + sender name + (You) badge + timestamp
				var nameBadge string
				if isSelected {
					nameBadge = lipgloss.NewStyle().Foreground(DefaultTheme.AccentColor).Bold(true).Render("▸ 👤 " + senderName)
				} else {
					nameBadge = lipgloss.NewStyle().Foreground(DefaultTheme.PrimaryText).Bold(true).Render("  👤 " + senderName)
				}

				if c.IsFromMe {
					youBadge := lipgloss.NewStyle().
						Foreground(DefaultTheme.OnlineStatus).
						Bold(true).
						Render(" (You)")
					nameBadge += youBadge
				}

				var timeBadge string
				if timeStr != "" {
					timeBadge = lipgloss.NewStyle().
						Foreground(DefaultTheme.SecondaryText).
						Render("   " + timeStr)
				}
				headerLine := nameBadge + timeBadge

				var bodyStyled string
				if isSelected {
					bodyStyled = lipgloss.NewStyle().Foreground(DefaultTheme.SelectedFg).Render(wrappedBody)
				} else {
					bodyStyled = lipgloss.NewStyle().Foreground(DefaultTheme.PrimaryText).Render(wrappedBody)
				}

				cardInner := lipgloss.JoinVertical(lipgloss.Left, headerLine, "", "    "+bodyStyled)

				cardStyle := lipgloss.NewStyle().
					Width(contentWidth-2).
					Padding(0, 1)

				if isSelected {
					cardStyle = cardStyle.
						Border(lipgloss.RoundedBorder()).
						BorderForeground(DefaultTheme.AccentColor).
						Background(DefaultTheme.SelectedBg)
				} else {
					cardStyle = cardStyle.
						Border(lipgloss.RoundedBorder()).
						BorderForeground(DefaultTheme.BorderColor).
						Background(DefaultTheme.SubtleBg)
				}

				rendered := cardStyle.Render(cardInner)
				allCards[i] = renderedCardInfo{
					rendered: rendered,
					height:   lipgloss.Height(rendered),
				}
			}

			// Viewport height calculation
			inputHeight := 4
			if f.windowHeight > 0 && f.windowHeight <= 26 {
				inputHeight = 3
			}
			availableHeight := 14
			if f.windowHeight > 0 {
				availableHeight = max(8, min(30, f.windowHeight-inputHeight-10))
			}

			startIdx := f.commentsCursor
			endIdx := f.commentsCursor + 1
			usedHeight := allCards[f.commentsCursor].height

			for {
				expanded := false
				if endIdx < len(allCards) && usedHeight+allCards[endIdx].height <= availableHeight {
					usedHeight += allCards[endIdx].height
					endIdx++
					expanded = true
				}
				if startIdx > 0 && usedHeight+allCards[startIdx-1].height <= availableHeight {
					usedHeight += allCards[startIdx-1].height
					startIdx--
					expanded = true
				}
				if !expanded {
					break
				}
			}

			var commentItems []string
			if startIdx > 0 {
				moreAbove := lipgloss.NewStyle().
					Foreground(DefaultTheme.SecondaryText).
					Italic(true).
					Padding(0, 1).
					Render(fmt.Sprintf("▲ %d older comments...", startIdx))
				commentItems = append(commentItems, moreAbove)
			}

			for i := startIdx; i < endIdx; i++ {
				commentItems = append(commentItems, allCards[i].rendered)
			}

			if endIdx < len(allCards) {
				moreBelow := lipgloss.NewStyle().
					Foreground(DefaultTheme.SecondaryText).
					Italic(true).
					Padding(0, 1).
					Render(fmt.Sprintf("▼ %d newer comments...", len(allCards)-endIdx))
				commentItems = append(commentItems, moreBelow)
			}

			listContent = lipgloss.JoinVertical(lipgloss.Left, commentItems...)
		}

		var bottomSection string
		if f.commentsDiscussionChat != nil && f.commentsDiscussionChat.IsLeft {
			var joinPrompt string
			if f.isJoining {
				joinPrompt = lipgloss.NewStyle().
					Foreground(DefaultTheme.AccentColor).
					Bold(true).
					Render(fmt.Sprintf("%s Joining...", spinView))
			} else if f.commentsError != nil {
				joinPrompt = lipgloss.NewStyle().
					Foreground(DefaultTheme.ErrorColor).
					Render(fmt.Sprintf("⚠️ %v — Press [Enter] to retry", f.commentsError))
			} else {
				enterKey := lipgloss.NewStyle().
					Foreground(DefaultTheme.PrimaryText).
					Bold(true).
					Render("[Enter] ")

				btnStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color("#FFFFFF")).
					Background(DefaultTheme.AccentColor).
					Bold(true).
					Padding(0, 2).
					Render("Join")

				joinPrompt = enterKey + btnStyle
			}

			bottomSection = lipgloss.NewStyle().
				Width(contentWidth-2).
				Background(DefaultTheme.SubtleBg).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(DefaultTheme.AccentColor).
				Padding(1, 1).
				Align(lipgloss.Center).
				Render(joinPrompt)
		} else if f.commentsDiscussionChat != nil && f.commentsDiscussionChat.IsMuted {
			mutedText := lipgloss.NewStyle().
				Foreground(DefaultTheme.ErrorColor).
				Bold(true).
				Render("🔇 Muted by admin") + "\n" +
				lipgloss.NewStyle().
					Foreground(DefaultTheme.SecondaryText).
					Render("You cannot write comments in this group")

			bottomSection = lipgloss.NewStyle().
				Width(contentWidth-2).
				Background(DefaultTheme.SubtleBg).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(DefaultTheme.ErrorColor).
				Padding(1, 2).
				Align(lipgloss.Center).
				Render(mutedText)
		} else {
			inputHeight := 4
			if f.windowHeight > 0 && f.windowHeight <= 26 {
				inputHeight = 3
			}
			f.commentInput.SetWidth(contentWidth - 6)
			f.commentInput.SetHeight(inputHeight)

			borderColor := DefaultTheme.BorderColor
			if f.focusedOn == SEARCH {
				borderColor = DefaultTheme.AccentColor
			}

			inputBoxStyle := lipgloss.NewStyle().
				Width(contentWidth-2).
				Background(DefaultTheme.InputBg).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(borderColor).
				Padding(0, 1)

			var inputView string
			if f.isPostingComment {
				inputView = lipgloss.NewStyle().
					Foreground(DefaultTheme.AccentColor).
					Bold(true).
					Render(fmt.Sprintf("%s Sending comment...", spinView))
			} else {
				inputView = f.commentInput.View()
			}

			bottomSection = inputBoxStyle.Render(inputView)
		}

		var footerText string
		if f.commentsDiscussionChat != nil && f.commentsDiscussionChat.IsLeft {
			footerText = "[Enter] Join    [Esc] Close"
		} else if f.commentsDiscussionChat != nil && f.commentsDiscussionChat.IsMuted {
			footerText = "[↑/↓/j/k] Navigate Comments    [Esc] Close"
		} else if f.focusedOn == LIST {
			footerText = "[↑/↓/j/k] Navigate Comments    [Enter/i/Tab] Comment    [Esc] Close"
		} else {
			footerText = "[Enter] Send comment    [↑/Tab] Browse Comments    [Esc] Close"
		}
		footerHints := lipgloss.NewStyle().
			Foreground(DefaultTheme.SecondaryText).
			Padding(0, 1).
			Render(footerText)

		layout := lipgloss.JoinVertical(
			lipgloss.Left,
			title,
			divider,
			"",
			listContent,
			"",
			bottomSection,
			"",
			footerHints,
		)
		return foreStyle.Width(contentWidth + 2).Render(layout)
	}

	title := lipgloss.NewStyle().Foreground(DefaultTheme.PrimaryText).Bold(true).Render("Search")
	contentWidth := max(40, min(70, f.windowWidth*2/3))
	f.input.Width = contentWidth - 4
	inputLine := lipgloss.NewStyle().
		Width(contentWidth).
		Background(DefaultTheme.InputBg).
		Padding(0, 1).
		Render(f.input.View())
	divider := lipgloss.NewStyle().
		Foreground(DefaultTheme.BorderColor).
		Render(strings.Repeat("─", contentWidth))
	var listContent string
	if len(f.searchResultCombined.Items()) == 0 {
		listContent = lipgloss.NewStyle().
			Foreground(DefaultTheme.SecondaryText).
			Italic(true).
			Padding(1, 2).
			Render("Type to search users, bots, channels...")
	} else {
		f.searchResultCombined.SetWidth(contentWidth)
		f.searchResultCombined.SetHeight(min(10, max(5, f.windowHeight/2)))
		listContent = f.searchResultCombined.View()
	}
	footerHints := lipgloss.NewStyle().
		Foreground(DefaultTheme.SecondaryText).
		Padding(0, 1).
		Render("[↑/↓] Navigate  [Enter] Select  [Esc] Close")
	layout := lipgloss.JoinVertical(lipgloss.Left, title, "", inputLine, divider, listContent, "", footerHints)
	return foreStyle.Width(contentWidth + 2).Render(layout)
}

func renderReactionsGrid(f *Foreground) string {
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

func renderEntityInfo(f *Foreground) string {
	var info strings.Builder
	detailStyle := lipgloss.NewStyle().Foreground(DefaultTheme.SecondaryText)

	if f.Entity.User != nil {
		u := f.Entity.User
		name := u.FirstName
		if u.LastName != "" {
			name += " " + u.LastName
		}
		info.WriteString(lipgloss.NewStyle().Foreground(DefaultTheme.PrimaryText).Bold(true).Render(name) + "\n")
		if u.Username != "" {
			info.WriteString(detailStyle.Render("@"+u.Username) + "\n")
		}
		info.WriteString("\n")

		if u.IsBot {
			info.WriteString(detailStyle.Render("🤖 Bot") + "\n")
		}

		if u.IsOnline {
			info.WriteString(lipgloss.NewStyle().Foreground(DefaultTheme.OnlineStatus).Render("● Online") + "\n")
		} else if u.LastSeen != nil {
			info.WriteString(detailStyle.Render("Last seen: "+*u.LastSeen) + "\n")
		}

		if u.Premium {
			info.WriteString(lipgloss.NewStyle().Foreground(DefaultTheme.AccentColor).Render("⭐ Premium User") + "\n")
		}
		info.WriteString(detailStyle.Render(fmt.Sprintf("ID: %s", u.PeerID)) + "\n")
	} else if f.Entity.Channel != nil || f.Entity.Group != nil {
		var c *types.ChannelInfo
		if f.Entity.Channel != nil {
			c = f.Entity.Channel
		} else {
			c = f.Entity.Group
		}

		info.WriteString(lipgloss.NewStyle().Foreground(DefaultTheme.PrimaryText).Bold(true).Render(c.ChannelTitle) + "\n")
		if c.Username != nil && *c.Username != "" {
			info.WriteString(detailStyle.Render("@"+*c.Username) + "\n")
		}
		info.WriteString("\n")

		if c.ParticipantsCount != nil {
			info.WriteString(detailStyle.Render(fmt.Sprintf("Members: %d", *c.ParticipantsCount)) + "\n")
		}

		if c.IsBroadcast {
			info.WriteString(detailStyle.Render("Broadcast Channel") + "\n")
		}
		if c.IsForum {
			info.WriteString(detailStyle.Render("Forum Enabled") + "\n")
		}
		info.WriteString(detailStyle.Render(fmt.Sprintf("ID: %s", c.ID)) + "\n")
	}
	return lipgloss.NewStyle().Padding(1, 0).Render(info.String())
}

func formatNumberWithCommas(n int) string {
	in := strconv.Itoa(n)
	if len(in) <= 3 {
		return in
	}
	var b strings.Builder
	remainder := len(in) % 3
	if remainder > 0 {
		b.WriteString(in[:remainder])
		if len(in) > remainder {
			b.WriteByte(',')
		}
	}
	for i := remainder; i < len(in); i += 3 {
		b.WriteString(in[i : i+3])
		if i+3 < len(in) {
			b.WriteByte(',')
		}
	}
	return b.String()
}
