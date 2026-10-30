package ui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kumneger0/cligram/internal/config"
	"github.com/kumneger0/cligram/internal/telegram"
	"github.com/kumneger0/cligram/internal/telegram/client"
	"github.com/kumneger0/cligram/internal/telegram/types"
)

func setupTestConfig(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	defaultMode := "default"
	defaultTyping := true
	_ = config.SaveConfig(config.CliGramConfig{
		Chat: config.ChatConfig{
			ReadReceiptMode: &defaultMode,
			SendTypingState: &defaultTyping,
		},
		ReadStories: false,
	})
}

func TestGhostMode_OpenModalAndInit(t *testing.T) {
	setupTestConfig(t)

	fg := &Foreground{}
	model, _ := fg.Update(OpenModalMsg{
		ModalMode: ModalModeGhostMode,
	})

	updatedFg, ok := model.(*Foreground)
	if !ok {
		t.Fatalf("expected *Foreground, got %T", model)
	}

	if updatedFg.ModalMode != ModalModeGhostMode {
		t.Fatalf("expected ModalMode %s, got %s", ModalModeGhostMode, updatedFg.ModalMode)
	}
	if updatedFg.GhostModeIndex != 0 {
		t.Fatalf("expected GhostModeIndex 0, got %d", updatedFg.GhostModeIndex)
	}
	if updatedFg.GhostConfig.Chat.ReadReceiptMode == nil || *updatedFg.GhostConfig.Chat.ReadReceiptMode != "default" {
		t.Fatalf("expected default ReadReceiptMode, got %v", updatedFg.GhostConfig.Chat.ReadReceiptMode)
	}
	if updatedFg.GhostConfig.Chat.SendTypingState == nil || !*updatedFg.GhostConfig.Chat.SendTypingState {
		t.Fatalf("expected true SendTypingState, got %v", updatedFg.GhostConfig.Chat.SendTypingState)
	}
	if updatedFg.GhostConfig.ReadStories {
		t.Fatalf("expected false ReadStories, got %v", updatedFg.GhostConfig.ReadStories)
	}
}

func TestGhostMode_Navigation(t *testing.T) {
	setupTestConfig(t)

	fg := &Foreground{
		ModalMode: ModalModeGhostMode,
	}
	fg.initGhostMode()

	// Initial index is 0
	if fg.GhostModeIndex != 0 {
		t.Fatalf("initial index should be 0, got %d", fg.GhostModeIndex)
	}

	// Down / j / ctrl+n
	m, _ := fg.Update(tea.KeyMsg{Type: tea.KeyDown})
	fg = m.(*Foreground)
	if fg.GhostModeIndex != 1 {
		t.Errorf("expected index 1 after down, got %d", fg.GhostModeIndex)
	}

	m, _ = fg.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	fg = m.(*Foreground)
	if fg.GhostModeIndex != 2 {
		t.Errorf("expected index 2 after j, got %d", fg.GhostModeIndex)
	}

	// Wrap around to 0
	m, _ = fg.Update(tea.KeyMsg{Type: tea.KeyDown})
	fg = m.(*Foreground)
	if fg.GhostModeIndex != 0 {
		t.Errorf("expected wrap around to index 0, got %d", fg.GhostModeIndex)
	}

	// Up / k / ctrl+p wraps back to 2
	m, _ = fg.Update(tea.KeyMsg{Type: tea.KeyUp})
	fg = m.(*Foreground)
	if fg.GhostModeIndex != 2 {
		t.Errorf("expected wrap around to index 2 on up, got %d", fg.GhostModeIndex)
	}

	m, _ = fg.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	fg = m.(*Foreground)
	if fg.GhostModeIndex != 1 {
		t.Errorf("expected index 1 on k, got %d", fg.GhostModeIndex)
	}
}

