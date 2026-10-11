package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kumneger0/cligram/internal/telegram/types"
)

func TestInputState_Determination(t *testing.T) {
	// 1. Normal user chat
	mUser := Model{
		Mode: ModeUsers,
		SelectedUser: types.UserInfo{
			PeerID: "1001",
		},
	}
	if state := mUser.CurrentInputState(); state != InputStateNormal {
		t.Errorf("expected InputStateNormal for users, got %v", state)
	}
	if !mUser.CanFocusInput() {
		t.Errorf("expected CanFocusInput to be true for normal user chat")
	}
	if !mUser.CanSendMessages() {
		t.Errorf("expected CanSendMessages to be true for normal user chat")
	}

	// 2. Broadcast channel where user is NOT creator
	mChannelReadOnly := Model{
		Mode: ModeChannels,
		SelectedChannel: types.ChannelInfo{
			ID:          "2001",
			IsBroadcast: true,
			IsCreator:   false,
		},
	}
	if state := mChannelReadOnly.CurrentInputState(); state != InputStateBroadcastReadOnly {
		t.Errorf("expected InputStateBroadcastReadOnly, got %v", state)
	}
	if mChannelReadOnly.CanFocusInput() {
		t.Errorf("expected CanFocusInput to be false for read-only broadcast channel")
	}
	if mChannelReadOnly.CanSendMessages() {
		t.Errorf("expected CanSendMessages to be false for read-only broadcast channel")
	}

	// 3. Broadcast channel where user IS creator
	mChannelCreator := Model{
		Mode: ModeChannels,
		SelectedChannel: types.ChannelInfo{
			ID:          "2002",
			IsBroadcast: true,
			IsCreator:   true,
		},
	}
	if state := mChannelCreator.CurrentInputState(); state != InputStateNormal {
		t.Errorf("expected InputStateNormal for channel creator, got %v", state)
	}
	if !mChannelCreator.CanFocusInput() {
		t.Errorf("expected CanFocusInput to be true for channel creator")
	}
	if !mChannelCreator.CanSendMessages() {
		t.Errorf("expected CanSendMessages to be true for channel creator")
	}

	// 4. Group where user is muted by admin
	mGroupMuted := Model{
		Mode: ModeGroups,
		SelectedGroup: types.ChannelInfo{
			ID:          "3001",
			IsBroadcast: false,
			IsMuted:     true,
		},
	}
	if state := mGroupMuted.CurrentInputState(); state != InputStateMutedByAdmin {
		t.Errorf("expected InputStateMutedByAdmin, got %v", state)
	}
	if mGroupMuted.CanFocusInput() {
		t.Errorf("expected CanFocusInput to be false for muted group")
	}
	if mGroupMuted.CanSendMessages() {
		t.Errorf("expected CanSendMessages to be false for muted group")
	}

	// 5. Unjoined public channel
	mUnjoined := Model{
		Mode: ModeChannels,
		SelectedChannel: types.ChannelInfo{
			ID:          "4001",
			IsBroadcast: true,
			IsLeft:      true,
		},
	}
	if state := mUnjoined.CurrentInputState(); state != InputStateUnjoined {
		t.Errorf("expected InputStateUnjoined, got %v", state)
	}
	if !mUnjoined.CanFocusInput() {
		t.Errorf("expected CanFocusInput to be true for unjoined channel (to allow Enter to join)")
	}
	if mUnjoined.CanSendMessages() {
		t.Errorf("expected CanSendMessages to be false for unjoined channel")
	}
}

