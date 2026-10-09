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
	linkPreviewStyle = lipgloss.NewStyle().
				Foreground(DefaultTheme.AccentColor).
				Padding(0, 1)

	linkPreviewTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(DefaultTheme.AccentColor)

	linkPreviewSiteStyle = lipgloss.NewStyle().
				Foreground(DefaultTheme.SecondaryText).
				Italic(true)

	linkPreviewDescStyle = lipgloss.NewStyle().
				Foreground(DefaultTheme.PrimaryText)

	linkPreviewBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), false, false, false, true).
				BorderForeground(DefaultTheme.AccentColor).
				PaddingLeft(1)
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

	activeTabStyle = lipgloss.NewStyle().
			Foreground(DefaultTheme.AccentColor).
			Bold(true)

	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(DefaultTheme.SecondaryText)

	tabBadgeStyle = lipgloss.NewStyle().
			Foreground(DefaultTheme.UnreadCountFg).
			Bold(true)

	tabShortcutStyle = lipgloss.NewStyle().
				Foreground(DefaultTheme.SecondaryText)

	listItemNormalStyle = lipgloss.NewStyle().
				Foreground(DefaultTheme.PrimaryText)

	listItemSelectedStyle = lipgloss.NewStyle().
				Foreground(DefaultTheme.SelectedFg).
				Background(DefaultTheme.SelectedBg).
				Bold(true)

	listItemUnfocusedStyle = lipgloss.NewStyle().
				Foreground(DefaultTheme.PrimaryText).
				Background(lipgloss.Color("#1E293B"))

	selectedIndicatorStyle = lipgloss.NewStyle().
				Foreground(DefaultTheme.AccentColor).
				Bold(true)

	incomingBubbleStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(DefaultTheme.BorderColor).
				Padding(0, 1)

	incomingBubbleSelectedStyle = lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					BorderForeground(DefaultTheme.AccentColor).
					Padding(0, 1)

	outgoingBubbleStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#334155")).
				Padding(0, 1)

	outgoingBubbleSelectedStyle = lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					BorderForeground(DefaultTheme.AccentColor).
					Padding(0, 1)

	senderHandleColors = []lipgloss.Color{
		lipgloss.Color("#38BDF8"), // Sky
		lipgloss.Color("#818CF8"), // Indigo
		lipgloss.Color("#F472B6"), // Pink
		lipgloss.Color("#34D399"), // Emerald
		lipgloss.Color("#FBBF24"), // Amber
		lipgloss.Color("#A78BFA"), // Violet
		lipgloss.Color("#FB923C"), // Orange
		lipgloss.Color("#4ADE80"), // Green
	}
)

func senderColor(name string) lipgloss.Color {
	if len(senderHandleColors) == 0 {
		return DefaultTheme.AccentColor
	}
	var hash uint
	for i := 0; i < len(name); i++ {
		hash = hash*31 + uint(name[i])
	}
	return senderHandleColors[hash%uint(len(senderHandleColors))]
}

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
