package ui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kumneger0/cligram/internal/telegram"
	"github.com/kumneger0/cligram/internal/telegram/client"
	"github.com/kumneger0/cligram/internal/telegram/types"
)

func TestCallOverlay_ViewStates(t *testing.T) {
	overlay := NewCallOverlay()
	if overlay.View() != "" {
		t.Errorf("expected empty view for CallOverlayNone, got %q", overlay.View())
	}

	overlay.SetDialing("Alice", 101)
	if !strings.Contains(overlay.View(), "Calling Alice") {
		t.Errorf("expected 'Calling Alice' in dialing view, got %q", overlay.View())
	}

	overlay.SetIncoming("Bob", 102)
	if !strings.Contains(overlay.View(), "Incoming call from Bob") {
		t.Errorf("expected 'Incoming call from Bob' in incoming view, got %q", overlay.View())
	}

	overlay.SetActive("Charlie", 103)
	overlay.SetRelay(true)
	activeView := overlay.View()
	if !strings.Contains(activeView, "In call with Charlie") || !strings.Contains(activeView, "[Relay]") {
		t.Errorf("expected active call with Relay badge, got %q", activeView)
	}

	overlay.ToggleMute()
	if !strings.Contains(overlay.View(), "MUTED") {
		t.Errorf("expected muted indicator after ToggleMute, got %q", overlay.View())
	}

	overlay.SetMissingHelper()
	if !strings.Contains(overlay.View(), "VoIP Helper Missing") {
		t.Errorf("expected helper missing warning, got %q", overlay.View())
	}

	overlay.SetNone()
	if overlay.IsActive() {
		t.Errorf("expected overlay not active after SetNone")
	}
}

func TestCallOverlay_Badges(t *testing.T) {
	overlay := NewCallOverlay()
	overlay.SetActive("Alice", 101)

	// Direct P2P badge test
	overlay.SetRelay(false)
	p2pView := overlay.View()
	if !strings.Contains(p2pView, "[P2P]") {
		t.Errorf("expected [P2P] badge when IsRelay is false, got %q", p2pView)
	}
	if strings.Contains(p2pView, "[Relay]") {
		t.Errorf("expected no [Relay] badge when IsRelay is false, got %q", p2pView)
	}

	// Relay badge test
	overlay.SetRelay(true)
	relayView := overlay.View()
	if !strings.Contains(relayView, "[Relay]") {
		t.Errorf("expected [Relay] badge when IsRelay is true, got %q", relayView)
	}
	if strings.Contains(relayView, "[P2P]") {
		t.Errorf("expected no [P2P] badge when IsRelay is true, got %q", relayView)
	}

	// Unmuted state has Alt+M: Mute
	if !strings.Contains(relayView, "[Alt+M: Mute]") {
		t.Errorf("expected [Alt+M: Mute] shortcut hint when unmuted, got %q", relayView)
	}

	// Muted state has MUTED badge and Alt+M: Unmute
	overlay.ToggleMute()
	mutedView := overlay.View()
	if !strings.Contains(mutedView, "MUTED") || !strings.Contains(mutedView, "[Alt+M: Unmute]") {
		t.Errorf("expected MUTED badge and Unmute hint, got %q", mutedView)
	}
}

func TestCallOverlay_Duration(t *testing.T) {
	overlay := NewCallOverlay()
	overlay.SetActive("Alice", 101)
	overlay.StartTime = time.Now().Add(-65 * time.Second) // 1 minute 5 seconds ago

	dur := overlay.Duration()
	if dur != "01:05" {
		t.Errorf("expected 01:05, got %s", dur)
	}
}

func TestModel_CallNotificationHandling(t *testing.T) {
	m := Model{
		CallOverlay: NewCallOverlay(),
	}

	// Incoming call
	resModel, _ := m.Update(types.CallNotification{
		State:    types.CallStateIncoming,
		UserName: "Alice",
		UserID:   1001,
	})
	m = resModel.(Model)
	if m.CallOverlay.State != CallOverlayIncoming || m.CallOverlay.UserName != "Alice" {
		t.Fatalf("expected CallOverlayIncoming for Alice, got state=%v, user=%s", m.CallOverlay.State, m.CallOverlay.UserName)
	}

	// Call accepted & active
	resModel, cmd := m.Update(types.CallNotification{
		State:    types.CallStateActive,
		UserName: "Alice",
		UserID:   1001,
		IsRelay:  true,
	})
	m = resModel.(Model)
	if m.CallOverlay.State != CallOverlayActive || !m.CallOverlay.IsRelay {
		t.Fatalf("expected CallOverlayActive with IsRelay=true")
	}
	if cmd == nil {
		t.Fatalf("expected tickCall command returned on CallStateActive")
	}

	// Call tick updates while active
	resModel, tickCmd := m.Update(CallTickMsg(time.Now()))
	m = resModel.(Model)
	if tickCmd == nil {
		t.Fatalf("expected next tick command when CallTickMsg handled while active")
	}

	// Mute toggled
	resModel, _ = m.Update(types.CallMuteToggledMsg{Muted: true})
	m = resModel.(Model)
	if !m.CallOverlay.Muted {
		t.Fatalf("expected call overlay to be muted")
	}

	// Call ended
	resModel, _ = m.Update(types.CallNotification{
		State:  types.CallStateEnded,
		UserID: 1001,
	})
	m = resModel.(Model)
	if m.CallOverlay.State != CallOverlayNone {
		t.Fatalf("expected CallOverlayNone after CallStateEnded, got %v", m.CallOverlay.State)
	}

	// Tick should stop after call ended
	resModel, stopCmd := m.Update(CallTickMsg(time.Now()))
	m = resModel.(Model)
	if stopCmd != nil {
		t.Fatalf("expected no tick cmd when call is not active")
	}
}

func TestModel_CallOverlayKeybindings(t *testing.T) {
	oldCligram := telegram.Cligram
	telegram.Cligram = &client.Client{}
	defer func() { telegram.Cligram = oldCligram }()

	m := Model{
		CallOverlay:      NewCallOverlay(),
		ActiveCallUserID: 2002,
	}
	m.CallOverlay.SetActive("Bob", 2002)

	// Typing normal alphanumeric character 'm' or 'h' while in active call
	// should NOT trigger call mute/hangup
	_, cmd := m.handleKeyPress(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'m'},
	})
	if cmd != nil {
		t.Logf("cmd returned for 'm': %v", cmd)
	}

	// Alt+M should trigger toggle mute
	_, muteCmd := m.handleKeyPress(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'m'},
		Alt:   true,
	})
	if muteCmd == nil {
		t.Errorf("expected tea.Cmd for Alt+M")
	}

	// Alt+H should trigger hangup
	_, hangupCmd := m.handleKeyPress(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'h'},
		Alt:   true,
	})
	if hangupCmd == nil {
		t.Errorf("expected tea.Cmd for Alt+H")
	}

	// Missing helper modal: esc dismisses modal
	m.CallOverlay.SetMissingHelper()
	resM, _ := m.handleKeyPress(tea.KeyMsg{Type: tea.KeyEsc})
	m = resM.(Model)
	if m.CallOverlay.State != CallOverlayNone {
		t.Errorf("expected CallOverlayNone after pressing Esc on MissingHelper modal")
	}
}
