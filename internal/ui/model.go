package ui

import (
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gotd/td/tg"
	"github.com/kumneger0/cligram/internal/config"
	"github.com/kumneger0/cligram/internal/telegram"
	"github.com/kumneger0/cligram/internal/telegram/types"
	"github.com/muesli/reflow/wordwrap"
	"go.dalton.dog/bubbleup"
)

var (
	dialogBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(DefaultTheme.AccentColor).
		Padding(1, 2).
		BorderTop(true).
		BorderLeft(true).
		BorderRight(true).
		BorderBottom(true)
)

type ForumTopicsDelegate struct {
	list.DefaultDelegate
	*Model
}

func (d ForumTopicsDelegate) Height() int                               { return 1 }
func (d ForumTopicsDelegate) Spacing() int                              { return 0 }
func (d ForumTopicsDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d ForumTopicsDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	var title string
	var unreadBadge string

	forumTopic, ok := item.(types.ForumTopicInfo)
	if !ok {
		return
	}

	title = forumTopic.Title()
	if forumTopic.UnreadCount > 0 {
		unreadBadge = unreadCountStyle.Render(strconv.Itoa(forumTopic.UnreadCount))
	}

	var content string
	if unreadBadge != "" {
		availWidth := m.Width() - lipgloss.Width(unreadBadge) - 2
		if availWidth < 0 {
			availWidth = 0
		}
		name := lipgloss.NewStyle().MaxWidth(availWidth).Render(title)

		occupied := lipgloss.Width(name) + lipgloss.Width(unreadBadge) + 2
		spacerWidth := m.Width() - occupied
		if spacerWidth < 0 {
			spacerWidth = 0
		}
		spacer := strings.Repeat(" ", spacerWidth)
		content = lipgloss.JoinHorizontal(lipgloss.Top, name, spacer, unreadBadge)
	} else {
		content = title
	}

	isMainViewFocused := d.Model.FocusedOn == Main
	if index == m.Index() && isMainViewFocused && d.Model.ShowForumTopics {
		fmt.Fprint(w, selectedStyle.Render(content))
	} else {
		fmt.Fprint(w, normalStyle.Render(content))
	}
}

type MessagesDelegate struct {
	list.DefaultDelegate
	*Model
}

