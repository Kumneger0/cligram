package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
)

type CallOverlayState int

const (
	CallOverlayNone CallOverlayState = iota
	CallOverlayIncoming
	CallOverlayActive
)

type CallOverlayModel struct {
	State     CallOverlayState
	UserName  string
	UserID    int64
	IsGroup   bool
	GroupName string
	Muted     bool
	StartTime time.Time
}

var (
	callOverlayBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#10B981")).
				Padding(1, 2)

	callStatusBarStyle = lipgloss.NewStyle().
				Background(DefaultTheme.SubtleBg).
				Foreground(DefaultTheme.PrimaryText).
				Padding(0, 1)

	callMutedStyle = lipgloss.NewStyle().
			Foreground(DefaultTheme.ErrorColor)
)

func NewCallOverlay() CallOverlayModel {
	return CallOverlayModel{
		State: CallOverlayNone,
	}
}

func (m CallOverlayModel) View() string {
	switch m.State {
	case CallOverlayNone:
		return ""
	case CallOverlayIncoming:
		prompt := fmt.Sprintf("📞 Incoming call from %s\n\n    [a] Accept    [d] Decline", m.UserName)
		return callOverlayBorderStyle.Render(prompt)
	case CallOverlayActive:
		duration := m.Duration()
		status := fmt.Sprintf("📞 In call with %s  %s", m.UserName, duration)

		muteStr := ""
		if m.Muted {
			muteStr = "  " + callMutedStyle.Render("🔇 Muted")
		}

		controls := "[m]ute [h]angup"

		barContent := fmt.Sprintf("%s%s  %s", status, muteStr, controls)
		return callStatusBarStyle.Render(barContent)
	}
	return ""
}

func (m *CallOverlayModel) SetIncoming(userName string, userID int64) {
	m.State = CallOverlayIncoming
	m.UserName = userName
	m.UserID = userID
	m.IsGroup = false
	m.GroupName = ""
}

func (m *CallOverlayModel) SetActive(userName string, userID int64) {
	m.State = CallOverlayActive
	m.UserName = userName
	m.UserID = userID
	m.StartTime = time.Now()
}

func (m *CallOverlayModel) SetNone() {
	m.State = CallOverlayNone
}

func (m *CallOverlayModel) ToggleMute() {
	m.Muted = !m.Muted
}

func (m CallOverlayModel) IsActive() bool {
	return m.State != CallOverlayNone
}

func (m CallOverlayModel) Duration() string {
	if m.StartTime.IsZero() {
		return "00:00"
	}
	d := time.Since(m.StartTime)
	mins := int(d.Minutes())
	secs := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d", mins, secs)
}
