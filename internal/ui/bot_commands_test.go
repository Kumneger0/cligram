package ui

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kumneger0/cligram/internal/telegram/client"
	"github.com/kumneger0/cligram/internal/telegram/types"
)

func newTestModelWithBotCommands() Model {
	ti := textinput.New()
	ti.Focus()

	m := Model{
		FocusedOn:    Input,
		Input:        ti,
		ActivePeerID: "12345",
		BotCommandsCache: map[string][]types.BotCommand{
			"12345": {
				{Command: "start", Description: "Start the bot"},
				{Command: "help", Description: "Show help message"},
				{Command: "settings", Description: "Configure preferences"},
				{Command: "status", Description: "Check server status"},
			},
		},
	}
	return m
}

func TestBotCommandSuggestionsFiltering(t *testing.T) {
	m := newTestModelWithBotCommands()

	// 1. Typing "/" should show all 4 commands
	m.Input.SetValue("/")
	m.updateBotCommandSuggestions()

	if !m.ShowBotCommands {
		t.Fatalf("expected ShowBotCommands to be true for '/'")
	}
	if len(m.FilteredBotCommands) != 4 {
		t.Fatalf("expected 4 filtered commands, got %d", len(m.FilteredBotCommands))
	}
	if m.FilteredBotCommands[0].Command != "start" {
		t.Errorf("expected first command to be 'start', got %q", m.FilteredBotCommands[0].Command)
	}

	// 2. Typing "/st" should match start, status, and settings (starts with "st")
	m.Input.SetValue("/st")
	m.updateBotCommandSuggestions()

	if !m.ShowBotCommands {
		t.Fatalf("expected ShowBotCommands to be true for '/st'")
	}
	if len(m.FilteredBotCommands) != 2 {
		t.Fatalf("expected 2 filtered commands for '/st', got %d", len(m.FilteredBotCommands))
	}
	if m.FilteredBotCommands[0].Command != "start" || m.FilteredBotCommands[1].Command != "status" {
		t.Errorf("unexpected commands for '/st': %+v", m.FilteredBotCommands)
	}

	// 3. Typing "/hel" should match help
	m.Input.SetValue("/hel")
	m.updateBotCommandSuggestions()

	if !m.ShowBotCommands {
		t.Fatalf("expected ShowBotCommands to be true for '/hel'")
	}
	if len(m.FilteredBotCommands) != 1 || m.FilteredBotCommands[0].Command != "help" {
		t.Errorf("unexpected commands for '/hel': %+v", m.FilteredBotCommands)
	}

	// 4. Typing non-matching query "/xyz" should hide popup
	m.Input.SetValue("/xyz")
	m.updateBotCommandSuggestions()

	if m.ShowBotCommands {
		t.Errorf("expected ShowBotCommands to be false for '/xyz'")
	}
	if len(m.FilteredBotCommands) != 0 {
		t.Errorf("expected 0 filtered commands, got %d", len(m.FilteredBotCommands))
	}

	// 5. Space after command "/start " should hide popup
	m.Input.SetValue("/start ")
	m.updateBotCommandSuggestions()

	if m.ShowBotCommands {
		t.Errorf("expected ShowBotCommands to be false when space is typed")
	}

	// 6. Value without leading slash should hide popup
	m.Input.SetValue("hello")
	m.updateBotCommandSuggestions()

	if m.ShowBotCommands {
		t.Errorf("expected ShowBotCommands to be false for non-slash input")
	}

	// 7. Defocused input should hide popup
	m.Input.SetValue("/start")
	m.FocusedOn = Main
	m.updateBotCommandSuggestions()

	if m.ShowBotCommands {
		t.Errorf("expected ShowBotCommands to be false when FocusedOn != Input")
	}
}

