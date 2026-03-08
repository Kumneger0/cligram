package ui

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kumneger0/cligram/internal/config"
	"github.com/kumneger0/cligram/internal/telegram"
	"github.com/kumneger0/cligram/internal/telegram/types"
	"github.com/muesli/reflow/wordwrap"
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

type MessagesDelegate struct {
	list.DefaultDelegate
	*Model
}

func (d MessagesDelegate) Height() int                               { return 1 }
func (d MessagesDelegate) Spacing() int                              { return 0 }
func (d MessagesDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

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

	if entry.IsFromMe {
		title = "You: " + title
	} else {
		if entry.SenderUserInfo != nil {
			title = entry.SenderUserInfo.FirstName + ": " + title
		} else {
			title = entry.Sender + ": " + title
		}
	}
	date := strings.Repeat(" ", 4) + timestampStyle.Render(entry.Date.Format("02/01/2006 03:04 PM"))

	isMainViewFocused := d.Model.FocusedOn == Mainview
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
	ModeUsers    Mode = "users"
	ModeChannels Mode = "channels"
	ModeGroups   Mode = "groups"
	ModeBots     Mode = "bot"
)

type FocusedOn string

const (
	SideBar  FocusedOn = "sideBar"
	Mainview FocusedOn = "mainView"
	Input    FocusedOn = "input"
)

type Model struct {
	Filepicker           filepicker.Model
	IsFilepickerVisible  bool
	SelectedFile         string
	Users                list.Model
	Bots                 list.Model
	SelectedUser         types.UserInfo
	Channels             list.Model
	IsModalVisible       bool
	ModalContent         string
	SelectedChannel      types.ChannelInfo
	Groups               list.Model
	SelectedGroup        types.ChannelInfo
	Height               int
	Width                int
	MainViewLoading      bool
	SideBarLoading       bool
	Mode                 Mode
	Input                textinput.Model
	viewport             viewport.Model
	FocusedOn            FocusedOn
	ChatUI               list.Model
	Conversations        [50]types.FormattedMessage
	IsReply              bool
	ReplyTo              *types.FormattedMessage
	EditMessage          *types.FormattedMessage
	SkipNextInput        bool
	OffsetDate, OffsetID int
	OnPagination         bool
	Stories              []types.Stories
}

func filterEmptyMessages(msgs [50]types.FormattedMessage) []types.FormattedMessage {
	var filteredMsgs []types.FormattedMessage
	for _, m := range msgs {
		if m.ID != 0 {
			filteredMsgs = append(filteredMsgs, m)
		}
	}
	return filteredMsgs
}

func formatMessages(msgs [50]types.FormattedMessage) []list.Item {
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
	modalContent.WriteString(errorMessage + "\n")
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
	return lipgloss.NewStyle().Background(DefaultTheme.SubtleBg).Render(lipgloss.JoinVertical(lipgloss.Top, row, inputView))
}

type layoutDimensions struct {
	sidebarWidth  int
	mainWidth     int
	contentHeight int
	inputHeight   int
}

func calculateLayoutDimensions(m *Model) layoutDimensions {
	sidebarWidth := m.Width * 30 / 100
	return layoutDimensions{
		sidebarWidth:  sidebarWidth,
		mainWidth:     m.Width - sidebarWidth,
		contentHeight: m.Height * 90 / 100,
		inputHeight:   m.Height - (m.Height * 90 / 100),
	}
}

