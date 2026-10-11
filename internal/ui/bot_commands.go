package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	botCmdCardBorder = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(DefaultTheme.AccentColor).
				Background(DefaultTheme.InputBg).
				Padding(0, 1)

	botCmdHeaderStyle = lipgloss.NewStyle().
				Foreground(DefaultTheme.AccentColor).
				Bold(true)

	botCmdSelectedStyle = lipgloss.NewStyle().
				Foreground(DefaultTheme.SelectedFg).
				Background(DefaultTheme.SelectedBg).
				Bold(true)

	botCmdNormalStyle = lipgloss.NewStyle().
				Foreground(DefaultTheme.PrimaryText)

	botCmdDescStyle = lipgloss.NewStyle().
			Foreground(DefaultTheme.SecondaryText)
)

func renderBotCommandsPopup(m *Model, mainWidth int) string {
	if !m.ShowBotCommands || len(m.FilteredBotCommands) == 0 {
		return ""
	}

	cardWidth := max(30, min(mainWidth-4, 55))
	innerWidth := cardWidth - 4 // border (2) + padding (2)
	if innerWidth < 10 {
		innerWidth = 10
	}

	maxVisible := 5
	totalCmds := len(m.FilteredBotCommands)
	visibleCount := min(totalCmds, maxVisible)

	start := 0
	if m.SelectedBotCommandIndex >= visibleCount {
		start = m.SelectedBotCommandIndex - visibleCount + 1
	}
	end := start + visibleCount
	if end > totalCmds {
		end = totalCmds
		start = max(0, end-visibleCount)
	}

	lines := make([]string, 0, visibleCount+1)
	header := botCmdHeaderStyle.Render("Commands")
	lines = append(lines, header)

	for i := start; i < end; i++ {
		cmd := m.FilteredBotCommands[i]
		cmdStr := "/" + cmd.Command

		isSelected := (i == m.SelectedBotCommandIndex)
		prefix := "  "
		if isSelected {
			prefix = "▸ "
		}

		var row string
		if cmd.Description != "" {
			availDesc := innerWidth - lipgloss.Width(prefix) - lipgloss.Width(cmdStr) - 3 // " - "
			descStr := cmd.Description
			if availDesc > 3 {
				if len(descStr) > availDesc {
					descStr = descStr[:availDesc-1] + "…"
				}
				if isSelected {
					row = botCmdSelectedStyle.Width(innerWidth).Render(fmt.Sprintf("%s%s - %s", prefix, cmdStr, descStr))
				} else {
					row = fmt.Sprintf("%s%s %s",
						prefix,
						botCmdNormalStyle.Bold(true).Render(cmdStr),
						botCmdDescStyle.Render("- "+descStr),
					)
				}
			} else {
				if isSelected {
					row = botCmdSelectedStyle.Width(innerWidth).Render(prefix + cmdStr)
				} else {
					row = prefix + botCmdNormalStyle.Bold(true).Render(cmdStr)
				}
			}
		} else {
			if isSelected {
				row = botCmdSelectedStyle.Width(innerWidth).Render(prefix + cmdStr)
			} else {
				row = prefix + botCmdNormalStyle.Bold(true).Render(cmdStr)
			}
		}
		lines = append(lines, row)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return botCmdCardBorder.Width(cardWidth).Render(content)
}
