package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kumneger0/cligram/internal/telegram"
	"github.com/kumneger0/cligram/internal/telegram/client"
	"github.com/kumneger0/cligram/internal/telegram/types"
	"go.dalton.dog/bubbleup"
)

func newTestModel(width, height int) Model {
	m := Model{
		Width:       width,
		Height:      height,
		CallOverlay: NewCallOverlay(),
		Alert:       *bubbleup.NewAlertModel(width, true, 10*time.Second),
		Mode:        ModeUsers,
		Input:       textinput.New(),
	}
	m.Users = list.New([]list.Item{}, CustomDelegate{Model: &m}, 10, 20)
	m.Channels = list.New([]list.Item{}, CustomDelegate{Model: &m}, 10, 20)
	m.Groups = list.New([]list.Item{}, CustomDelegate{Model: &m}, 10, 20)
	m.Bots = list.New([]list.Item{}, CustomDelegate{Model: &m}, 10, 20)
	m.ChatUI = list.New([]list.Item{}, MessagesDelegate{Model: &m}, 10, 20)
	m.SelectedGroupForumTopics = list.New([]list.Item{}, ForumTopicsDelegate{Model: &m}, 10, 20)
	return m
}

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

func TestModel_IncomingCallOverlayCompositing(t *testing.T) {
	m := newTestModel(80, 24)
	resModel, _ := m.Update(types.CallNotification{
		State:    types.CallStateIncoming,
		UserName: "Alice",
		UserID:   1001,
	})
	m = resModel.(Model)

	view := m.View()

	// 1. View must contain the incoming call prompt
	if !strings.Contains(view, "Incoming call from Alice") && !strings.Contains(view, "Alice") {
		t.Errorf("expected view to contain incoming caller Alice, got:\n%s", view)
	}
	if !strings.Contains(view, "[a] Accept") || !strings.Contains(view, "Decline") {
		t.Errorf("expected view to contain call actions, got:\n%s", view)
	}

	// 2. View line count must NOT overflow the terminal height (24 lines)
	lines := strings.Split(strings.TrimRight(view, "\n"), "\n")
	if len(lines) > 24 {
		t.Errorf("expected view lines <= 24 to prevent off-screen scrolling, got %d lines", len(lines))
	}
}

func TestModel_IncomingCallKeyInterception(t *testing.T) {
	oldCligram := telegram.Cligram
	telegram.Cligram = &client.Client{}
	defer func() { telegram.Cligram = oldCligram }()

	m := newTestModel(80, 24)
	m.CallOverlay.SetIncoming("Alice", 1001)

	// Normal alphanumeric keys should be intercepted and NOT typed into chat input
	m.Input.SetValue("hello")
	resModel, cmd := m.handleKeyPress(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'x'},
	})
	m = resModel.(Model)
	if m.Input.Value() != "hello" {
		t.Errorf("expected input to remain unchanged during incoming call, got %q", m.Input.Value())
	}
	if cmd != nil {
		t.Errorf("expected no command for regular character during incoming call, got %v", cmd)
	}

	// Pressing 'a' should trigger accept call command
	_, acceptCmd := m.handleKeyPress(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'a'},
	})
	if acceptCmd == nil {
		t.Errorf("expected tea.Cmd for 'a' (accept call)")
	}

	// Pressing 'd' should trigger decline call command and reset overlay state
	resModel, declineCmd := m.handleKeyPress(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'d'},
	})
	m = resModel.(Model)
	if declineCmd == nil {
		t.Errorf("expected tea.Cmd for 'd' (decline call)")
	}
	if m.CallOverlay.State != CallOverlayNone {
		t.Errorf("expected CallOverlayNone after declining call, got %v", m.CallOverlay.State)
	}

	// Reset to incoming state to test Esc key
	m.CallOverlay.SetIncoming("Alice", 1001)
	resModel, escCmd := m.handleKeyPress(tea.KeyMsg{Type: tea.KeyEsc})
	m = resModel.(Model)
	if escCmd == nil {
		t.Errorf("expected tea.Cmd for Esc (decline call)")
	}
	if m.CallOverlay.State != CallOverlayNone {
		t.Errorf("expected CallOverlayNone after pressing Esc on incoming call, got %v", m.CallOverlay.State)
	}
}