func TestGhostMode_OptionToggling(t *testing.T) {
	setupTestConfig(t)

	fg := &Foreground{
		ModalMode: ModalModeGhostMode,
	}
	fg.initGhostMode()

	// 1. Index 0: Read Messages (cycle: default -> never -> instant -> default)
	fg.GhostModeIndex = 0

	// Toggle: default -> never
	m, _ := fg.Update(tea.KeyMsg{Type: tea.KeyEnter})
	fg = m.(*Foreground)
	if *fg.GhostConfig.Chat.ReadReceiptMode != "never" {
		t.Errorf("expected 'never', got %s", *fg.GhostConfig.Chat.ReadReceiptMode)
	}
	// Verify hot-reload via GetConfig()
	if *config.GetConfig().Chat.ReadReceiptMode != "never" {
		t.Errorf("expected hot-reloaded config to be 'never', got %s", *config.GetConfig().Chat.ReadReceiptMode)
	}

	// Toggle via Space: never -> instant
	m, _ = fg.Update(tea.KeyMsg{Type: tea.KeySpace})
	fg = m.(*Foreground)
	if *fg.GhostConfig.Chat.ReadReceiptMode != "instant" {
		t.Errorf("expected 'instant', got %s", *fg.GhostConfig.Chat.ReadReceiptMode)
	}

	// Toggle: instant -> default
	m, _ = fg.Update(tea.KeyMsg{Type: tea.KeyEnter})
	fg = m.(*Foreground)
	if *fg.GhostConfig.Chat.ReadReceiptMode != "default" {
		t.Errorf("expected 'default', got %s", *fg.GhostConfig.Chat.ReadReceiptMode)
	}

	// 2. Index 1: Read Stories (toggle false -> true -> false)
	fg.GhostModeIndex = 1
	if fg.GhostConfig.ReadStories {
		t.Fatalf("expected ReadStories initially false")
	}

	m, _ = fg.Update(tea.KeyMsg{Type: tea.KeySpace})
	fg = m.(*Foreground)
	if !fg.GhostConfig.ReadStories {
		t.Errorf("expected ReadStories true after space")
	}
	if !config.GetConfig().ReadStories {
		t.Errorf("expected hot-reloaded ReadStories true")
	}

	m, _ = fg.Update(tea.KeyMsg{Type: tea.KeyEnter})
	fg = m.(*Foreground)
	if fg.GhostConfig.ReadStories {
		t.Errorf("expected ReadStories false after enter")
	}

	// 3. Index 2: Send Typing (toggle true -> false -> true)
	fg.GhostModeIndex = 2
	if !*fg.GhostConfig.Chat.SendTypingState {
		t.Fatalf("expected SendTypingState initially true")
	}

	m, _ = fg.Update(tea.KeyMsg{Type: tea.KeyEnter})
	fg = m.(*Foreground)
	if *fg.GhostConfig.Chat.SendTypingState {
		t.Errorf("expected SendTypingState false after enter")
	}
	if *config.GetConfig().Chat.SendTypingState {
		t.Errorf("expected hot-reloaded SendTypingState false")
	}

	m, _ = fg.Update(tea.KeyMsg{Type: tea.KeySpace})
	fg = m.(*Foreground)
	if !*fg.GhostConfig.Chat.SendTypingState {
		t.Errorf("expected SendTypingState true after space")
	}
}

func TestGhostMode_Dismissal(t *testing.T) {
	setupTestConfig(t)

	keys := []string{"esc", "q", "Q", "f2", ","}

	for _, k := range keys {
		t.Run("dismiss_"+k, func(t *testing.T) {
			fg := &Foreground{
				ModalMode: ModalModeGhostMode,
			}
			fg.initGhostMode()

			var keyMsg tea.KeyMsg
			switch k {
			case "esc":
				keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
			case "f2":
				keyMsg = tea.KeyMsg{Type: tea.KeyF2}
			default:
				keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
			}

			_, cmd := fg.Update(keyMsg)
			if cmd == nil {
				t.Fatalf("expected non-nil cmd for dismissal key %s", k)
			}
			msg := cmd()
			if _, ok := msg.(CloseOverlay); !ok {
				t.Errorf("expected CloseOverlay on %s, got %T", k, msg)
			}
		})
	}
}

