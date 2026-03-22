package ui

import (
	"github.com/charmbracelet/lipgloss"
)

type Theme struct {
	PrimaryText    lipgloss.Color
	SecondaryText  lipgloss.Color
	AccentColor    lipgloss.Color
	BorderColor    lipgloss.Color
	SelectedBg     lipgloss.Color
	SelectedFg     lipgloss.Color
	SubtleBg       lipgloss.Color
	WarningColor   lipgloss.Color
	ErrorColor     lipgloss.Color
	OnlineStatus   lipgloss.Color
	OfflineStatus  lipgloss.Color
	UnreadCountBg  lipgloss.Color
	UnreadCountFg  lipgloss.Color
	InputBg        lipgloss.Color
	TimestampColor lipgloss.Color
}

var DefaultTheme = Theme{
	PrimaryText:    lipgloss.Color("#E2E8F0"),
	SecondaryText:  lipgloss.Color("#94A3B8"),
	AccentColor:    lipgloss.Color("#818CF8"),
	BorderColor:    lipgloss.Color("#1E293B"),
	SelectedBg:     lipgloss.Color("#334155"),
	SelectedFg:     lipgloss.Color("#F0F0F0"),
	SubtleBg:       lipgloss.Color("#0A0F1D"),
	WarningColor:   lipgloss.Color("#F59E0B"),
	ErrorColor:     lipgloss.Color("#EF4444"),
	OnlineStatus:   lipgloss.Color("#10B981"),
	OfflineStatus:  lipgloss.Color("#F1F5F9"),
	UnreadCountBg:  lipgloss.Color("#1E293B"),
	UnreadCountFg:  lipgloss.Color("#818CF8"),
	InputBg:        lipgloss.Color("#0F172A"),
	TimestampColor: lipgloss.Color("#CBD5E1"),
}

var (
	normalStyle = lipgloss.NewStyle().
			Foreground(DefaultTheme.PrimaryText).
			PaddingLeft(1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(DefaultTheme.SelectedFg).
			Background(DefaultTheme.SelectedBg).
			PaddingLeft(1).
			Bold(true)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(DefaultTheme.AccentColor).Padding(0, 1)

	sidebarHeaderStyle = lipgloss.NewStyle().
				Foreground(DefaultTheme.SecondaryText).
				Bold(true).
				Padding(0, 1)

	timestampStyle = lipgloss.NewStyle().
			Foreground(DefaultTheme.TimestampColor).
			Italic(true)

	messageStyle = lipgloss.NewStyle().
			PaddingTop(1).
			PaddingBottom(1).
			Foreground(DefaultTheme.PrimaryText)

	replyMessageStyle = lipgloss.NewStyle().
				Foreground(DefaultTheme.SecondaryText)

	unreadCountStyle = lipgloss.NewStyle().
				Background(DefaultTheme.UnreadCountBg).
				Foreground(DefaultTheme.UnreadCountFg).
				Padding(0, 1).
				Bold(true)

	reactionBadgeStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#1E293B")).
				Foreground(DefaultTheme.PrimaryText).
				Padding(0, 1)

	myReactionBadgeStyle = lipgloss.NewStyle().
				Background(DefaultTheme.AccentColor).
				Foreground(lipgloss.Color("#FFFFFF")).
				Padding(0, 1).
				Bold(true)

	viewCountStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#1A202C")).  // A slightly darker subtle background
			Foreground(DefaultTheme.SecondaryText). // Use secondary text color for view count
			Padding(0, 1).
			Bold(false)

	readStateStyleSingle = lipgloss.NewStyle().
				Foreground(DefaultTheme.SecondaryText).
				Padding(0, 1)

	readStateStyleDouble = lipgloss.NewStyle().
				Foreground(DefaultTheme.AccentColor).
				Padding(0, 1)
)

func getSideBarStyles(sidebarWidth int, contentHeight int, m *Model) lipgloss.Style {
	sideBarStyle := lipgloss.NewStyle().
		Width(max(0, sidebarWidth-2)).
		Height(max(0, contentHeight-2)).
		Padding(1).
		Border(getItemBorder(m.FocusedOn == "sideBar")).
		BorderForeground(getBorderColor(m.FocusedOn == "sideBar")).
		MaxHeight(contentHeight)
	return sideBarStyle
}

func getInputStyle(m *Model, inputHeight int) lipgloss.Style {
	inputStyle := lipgloss.NewStyle().
		Width(max(0, m.Width-2)).
		Height(max(0, inputHeight-2)).
		Padding(0, 1).
		Border(getItemBorder(m.FocusedOn == "input")).
		BorderForeground(getBorderColor(m.FocusedOn == "input")).
		Background(DefaultTheme.InputBg)
	return inputStyle
}

func getMainStyle(mainWidth int, contentHeight int, m *Model) lipgloss.Style {
	return lipgloss.NewStyle().
		Width(max(0, mainWidth-2)).
		Height(max(0, contentHeight-2)).
		Padding(1).
		Border(getItemBorder(m.FocusedOn == "mainView")).
		BorderForeground(getBorderColor(m.FocusedOn == "mainView")).
		MaxHeight(contentHeight).
		MaxWidth(mainWidth)
}

func getItemBorder(isSelected bool) lipgloss.Border {
	if isSelected {
		return lipgloss.DoubleBorder()
	}
	return lipgloss.NormalBorder()
}

func getBorderColor(isSelected bool) lipgloss.Color {
	if isSelected {
		return DefaultTheme.AccentColor
	}
	return DefaultTheme.BorderColor
}
