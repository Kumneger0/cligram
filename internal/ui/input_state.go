package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/kumneger0/cligram/internal/telegram/types"
)

// InputState represents the functional state of the message input area.
type InputState int

const (
	InputStateNormal InputState = iota
	InputStateBroadcastReadOnly
	InputStateMutedByAdmin
	InputStateUnjoined
)

func (s InputState) String() string {
	switch s {
	case InputStateNormal:
		return "Normal"
	case InputStateBroadcastReadOnly:
		return "BroadcastReadOnly"
	case InputStateMutedByAdmin:
		return "MutedByAdmin"
	case InputStateUnjoined:
		return "Unjoined"
	default:
		return "Unknown"
	}
}

// CurrentInputState determines the active input mode for the current chat.
func (m *Model) CurrentInputState() InputState {
	if m.isUnjoinedChat() {
		return InputStateUnjoined
	}
	if m.isBroadcastReadOnly() {
		return InputStateBroadcastReadOnly
	}
	if m.isMutedByAdmin() {
		return InputStateMutedByAdmin
	}
	return InputStateNormal
}

// CanFocusInput indicates whether keyboard focus (Tab/Shift-Tab) can land on the input component.
func (m *Model) CanFocusInput() bool {
	state := m.CurrentInputState()
	return state == InputStateNormal || state == InputStateUnjoined
}

// CanSendMessages indicates whether the user can type and send new messages in the current chat.
func (m *Model) CanSendMessages() bool {
	return m.CurrentInputState() == InputStateNormal
}

func (m *Model) isBroadcastReadOnly() bool {
	if m.Mode == ModeChannels {
		return m.SelectedChannel.ID != "" && m.SelectedChannel.IsBroadcast && !m.SelectedChannel.IsCreator
	}
	if m.Mode == ModeAll {
		if sel := m.All.SelectedItem(); sel != nil {
			if ch, ok := sel.(types.ChannelInfo); ok && ch.IsBroadcast && !ch.IsCreator {
				return true
			}
		} else if m.SelectedChannel.ID != "" && m.SelectedChannel.IsBroadcast && !m.SelectedChannel.IsCreator {
			return true
		}
	}
	return false
}

func (m *Model) isMutedByAdmin() bool {
	if m.Mode == ModeGroups {
		return m.SelectedGroup.ID != "" && m.SelectedGroup.IsMuted
	}
	if m.Mode == ModeChannels {
		return m.SelectedChannel.ID != "" && m.SelectedChannel.IsMuted
	}
	if m.Mode == ModeAll {
		if sel := m.All.SelectedItem(); sel != nil {
			if ch, ok := sel.(types.ChannelInfo); ok {
				return ch.IsMuted
			}
		}
		if m.SelectedGroup.ID != "" && m.SelectedGroup.IsMuted {
			return true
		}
		if m.SelectedChannel.ID != "" && m.SelectedChannel.IsMuted {
			return true
		}
	}
	return false
}

// SelectedMessageHasComments checks if the currently highlighted message in the stream has associated comments.
func (m *Model) SelectedMessageHasComments() (bool, int, int64) {
	if m.SelectedMessageIndex >= 0 && m.SelectedMessageIndex < len(m.Conversations) {
		msg := m.Conversations[m.SelectedMessageIndex]
		if msg.HasComments || msg.RepliesCount > 0 || msg.RepliesChannelID != 0 {
			return true, msg.RepliesCount, msg.RepliesChannelID
		}
	}
	return false, 0, 0
}

// renderRestrictedInputBanner renders an un-focusable styled banner in place of the input box.
func renderRestrictedInputBanner(m *Model, d layoutDimensions) string {
	boxWidth := max(0, d.mainWidth-2)
	if boxWidth <= 0 {
		boxWidth = max(0, m.Width-2)
	}

	state := m.CurrentInputState()
	var bannerContent string

	switch state {
	case InputStateMutedByAdmin:
		bannerContent = lipgloss.NewStyle().
			Foreground(DefaultTheme.WarningColor).
			Bold(true).
			Render("🔇 Muted by admin")
	case InputStateBroadcastReadOnly:
		channelNotice := lipgloss.NewStyle().
			Foreground(DefaultTheme.SecondaryText).
			Bold(true).
			Render("📢 Broadcast Channel (Read-Only)")

		if hasComments, count, _ := m.SelectedMessageHasComments(); hasComments {
			var commentsText string
			if count > 0 {
				commentsText = fmt.Sprintf("[💬 View Comments (%d)]", count)
			} else {
				commentsText = "[💬 View Comments]"
			}
			commentShortcut := lipgloss.NewStyle().
				Foreground(DefaultTheme.AccentColor).
				Bold(true).
				Render("  •  " + commentsText)
			bannerContent = channelNotice + commentShortcut
		} else {
			bannerContent = channelNotice
		}
	default:
		bannerContent = lipgloss.NewStyle().
			Foreground(DefaultTheme.SecondaryText).
			Italic(true).
			Render("Not Allowed To Type")
	}

	bannerBox := lipgloss.NewStyle().
		Width(boxWidth).
		Height(max(1, d.inputHeight-2)).
		Align(lipgloss.Center, lipgloss.Center).
		Border(getItemBorder(false)).
		BorderForeground(getBorderColor(false)).
		Background(DefaultTheme.InputBg).
		Render(bannerContent)

	return bannerBox
}