func TestBotCommandNavigationAndWrapAround(t *testing.T) {
	m := newTestModelWithBotCommands()
	m.Input.SetValue("/")
	m.updateBotCommandSuggestions()

	if len(m.FilteredBotCommands) != 4 {
		t.Fatalf("expected 4 commands, got %d", len(m.FilteredBotCommands))
	}
	if m.SelectedBotCommandIndex != 0 {
		t.Fatalf("expected initial index 0, got %d", m.SelectedBotCommandIndex)
	}

	// Down arrow key
	downKey := tea.KeyMsg{Type: tea.KeyDown}
	res, _ := m.handleKeyPress(downKey)
	m = res.(Model)
	if m.SelectedBotCommandIndex != 1 {
		t.Errorf("expected index 1 after Down, got %d", m.SelectedBotCommandIndex)
	}
	if !m.SkipNextInput {
		t.Errorf("expected SkipNextInput to be true after Down navigation")
	}

	// Down again -> index 2
	res, _ = m.handleKeyPress(downKey)
	m = res.(Model)
	if m.SelectedBotCommandIndex != 2 {
		t.Errorf("expected index 2, got %d", m.SelectedBotCommandIndex)
	}

	// Down again -> index 3 (last item)
	res, _ = m.handleKeyPress(downKey)
	m = res.(Model)
	if m.SelectedBotCommandIndex != 3 {
		t.Errorf("expected index 3, got %d", m.SelectedBotCommandIndex)
	}

	// Down again -> wrap around to 0
	res, _ = m.handleKeyPress(downKey)
	m = res.(Model)
	if m.SelectedBotCommandIndex != 0 {
		t.Errorf("expected wrap around to index 0, got %d", m.SelectedBotCommandIndex)
	}

	// Up arrow key -> wrap around to 3
	upKey := tea.KeyMsg{Type: tea.KeyUp}
	res, _ = m.handleKeyPress(upKey)
	m = res.(Model)
	if m.SelectedBotCommandIndex != 3 {
		t.Errorf("expected wrap around to index 3, got %d", m.SelectedBotCommandIndex)
	}
	if !m.SkipNextInput {
		t.Errorf("expected SkipNextInput to be true after Up navigation")
	}

	// Up again -> index 2
	res, _ = m.handleKeyPress(upKey)
	m = res.(Model)
	if m.SelectedBotCommandIndex != 2 {
		t.Errorf("expected index 2, got %d", m.SelectedBotCommandIndex)
	}
}

func TestBotCommandCompletionWithTab(t *testing.T) {
	m := newTestModelWithBotCommands()
	m.Input.SetValue("/st")
	m.updateBotCommandSuggestions()

	// Select second match ("status")
	downKey := tea.KeyMsg{Type: tea.KeyDown}
	res, _ := m.handleKeyPress(downKey)
	m = res.(Model)
	if m.SelectedBotCommandIndex != 1 {
		t.Fatalf("expected index 1, got %d", m.SelectedBotCommandIndex)
	}

	// Press Tab
	tabKey := tea.KeyMsg{Type: tea.KeyTab}
	res, _ = m.handleKeyPress(tabKey)
	m = res.(Model)

	if m.Input.Value() != "/status " {
		t.Errorf("expected input to be completed to '/status ', got %q", m.Input.Value())
	}
	if m.ShowBotCommands {
		t.Errorf("expected ShowBotCommands to be false after completion")
	}
	if !m.SkipNextInput {
		t.Errorf("expected SkipNextInput to be true after Tab completion")
	}
}