func updateListDimensions(m *Model, d layoutDimensions) {
	listHeight := d.contentHeight - 4
	listWidth := d.sidebarWidth - 4
	m.Users.SetHeight(listHeight)
	m.Users.SetWidth(listWidth)
	m.Bots.SetHeight(listHeight)
	m.Bots.SetWidth(listWidth)
	m.Channels.SetWidth(listWidth)
	m.Channels.SetHeight(listHeight)
	m.Groups.SetWidth(listWidth)
	m.Groups.SetHeight(listHeight)
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
	switch m.Mode {
	case ModeUsers, ModeBots:
		return formatUserName(m.SelectedUser)
	case ModeChannels:
		return formatChannelName(m.SelectedChannel)
	case ModeGroups:
		return formatGroupName(m.SelectedGroup)
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
	m.ChatUI.SetWidth(d.mainWidth - 4)
	//the terminal height is determined by charater
	// one list items takes one 1 charater space since we are showing extra info on chats like times
	// using the d.contentHeight will make the content out of view
	m.ChatUI.SetHeight(int(d.contentHeight / 5))

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

	chatsView := m.ChatUI.View()

	if len(m.ChatUI.Items()) > 0 {
		m.ChatUI.Select(len(m.ChatUI.Items()) - 1)
	}

	if m.IsFilepickerVisible {
		chatsView = prepareFilepickerView(m)
	}

	mainContent := lipgloss.JoinVertical(
		lipgloss.Top,
		headerView,
		chatsView,
	)

	return mainStyle.Render(mainContent)
}

func prepareFilepickerView(m *Model) string {
	var s strings.Builder
	s.WriteString("\n  ")
	if m.SelectedFile == "" {
		s.WriteString("Pick a file:")
	} else {
		s.WriteString("Selected file: click ctrl + a to close file picker\n" + m.Filepicker.Styles.Selected.Render(m.SelectedFile))
	}
	s.WriteString("\n\n" + m.Filepicker.View() + "\n")
	return s.String()
}

func prepareSidebarContent(m *Model, d layoutDimensions) string {
	var content string
	switch m.Mode {
	case ModeBots:
		content = m.Bots.View()
	case ModeUsers:
		content = m.Users.View()
	case ModeChannels:
		content = m.Channels.View()
	case ModeGroups:
		content = m.Groups.View()
	}

	storiesIndicator := sidebarHeaderStyle.Render(fmt.Sprintf("📖 Stories (%d)", len(m.Stories)))
	itemsCount := sidebarHeaderStyle.Render(fmt.Sprintf("💬 Chats (%d)", len(m.Users.Items())))
	if m.Mode == ModeChannels {
		itemsCount = sidebarHeaderStyle.Render(fmt.Sprintf("📢 Channels (%d)", len(m.Channels.Items())))
	} else if m.Mode == ModeGroups {
		itemsCount = sidebarHeaderStyle.Render(fmt.Sprintf("👥 Groups (%d)", len(m.Groups.Items())))
	} else if m.Mode == ModeBots {
		itemsCount = sidebarHeaderStyle.Render(fmt.Sprintf("🤖 Bots (%d)", len(m.Bots.Items())))
	}

	header := lipgloss.JoinVertical(lipgloss.Left, storiesIndicator, "", itemsCount)
	joinedView := lipgloss.JoinVertical(lipgloss.Top, header, content)
	return getSideBarStyles(d.sidebarWidth, d.contentHeight, m).Render(joinedView)
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
	if m.Mode == ModeUsers || m.Mode == ModeBots {
		if m.Mode == ModeUsers {
			m.SelectedUser = m.Users.SelectedItem().(types.UserInfo)
			cType = types.UserChat
		}
		if m.Mode == ModeBots {
			m.SelectedUser = m.Bots.SelectedItem().(types.UserInfo)
			cType = types.BotChat
		}
		pInfo = types.Peer{
			AccessHash: m.SelectedUser.AccessHash,
			ID:         m.SelectedUser.PeerID,
			ChatType:   cType,
		}
	}
	if m.Mode == ModeChannels {
		m.SelectedChannel = m.Channels.SelectedItem().(types.ChannelInfo)
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
	if m.Mode == ModeGroups {
		m.SelectedGroup = m.Groups.SelectedItem().(types.ChannelInfo)
		cType = types.ChatType(types.GroupChat)
		pInfo = types.Peer{
			AccessHash: m.SelectedGroup.AccessHash,
			ID:         m.SelectedGroup.ID,
			ChatType:   cType,
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