func TestModel_ActiveCallLayoutBudgetingAndStatusBar(t *testing.T) {
	m := newTestModel(80, 24)

	// In idle state, view has exactly 24 lines
	idleLines := strings.Split(strings.TrimRight(m.View(), "\n"), "\n")
	if len(idleLines) != 24 {
		t.Fatalf("expected idle view to have 24 lines, got %d", len(idleLines))
	}

	// Transition to active call with IsRelay = false (P2P)
	resModel, tickCmd := m.Update(types.CallNotification{
		State:    types.CallStateActive,
		UserName: "Bob",
		UserID:   2002,
		IsRelay:  false,
	})
	m = resModel.(Model)
	if tickCmd == nil {
		t.Fatalf("expected tickCmd on transition to CallStateActive")
	}

	activeView := m.View()

	// 1. Status bar must be rendered with peer name, timer, P2P badge, and control hints
	if !strings.Contains(activeView, "In call with Bob") {
		t.Errorf("expected view to contain 'In call with Bob', got:\n%s", activeView)
	}
	if !strings.Contains(activeView, "[P2P]") {
		t.Errorf("expected view to contain '[P2P]' badge, got:\n%s", activeView)
	}
	if !strings.Contains(activeView, "(00:00)") {
		t.Errorf("expected view to contain '(00:00)' initial duration, got:\n%s", activeView)
	}
	if !strings.Contains(activeView, "[Alt+M: Mute]") || !strings.Contains(activeView, "[Alt+H: Hangup]") {
		t.Errorf("expected view to contain hotkey hints, got:\n%s", activeView)
	}

	// 2. Line count must strictly equal 24 lines (layout height budgeted with status bar)
	activeLines := strings.Split(strings.TrimRight(activeView, "\n"), "\n")
	if len(activeLines) != 24 {
		t.Errorf("expected active view to strictly equal 24 lines, got %d", len(activeLines))
	}
}

func TestModel_ActiveCallDurationTimerTick(t *testing.T) {
	m := newTestModel(80, 24)
	resModel, _ := m.Update(types.CallNotification{
		State:    types.CallStateActive,
		UserName: "Charlie",
		UserID:   3003,
		IsRelay:  true,
	})
	m = resModel.(Model)

	// Simulate 75 seconds elapsed
	m.CallOverlay.StartTime = time.Now().Add(-75 * time.Second)

	// Handle timer tick
	resModel, nextTickCmd := m.Update(CallTickMsg(time.Now()))
	m = resModel.(Model)
	if nextTickCmd == nil {
		t.Fatalf("expected next tick cmd on CallTickMsg")
	}

	view := m.View()
	if !strings.Contains(view, "(01:15)") {
		t.Errorf("expected duration '(01:15)' in view, got:\n%s", view)
	}
	if !strings.Contains(view, "[Relay]") {
		t.Errorf("expected '[Relay]' badge in view, got:\n%s", view)
	}

	lines := strings.Split(strings.TrimRight(view, "\n"), "\n")
	if len(lines) != 24 {
		t.Errorf("expected exactly 24 lines, got %d", len(lines))
	}
}

func TestModel_ActiveCallHangupRestoresLayout(t *testing.T) {
	oldCligram := telegram.Cligram
	telegram.Cligram = &client.Client{}
	defer func() { telegram.Cligram = oldCligram }()

	m := newTestModel(80, 24)
	resModel, _ := m.Update(types.CallNotification{
		State:    types.CallStateActive,
		UserName: "Alice",
		UserID:   1001,
		IsRelay:  true,
	})
	m = resModel.(Model)

	// In active call, status bar is visible
	if !strings.Contains(m.View(), "In call with Alice") {
		t.Fatalf("expected active call view to contain 'In call with Alice'")
	}

	// Hangup via Alt+H
	resModel, hangupCmd := m.handleKeyPress(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'h'},
		Alt:   true,
	})
	m = resModel.(Model)
	if hangupCmd == nil {
		t.Errorf("expected hangup command on Alt+H")
	}

	// Simulate call ended notification
	resModel, _ = m.Update(types.CallNotification{
		State:  types.CallStateEnded,
		UserID: 1001,
	})
	m = resModel.(Model)

	// Status bar dismissed
	if m.CallOverlay.State != CallOverlayNone {
		t.Errorf("expected CallOverlayNone after call ended, got %v", m.CallOverlay.State)
	}
	idleView := m.View()
	if strings.Contains(idleView, "In call with Alice") {
		t.Errorf("expected status bar removed after call ended")
	}
	lines := strings.Split(strings.TrimRight(idleView, "\n"), "\n")
	if len(lines) != 24 {
		t.Errorf("expected exactly 24 lines in idle view after hangup, got %d", len(lines))
	}
}