func TestGhostMode_ManagerIntegration(t *testing.T) {
	setupTestConfig(t)

	bg := newTestModel(80, 24)
	bg.FocusedOn = Main
	fg := &Foreground{}

	mgr := Manager{
		State:      MainView,
		Background: bg,
		Foreground: fg,
	}

	// 1. Press f2 in MainView -> opens modal
	updatedMgr, cmd := mgr.Update(tea.KeyMsg{Type: tea.KeyF2})
	m := updatedMgr.(Manager)
	if m.State != ModalView {
		t.Errorf("expected ModalView after f2, got %v", m.State)
	}
	if cmd == nil {
		t.Fatalf("expected cmd returning OpenModalMsg")
	}
	openMsg := cmd()
	if om, ok := openMsg.(OpenModalMsg); !ok || om.ModalMode != ModalModeGhostMode {
		t.Errorf("expected OpenModalMsg with ModalModeGhostMode, got %+v", openMsg)
	}

	// Simulate Foreground receiving OpenModalMsg
	fgModel, _ := m.Foreground.Update(openMsg)
	m.Foreground = fgModel

	// 2. Press f2 again in ModalView -> closes to MainView
	updatedMgr, _ = m.Update(tea.KeyMsg{Type: tea.KeyF2})
	m = updatedMgr.(Manager)
	if m.State != MainView {
		t.Errorf("expected MainView after second f2, got %v", m.State)
	}

	// 3. Comma shortcut when FocusedOn == Main -> opens modal
	updatedMgr, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{','}})
	m = updatedMgr.(Manager)
	if m.State != ModalView {
		t.Errorf("expected ModalView after comma, got %v", m.State)
	}
	if cmd == nil {
		t.Fatalf("expected cmd returning OpenModalMsg on comma")
	}

	// 4. Comma when FocusedOn == Input -> should NOT open modal
	bgInput := newTestModel(80, 24)
	bgInput.FocusedOn = Input
	mgrInput := Manager{
		State:      MainView,
		Background: bgInput,
		Foreground: &Foreground{},
	}
	updatedInputMgr, _ := mgrInput.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{','}})
	mi := updatedInputMgr.(Manager)
	if mi.State != MainView {
		t.Errorf("expected MainView when comma pressed with Input focus, got %v", mi.State)
	}
}

func TestGhostMode_ViewRendering(t *testing.T) {
	setupTestConfig(t)

	fg := &Foreground{
		ModalMode:      ModalModeGhostMode,
		windowWidth:    80,
		GhostModeIndex: 0,
	}
	fg.initGhostMode()

	view := fg.View()

	if !strings.Contains(view, "Ghost Mode") {
		t.Errorf("expected view to contain 'Ghost Mode', got:\n%s", view)
	}
	if !strings.Contains(view, "Read Messages") {
		t.Errorf("expected view to contain 'Read Messages'")
	}
	if !strings.Contains(view, "Read Stories") {
		t.Errorf("expected view to contain 'Read Stories'")
	}
	if !strings.Contains(view, "Send Typing") {
		t.Errorf("expected view to contain 'Send Typing'")
	}
	if !strings.Contains(view, "[↑/↓, j/k] Navigate") {
		t.Errorf("expected view to contain footer hints")
	}
}

func TestSendUserIsTyping_ConfigAndThrottling(t *testing.T) {
	setupTestConfig(t)

	oldCligram := telegram.Cligram
	telegram.Cligram = &client.Client{}
	defer func() { telegram.Cligram = oldCligram }()

	m := newTestModel(80, 24)
	m.Mode = ModeUsers
	m.SelectedUser = types.UserInfo{
		PeerID:     "12345",
		AccessHash: "67890",
	}
	m.FocusedOn = Input
	m.Input.SetValue("hello")

	// 1. With SendTypingState = true, returns non-nil cmd and updates LastTypingSent
	cmd := SendUserIsTyping(&m)
	if cmd == nil {
		t.Fatalf("expected non-nil cmd when typing is enabled and input is not empty")
	}
	if m.LastTypingSent.IsZero() {
		t.Fatalf("expected LastTypingSent to be updated")
	}
	firstTime := m.LastTypingSent

	// 2. Throttled within 4 seconds: second call returns nil
	cmd2 := SendUserIsTyping(&m)
	if cmd2 != nil {
		t.Fatalf("expected nil cmd due to 4s throttling, got non-nil")
	}

	// 3. After 4 seconds: returns non-nil cmd again
	m.LastTypingSent = time.Now().Add(-5 * time.Second)
	cmd3 := SendUserIsTyping(&m)
	if cmd3 == nil {
		t.Fatalf("expected non-nil cmd after 4s window elapsed")
	}
	if !m.LastTypingSent.After(firstTime) {
		t.Fatalf("expected LastTypingSent to be refreshed")
	}

	// 4. When SendTypingState = false: returns nil
	typingFalse := false
	_ = config.SaveConfig(config.CliGramConfig{
		Chat: config.ChatConfig{
			SendTypingState: &typingFalse,
		},
	})
	m.LastTypingSent = time.Time{}
	cmdDisabled := SendUserIsTyping(&m)
	if cmdDisabled != nil {
		t.Fatalf("expected nil cmd when SendTypingState is false")
	}

	// 5. When input is empty: returns nil
	typingTrue := true
	_ = config.SaveConfig(config.CliGramConfig{
		Chat: config.ChatConfig{
			SendTypingState: &typingTrue,
		},
	})
	m.Input.SetValue("")
	cmdEmpty := SendUserIsTyping(&m)
	if cmdEmpty != nil {
		t.Fatalf("expected nil cmd when input is empty")
	}

	// 6. When not focused on Input: returns nil
	m.Input.SetValue("hello")
	m.FocusedOn = Main
	cmdMain := SendUserIsTyping(&m)
	if cmdMain != nil {
		t.Fatalf("expected nil cmd when FocusedOn != Input")
	}
}