func TestBotCommandCompletionWithEnter(t *testing.T) {
	m := newTestModelWithBotCommands()
	m.Input.SetValue("/h")
	m.updateBotCommandSuggestions()

	if len(m.FilteredBotCommands) != 1 || m.FilteredBotCommands[0].Command != "help" {
		t.Fatalf("expected 1 command 'help', got %+v", m.FilteredBotCommands)
	}

	// Press Enter
	res, cmd := m.handleEnterKey()
	m = res.(Model)

	if cmd != nil {
		t.Errorf("expected nil cmd on autocomplete Enter completion, got %v", cmd)
	}
	if m.Input.Value() != "/help " {
		t.Errorf("expected input to be completed to '/help ', got %q", m.Input.Value())
	}
	if m.ShowBotCommands {
		t.Errorf("expected ShowBotCommands to be false after Enter completion")
	}
	if !m.SkipNextInput {
		t.Errorf("expected SkipNextInput to be true after Enter completion")
	}
}

func TestBotCommandDismissWithEsc(t *testing.T) {
	m := newTestModelWithBotCommands()
	m.Input.SetValue("/sta")
	m.updateBotCommandSuggestions()

	if !m.ShowBotCommands {
		t.Fatalf("expected ShowBotCommands to be true")
	}

	// Press Esc
	escKey := tea.KeyMsg{Type: tea.KeyEsc}
	res, _ := m.handleKeyPress(escKey)
	m = res.(Model)

	if m.ShowBotCommands {
		t.Errorf("expected ShowBotCommands to be false after Esc")
	}
	if m.Input.Value() != "/sta" {
		t.Errorf("expected input value to remain '/sta', got %q", m.Input.Value())
	}
	if m.FocusedOn != Input {
		t.Errorf("expected focus to remain on Input after dismissing popup, got %v", m.FocusedOn)
	}
	if !m.SkipNextInput {
		t.Errorf("expected SkipNextInput to be true after Esc")
	}
}

func TestBotCommandPeerBotCommandsMsgAndCache(t *testing.T) {
	m := Model{
		FocusedOn:    Input,
		Input:        textinput.New(),
		ActivePeerID: "9999",
	}
	m.Input.SetValue("/")

	// Dispatch PeerBotCommandsMsg for the active peer
	msg := types.PeerBotCommandsMsg{
		PeerID: "9999",
		Commands: []types.BotCommand{
			{Command: "ping", Description: "Pong response"},
			{Command: "info", Description: "Bot information"},
		},
	}

	newM, _ := m.Update(msg)
	m = newM.(Model)

	if !m.ShowBotCommands {
		t.Fatalf("expected ShowBotCommands to become true after receiving PeerBotCommandsMsg")
	}
	if len(m.FilteredBotCommands) != 2 {
		t.Fatalf("expected 2 filtered commands, got %d", len(m.FilteredBotCommands))
	}
	if m.BotCommandsCache["9999"] == nil || len(m.BotCommandsCache["9999"]) != 2 {
		t.Errorf("expected cache to contain 2 commands for peer 9999")
	}

	// Client level cache helpers
	client.ClearCachedBotCommands()
	client.SetCachedBotCommands("test_peer", []types.BotCommand{
		{Command: "test", Description: "Test command"},
	})
	client.ClearCachedBotCommands()
}

func TestRenderBotCommandsPopup(t *testing.T) {
	m := newTestModelWithBotCommands()

	// When ShowBotCommands is false
	popup := renderBotCommandsPopup(&m, 80)
	if popup != "" {
		t.Errorf("expected empty string when popup is hidden, got %q", popup)
	}

	// When ShowBotCommands is true
	m.Input.SetValue("/")
	m.updateBotCommandSuggestions()

	popup = renderBotCommandsPopup(&m, 80)
	if popup == "" {
		t.Fatalf("expected non-empty rendered popup card")
	}
	if len(popup) < 20 {
		t.Errorf("rendered popup unexpectedly short: %q", popup)
	}

	// Check that popup contains command names and descriptions
	if !containsText(popup, "Commands") {
		t.Errorf("expected popup to contain 'Commands' header")
	}
	if !containsText(popup, "start") {
		t.Errorf("expected popup to contain 'start'")
	}
	if !containsText(popup, "help") {
		t.Errorf("expected popup to contain 'help'")
	}
}

func containsText(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
