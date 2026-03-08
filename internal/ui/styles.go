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
	PrimaryText:    lipgloss.Color("#E0E0E0"),
	SecondaryText:  lipgloss.Color("#A0A0A0"),
	AccentColor:    lipgloss.Color("#8A68F8"),
	BorderColor:    lipgloss.Color("#606060"),
	SelectedBg:     lipgloss.Color("#8A68F8"),
	SelectedFg:     lipgloss.Color("#FFFFFF"),
	SubtleBg:       lipgloss.Color("#303030"),
	WarningColor:   lipgloss.Color("#FFD700"),
	ErrorColor:     lipgloss.Color("#FF6347"),
	OnlineStatus:   lipgloss.Color("#00FF00"),
	OfflineStatus:  lipgloss.Color("#FFA500"),
	UnreadCountBg:  lipgloss.Color("#FF6347"),
	UnreadCountFg:  lipgloss.Color("#FFFFFF"),
	InputBg:        lipgloss.Color("#252525"),
	TimestampColor: lipgloss.Color("#707070"),
}

var (
	normalStyle = lipgloss.NewStyle().
			Foreground(DefaultTheme.PrimaryText)

	selectedStyle = lipgloss.NewStyle().
			Foreground(DefaultTheme.SelectedFg).
			Background(DefaultTheme.SelectedBg).
			Bold(true)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(DefaultTheme.AccentColor).Padding(0, 1)

	timestampStyle = lipgloss.NewStyle().Height(1).
			Foreground(DefaultTheme.TimestampColor).
			Italic(true).
			PaddingRight(1).PaddingLeft(4)

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
				SetString(" ")
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