func TestRestrictedInputBanner_Rendering(t *testing.T) {
	d := layoutDimensions{mainWidth: 80, inputHeight: 3}

	// 1. Broadcast read-only without comments
	mReadOnly := Model{
		Mode: ModeChannels,
		SelectedChannel: types.ChannelInfo{
			ID:          "2001",
			IsBroadcast: true,
			IsCreator:   false,
		},
		Conversations: []types.FormattedMessage{
			{ID: 10, Content: "Announcement without comments", HasComments: false},
		},
		SelectedMessageIndex: 0,
	}
	viewReadOnly := prepareInputView(&mReadOnly, d)
	if !strings.Contains(viewReadOnly, "Broadcast Channel (Read-Only)") {
		t.Errorf("expected 'Broadcast Channel (Read-Only)' banner, got:\n%s", viewReadOnly)
	}
	if strings.Contains(viewReadOnly, "View Comments") {
		t.Errorf("expected no comments shortcut when post has no comments, got:\n%s", viewReadOnly)
	}

	// 2. Broadcast read-only with comments
	mWithComments := Model{
		Mode: ModeChannels,
		SelectedChannel: types.ChannelInfo{
			ID:          "2001",
			IsBroadcast: true,
			IsCreator:   false,
		},
		Conversations: []types.FormattedMessage{
			{ID: 11, Content: "Announcement with comments", HasComments: true, RepliesCount: 7},
		},
		SelectedMessageIndex: 0,
	}
	viewWithComments := prepareInputView(&mWithComments, d)
	if !strings.Contains(viewWithComments, "Broadcast Channel (Read-Only)") {
		t.Errorf("expected 'Broadcast Channel (Read-Only)' banner, got:\n%s", viewWithComments)
	}
	if !strings.Contains(viewWithComments, "View Comments (7)") {
		t.Errorf("expected '[💬 View Comments (7)]' shortcut in banner, got:\n%s", viewWithComments)
	}

	// 3. Muted by admin
	mMuted := Model{
		Mode: ModeGroups,
		SelectedGroup: types.ChannelInfo{
			ID:          "3001",
			IsBroadcast: false,
			IsMuted:     true,
		},
	}
	viewMuted := prepareInputView(&mMuted, d)
	if !strings.Contains(viewMuted, "Muted by admin") {
		t.Errorf("expected 'Muted by admin' banner, got:\n%s", viewMuted)
	}
}

func TestRestrictedInput_NoDummyStringInjection(t *testing.T) {
	ch := types.ChannelInfo{
		ID:           "2001",
		ChannelTitle: "News Channel",
		IsBroadcast:  true,
		IsCreator:    false,
	}
	m := Model{
		Mode:            ModeAll,
		FocusedOn:       SideBar,
		SelectedChannel: ch,
		Channels:        list.New([]list.Item{ch}, list.NewDefaultDelegate(), 30, 20),
	}

	// Switching to channels sidebar mode with 'c'
	newM, _ := changeSideBarMode(&m, "c")
	if newM.Input.Value() == "Not Allowed To Type" {
		t.Errorf("input value must NOT be set to 'Not Allowed To Type', got: %q", newM.Input.Value())
	}
	if newM.Input.Value() != "" {
		t.Errorf("expected clean empty input, got: %q", newM.Input.Value())
	}
}

func TestRestrictedInput_FocusCyclingSkipsInput(t *testing.T) {
	ch := types.ChannelInfo{
		ID:           "2001",
		ChannelTitle: "News Channel",
		IsBroadcast:  true,
		IsCreator:    false,
	}
	m := Model{
		Mode:            ModeChannels,
		FocusedOn:       SideBar,
		SelectedChannel: ch,
		Channels:        list.New([]list.Item{ch}, list.NewDefaultDelegate(), 30, 20),
		Conversations: []types.FormattedMessage{
			{ID: 1, Content: "First post", Date: time.Now()},
		},
	}

	// 1. From SideBar, Tab goes directly to Main (skips Input)
	m1 := m
	m1.FocusedOn = SideBar
	mFromSidebar, _ := changeFocusMode(&m1, "tab", false)
	if mFromSidebar.FocusedOn != Main {
		t.Errorf("expected Tab from SideBar to focus Main, got %v", mFromSidebar.FocusedOn)
	}

	// 2. From Main, Tab goes directly to SideBar (skips Input)
	m2 := m
	m2.FocusedOn = Main
	mFromMain, _ := changeFocusMode(&m2, "tab", false)
	if mFromMain.FocusedOn != SideBar {
		t.Errorf("expected Tab from Main to focus SideBar, got %v", mFromMain.FocusedOn)
	}

	// 3. From SideBar, Shift+Tab goes to Main (skips Input)
	m3 := m
	m3.FocusedOn = SideBar
	mShiftFromSidebar, _ := changeFocusMode(&m3, "tab", true)
	if mShiftFromSidebar.FocusedOn != Main {
		t.Errorf("expected Shift+Tab from SideBar to focus Main, got %v", mShiftFromSidebar.FocusedOn)
	}

	// 4. From Main, Shift+Tab goes to SideBar (skips Input)
	m4 := m
	m4.FocusedOn = Main
	mShiftFromMain, _ := changeFocusMode(&m4, "tab", true)
	if mShiftFromMain.FocusedOn != SideBar {
		t.Errorf("expected Shift+Tab from Main to focus SideBar, got %v", mShiftFromMain.FocusedOn)
	}
}

