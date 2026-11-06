package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kumneger0/cligram/internal/telegram/types"
	"go.dalton.dog/bubbleup"
)

func TestActionMenu_Message_OpenAndFilter(t *testing.T) {
	now := time.Now()

	// 1. Own recent message with content, media, and comments
	ownRecentMsg := types.FormattedMessage{
		ID:       101,
		Sender:   "You",
		IsFromMe: true,
		Date:     now.Add(-1 * time.Hour),
		Content:  "Check out https://example.com",
		MediaAttachment: &types.MediaAttachment{
			Status: types.MediaStatusDownloaded,
		},
		HasComments:  true,
		RepliesCount: 5,
	}

	m := Model{
		FocusedOn:            Main,
		Conversations:        []types.FormattedMessage{ownRecentMsg},
		SelectedMessageIndex: 0,
	}

	resModel, cmd := m.handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if cmd == nil {
		t.Fatalf("expected command on 'x' key press, got nil")
	}

	msg := cmd()
	modalMsg, ok := msg.(OpenModalMsg)
	if !ok {
		t.Fatalf("expected OpenModalMsg, got %T", msg)
	}
	if modalMsg.ModalMode != ModalModeActionMenu {
		t.Errorf("expected ModalModeActionMenu, got %v", modalMsg.ModalMode)
	}
	if modalMsg.ActionMenuTitle != "Message Actions" {
		t.Errorf("expected 'Message Actions' title, got %q", modalMsg.ActionMenuTitle)
	}

	findItem := func(actionID string) *ActionMenuItem {
		for _, item := range modalMsg.ActionMenuItems {
			if item.ActionID == actionID {
				return &item
			}
		}
		return nil
	}

	// Should contain reply, edit, delete, forward, react, copy, open_media, comments
	for _, id := range []string{"reply", "edit", "delete", "forward", "react", "copy", "open_media", "comments"} {
		if findItem(id) == nil {
			t.Errorf("expected action %q to be present in menu", id)
		}
	}

	// 2. Someone else's message older than 48 hours without media or links
	otherOldMsg := types.FormattedMessage{
		ID:       102,
		Sender:   "Alice",
		IsFromMe: false,
		Date:     now.Add(-50 * time.Hour),
		Content:  "Just plain text",
	}

	m.Conversations = []types.FormattedMessage{otherOldMsg}
	m.SelectedMessageIndex = 0

	_, cmd = m.handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if cmd == nil {
		t.Fatalf("expected cmd on 'x', got nil")
	}
	modalMsg = cmd().(OpenModalMsg)

	findItemInModal := func(actionID string) *ActionMenuItem {
		for _, item := range modalMsg.ActionMenuItems {
			if item.ActionID == actionID {
				return &item
			}
		}
		return nil
	}

	// Edit should NOT be present
	if findItemInModal("edit") != nil {
		t.Errorf("edit should NOT be present on other's old message")
	}
	// Open media should NOT be present
	if findItemInModal("open_media") != nil {
		t.Errorf("open_media should NOT be present on text-only message")
	}
	// Comments should NOT be present
	if findItemInModal("comments") != nil {
		t.Errorf("comments should NOT be present on non-reply message")
	}
	// Reply, delete, forward, react, copy should be present
	for _, id := range []string{"reply", "delete", "forward", "react", "copy"} {
		if findItemInModal(id) == nil {
			t.Errorf("expected action %q to be present", id)
		}
	}

	_ = resModel
}

func TestActionMenu_Message_EmptySelection(t *testing.T) {
	m := Model{
		FocusedOn:            Main,
		Conversations:        nil,
		SelectedMessageIndex: -1,
	}

	_, cmd := m.handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if cmd != nil {
		t.Errorf("expected nil cmd when no message is selected, got %v", cmd)
	}
}

