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
	CallOverlayDialing
	CallOverlayActive
	CallOverlayMissingHelper
)

type CallOverlayModel struct {
	State     CallOverlayState
	UserName  string
	UserID    int64
	Muted     bool
	IsRelay   bool
	StartTime time.Time
}

var (
	callOverlayBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#10B981")).
				Padding(1, 2)

	callOverlayWarningStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#EF4444")).
				Padding(1, 2)

	callStatusBarStyle = lipgloss.NewStyle().
				Background(DefaultTheme.SubtleBg).
				Foreground(DefaultTheme.PrimaryText).
				Padding(0, 1)

	callMutedStyle = lipgloss.NewStyle().
			Foreground(DefaultTheme.ErrorColor).
			Bold(true)

	callBadgeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3B82F6")).
			Bold(true)
)

func NewCallOverlay() CallOverlayModel {
	return CallOverlayModel{
		State:   CallOverlayNone,
		IsRelay: true, // Default safe display
	}
}

func (m CallOverlayModel) View() string {
	switch m.State {
	case CallOverlayNone:
		return ""
	case CallOverlayMissingHelper:
		prompt := "⚠️  VoIP Helper Missing\n\nVoice calling requires 'cligram-voip'.\nPlease install it to ~/.local/bin or your PATH.\n\n    [Enter/Esc] Dismiss"
		return callOverlayWarningStyle.Render(prompt)
	case CallOverlayDialing:
		prompt := fmt.Sprintf("📞 Calling %s...\n\n    [Alt+H] Cancel", m.UserName)
		return callOverlayBorderStyle.Render(prompt)
	case CallOverlayIncoming:
		prompt := fmt.Sprintf("📞 Incoming call from %s\n\n    [a] Accept    [d] Decline", m.UserName)
		return callOverlayBorderStyle.Render(prompt)
	case CallOverlayActive:
		duration := m.Duration()
		badge := "[Relay]"
		if !m.IsRelay {
			badge = "[P2P]"
		}

		status := fmt.Sprintf("📞 In call with %s (%s)  %s", m.UserName, duration, callBadgeStyle.Render(badge))

		muteStr := ""
		muteAction := "Mute"
		if m.Muted {
			muteStr = "  " + callMutedStyle.Render("🔇 MUTED")
			muteAction = "Unmute"
		}

		controls := fmt.Sprintf("[Alt+M: %s]  [Alt+H: Hangup]", muteAction)

		barContent := fmt.Sprintf("%s%s  %s", status, muteStr, controls)
		return callStatusBarStyle.Render(barContent)
	}
	return ""
}

func (m *CallOverlayModel) SetDialing(userName string, userID int64) {
	m.State = CallOverlayDialing
	m.UserName = userName
	m.UserID = userID
	m.Muted = false
}

func (m *CallOverlayModel) SetIncoming(userName string, userID int64) {
	m.State = CallOverlayIncoming
	m.UserName = userName
	m.UserID = userID
	m.Muted = false
}

func (m *CallOverlayModel) SetActive(userName string, userID int64) {
	m.State = CallOverlayActive
	m.UserName = userName
	m.UserID = userID
	if m.StartTime.IsZero() {
		m.StartTime = time.Now()
	}
}

func (m *CallOverlayModel) SetMissingHelper() {
	m.State = CallOverlayMissingHelper
}

func (m *CallOverlayModel) SetRelay(isRelay bool) {
	m.IsRelay = isRelay
}

func (m *CallOverlayModel) SetNone() {
	m.State = CallOverlayNone
	m.Muted = false
	m.StartTime = time.Time{}
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