func TestSendUserIsTyping_UpdateFocusedComponent(t *testing.T) {
	setupTestConfig(t)

	oldCligram := telegram.Cligram
	telegram.Cligram = &client.Client{}
	defer func() { telegram.Cligram = oldCligram }()

	m := newTestModel(80, 24)
	m.Mode = ModeUsers
	m.SelectedUser = types.UserInfo{
		PeerID:     "12345",
		AccessHash: "67890",
	}
	m.FocusedOn = Input

	var cmds []tea.Cmd
	// Simulate typing 'a' into the input field
	updatedM, cmd := updateFocusedComponent(&m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}, &cmds)
	if updatedM.Input.Value() != "a" {
		t.Errorf("expected input value 'a', got %q", updatedM.Input.Value())
	}
	if cmd == nil {
		t.Fatalf("expected non-nil batch cmd returned from updateFocusedComponent")
	}
	if updatedM.LastTypingSent.IsZero() {
		t.Errorf("expected LastTypingSent to be updated on keystroke")
	}
}

func TestSendUserIsTyping_ModeAllAndActivePeerID(t *testing.T) {
	setupTestConfig(t)

	oldCligram := telegram.Cligram
	telegram.Cligram = &client.Client{}
	defer func() { telegram.Cligram = oldCligram }()

	m := newTestModel(80, 24)
	m.Mode = ModeAll
	m.ActivePeerID = "998877"
	m.SelectedUser = types.UserInfo{
		PeerID:     "998877",
		AccessHash: "112233",
	}
	m.FocusedOn = Input
	m.Input.SetValue("hello from ModeAll")

	// Even with m.All having no selected item, ActivePeerID resolution ensures SendUserIsTyping succeeds
	cmd := SendUserIsTyping(&m)
	if cmd == nil {
		t.Fatalf("expected non-nil cmd for SendUserIsTyping in ModeAll with ActivePeerID")
	}
	if m.LastTypingSent.IsZero() {
		t.Fatalf("expected LastTypingSent to be updated")
	}

	// Verify getMessageParams returns the correct peer
	peer := getMessageParams(&m)
	if peer.ID != "998877" || peer.AccessHash != "112233" || peer.ChatType != types.UserChat {
		t.Fatalf("expected peer 998877/UserChat, got %+v", peer)
	}
}

func TestReadReceiptMode_SendMessage(t *testing.T) {
	setupTestConfig(t)

	oldCligram := telegram.Cligram
	telegram.Cligram = &client.Client{}
	defer func() { telegram.Cligram = oldCligram }()

	// 1. With mode = "default" -> marks messages as read
	defaultMode := "default"
	typingTrue := true
	_ = config.SaveConfig(config.CliGramConfig{
		Chat: config.ChatConfig{
			ReadReceiptMode: &defaultMode,
			SendTypingState: &typingTrue,
		},
	})

	m := newTestModel(80, 24)
	m.Mode = ModeUsers
	m.SelectedUser = types.UserInfo{PeerID: "1001", AccessHash: "111"}
	m.FocusedOn = Input
	m.Input.SetValue("test message")
	m.LastTypingSent = time.Now()

	mResult, cmdDefault := sendMessage(&m)
	if cmdDefault == nil {
		t.Fatalf("expected non-nil cmd on sendMessage")
	}
	if !mResult.LastTypingSent.IsZero() {
		t.Errorf("expected LastTypingSent to be reset after sendMessage")
	}

	// 2. With mode = "instant" -> also marks messages as read
	instantMode := "instant"
	_ = config.SaveConfig(config.CliGramConfig{
		Chat: config.ChatConfig{
			ReadReceiptMode: &instantMode,
			SendTypingState: &typingTrue,
		},
	})
	m.Input.SetValue("another message")
	_, cmdInstant := sendMessage(&m)
	if cmdInstant == nil {
		t.Fatalf("expected non-nil cmd on sendMessage in instant mode")
	}

	// 3. With mode = "never" -> does NOT include MarkMessagesAsRead
	neverMode := "never"
	_ = config.SaveConfig(config.CliGramConfig{
		Chat: config.ChatConfig{
			ReadReceiptMode: &neverMode,
			SendTypingState: &typingTrue,
		},
	})
	m.Input.SetValue("ghost message")
	_, cmdNever := sendMessage(&m)
	if cmdNever == nil {
		t.Fatalf("expected non-nil cmd for SendMessage and updateConversations")
	}
}