func TestActionMenu_Chat_Open(t *testing.T) {
	userItem := types.UserInfo{
		PeerID:            "12345",
		FirstName:         "Alice",
		LastName:          "Smith",
		Username:          "alicesmith",
		PhoneCallsPrivate: false,
		IsBot:             false,
	}

	userList := list.New([]list.Item{userItem}, list.NewDefaultDelegate(), 40, 20)

	m := Model{
		FocusedOn: SideBar,
		Mode:      ModeUsers,
		Users:     userList,
	}

	_, cmd := m.handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if cmd == nil {
		t.Fatalf("expected cmd on 'x', got nil")
	}
	modalMsg, ok := cmd().(OpenModalMsg)
	if !ok || modalMsg.ModalMode != ModalModeActionMenu {
		t.Fatalf("expected ModalModeActionMenu, got %v", modalMsg.ModalMode)
	}
	if !strings.Contains(modalMsg.ActionMenuTitle, "Alice") {
		t.Errorf("expected title to contain Alice, got %q", modalMsg.ActionMenuTitle)
	}

	findItem := func(id string) *ActionMenuItem {
		for _, item := range modalMsg.ActionMenuItems {
			if item.ActionID == id {
				return &item
			}
		}
		return nil
	}

	if findItem("open_chat") == nil {
		t.Errorf("expected open_chat action")
	}
	if findItem("call") == nil {
		t.Errorf("expected call action for user with calls allowed")
	}
	if findItem("stories") != nil {
		t.Errorf("stories should NOT be present in chat action menu")
	}
}

func TestActionMenu_Chat_GroupMembers(t *testing.T) {
	count := 42
	groupItem := types.ChannelInfo{
		ID:                "9876",
		ChannelTitle:      "Go Developers",
		IsBroadcast:       false,
		ParticipantsCount: &count,
	}

	groupList := list.New([]list.Item{groupItem}, list.NewDefaultDelegate(), 40, 20)

	m := Model{
		FocusedOn: SideBar,
		Mode:      ModeGroups,
		Groups:    groupList,
	}

	_, cmd := m.handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if cmd == nil {
		t.Fatalf("expected cmd on 'x', got nil")
	}
	modalMsg := cmd().(OpenModalMsg)

	findItem := func(id string) *ActionMenuItem {
		for _, item := range modalMsg.ActionMenuItems {
			if item.ActionID == id {
				return &item
			}
		}
		return nil
	}

	if findItem("members") == nil {
		t.Errorf("expected members action for group")
	}
	if findItem("call") != nil {
		t.Errorf("call action should NOT be present for group")
	}
}

func TestActionMenu_Foreground_KeyNavigation(t *testing.T) {
	fg := &Foreground{
		ModalMode:       ModalModeActionMenu,
		ActionMenuTitle: "Test Actions",
		ActionMenuItems: []ActionMenuItem{
			{Key: "r", Label: "Reply", ActionID: "reply"},
			{Key: "e", Label: "Edit", ActionID: "edit"},
			{Key: "d", Label: "Delete", ActionID: "delete"},
		},
		ActionMenuIndex: 0,
	}

	// Move Down (j)
	mRes, _ := fg.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	fg = mRes.(*Foreground)
	if fg.ActionMenuIndex != 1 {
		t.Errorf("expected ActionMenuIndex 1 after 'j', got %d", fg.ActionMenuIndex)
	}

	// Move Down again
	mRes, _ = fg.Update(tea.KeyMsg{Type: tea.KeyDown})
	fg = mRes.(*Foreground)
	if fg.ActionMenuIndex != 2 {
		t.Errorf("expected ActionMenuIndex 2 after Down arrow, got %d", fg.ActionMenuIndex)
	}

	// Wrap around to 0
	mRes, _ = fg.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	fg = mRes.(*Foreground)
	if fg.ActionMenuIndex != 0 {
		t.Errorf("expected ActionMenuIndex wrap to 0, got %d", fg.ActionMenuIndex)
	}

	// Move Up (k) wraps to 2
	mRes, _ = fg.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	fg = mRes.(*Foreground)
	if fg.ActionMenuIndex != 2 {
		t.Errorf("expected ActionMenuIndex wrap to 2 after 'k', got %d", fg.ActionMenuIndex)
	}

	// Press Enter to select item 2 ("delete")
	_, cmd := fg.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected batch cmd on Enter, got nil")
	}

	// Execute batch and inspect emitted messages
	rawMsg := cmd()
	batchMsgs, ok := rawMsg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("expected BatchMsg, got %T", rawMsg)
	}

	var hasCloseOverlay, hasExecuteAction bool
	var executedActionID string
	for _, c := range batchMsgs {
		if c == nil {
			continue
		}
		inner := c()
		switch v := inner.(type) {
		case CloseOverlay:
			hasCloseOverlay = true
		case ExecuteActionMsg:
			hasExecuteAction = true
			executedActionID = v.ActionID
		}
	}

	if !hasCloseOverlay {
		t.Errorf("expected CloseOverlay to be emitted on Enter")
	}
	if !hasExecuteAction || executedActionID != "delete" {
		t.Errorf("expected ExecuteActionMsg with delete, got action %q", executedActionID)
	}
}

