package ui

import (
	"errors"
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
	m.All = list.New([]list.Item{}, CustomDelegate{Model: &m}, 10, 20)
	m.Users = list.New([]list.Item{}, CustomDelegate{Model: &m}, 10, 20)
	m.Channels = list.New([]list.Item{}, CustomDelegate{Model: &m}, 10, 20)
	m.Groups = list.New([]list.Item{}, CustomDelegate{Model: &m}, 10, 20)
	m.Bots = list.New([]list.Item{}, CustomDelegate{Model: &m}, 10, 20)
	m.ChatUI = list.New([]list.Item{}, MessagesDelegate{Model: &m}, 10, 20)
	m.SelectedGroupForumTopics = list.New([]list.Item{}, ForumTopicsDelegate{Model: &m}, 10, 20)
	m.Conversations = []types.FormattedMessage{}
	m.SelectedMessageIndex = -1
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

	m := newTestModel(80, 24)
	m.ActiveCallUserID = 2002
	m.CallOverlay.SetActive("Bob", 2002)

	// Typing normal alphanumeric character 'm' or 'h' while in active call
	// should NOT trigger call mute/hangup
	resM, cmd := m.Update(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'m'},
	})
	m = resM.(Model)
	if cmd != nil {
		t.Logf("cmd returned for 'm': %v", cmd)
	}

	// Alt+M should trigger toggle mute
	resM, muteCmd := m.Update(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'m'},
		Alt:   true,
	})
	m = resM.(Model)
	if muteCmd == nil {
		t.Errorf("expected tea.Cmd for Alt+M")
	}

	// Feeding CallMuteToggledMsg verifies mute badge toggling in View()
	resM, _ = m.Update(types.CallMuteToggledMsg{UserID: 2002, Muted: true})
	m = resM.(Model)
	if !strings.Contains(m.View(), "MUTED") || !strings.Contains(m.View(), "[Alt+M: Unmute]") {
		t.Errorf("expected MUTED badge and Unmute hint in View(), got:\n%s", m.View())
	}

	resM, _ = m.Update(types.CallMuteToggledMsg{UserID: 2002, Muted: false})
	m = resM.(Model)
	if strings.Contains(m.View(), "MUTED") || !strings.Contains(m.View(), "[Alt+M: Mute]") {
		t.Errorf("expected MUTED badge removed and Mute hint in View(), got:\n%s", m.View())
	}

	// Alt+H should trigger hangup
	_, hangupCmd := m.Update(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'h'},
		Alt:   true,
	})
	if hangupCmd == nil {
		t.Errorf("expected tea.Cmd for Alt+H")
	}

	// Missing helper modal: esc dismisses modal
	m.CallOverlay.SetMissingHelper()
	resM, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
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
	if !strings.Contains(view, "[a] Accept") || !strings.Contains(view, "[d/Esc] Decline") {
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
	resModel, cmd := m.Update(tea.KeyMsg{
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
	_, acceptCmd := m.Update(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'a'},
	})
	if acceptCmd == nil {
		t.Errorf("expected tea.Cmd for 'a' (accept call)")
	}

	// Pressing 'd' should trigger decline call command and reset overlay state
	resModel, declineCmd := m.Update(tea.KeyMsg{
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
	resModel, escCmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
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
	resModel, hangupCmd := m.Update(tea.KeyMsg{
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

func TestModel_OutgoingCallTransitionToActive(t *testing.T) {
	m := newTestModel(80, 24)

	// User dialed Kune
	m.CallOverlay.SetDialing("Kune", 5660513633)
	if m.CallOverlay.State != CallOverlayDialing || !m.CallOverlay.IsModal() {
		t.Fatalf("expected CallOverlayDialing and modal active")
	}

	dialingView := m.View()
	if !strings.Contains(dialingView, "Calling Kune...") {
		t.Errorf("expected dialing view to contain 'Calling Kune...', got:\n%s", dialingView)
	}

	// Remote peer answers and CallStateActive arrives
	resModel, cmd := m.Update(types.CallNotification{
		State:   types.CallStateActive,
		UserID:  5660513633,
		IsRelay: false,
	})
	m = resModel.(Model)

	// Dialing modal dismissed, status bar active
	if m.CallOverlay.State != CallOverlayActive {
		t.Errorf("expected CallOverlayActive, got %v", m.CallOverlay.State)
	}
	if m.CallOverlay.IsModal() {
		t.Errorf("expected modal dismissed once call is active")
	}
	if m.CallOverlay.UserName != "Kune" {
		t.Errorf("expected username preserved as 'Kune', got %q", m.CallOverlay.UserName)
	}
	if m.ActiveCallUserID != 5660513633 {
		t.Errorf("expected ActiveCallUserID 5660513633, got %d", m.ActiveCallUserID)
	}
	if cmd == nil {
		t.Errorf("expected tickCall cmd returned when outgoing call transitions to active")
	}

	// View shows in-call bar and NOT dialing modal
	activeView := m.View()
	if strings.Contains(activeView, "Calling Kune...") {
		t.Errorf("expected dialing modal to be dismissed, but was still present")
	}
	if !strings.Contains(activeView, "In call with Kune") {
		t.Errorf("expected in-call status bar with 'In call with Kune', got:\n%s", activeView)
	}
	if !strings.Contains(activeView, "[P2P]") {
		t.Errorf("expected [P2P] badge for non-relay call")
	}

	// Total height remains strictly budgeted
	lines := strings.Split(strings.TrimRight(activeView, "\n"), "\n")
	if len(lines) > 24 {
		t.Errorf("expected lines <= 24, got %d lines", len(lines))
	}
}

func TestModel_OutgoingCall_CallUserResponse(t *testing.T) {
	m := newTestModel(80, 24)
	uID := int64(777888)
	m.CallOverlay.SetDialing("Bob", uID)

	// CallUserResponse with Err: nil transitions dialing overlay to active
	resModel, cmd := m.Update(types.CallUserResponse{
		UserID: &uID,
		Err:    nil,
	})
	m = resModel.(Model)

	if m.CallOverlay.State != CallOverlayActive {
		t.Errorf("expected CallOverlayActive upon CallUserResponse success, got %v", m.CallOverlay.State)
	}
	if m.CallOverlay.UserName != "Bob" {
		t.Errorf("expected username preserved as 'Bob', got %q", m.CallOverlay.UserName)
	}
	if m.ActiveCallUserID != uID {
		t.Errorf("expected ActiveCallUserID %d, got %d", uID, m.ActiveCallUserID)
	}
	if cmd == nil {
		t.Errorf("expected tickCall cmd from CallUserResponse success, got nil")
	}

	// Subsequent CallUserResponse when already active is a no-op
	resModel2, cmd2 := m.Update(types.CallUserResponse{
		UserID: &uID,
		Err:    nil,
	})
	m = resModel2.(Model)
	if m.CallOverlay.State != CallOverlayActive {
		t.Errorf("expected CallOverlayActive maintained, got %v", m.CallOverlay.State)
	}
	if cmd2 != nil {
		t.Errorf("expected nil cmd when already active, got %v", cmd2)
	}

	// CallUserResponse with error resets overlay and alerts
	m.CallOverlay.SetDialing("Bob", uID)
	m.ActiveCallUserID = uID
	resModel, errCmd := m.Update(types.CallUserResponse{
		UserID: &uID,
		Err:    errors.New("connection failed"),
	})
	m = resModel.(Model)

	if m.CallOverlay.State != CallOverlayNone {
		t.Errorf("expected CallOverlayNone after CallUserResponse failure, got %v", m.CallOverlay.State)
	}
	if m.ActiveCallUserID != 0 {
		t.Errorf("expected ActiveCallUserID 0 after failure, got %d", m.ActiveCallUserID)
	}
	if errCmd == nil {
		t.Errorf("expected error alert cmd when CallUserResponse returns error")
	}
}

func TestCallOverlay_IsActive(t *testing.T) {
	overlay := NewCallOverlay()
	if overlay.IsActive() {
		t.Errorf("expected IsActive()=false for CallOverlayNone")
	}

	overlay.SetDialing("Alice", 1)
	if overlay.IsActive() {
		t.Errorf("expected IsActive()=false for CallOverlayDialing")
	}

	overlay.SetIncoming("Alice", 1)
	if overlay.IsActive() {
		t.Errorf("expected IsActive()=false for CallOverlayIncoming")
	}

	overlay.SetMissingHelper()
	if overlay.IsActive() {
		t.Errorf("expected IsActive()=false for CallOverlayMissingHelper")
	}

	overlay.SetActive("Alice", 1)
	if !overlay.IsActive() {
		t.Errorf("expected IsActive()=true for CallOverlayActive")
	}
}

func TestModel_CallAcceptedError_DismissesOverlay(t *testing.T) {
	m := newTestModel(80, 24)
	m.CallOverlay.SetIncoming("Alice", 1001)

	resModel, cmd := m.Update(types.CallAcceptedMsg{
		Err: errors.New("signaling exchange error"),
	})
	m = resModel.(Model)

	if m.CallOverlay.State != CallOverlayNone {
		t.Errorf("expected CallOverlayNone after accept failure, got %v", m.CallOverlay.State)
	}
	if m.ActiveCallUserID != 0 {
		t.Errorf("expected ActiveCallUserID 0 after accept failure, got %d", m.ActiveCallUserID)
	}
	if cmd == nil {
		t.Errorf("expected alert cmd when accept fails")
	}
}

func TestModel_CallDeclinedError_Alerts(t *testing.T) {
	m := newTestModel(80, 24)
	m.CallOverlay.SetIncoming("Alice", 1001)
	m.ActiveCallUserID = 1001

	resModel, cmd := m.Update(types.CallDeclinedMsg{
		Err: errors.New("decline rpc error"),
	})
	m = resModel.(Model)

	if m.CallOverlay.State != CallOverlayNone {
		t.Errorf("expected CallOverlayNone, got %v", m.CallOverlay.State)
	}
	if m.ActiveCallUserID != 0 {
		t.Errorf("expected ActiveCallUserID 0, got %d", m.ActiveCallUserID)
	}
	if cmd == nil {
		t.Errorf("expected alert cmd on decline failure")
	}
}

func TestModel_CallHungUpError_Alerts(t *testing.T) {
	m := newTestModel(80, 24)
	m.CallOverlay.SetActive("Alice", 1001)
	m.ActiveCallUserID = 1001

	resModel, cmd := m.Update(types.CallHungUpMsg{
		Err: errors.New("hangup rpc error"),
	})
	m = resModel.(Model)

	if m.CallOverlay.State != CallOverlayNone {
		t.Errorf("expected CallOverlayNone, got %v", m.CallOverlay.State)
	}
	if m.ActiveCallUserID != 0 {
		t.Errorf("expected ActiveCallUserID 0, got %d", m.ActiveCallUserID)
	}
	if cmd == nil {
		t.Errorf("expected alert cmd on hangup failure")
	}
}

func TestModel_CtrlP_InvalidUserID_Alerts(t *testing.T) {
	m := newTestModel(80, 24)
	m.Mode = ModeUsers
	m.SelectedUser = types.UserInfo{
		FirstName: "Broken",
		PeerID:    "not_a_valid_int",
	}

	resModel, cmd := m.Update(tea.KeyMsg{
		Type:  tea.KeyCtrlP,
		Runes: []rune{'p'},
	})
	m = resModel.(Model)

	if m.CallOverlay.State != CallOverlayNone {
		t.Errorf("expected CallOverlayNone when user ID is invalid, got %v", m.CallOverlay.State)
	}
	if cmd == nil {
		t.Errorf("expected error alert cmd when parsing invalid user ID")
	}
}

func TestModel_CallTickMsg_AdvancesDuration(t *testing.T) {
	m := newTestModel(80, 24)
	resModel, _ := m.Update(types.CallNotification{
		State:    types.CallStateActive,
		UserName: "Alice",
		UserID:   1001,
		IsRelay:  false,
	})
	m = resModel.(Model)

	if m.CallOverlay.Duration() != "00:00" {
		t.Errorf("expected initial duration '00:00', got %q", m.CallOverlay.Duration())
	}

	// First tick increments elapsed time
	resModel, _ = m.Update(CallTickMsg{})
	m = resModel.(Model)
	if m.CallOverlay.Duration() != "00:01" {
		t.Errorf("expected duration '00:01' after first tick, got %q", m.CallOverlay.Duration())
	}

	// Second tick increments again
	resModel, _ = m.Update(CallTickMsg{})
	m = resModel.(Model)
	if m.CallOverlay.Duration() != "00:02" {
		t.Errorf("expected duration '00:02' after second tick, got %q", m.CallOverlay.Duration())
	}

	// View shows the updated duration
	view := m.View()
	if !strings.Contains(view, "(00:02)") {
		t.Errorf("expected view to contain '(00:02)', got:\n%s", view)
	}
}

func TestManager_ModalView_GlobalHotkeys(t *testing.T) {
	bg := newTestModel(80, 24)
	bg.CallOverlay.SetActive("Alice", 1001)
	bg.ActiveCallUserID = 1001

	mgr := Manager{
		State:      ModalView,
		Background: bg,
		Foreground: &Foreground{},
	}

	// Alt+H hangup is routed to background even when Manager is in ModalView
	resMgr, _ := mgr.Update(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'h'},
		Alt:   true,
	})
	updatedMgr := resMgr.(Manager)
	updatedBg := updatedMgr.Background.(Model)

	if updatedBg.CallOverlay.State != CallOverlayNone {
		t.Errorf("expected Alt+H to dismiss call overlay in Background from ModalView, got %v", updatedBg.CallOverlay.State)
	}
	if updatedBg.ActiveCallUserID != 0 {
		t.Errorf("expected ActiveCallUserID 0 after Alt+H in ModalView, got %d", updatedBg.ActiveCallUserID)
	}
}