func (d MessagesDelegate) Height() int                               { return 1 }
func (d MessagesDelegate) Spacing() int                              { return 0 }
func (d MessagesDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (m *Model) renderReactions(tgReactions *tg.MessageReactions) string {
	var reactions string
	if tgReactions != nil {
		if len(tgReactions.Results) > 0 {
			var reactionBadges []string
			for _, r := range tgReactions.Results {
				var content string
				var isMeReacted bool

				switch reaction := r.Reaction.(type) {
				case *tg.ReactionEmoji:
					content = reaction.Emoticon + " " + strconv.Itoa(int(r.Count))
				case *tg.ReactionPaid:
					content = "⭐" + " " + strconv.Itoa(int(r.Count))
				case *tg.ReactionCustomEmoji:
					if m != nil && m.CustomEmojis != nil {
						if doc, found := m.CustomEmojis[reaction.DocumentID]; found {
							if len(doc.Thumbs) > 0 {
								thumb := doc.Thumbs[len(doc.Thumbs)-1]
								_ = thumb
							}
						}
					}
				}

				if _, ok := r.GetChosenOrder(); ok {
					isMeReacted = true
				}

				if isMeReacted {
					reactionBadges = append(reactionBadges, myReactionBadgeStyle.Render(content))
				} else {
					reactionBadges = append(reactionBadges, reactionBadgeStyle.Render(content))
				}
			}
			reactions = lipgloss.JoinHorizontal(lipgloss.Top, reactionBadges...)
		}
	}
	return reactions
}

func (m *Model) renderWebPagePreview(webPageMedia *tg.MessageMediaWebPage, width int) string {
	if webPageMedia == nil {
		return ""
	}

	webpage, ok := webPageMedia.Webpage.(*tg.WebPage)
	if !ok {
		return ""
	}

	var builder strings.Builder

	if siteName, ok := webpage.GetSiteName(); ok && siteName != "" {
		builder.WriteString(linkPreviewSiteStyle.Render(siteName) + "\n")
	}

	if title, ok := webpage.GetTitle(); ok && title != "" {
		builder.WriteString(linkPreviewTitleStyle.Render(wordwrap.String(title, width-4)) + "\n")
	}

	if description, ok := webpage.GetDescription(); ok && description != "" {
		builder.WriteString(linkPreviewDescStyle.Render(wordwrap.String(description, width-4)) + "\n")
	}

	if url := webpage.URL; url != "" {
		builder.WriteString(linkPreviewStyle.Render(url))
	}

	content := strings.TrimSpace(builder.String())
	if content == "" {
		return ""
	}

	return "\n" + linkPreviewBorderStyle.Render(content)
}

func (d MessagesDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	var title string

	entry, ok := item.(types.FormattedMessage)

	if !ok {
		return
	}
	if entry.ReplyTo != nil {
		var strBuilder strings.Builder
		messageReplayedTo := entry.ReplyTo.Content
		strBuilder.WriteString("> ")
		strBuilder.WriteString(replyMessageStyle.Render(messageReplayedTo))
		strBuilder.WriteString("\n")
		strBuilder.WriteString(wordwrap.String(entry.Title(), m.Width()))
		title = strBuilder.String()
	} else {
		title = wordwrap.String(entry.Title(), m.Width())
	}

	var readMaxOutboxID int
	if d.Model != nil {
		readMaxOutboxID = d.Model.ActiveReadOutboxMaxID()
	}

	var readState string
	if entry.IsFromMe && entry.ID <= readMaxOutboxID {
		readState = readStateStyleDouble.Render("✓✓")
	} else if entry.IsFromMe {
		readState = readStateStyleSingle.Render("✓")
	}

	var reactions string
	if entry.Reactions != nil && d.Model != nil {
		reactions = d.Model.renderReactions(entry.Reactions)
	}

	var preview string
	if entry.MessageMediaWebPage != nil && d.Model != nil {
		preview = d.Model.renderWebPagePreview(entry.MessageMediaWebPage, m.Width())
	}

	if entry.IsFromMe {
		title = "You: " + title
	} else {
		if entry.SenderUserInfo != nil {
			title = entry.SenderUserInfo.FirstName + ": " + title
		} else {
			title = entry.Sender + ": " + title
		}
	}

	if preview != "" {
		title = title + preview
	}

	date := strings.Repeat(" ", 4) + timestampStyle.Render(entry.Date.Format("02/01/2006 03:04 PM")) + readState

	if reactions != "" {
		title = title + "\n" + reactions
	}

	if entry.Views != 0 {
		viewCountStr := viewCountStyle.Render(fmt.Sprintf("👁️ %d", entry.Views))
		if reactions != "" {
			title = title + " " + viewCountStr
		} else {
			title = title + "\n" + viewCountStr
		}
	}

	isMainViewFocused := d.Model.FocusedOn == Main
	if index == m.Index() && isMainViewFocused {
		str := title + "\n" + date

		lines := strings.Split(str, "\n")
		var styledLines []string

		lineStyle := selectedStyle.Width(m.Width())

		for i, line := range lines {
			style := lineStyle
			if i == 0 {
				style = style.PaddingTop(1)
			}
			if i == len(lines)-1 {
				style = style.PaddingBottom(1)
			}
			styledLines = append(styledLines, style.Render(line))
		}

		fmt.Fprint(w, strings.Join(styledLines, "\n"))
	} else {
		str := messageStyle.Render(title + "\n" + date)
		fmt.Fprint(w, normalStyle.Width(m.Width()).Render(str))
	}
}

type Mode string

const (
	ModeAll         Mode = "all"
	ModeUsers       Mode = "users"
	ModeChannels    Mode = "channels"
	ModeGroups      Mode = "groups"
	ModeBots        Mode = "bot"
	ModeForumTopics Mode = "forumTopics"
)

type FocusedOn string

const (
	SideBar FocusedOn = "sideBar"
	Main    FocusedOn = "mainView"
	Input   FocusedOn = "input"
)

type PhoneCallDghConfigs struct {
	Configs map[int64]*types.DHConfig
}

type messageLineRange struct {
	startLine int
	endLine   int
}

type Model struct {
	PhoneCallDhConfigs       *PhoneCallDghConfigs
	Alert                    bubbleup.AlertModel
	Filepicker               filepicker.Model
	IsFilepickerVisible      bool
	SelectedFile             string
	All                      list.Model
	AllLoaded                bool
	AllLoading               bool
	Users                    list.Model
	Bots                     list.Model
	SelectedUser             types.UserInfo
	Channels                 list.Model
	IsModalVisible           bool
	ModalContent             string
	SelectedChannel          types.ChannelInfo
	Groups                   list.Model
	SelectedGroup            types.ChannelInfo
	Height                   int
	Width                    int
	MainViewLoading          bool
	SideBarLoading           bool
	Mode                     Mode
	Input                    textinput.Model
	viewport                 viewport.Model
	FocusedOn                FocusedOn
	ChatUI                   list.Model
	Conversations            []types.FormattedMessage
	ActivePeerID             string
	SelectedMessageIndex     int
	messageLineRanges        []messageLineRange
	IsReply                  bool
	ReplyTo                  *types.FormattedMessage
	EditMessage              *types.FormattedMessage
	SkipNextInput            bool
	OffsetDate, OffsetID     int
	OnPagination             bool
	UsersLoaded              bool
	UsersLoading             bool
	UsersOffsetDate          int
	UsersOffsetID            int
	BotsLoaded               bool
	BotsLoading              bool
	BotsOffsetDate           int
	BotsOffsetID             int
	ChannelsLoaded           bool
	ChannelsLoading          bool
	ChannelsOffsetDate       int
	ChannelsOffsetID         int
	GroupsLoaded             bool
	GroupsLoading            bool
	GroupsOffsetDate         int
	GroupsOffsetID           int
	Stories                  []types.Stories
	StoriesLoading           bool
	CurrentUser              *types.UserInfo
	Error                    error
	CustomEmojis             map[int64]*tg.Document
	SelectedGroupForumTopics list.Model
	ForumTopicLoading        bool
	ShowForumTopics          bool
	SelectedForumTopic       *types.ForumTopicInfo
	CallOverlay              CallOverlayModel
	ActiveCallUserID         int64
	SidebarSpinner           spinner.Model
}

type CustomEmojiDocumentMsg struct {
	DocumentID int64
	Document   *tg.Document
	Err        error
}

func FetchCustomEmojiDocumentCmd(documentID int64) tea.Cmd {
	return func() tea.Msg {
		docs, err := telegram.Cligram.API().MessagesGetCustomEmojiDocuments(telegram.Cligram.Context(), []int64{documentID})
		if err != nil {
			slog.Error("failed to get custom emoji documents", "error", err, "document_id", documentID)
			return CustomEmojiDocumentMsg{DocumentID: documentID, Err: err}
		}
		if len(docs) > 0 {
			if doc, ok := docs[0].(*tg.Document); ok {
				return CustomEmojiDocumentMsg{DocumentID: documentID, Document: doc}
			}
		}
		return CustomEmojiDocumentMsg{DocumentID: documentID, Err: fmt.Errorf("document not found for ID: %d", documentID)}
	}
}

func filterEmptyMessages(msgs []types.FormattedMessage) []types.FormattedMessage {
	var filteredMsgs []types.FormattedMessage
	for _, m := range msgs {
		if m.ID != 0 {
			filteredMsgs = append(filteredMsgs, m)
		}
	}
	return filteredMsgs
}

func formatMessages(msgs []types.FormattedMessage) []list.Item {
	filteredMsgs := filterEmptyMessages(msgs)
	var lines []list.Item
	for _, m := range filteredMsgs {
		lines = append(lines, m)
	}

	return lines
}

// this is just temporary just to get things working
// definitely i need to remove this
func GetModalContent(errorMessage string) string {
	var modalContent strings.Builder
	modalContent.WriteString(errorMessage)
	modalContent.WriteString("\n" + "press ctrl + c or q to close")
	modalWidth := max(40, len(errorMessage)+4)
	return dialogBoxStyle.Width(modalWidth).Render(modalContent.String())
}

func setItemStyles(m *Model) string {
	if m.IsModalVisible {
		return renderModal(m)
	}

	dimensions := calculateLayoutDimensions(m)

	updateListDimensions(m, dimensions)

	mainContent := prepareMainContent(m, dimensions)

	sidebarContent := prepareSidebarContent(m, dimensions)

	inputView := prepareInputView(m, dimensions)

	row := lipgloss.JoinHorizontal(lipgloss.Top, sidebarContent, mainContent)

	var fullLayout string
	if m.CallOverlay.State == CallOverlayActive {
		callBar := m.CallOverlay.View()
		fullLayout = lipgloss.JoinVertical(lipgloss.Top, callBar, row, inputView)
	} else {
		fullLayout = lipgloss.JoinVertical(lipgloss.Top, row, inputView)
	}

	return lipgloss.NewStyle().Background(DefaultTheme.SubtleBg).Render(fullLayout)
}

type layoutDimensions struct {
	sidebarWidth  int
	mainWidth     int
	contentHeight int
	inputHeight   int
}

func calculateLayoutDimensions(m *Model) layoutDimensions {
	sidebarWidth := m.Width * 30 / 100
	totalHeight := m.Height
	if m.CallOverlay.State == CallOverlayActive {
		totalHeight = max(0, totalHeight-1)
	}
	contentHeight := totalHeight * 90 / 100
	return layoutDimensions{
		sidebarWidth:  sidebarWidth,
		mainWidth:     m.Width - sidebarWidth,
		contentHeight: contentHeight,
		inputHeight:   totalHeight - contentHeight,
	}
}

func updateListDimensions(m *Model, d layoutDimensions) {
	tabBarHeight := lipgloss.Height(m.renderSidebarTabBar(max(0, d.sidebarWidth-4)))
	listHeight := max(0, d.contentHeight-(tabBarHeight+4))
	listWidth := max(0, d.sidebarWidth-4)
	m.All.SetHeight(listHeight)
	m.All.SetWidth(listWidth)
	m.Users.SetHeight(listHeight)
	m.Users.SetWidth(listWidth)
	m.Bots.SetHeight(listHeight)
	m.Bots.SetWidth(listWidth)
	m.Channels.SetWidth(listWidth)
	m.Channels.SetHeight(listHeight)
	m.Groups.SetWidth(listWidth)
	m.Groups.SetHeight(listHeight)
	// Forum topics are displayed in the main view area
	mainListHeight := max(0, d.contentHeight-8)
	mainListWidth := max(0, d.mainWidth-4)
	m.SelectedGroupForumTopics.SetHeight(mainListHeight)
	m.SelectedGroupForumTopics.SetWidth(mainListWidth)
}

func renderModal(m *Model) string {
	return lipgloss.Place(
		m.Width,
		m.Height,
		lipgloss.Center,
		lipgloss.Center,
		m.ModalContent,
	)
}

func getUserOrChannelName(m *Model) string {
	if m.ActivePeerID != "" {
		if m.SelectedUser.PeerID == m.ActivePeerID {
			return formatUserName(m.SelectedUser)
		}
		if m.SelectedChannel.ID == m.ActivePeerID {
			return formatChannelName(m.SelectedChannel)
		}
		if m.SelectedGroup.ID == m.ActivePeerID {
			groupName := formatGroupName(m.SelectedGroup)
			if m.SelectedForumTopic != nil {
				return groupName + " > " + m.SelectedForumTopic.TopicTitle
			}
			if m.ShowForumTopics {
				return groupName + " (Forum)"
			}
			return groupName
		}
	}

	switch m.Mode {
	case ModeAll:
		selected := m.All.SelectedItem()
		switch it := selected.(type) {
		case types.UserInfo:
			return formatUserName(it)
		case types.ChannelInfo:
			if it.IsBroadcast {
				return formatChannelName(it)
			}
			return formatGroupName(it)
		default:
			return ""
		}
	case ModeUsers, ModeBots:
		return formatUserName(m.SelectedUser)
	case ModeChannels:
		return formatChannelName(m.SelectedChannel)
	case ModeGroups:
		groupName := formatGroupName(m.SelectedGroup)
		if m.SelectedForumTopic != nil {
			return groupName + " > " + m.SelectedForumTopic.TopicTitle
		}
		if m.ShowForumTopics {
			return groupName + " (Forum)"
		}
		return groupName
	default:
		return ""
	}
}

func formatUserName(user types.UserInfo) string {
	name := user.Title()
	if user.IsTyping {
		return name + " Typing..."
	}
	if user.IsOnline && !user.IsBot {
		return name + " Online"
	}
	if user.LastSeen != nil && !user.IsBot {
		return name + " " + *user.LastSeen
	}
	return name
}

func formatChannelOrGroupName(name string, count *int) string {
	if count == nil {
		return name
	}
	return fmt.Sprintf("%s %d Members", name, *count)
}

func formatChannelName(channel types.ChannelInfo) string {
	return formatChannelOrGroupName(channel.FilterValue(), channel.ParticipantsCount)
}

func formatGroupName(group types.ChannelInfo) string {
	return formatChannelOrGroupName(group.FilterValue(), group.ParticipantsCount)
}

func prepareMainContent(m *Model, d layoutDimensions) string {
	mainStyle := getMainStyle(d.mainWidth, d.contentHeight, m)

	if m.MainViewLoading {
		return mainStyle.Render("Loading...")
	}

	if m.ForumTopicLoading {
		return mainStyle.Render("Loading forum topics...")
	}

	userNameOrChannelName := getUserOrChannelName(m)
	title := lipgloss.NewStyle().
		Foreground(DefaultTheme.PrimaryText).
		Bold(true).
		Padding(0, 1).
		Render(userNameOrChannelName)

	separatorLine := lipgloss.NewStyle().
		Foreground(DefaultTheme.BorderColor).
		SetString(strings.Repeat("─", max(0, d.mainWidth-2))). // Full width line
		String()

	headerView := lipgloss.JoinVertical(lipgloss.Left, title, "", separatorLine)

	// Show forum topics list when in forum mode and no topic is selected
	if m.ShowForumTopics && m.SelectedForumTopic == nil {
		m.SelectedGroupForumTopics.SetShowFilter(false)
		m.SelectedGroupForumTopics.SetShowStatusBar(false)
		m.SelectedGroupForumTopics.SetShowTitle(false)
		m.SelectedGroupForumTopics.SetShowHelp(false)

		forumView := m.SelectedGroupForumTopics.View()

		mainContent := lipgloss.JoinVertical(
			lipgloss.Top,
			headerView,
			forumView,
		)
		return mainStyle.Render(mainContent)
	}

	var chatsView string
	if m.IsFilepickerVisible {
		chatsView = prepareFilepickerView(m)
	} else {
		headerHeight := lipgloss.Height(headerView)
		availHeight := max(4, d.contentHeight-4-headerHeight)
		availWidth := max(10, d.mainWidth-6)
		chatsView = m.renderMessagesViewport(availWidth, availHeight)
	}

	mainContent := lipgloss.JoinVertical(
		lipgloss.Top,
		headerView,
		chatsView,
	)

	return mainStyle.Render(mainContent)
}

func (m *Model) CurrentPeerID() string {
	if m.ActivePeerID != "" {
		return m.ActivePeerID
	}
	switch m.Mode {
	case ModeAll:
		selected := m.All.SelectedItem()
		switch it := selected.(type) {
		case types.UserInfo:
			return it.PeerID
		case types.ChannelInfo:
			return it.ID
		}
	case ModeUsers, ModeBots:
		return m.SelectedUser.PeerID
	case ModeChannels:
		return m.SelectedChannel.ID
	case ModeGroups:
		return m.SelectedGroup.ID
	}
	return ""
}

func (m *Model) ActiveReadOutboxMaxID() int {
	if m.ActivePeerID != "" {
		if m.SelectedUser.PeerID == m.ActivePeerID {
			return m.SelectedUser.ReadOutboxMaxID
		}
		if m.SelectedChannel.ID == m.ActivePeerID {
			return m.SelectedChannel.ReadOutboxMaxID
		}
		if m.SelectedGroup.ID == m.ActivePeerID {
			return m.SelectedGroup.ReadOutboxMaxID
		}
	}
	switch m.Mode {
	case ModeAll:
		selected := m.All.SelectedItem()
		switch it := selected.(type) {
		case types.UserInfo:
			return it.ReadOutboxMaxID
		case types.ChannelInfo:
			return it.ReadOutboxMaxID
		}
	case ModeUsers, ModeBots:
		return m.SelectedUser.ReadOutboxMaxID
	case ModeChannels:
		return m.SelectedChannel.ReadOutboxMaxID
	case ModeGroups:
		return m.SelectedGroup.ReadOutboxMaxID
	}
	return 0
}

func (m *Model) SelectedMessage() *types.FormattedMessage {
	if len(m.Conversations) > 0 {
		if m.SelectedMessageIndex < 0 {
			m.SelectedMessageIndex = 0
		}
		if m.SelectedMessageIndex >= len(m.Conversations) {
			m.SelectedMessageIndex = len(m.Conversations) - 1
		}
		return &m.Conversations[m.SelectedMessageIndex]
	}
	if sel, ok := m.ChatUI.SelectedItem().(types.FormattedMessage); ok {
		return &sel
	}
	return nil
}

func (m *Model) scrollSelectedMessageIntoView() {
	if len(m.Conversations) == 0 || m.SelectedMessageIndex < 0 || m.SelectedMessageIndex >= len(m.messageLineRanges) {
		return
	}
	target := m.messageLineRanges[m.SelectedMessageIndex]
	viewportHeight := m.viewport.Height
	if viewportHeight <= 0 {
		return
	}

	// Card taller than viewport: align top
	if (target.endLine - target.startLine + 1) >= viewportHeight {
		m.viewport.SetYOffset(target.startLine)
		return
	}

	// If card starts above viewport, scroll up so start is visible
	if target.startLine < m.viewport.YOffset {
		m.viewport.SetYOffset(target.startLine)
		return
	}

	// If card ends below viewport, scroll down so end is visible
	if target.endLine >= m.viewport.YOffset+viewportHeight {
		m.viewport.SetYOffset(target.endLine - viewportHeight + 1)
	}
}

func (m *Model) renderMessagesViewport(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	m.viewport.Width = width
	m.viewport.Height = height

	if len(m.Conversations) == 0 {
		m.messageLineRanges = nil
		emptyView := lipgloss.NewStyle().
			Width(width).
			Height(height).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(DefaultTheme.SecondaryText).
			Render("No messages yet")
		m.viewport.SetContent(emptyView)
		return m.viewport.View()
	}

	readMaxOutboxID := m.ActiveReadOutboxMaxID()

	bubbleWidth := max(24, int(float64(width)*0.80))
	if bubbleWidth > width-2 {
		bubbleWidth = width - 2
	}

	var renderedCards []string
	m.messageLineRanges = make([]messageLineRange, len(m.Conversations))
	currentLine := 0

	for i, entry := range m.Conversations {
		cardStr := m.renderMessageBubble(entry, i, bubbleWidth, width, readMaxOutboxID)
		lines := strings.Split(cardStr, "\n")
		cardLines := len(lines)

		m.messageLineRanges[i] = messageLineRange{
			startLine: currentLine,
			endLine:   currentLine + cardLines - 1,
		}
		currentLine += cardLines + 1 // +1 for spacing between cards
		renderedCards = append(renderedCards, cardStr)
	}

	content := strings.Join(renderedCards, "\n\n")
	m.viewport.SetContent(content)
	return m.viewport.View()
}

func (m *Model) renderMessageBubble(entry types.FormattedMessage, idx int, bubbleWidth int, viewportWidth int, readMaxOutboxID int) string {
	isSelected := (m.FocusedOn == Main && idx == m.SelectedMessageIndex)

	var handle string
	if entry.IsFromMe {
		handle = lipgloss.NewStyle().
			Foreground(DefaultTheme.AccentColor).
			Bold(true).
			Render("You")
	} else {
		name := entry.Sender
		if entry.SenderUserInfo != nil && entry.SenderUserInfo.FirstName != "" {
			name = entry.SenderUserInfo.FirstName
		}
		handle = lipgloss.NewStyle().
			Foreground(senderColor(name)).
			Bold(true).
			Render(name)
	}

	if isSelected {
		handle = selectedIndicatorStyle.Render("▎ ") + handle
	}

	var cardElements []string
	cardElements = append(cardElements, handle)

	if entry.ReplyTo != nil {
		replyText := entry.ReplyTo.Content
		if len(replyText) > 60 {
			replyText = replyText[:57] + "..."
		}
		replyQuote := lipgloss.NewStyle().
			Foreground(DefaultTheme.SecondaryText).
			Italic(true).
			Render("┌ " + wordwrap.String(replyText, max(10, bubbleWidth-6)))
		cardElements = append(cardElements, replyQuote)
	}

	rawTitle := entry.Title()
	if rawTitle != "" {
		body := wordwrap.String(rawTitle, max(10, bubbleWidth-6))
		cardElements = append(cardElements, body)
	}

	if entry.MessageMediaWebPage != nil {
		preview := m.renderWebPagePreview(entry.MessageMediaWebPage, max(10, bubbleWidth-6))
		if preview != "" {
			cardElements = append(cardElements, preview)
		}
	}

	var badges []string
	if entry.Reactions != nil {
		if rStr := m.renderReactions(entry.Reactions); rStr != "" {
			badges = append(badges, rStr)
		}
	}
	if entry.Views != 0 {
		badges = append(badges, viewCountStyle.Render(fmt.Sprintf("👁️ %d", entry.Views)))
	}
	if len(badges) > 0 {
		cardElements = append(cardElements, strings.Join(badges, " "))
	}

	timeStr := entry.Date.Format("03:04 PM")
	var receiptStr string
	if entry.IsFromMe {
		if entry.ID > 0 && entry.ID <= readMaxOutboxID {
			receiptStr = " " + readStateStyleDouble.Render("✓✓")
		} else {
			receiptStr = " " + readStateStyleSingle.Render("✓")
		}
	}
	footer := timestampStyle.Render(timeStr) + receiptStr
	footerLine := lipgloss.NewStyle().
		Width(max(10, bubbleWidth-4)).
		Align(lipgloss.Right).
		Render(footer)
	cardElements = append(cardElements, footerLine)

	innerContent := strings.Join(cardElements, "\n")

	if entry.IsFromMe {
		style := outgoingBubbleStyle.Width(bubbleWidth)
		if isSelected {
			style = outgoingBubbleSelectedStyle.Width(bubbleWidth)
		}
		indent := max(0, viewportWidth-bubbleWidth-4)
		return style.MarginLeft(indent).Render(innerContent)
	}

	style := incomingBubbleStyle.Width(bubbleWidth)
	if isSelected {
		style = incomingBubbleSelectedStyle.Width(bubbleWidth)
	}
	return style.MarginLeft(1).Render(innerContent)
}

func prepareFilepickerView(m *Model) string {
	var s strings.Builder
	s.WriteString("\n  ")
	if m.SelectedFile == "" {
		s.WriteString("Pick a file:")
	} else {
		s.WriteString("Selected file: click ctrl + a to close file picker\n")
		s.WriteString(m.Filepicker.Styles.Selected.Render(m.SelectedFile))
	}
	s.WriteString("\n\n")
	s.WriteString(m.Filepicker.View())
	s.WriteString("\n")
	return s.String()
}

func (m *Model) ensureSidebarSpinner() {
	if len(m.SidebarSpinner.Spinner.Frames) == 0 {
		s := spinner.New()
		s.Spinner = spinner.Dot
		s.Style = lipgloss.NewStyle().Foreground(DefaultTheme.AccentColor).Bold(true)
		m.SidebarSpinner = s
	}
}

func (m *Model) isCurrentCategoryLoading() bool {
	switch m.Mode {
	case ModeAll:
		return m.AllLoading || m.SideBarLoading
	case ModeUsers:
		return m.UsersLoading
	case ModeChannels:
		return m.ChannelsLoading
	case ModeGroups:
		return m.GroupsLoading
	case ModeBots:
		return m.BotsLoading
	}
	return false
}

func (m *Model) isAnyCategoryLoading() bool {
	return m.AllLoading || m.UsersLoading || m.ChannelsLoading || m.GroupsLoading || m.BotsLoading || m.SideBarLoading
}

func (m *Model) currentCategoryItemCount() int {
	switch m.Mode {
	case ModeAll:
		return len(m.All.Items())
	case ModeUsers:
		return len(m.Users.Items())
	case ModeChannels:
		return len(m.Channels.Items())
	case ModeGroups:
		return len(m.Groups.Items())
	case ModeBots:
		return len(m.Bots.Items())
	}
	return 0
}

func (m *Model) setCategoryLoading(mode Mode, loading bool) {
	switch mode {
	case ModeAll:
		m.AllLoading = loading
	case ModeUsers:
		m.UsersLoading = loading
	case ModeChannels:
		m.ChannelsLoading = loading
	case ModeGroups:
		m.GroupsLoading = loading
	case ModeBots:
		m.BotsLoading = loading
	}
}

func (m *Model) checkThresholdAndBackfill(mode Mode) (Model, tea.Cmd) {
	m.ensureSidebarSpinner()
	if m.currentCategoryItemCount() < 10 && m.OffsetDate != -1 && m.OffsetID != -1 && !m.isCurrentCategoryLoading() && !m.SideBarLoading && !m.OnPagination {
		m.setCategoryLoading(mode, true)
		if telegram.Cligram != nil {
			return *m, tea.Batch(
				telegram.Cligram.GetAllChats(telegram.Cligram.Context(), m.OffsetDate, m.OffsetID, 100),
				m.SidebarSpinner.Tick,
			)
		}
		return *m, m.SidebarSpinner.Tick
	}
	return *m, nil
}

func prepareSidebarContent(m *Model, d layoutDimensions) string {
	m.ensureSidebarSpinner()

	var content string
	if (m.isCurrentCategoryLoading() || m.SideBarLoading) && m.currentCategoryItemCount() == 0 {
		catName := "dialogs"
		switch m.Mode {
		case ModeAll:
			catName = "all dialogs"
		case ModeChannels:
			catName = "channels"
		case ModeGroups:
			catName = "groups"
		case ModeBots:
			catName = "bots"
		case ModeUsers:
			catName = "chats"
		}
		spinView := m.SidebarSpinner.View()
		spinText := lipgloss.NewStyle().
			Foreground(DefaultTheme.SecondaryText).
			Render(fmt.Sprintf("Loading %s...", catName))
		spinBlock := lipgloss.JoinHorizontal(lipgloss.Center, spinView, " ", spinText)
		content = lipgloss.NewStyle().
			Width(max(0, d.sidebarWidth-4)).
			Height(max(0, d.contentHeight-8)).
			Align(lipgloss.Center, lipgloss.Center).
			Render(spinBlock)
	} else {
		switch m.Mode {
		case ModeAll:
			content = m.All.View()
		case ModeBots:
			content = m.Bots.View()
		case ModeUsers:
			content = m.Users.View()
		case ModeChannels:
			content = m.Channels.View()
		case ModeGroups:
			content = m.Groups.View()
		}
	}

	storiesIndicatorText := fmt.Sprintf("📖 Stories (%d)", len(m.Stories))
	if m.StoriesLoading && len(m.Stories) == 0 {
		storiesIndicatorText = "📖 Stories (⠋ Loading...)"
	}
	storiesIndicator := sidebarHeaderStyle.Render(storiesIndicatorText)
	tabBar := m.renderSidebarTabBar(max(0, d.sidebarWidth-4))

	header := lipgloss.JoinVertical(lipgloss.Left, storiesIndicator, "", tabBar, "")
	joinedView := lipgloss.JoinVertical(lipgloss.Top, header, content)
	return getSideBarStyles(d.sidebarWidth, d.contentHeight, m).Render(joinedView)
}

func (m *Model) categoryUnreadCount(mode Mode) int {
	var count int
	var items []list.Item
	switch mode {
	case ModeAll:
		items = m.All.Items()
	case ModeUsers:
		items = m.Users.Items()
	case ModeGroups:
		items = m.Groups.Items()
	case ModeChannels:
		items = m.Channels.Items()
	case ModeBots:
		items = m.Bots.Items()
	}
	for _, item := range items {
		switch v := item.(type) {
		case types.UserInfo:
			count += v.UnreadCount
		case types.ChannelInfo:
			count += v.UnreadCount
		}
	}
	return count
}

func (m *Model) renderSidebarTabBar(width int) string {
	m.ensureSidebarSpinner()

	type tabInfo struct {
		mode     Mode
		shortcut string
		name     string
	}
	tabs := []tabInfo{
		{mode: ModeAll, shortcut: "a", name: "All"},
		{mode: ModeUsers, shortcut: "u", name: "Chats"},
		{mode: ModeGroups, shortcut: "g", name: "Groups"},
		{mode: ModeChannels, shortcut: "c", name: "Channels"},
		{mode: ModeBots, shortcut: "b", name: "Bots"},
	}

	renderedTabs := make([]string, len(tabs))
	for i, t := range tabs {
		isLoading := false
		switch t.mode {
		case ModeAll:
			isLoading = m.AllLoading
		case ModeUsers:
			isLoading = m.UsersLoading
		case ModeChannels:
			isLoading = m.ChannelsLoading
		case ModeGroups:
			isLoading = m.GroupsLoading
		case ModeBots:
			isLoading = m.BotsLoading
		}

		unread := m.categoryUnreadCount(t.mode)
		badge := ""
		if unread > 0 {
			badge = fmt.Sprintf(" (%d)", unread)
		}
		if isLoading {
			badge += " " + m.SidebarSpinner.View()
		}
		label := fmt.Sprintf("[%s] %s%s", t.shortcut, t.name, badge)
		if m.Mode == t.mode {
			renderedTabs[i] = activeTabStyle.Render(label)
		} else {
			renderedTabs[i] = inactiveTabStyle.Render(label)
		}
	}

	// Try single line if it fits width
	singleLine := strings.Join(renderedTabs, " ")
	if width <= 0 || lipgloss.Width(singleLine) <= width {
		return singleLine
	}

	// 2 rows: 3 tabs on row1, 2 on row2
	row1 := strings.Join(renderedTabs[:3], " ")
	row2 := strings.Join(renderedTabs[3:], " ")
	if lipgloss.Width(row1) <= width && lipgloss.Width(row2) <= width {
		return lipgloss.JoinVertical(lipgloss.Left, row1, row2)
	}

	// Fallback to vertical stack
	return lipgloss.JoinVertical(lipgloss.Left, renderedTabs...)
}

func prepareInputView(m *Model, d layoutDimensions) string {
	if m.FocusedOn == Input {
		m.Input.Focus()
	}

	inputView := getInputStyle(m, d.inputHeight).Render(m.Input.View())

	if m.IsReply && m.ReplyTo != nil {
		replyContext := fmt.Sprintf("Reply to \n%s", strings.Split(m.ReplyTo.Content, "\n")[0])
		inputView = lipgloss.JoinVertical(lipgloss.Top, replyContext, inputView)
	}

	if m.SelectedFile != "" {
		fileContext := fmt.Sprintf("File \n%s", strings.Split(m.SelectedFile, "\n")[0])
		inputView = lipgloss.JoinVertical(lipgloss.Top, fileContext, inputView)
	}

	return inputView
}

func Debounce(fn func(args ...any) tea.Msg, delay time.Duration) func(args ...any) tea.Cmd {
	var mu sync.Mutex
	var timer *time.Timer
	var lastArgs []any

	return func(args ...any) tea.Cmd {
		mu.Lock()
		defer mu.Unlock()

		lastArgs = args

		if timer != nil {
			timer.Stop()
		}

		return func() tea.Msg {
			msgChan := make(chan tea.Msg, 1)

			timer = time.AfterFunc(delay, func() {
				mu.Lock()
				defer mu.Unlock()

				argsToPass := make([]any, len(lastArgs))
				copy(argsToPass, lastArgs)

				msgChan <- fn(argsToPass...)
				close(msgChan)
			})

			return <-msgChan
		}
	}
}

func getMessageParams(m *Model) types.Peer {
	var cType types.ChatType
	var pInfo types.Peer
	if m.Mode == ModeAll {
		selected := m.All.SelectedItem()
		switch it := selected.(type) {
		case types.UserInfo:
			m.SelectedUser = it
			if it.IsBot {
				cType = types.BotChat
			} else {
				cType = types.UserChat
			}
			pInfo = types.Peer{
				AccessHash: it.AccessHash,
				ID:         it.PeerID,
				ChatType:   cType,
			}
		case types.ChannelInfo:
			if it.IsBroadcast {
				m.SelectedChannel = it
				cType = types.ChannelChat
			} else {
				m.SelectedGroup = it
				cType = types.GroupChat
			}
			pInfo = types.Peer{
				AccessHash: it.AccessHash,
				ID:         it.ID,
				ChatType:   cType,
			}
			if it.IsBroadcast && it.IsCreator {
				m.Input.Reset()
			}
		}
	}
	if m.Mode == ModeUsers || m.Mode == ModeBots {
		if m.Mode == ModeUsers {
			if item, ok := m.Users.SelectedItem().(types.UserInfo); ok {
				m.SelectedUser = item
				cType = types.UserChat
				pInfo = types.Peer{
					AccessHash: m.SelectedUser.AccessHash,
					ID:         m.SelectedUser.PeerID,
					ChatType:   cType,
				}
			}
		}
		if m.Mode == ModeBots {
			if item, ok := m.Bots.SelectedItem().(types.UserInfo); ok {
				m.SelectedUser = item
				cType = types.BotChat
				pInfo = types.Peer{
					AccessHash: m.SelectedUser.AccessHash,
					ID:         m.SelectedUser.PeerID,
					ChatType:   cType,
				}
			}
		}
	}
	if m.Mode == ModeChannels {
		if item, ok := m.Channels.SelectedItem().(types.ChannelInfo); ok {
			m.SelectedChannel = item
			cType = types.ChannelChat
			pInfo = types.Peer{
				AccessHash: m.SelectedChannel.AccessHash,
				ID:         m.SelectedChannel.ID,
				ChatType:   cType,
			}
			if m.SelectedChannel.IsCreator {
				m.Input.Reset()
			}
		}
	}
	if m.Mode == ModeGroups {
		if item, ok := m.Groups.SelectedItem().(types.ChannelInfo); ok {
			m.SelectedGroup = item
			cType = types.GroupChat
			pInfo = types.Peer{
				AccessHash: m.SelectedGroup.AccessHash,
				ID:         m.SelectedGroup.ID,
				ChatType:   cType,
			}
		}
	}
	return pInfo
}

func SendUserIsTyping(m *Model) tea.Cmd {
	userConfig := config.GetConfig()
	if !*userConfig.Chat.SendTypingState {
		return nil
	}

	if (m.Mode == ModeUsers || m.Mode == ModeGroups) && m.FocusedOn == Input {
		var pInfo types.Peer
		if m.Mode == ModeUsers {
			pInfo = types.Peer{
				ID:         m.SelectedUser.PeerID,
				AccessHash: m.SelectedUser.AccessHash,
				ChatType:   types.UserChat,
			}
		}
		if m.Mode == ModeGroups {
			pInfo = types.Peer{
				ID:         m.SelectedGroup.ID,
				AccessHash: m.SelectedGroup.AccessHash,
				ChatType:   types.GroupChat,
			}
		}
		go func() {
			err := telegram.Cligram.SetUserTyping(telegram.Cligram.Context(), types.SetTypingRequest{
				Peer: pInfo,
			})
			if err != nil {
				slog.Error(err.Error())
			}
		}()
	}
	return nil
}