func TestActionMenu_Foreground_DirectHotkey(t *testing.T) {
	fg := &Foreground{
		ModalMode:       ModalModeActionMenu,
		ActionMenuTitle: "Test Actions",
		ActionMenuItems: []ActionMenuItem{
			{Key: "r", Label: "Reply", ActionID: "reply"},
			{Key: "e", Label: "Edit", ActionID: "edit"},
			{Key: "d", Label: "Delete", ActionID: "delete"},
		},
		ActionMenuIndex: 0,
	}

	// Direct hotkey 'r'
	_, cmd := fg.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if cmd == nil {
		t.Fatalf("expected batch cmd on 'r', got nil")
	}

	batchMsgs, ok := cmd().(tea.BatchMsg)
	if !ok {
		t.Fatalf("expected BatchMsg, got %T", cmd())
	}

	var hasCloseOverlay, hasExecuteAction bool
	var executedActionID string
	for _, c := range batchMsgs {
		if c == nil {
			continue
		}
		inner := c()
		switch v := inner.(type) {
		case CloseOverlay:
			hasCloseOverlay = true
		case ExecuteActionMsg:
			hasExecuteAction = true
			executedActionID = v.ActionID
		}
	}

	if !hasCloseOverlay {
		t.Errorf("expected CloseOverlay on hotkey")
	}
	if !hasExecuteAction || executedActionID != "reply" {
		t.Errorf("expected ExecuteActionMsg with reply, got %q", executedActionID)
	}
}

func TestActionMenu_Foreground_Dismissal(t *testing.T) {
	fg := &Foreground{
		ModalMode:       ModalModeActionMenu,
		ActionMenuTitle: "Test Actions",
		ActionMenuItems: []ActionMenuItem{
			{Key: "r", Label: "Reply", ActionID: "reply"},
		},
	}

	// Pressing 'esc', 'q', or 'x' should return CloseOverlay
	for _, keyStr := range []string{"esc", "q", "x"} {
		var keyMsg tea.KeyMsg
		if keyStr == "esc" {
			keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
		} else {
			keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(keyStr)}
		}

		_, cmd := fg.Update(keyMsg)
		if cmd == nil {
			t.Fatalf("expected cmd on key %s, got nil", keyStr)
		}
		msg := cmd()
		if _, ok := msg.(CloseOverlay); !ok {
			t.Errorf("expected CloseOverlay on key %s, got %T", keyStr, msg)
		}
	}
}

func TestActionMenu_ExecuteAction_Copy(t *testing.T) {
	testText := "Hello Cligram clipboard test"
	m := Model{
		FocusedOn: Main,
		Conversations: []types.FormattedMessage{
			{ID: 1, Content: testText},
		},
		SelectedMessageIndex: 0,
		Alert:                *bubbleup.NewAlertModel(80, true, 10*time.Second),
	}

	newM, cmd := m.handleExecuteAction(ExecuteActionMsg{ActionID: "copy"})
	if cmd == nil {
		t.Fatalf("expected alertCmd on copy, got nil")
	}

	// Verify clipboard was written (ignore error in CI/headless if no display/clipboard available)
	clipContent, err := clipboard.ReadAll()
	if err == nil {
		if clipContent != testText {
			t.Errorf("expected clipboard content %q, got %q", testText, clipContent)
		}
	}
	_ = newM
}

func TestActionMenu_Manager_Dismissal(t *testing.T) {
	fg := &Foreground{
		ModalMode: ModalModeActionMenu,
	}
	bg := Model{
		FocusedOn: Main,
	}
	mgr := Manager{
		State:      ModalView,
		Foreground: fg,
		Background: bg,
	}

	// Pressing 'x' in ModalView with ModalModeActionMenu switches State to MainView
	updatedMgr, _ := mgr.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m := updatedMgr.(Manager)
	if m.State != MainView {
		t.Errorf("expected Manager state to be MainView after 'x', got %v", m.State)
	}
}
