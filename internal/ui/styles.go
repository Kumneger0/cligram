package ui

import "github.com/charmbracelet/lipgloss"

var (
	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF"))

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#7D56F4")).
			Bold(true)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).Padding(0, 1)

	timestampStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#999999")).
			Italic(true).
			MarginRight(1)

	senderStyle = lipgloss.NewStyle().
			Bold(true).
			MarginRight(1)

	myMessageStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FFFF")).
			Bold(true).
			MarginRight(1)

	contentStyle = lipgloss.NewStyle().
			PaddingLeft(1)

	messageStyle = lipgloss.NewStyle().
			PaddingTop(1).
			PaddingBottom(1)
)

func getSideBarStyles(sidebarWidth int, contentHeight int, m *Model) lipgloss.Style {
	sideBarStyle := lipgloss.NewStyle().Width(sidebarWidth).Height(contentHeight).Padding(1).Border(getItemBorder(m.FocusedOn == "sideBar"))
	return sideBarStyle
}

func getInputStyle(m *Model, inputHeight int) lipgloss.Style {
	inputStyle := lipgloss.NewStyle().Width(m.Width+2).Height(inputHeight).Padding(0, 1).Border(getItemBorder(m.FocusedOn == "input"))
	return inputStyle
}

func getMainStyle(mainWidth int, contentHeight int, m *Model) lipgloss.Style {
	return lipgloss.NewStyle().
		Width(mainWidth).
		Height(contentHeight - 6).
		Padding(1).
		Border(getItemBorder(m.FocusedOn == "mainView"))
}