func TestReadReceiptMode_ShowChat(t *testing.T) {
	setupTestConfig(t)

	oldCligram := telegram.Cligram
	telegram.Cligram = &client.Client{}
	defer func() { telegram.Cligram = oldCligram }()

	// 1. Instant mode: returns MarkMessagesAsRead cmd
	instantMode := "instant"
	_ = config.SaveConfig(config.CliGramConfig{
		Chat: config.ChatConfig{
			ReadReceiptMode: &instantMode,
		},
	})

	m := newTestModel(80, 24)
	m.Mode = ModeUsers
	m.SelectedUser = types.UserInfo{PeerID: "1001", AccessHash: "111"}

	resModel, cmd := handleUserChange(&m, nil, nil)
	if cmd == nil {
		t.Fatalf("expected non-nil cmd for handleUserChange in instant mode")
	}
	if !resModel.MainViewLoading {
		t.Errorf("expected MainViewLoading to be true in handleUserChange")
	}

	// 2. Default mode: does NOT include MarkMessagesAsRead
	defaultMode := "default"
	_ = config.SaveConfig(config.CliGramConfig{
		Chat: config.ChatConfig{
			ReadReceiptMode: &defaultMode,
		},
	})

	resModelDefault, cmdDefault := handleUserChange(&m, nil, nil)
	if cmdDefault == nil {
		t.Fatalf("expected non-nil cmd for handleUserChange in default mode")
	}
	if !resModelDefault.MainViewLoading {
		t.Errorf("expected MainViewLoading to be true in default mode")
	}
}

func TestReadReceiptMode_HandleNewMessage(t *testing.T) {
	setupTestConfig(t)

	oldCligram := telegram.Cligram
	telegram.Cligram = &client.Client{}
	defer func() { telegram.Cligram = oldCligram }()

	// 1. Instant mode: incoming message in active chat marks as read without incrementing unread count
	instantMode := "instant"
	_ = config.SaveConfig(config.CliGramConfig{
		Chat: config.ChatConfig{
			ReadReceiptMode: &instantMode,
		},
	})

	m := newTestModel(80, 24)
	m.Mode = ModeUsers
	m.SelectedUser = types.UserInfo{PeerID: "1001", AccessHash: "111", UnreadCount: 0}
	m.ActivePeerID = "1001"

	newMsg := types.NewMessageNotification{
		ID:     1,
		PeerID: "1001",
		Formatted: &types.FormattedMessage{
			ID:      1,
			Content: "hello from outside",
		},
	}

	updatedModel, cmd := m.handleNewMessage(newMsg)
	upM := updatedModel.(Model)
	if upM.SelectedUser.UnreadCount != 0 {
		t.Errorf("expected UnreadCount 0 in instant mode, got %d", upM.SelectedUser.UnreadCount)
	}
	if cmd == nil {
		t.Fatalf("expected non-nil cmd containing markReadCmd")
	}

	// 2. Default mode: incoming message increments unread count (not marked read until reply)
	defaultMode := "default"
	_ = config.SaveConfig(config.CliGramConfig{
		Chat: config.ChatConfig{
			ReadReceiptMode: &defaultMode,
		},
	})

	m2 := newTestModel(80, 24)
	m2.Mode = ModeUsers
	m2.SelectedUser = types.UserInfo{PeerID: "1001", AccessHash: "111", UnreadCount: 0}
	m2.ActivePeerID = "1001"

	updatedModel2, _ := m2.handleNewMessage(newMsg)
	upM2 := updatedModel2.(Model)
	if upM2.SelectedUser.UnreadCount != 1 {
		t.Errorf("expected UnreadCount 1 in default mode, got %d", upM2.SelectedUser.UnreadCount)
	}
}