func TestRestrictedInput_TypingBlocked(t *testing.T) {
	ch := types.ChannelInfo{
		ID:           "2001",
		ChannelTitle: "News Channel",
		IsBroadcast:  true,
		IsCreator:    false,
	}
	m := Model{
		Mode:            ModeChannels,
		FocusedOn:       Main,
		SelectedChannel: ch,
		Conversations: []types.FormattedMessage{
			{ID: 1, Content: "First post", Date: time.Now()},
		},
		SelectedMessageIndex: 0,
	}

	// Pressing 'i' while focused on Main should NOT switch focus to Input in restricted mode
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	mod := newM.(Model)
	if mod.FocusedOn == Input {
		t.Errorf("pressing 'i' in read-only channel must not focus Input")
	}

	// Keystrokes directly on Input if somehow focused must be dropped
	mInput := m
	mInput.FocusedOn = Input
	var cmds []tea.Cmd
	updatedModel, _ := updateFocusedComponent(&mInput, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h', 'e', 'l', 'l', 'o'}}, &cmds)
	if updatedModel.Input.Value() != "" {
		t.Errorf("input must remain empty in restricted mode, got %q", updatedModel.Input.Value())
	}
}

func TestRestrictedInput_ViewCommentsShortcut(t *testing.T) {
	discGroup := types.ChannelInfo{
		ID:           "9999",
		ChannelTitle: "News Discussion",
		IsBroadcast:  false,
	}
	ch := types.ChannelInfo{
		ID:           "2001",
		ChannelTitle: "News Channel",
		IsBroadcast:  true,
		IsCreator:    false,
	}
	post := types.FormattedMessage{
		ID:               101,
		Content:          "Big breaking news",
		HasComments:      true,
		RepliesCount:     3,
		RepliesChannelID: 9999,
	}
	m := Model{
		Mode:                 ModeChannels,
		FocusedOn:            Main,
		SelectedChannel:      ch,
		Groups:               list.New([]list.Item{discGroup}, list.NewDefaultDelegate(), 30, 20),
		Channels:             list.New([]list.Item{ch}, list.NewDefaultDelegate(), 30, 20),
		All:                  list.New([]list.Item{discGroup, ch}, list.NewDefaultDelegate(), 30, 20),
		Conversations:        []types.FormattedMessage{post},
		SelectedMessageIndex: 0,
	}

	// 1. Pressing 'c' while on post with comments opens modal without leaving the channel
	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	mod := newM.(Model)
	if mod.Mode != ModeChannels {
		t.Errorf("expected 'c' to keep channel mode, got mode %v", mod.Mode)
	}
	if !mod.CommentsModalOpen {
		t.Errorf("expected CommentsModalOpen to be true after pressing 'c'")
	}
	if cmd == nil {
		t.Fatalf("expected command from 'c' comments shortcut")
	}

	// Check that one of the commands is OpenModalMsg for ModalModeComments
	var foundCommentsModal bool
	switch msg := cmd().(type) {
	case OpenModalMsg:
		if msg.ModalMode == ModalModeComments {
			foundCommentsModal = true
		}
	case tea.BatchMsg:
		for _, c := range msg {
			if c != nil {
				if mVal, ok := c().(OpenModalMsg); ok && mVal.ModalMode == ModalModeComments {
					foundCommentsModal = true
					break
				}
			}
		}
	}
	if !foundCommentsModal {
		t.Errorf("expected OpenModalMsg with ModalModeComments from 'c' shortcut")
	}

	// 2. Pressing 'Enter' while on post with comments opens modal without leaving the channel
	mEnter := m
	newMEnter, cmdEnter := mEnter.handleEnterKey()
	modEnter := newMEnter.(Model)
	if modEnter.Mode != ModeChannels {
		t.Errorf("expected Enter to keep channel mode, got mode %v", modEnter.Mode)
	}
	if !modEnter.CommentsModalOpen {
		t.Errorf("expected CommentsModalOpen to be true after pressing Enter")
	}
	if cmdEnter == nil {
		t.Fatalf("expected command from Enter comments shortcut")
	}
}

func TestCommentsModal_Rendering_LoadingAndPopulated(t *testing.T) {
	post := types.FormattedMessage{
		ID:          101,
		Content:     "Headline: Major discovery in physics announced today",
		HasComments: true,
	}

	fg := &Foreground{
		windowWidth:  80,
		windowHeight: 30,
	}

	// 1. Open modal for comments
	updatedFgModel, _ := fg.Update(OpenModalMsg{
		ModalMode:    ModalModeComments,
		CommentsPost: &post,
	})
	fg = updatedFgModel.(*Foreground)

	if fg.ModalMode != ModalModeComments {
		t.Fatalf("expected ModalModeComments, got %v", fg.ModalMode)
	}
	if !fg.commentsLoading {
		t.Errorf("expected commentsLoading to be true initially")
	}

	viewLoading := fg.View()
	if !strings.Contains(viewLoading, "💬 Comments • Headline: Major discovery in") {
		t.Errorf("expected header snippet in view, got:\n%s", viewLoading)
	}
	if !strings.Contains(viewLoading, "Fetching comments") && !strings.Contains(viewLoading, "Loading") {
		t.Errorf("expected loading indicator in view, got:\n%s", viewLoading)
	}

	// 2. Deliver comments via DiscussionMessageMsg
	discChat := types.ChannelInfo{
		ID:           "9999",
		ChannelTitle: "Physics Discussion",
		IsBroadcast:  false,
		IsLeft:       false,
		IsMuted:      false,
	}
	comments := []types.FormattedMessage{
		{
			ID:      201,
			Sender:  "Alice",
			Content: "This is truly astonishing!",
			Date:    time.Now().Add(-10 * time.Minute),
		},
		{
			ID:      202,
			Sender:  "Bob",
			Content: "Has this been peer reviewed yet?",
			Date:    time.Now().Add(-5 * time.Minute),
		},
	}

	updatedFgModel2, _ := fg.Update(types.DiscussionMessageMsg{
		Chat:      &discChat,
		MessageID: post.ID,
		TopMsgID:  500,
		Comments:  comments,
	})
	fg = updatedFgModel2.(*Foreground)

	if fg.commentsLoading {
		t.Errorf("commentsLoading should be false after receiving DiscussionMessageMsg")
	}
	if len(fg.comments) != 2 {
		t.Errorf("expected 2 comments, got %d", len(fg.comments))
	}

	viewPopulated := fg.View()
	if !strings.Contains(viewPopulated, "Alice") || !strings.Contains(viewPopulated, "This is truly astonishing!") {
		t.Errorf("expected Alice's comment in view, got:\n%s", viewPopulated)
	}
	if !strings.Contains(viewPopulated, "Bob") || !strings.Contains(viewPopulated, "Has this been peer reviewed yet?") {
		t.Errorf("expected Bob's comment in view, got:\n%s", viewPopulated)
	}
	if !strings.Contains(viewPopulated, "Navigate") {
		t.Errorf("expected navigate comments hint in view, got:\n%s", viewPopulated)
	}
	if !strings.Contains(viewPopulated, "Reply to comment") {
		t.Errorf("expected reply to comment hint in view, got:\n%s", viewPopulated)
	}
	// Focus input and verify send comment hint is displayed
	fg.focusedOn = SEARCH
	fg.commentInput.Focus()
	viewInputFocused := fg.View()
	if !strings.Contains(viewInputFocused, "[Enter] Send comment") {
		t.Errorf("expected send comment hint when input focused, got:\n%s", viewInputFocused)
	}
	// Verify wide and tall input box is visible
	if fg.commentInput.Height() < 3 {
		t.Errorf("expected comment input height >= 3, got %d", fg.commentInput.Height())
	}
}

func TestCommentsModal_InModalJoin(t *testing.T) {
	post := types.FormattedMessage{
		ID:          101,
		Content:     "Important update for members",
		HasComments: true,
	}

	fg := &Foreground{
		windowWidth:  80,
		windowHeight: 30,
	}

	// 1. Open modal
	updatedFgModel, _ := fg.Update(OpenModalMsg{
		ModalMode:    ModalModeComments,
		CommentsPost: &post,
	})
	fg = updatedFgModel.(*Foreground)

	// 2. Deliver DiscussionMessageMsg where user has NOT joined (IsLeft = true)
	discChat := types.ChannelInfo{
		ID:           "9999",
		AccessHash:   "12345678",
		ChannelTitle: "Members Discussion",
		IsBroadcast:  false,
		IsLeft:       true,
		IsMuted:      false,
	}

	updatedFgModel2, _ := fg.Update(types.DiscussionMessageMsg{
		Chat:      &discChat,
		MessageID: post.ID,
		TopMsgID:  500,
		Comments:  nil,
	})
	fg = updatedFgModel2.(*Foreground)

	viewUnjoined := fg.View()
	if !strings.Contains(viewUnjoined, "Join") {
		t.Errorf("expected join prompt in view, got:\n%s", viewUnjoined)
	}
	if !strings.Contains(viewUnjoined, "[Enter]") {
		t.Errorf("expected '[Enter]' hint in view, got:\n%s", viewUnjoined)
	}
	if fg.commentInput.Focused() {
		t.Errorf("comment input must not be focused when user has not joined group")
	}

	// 3. Press Enter to join directly in modal
	updatedFgModel3, joinCmd := fg.Update(tea.KeyMsg{Type: tea.KeyEnter})
	fg = updatedFgModel3.(*Foreground)

	if !fg.isJoining {
		t.Errorf("expected isJoining to be true after pressing Enter")
	}
	if joinCmd == nil {
		t.Errorf("expected join command after pressing Enter")
	}

	viewJoining := fg.View()
	if !strings.Contains(viewJoining, "Joining") {
		t.Errorf("expected joining indicator in view, got:\n%s", viewJoining)
	}

	// 4. Deliver JoinChannelMsg successfully
	updatedFgModel4, _ := fg.Update(types.JoinChannelMsg{
		ChannelID: 9999,
		Err:       nil,
	})
	fg = updatedFgModel4.(*Foreground)

	if fg.isJoining {
		t.Errorf("isJoining should be false after JoinChannelMsg")
	}
	if fg.commentsDiscussionChat.IsLeft {
		t.Errorf("IsLeft should be false after successful join")
	}
	if !fg.commentInput.Focused() {
		t.Errorf("comment input should be focused after joining")
	}

	viewJoined := fg.View()
	if !strings.Contains(viewJoined, "[Enter] Send comment") {
		t.Errorf("expected send comment hint in view after joining, got:\n%s", viewJoined)
	}
}

func TestCommentsModal_MutedByAdmin(t *testing.T) {
	post := types.FormattedMessage{
		ID:          101,
		Content:     "Announcement post",
		HasComments: true,
	}

	fg := &Foreground{
		windowWidth:  80,
		windowHeight: 30,
	}

	updatedFgModel, _ := fg.Update(OpenModalMsg{
		ModalMode:    ModalModeComments,
		CommentsPost: &post,
	})
	fg = updatedFgModel.(*Foreground)

	// User is a member (IsLeft = false) but muted by admin (IsMuted = true)
	discChat := types.ChannelInfo{
		ID:           "9999",
		ChannelTitle: "Discussion",
		IsBroadcast:  false,
		IsLeft:       false,
		IsMuted:      true,
	}

	updatedFgModel2, _ := fg.Update(types.DiscussionMessageMsg{
		Chat:      &discChat,
		MessageID: post.ID,
		TopMsgID:  500,
		Comments:  nil,
	})
	fg = updatedFgModel2.(*Foreground)

	if fg.commentInput.Focused() {
		t.Errorf("comment input must not be focused when user is muted")
	}

	viewMuted := fg.View()
	if !strings.Contains(viewMuted, "Muted by admin") {
		t.Errorf("expected 'Muted by admin' banner in view, got:\n%s", viewMuted)
	}
	if !strings.Contains(viewMuted, "You cannot write comments in this group") {
		t.Errorf("expected explanation in view, got:\n%s", viewMuted)
	}
}

func TestCommentsModal_PostCommentAndNavigation(t *testing.T) {
	post := types.FormattedMessage{
		ID:          101,
		Content:     "New feature released!",
		HasComments: true,
	}

	fg := &Foreground{
		windowWidth:  80,
		windowHeight: 30,
	}

	updatedFgModel, _ := fg.Update(OpenModalMsg{
		ModalMode:    ModalModeComments,
		CommentsPost: &post,
	})
	fg = updatedFgModel.(*Foreground)

	discChat := types.ChannelInfo{
		ID:           "9999",
		AccessHash:   "987654321",
		ChannelTitle: "Feedback Group",
		IsBroadcast:  false,
		IsLeft:       false,
		IsMuted:      false,
	}

	updatedFgModel2, _ := fg.Update(types.DiscussionMessageMsg{
		Chat:      &discChat,
		MessageID: post.ID,
		TopMsgID:  500,
		Comments:  nil,
	})
	fg = updatedFgModel2.(*Foreground)

	// 1. Type comment
	fg.commentInput.SetValue("Awesome work on this!")

	// 2. Press Enter to send
	updatedFgModel3, sendCmd := fg.Update(tea.KeyMsg{Type: tea.KeyEnter})
	fg = updatedFgModel3.(*Foreground)

	if !fg.isPostingComment {
		t.Errorf("expected isPostingComment to be true after sending")
	}
	if sendCmd == nil {
		t.Fatalf("expected send command after pressing Enter")
	}
	if len(fg.comments) != 1 || fg.comments[0].Content != "Awesome work on this!" {
		t.Errorf("expected optimistic comment added to comments list")
	}
	if fg.commentInput.Value() != "" {
		t.Errorf("expected comment input to be cleared after sending, got %q", fg.commentInput.Value())
	}

	// 3. Deliver SendMessageMsg
	updatedFgModel4, _ := fg.Update(types.SendMessageMsg{
		Err: nil,
	})
	fg = updatedFgModel4.(*Foreground)
	if fg.isPostingComment {
		t.Errorf("isPostingComment should be false after SendMessageMsg")
	}

	// 4. Test Tab toggles focus between comment input and comment list
	if fg.focusedOn != SEARCH {
		t.Errorf("expected focusedOn to be SEARCH initially")
	}
	updatedFgModel5, _ := fg.Update(tea.KeyMsg{Type: tea.KeyTab})
	fg = updatedFgModel5.(*Foreground)
	if fg.focusedOn != LIST {
		t.Errorf("expected focusedOn to be LIST after Tab, got %v", fg.focusedOn)
	}

	// When focusedOn == LIST, test up/down/j/k adjusts comments cursor
	fg.comments = append(fg.comments, types.FormattedMessage{ID: 2, Content: "Another comment"})
	updatedFgModel6, _ := fg.Update(tea.KeyMsg{Type: tea.KeyDown})
	fg = updatedFgModel6.(*Foreground)
	if fg.commentsCursor != 1 {
		t.Errorf("expected commentsCursor to be 1 after Down, got %d", fg.commentsCursor)
	}

	// Test Up moves cursor back to 0
	updatedFgModelUp, _ := fg.Update(tea.KeyMsg{Type: tea.KeyUp})
	fg = updatedFgModelUp.(*Foreground)
	if fg.commentsCursor != 0 {
		t.Errorf("expected commentsCursor to be 0 after Up, got %d", fg.commentsCursor)
	}

	// Test 'j' moves cursor to 1
	updatedFgModelJ, _ := fg.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	fg = updatedFgModelJ.(*Foreground)
	if fg.commentsCursor != 1 {
		t.Errorf("expected commentsCursor to be 1 after 'j', got %d", fg.commentsCursor)
	}

	// Test 'k' moves cursor back to 0
	updatedFgModelK, _ := fg.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	fg = updatedFgModelK.(*Foreground)
	if fg.commentsCursor != 0 {
		t.Errorf("expected commentsCursor to be 0 after 'k', got %d", fg.commentsCursor)
	}

	// Test 'i' switches focus to SEARCH
	updatedFgModelI, _ := fg.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	fg = updatedFgModelI.(*Foreground)
	if fg.focusedOn != SEARCH {
		t.Errorf("expected focusedOn to be SEARCH after 'i', got %v", fg.focusedOn)
	}

	// Switch back to LIST with Tab and test Enter switches to SEARCH
	updatedFgModelTabBack, _ := fg.Update(tea.KeyMsg{Type: tea.KeyTab})
	fg = updatedFgModelTabBack.(*Foreground)
	if fg.focusedOn != LIST {
		t.Fatalf("expected focusedOn to be LIST after Tab, got %v", fg.focusedOn)
	}
	updatedFgModelEnter, _ := fg.Update(tea.KeyMsg{Type: tea.KeyEnter})
	fg = updatedFgModelEnter.(*Foreground)
	if fg.focusedOn != SEARCH {
		t.Errorf("expected focusedOn to be SEARCH after Enter on LIST, got %v", fg.focusedOn)
	}

	// 5. Test Esc: first Esc cancels reply target armed by Enter on LIST
	updatedFgModelEsc1, cmdEsc1 := fg.Update(tea.KeyMsg{Type: tea.KeyEsc})
	fg = updatedFgModelEsc1.(*Foreground)
	if cmdEsc1 != nil {
		t.Errorf("first Esc while replying should cancel reply, not close overlay")
	}

	// Second Esc closes overlay
	updatedFgModel7, closeCmd := fg.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if closeCmd == nil {
		t.Fatalf("expected CloseOverlay cmd on Esc")
	}
	if _, ok := closeCmd().(CloseOverlay); !ok {
		t.Errorf("expected CloseOverlay msg on Esc, got %T", closeCmd())
	}
	_ = updatedFgModel7
}

func TestCommentsModal_ReplyingToComment(t *testing.T) {
	post := types.FormattedMessage{
		ID:          101,
		Content:     "Original post announcing new release",
		HasComments: true,
	}

	fg := &Foreground{
		windowWidth:  90,
		windowHeight: 30,
	}

	updatedFgModel, _ := fg.Update(OpenModalMsg{
		ModalMode:    ModalModeComments,
		CommentsPost: &post,
	})
	fg = updatedFgModel.(*Foreground)

	discChat := types.ChannelInfo{
		ID:           "9999",
		AccessHash:   "987654321",
		ChannelTitle: "Feedback Group",
		IsBroadcast:  false,
		IsLeft:       false,
		IsMuted:      false,
	}

	commentA := types.FormattedMessage{
		ID:      501,
		Sender:  "Alice",
		Content: "This looks fantastic!",
		Date:    time.Now().Add(-10 * time.Minute),
	}
	commentB := types.FormattedMessage{
		ID:      502,
		Sender:  "Bob",
		Content: "Agreed, great work team!",
		Date:    time.Now().Add(-5 * time.Minute),
		ReplyTo: &commentA,
	}

	updatedFgModel2, _ := fg.Update(types.DiscussionMessageMsg{
		Chat:      &discChat,
		MessageID: post.ID,
		TopMsgID:  500,
		Comments:  []types.FormattedMessage{commentA, commentB},
	})
	fg = updatedFgModel2.(*Foreground)

	// 1. Verify that commentB renders the reply quote to Alice in the view
	viewComments := fg.View()
	if !strings.Contains(viewComments, "Replying to Alice") {
		t.Errorf("expected comment card to render reply quote to Alice, got:\n%s", viewComments)
	}

	// 2. Focused on LIST; cursor is at 0 (commentA). Press 'r' to reply to Alice.
	if fg.focusedOn != LIST {
		t.Fatalf("expected focusedOn to be LIST initially")
	}
	updatedFgModel3, _ := fg.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	fg = updatedFgModel3.(*Foreground)

	if fg.focusedOn != SEARCH {
		t.Errorf("expected focusedOn to be SEARCH after 'r', got %v", fg.focusedOn)
	}
	if fg.commentsReplyTo == nil || fg.commentsReplyTo.ID != commentA.ID {
		t.Fatalf("expected commentsReplyTo to target Alice (ID %d), got %+v", commentA.ID, fg.commentsReplyTo)
	}

	// 3. View should show reply preview bar above comment input
	viewReplying := fg.View()
	if !strings.Contains(viewReplying, "Replying to Alice") || !strings.Contains(viewReplying, "Esc to cancel") {
		t.Errorf("expected reply preview bar in view, got:\n%s", viewReplying)
	}

	// 4. Test Esc in SEARCH cancels the reply preview without closing modal
	updatedFgModelEsc, cmdEsc := fg.Update(tea.KeyMsg{Type: tea.KeyEsc})
	fg = updatedFgModelEsc.(*Foreground)
	if fg.commentsReplyTo != nil {
		t.Errorf("expected commentsReplyTo to be nil after Esc, got %+v", fg.commentsReplyTo)
	}
	if cmdEsc != nil {
		t.Errorf("expected Esc while replying to cancel reply without closing modal, got cmd")
	}

	// 5. Navigate to Bob (index 1) with 'down' and press Enter to reply to Bob
	fg.focusedOn = LIST
	fg.commentInput.Blur()
	updatedFgModelDown, _ := fg.Update(tea.KeyMsg{Type: tea.KeyDown})
	fg = updatedFgModelDown.(*Foreground)
	if fg.commentsCursor != 1 {
		t.Fatalf("expected commentsCursor to be 1, got %d", fg.commentsCursor)
	}

	updatedFgModelEnter, _ := fg.Update(tea.KeyMsg{Type: tea.KeyEnter})
	fg = updatedFgModelEnter.(*Foreground)
	if fg.focusedOn != SEARCH {
		t.Errorf("expected focusedOn to be SEARCH after Enter on comment card, got %v", fg.focusedOn)
	}
	if fg.commentsReplyTo == nil || fg.commentsReplyTo.ID != commentB.ID {
		t.Fatalf("expected commentsReplyTo to target Bob (ID %d), got %+v", commentB.ID, fg.commentsReplyTo)
	}

	// 6. Type reply text and send with Enter
	fg.commentInput.SetValue("I have a question about this too!")
	updatedFgModelSend, sendCmd := fg.Update(tea.KeyMsg{Type: tea.KeyEnter})
	fg = updatedFgModelSend.(*Foreground)

	if sendCmd == nil {
		t.Fatalf("expected send command after pressing Enter")
	}
	if len(fg.comments) != 3 {
		t.Fatalf("expected 3 comments after sending reply, got %d", len(fg.comments))
	}
	newComment := fg.comments[2]
	if newComment.ReplyTo == nil || newComment.ReplyTo.ID != commentB.ID {
		t.Errorf("expected new comment ReplyTo to target Bob (ID %d), got %+v", commentB.ID, newComment.ReplyTo)
	}
	if fg.commentsReplyTo != nil {
		t.Errorf("expected commentsReplyTo to be cleared after sending")
	}

	// 7. Verify newly posted comment renders reply quote to Bob
	viewAfterSend := fg.View()
	if !strings.Contains(viewAfterSend, "Replying to Bob") {
		t.Errorf("expected view after send to display reply quote to Bob, got:\n%s", viewAfterSend)
	}
}
