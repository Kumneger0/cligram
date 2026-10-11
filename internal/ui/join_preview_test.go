package ui

import (
	"errors"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kumneger0/cligram/internal/telegram/types"
)

func TestJoinPreviewModal_ViewRendering(t *testing.T) {
	// 1. Group preview
	f := &Foreground{
		windowWidth:  80,
		windowHeight: 24,
		ModalMode:    ModalModeJoinPreview,
		JoinPreview: &types.ChatInvitePreview{
			Hash:              "abc123XYZ",
			Title:             "Golang Enthusiasts",
			About:             "A community for Gophers discussing Go language, tooling, and tips.",
			ParticipantsCount: 14500,
			IsGroup:           true,
			RequestNeeded:     false,
		},
	}

	view := f.View()

	if !strings.Contains(view, "Golang Enthusiasts") {
		t.Errorf("expected view to contain chat title 'Golang Enthusiasts', got:\n%s", view)
	}
	if !strings.Contains(view, "14,500 members") {
		t.Errorf("expected formatted member count '14,500 members', got:\n%s", view)
	}
	if !strings.Contains(view, "A community for Gophers") {
		t.Errorf("expected view to contain about text, got:\n%s", view)
	}
	if !strings.Contains(view, "[Enter] Join") || !strings.Contains(view, "[Esc] Cancel") {
		t.Errorf("expected footer action buttons in view, got:\n%s", view)
	}

	// 2. Channel preview with RequestNeeded
	fChannel := &Foreground{
		windowWidth:  80,
		windowHeight: 24,
		ModalMode:    ModalModeJoinPreview,
		JoinPreview: &types.ChatInvitePreview{
			Hash:              "newsChannelHash",
			Title:             "Tech Announcements",
			About:             "Official announcements channel.",
			ParticipantsCount: 520,
			IsChannel:         true,
			RequestNeeded:     true,
		},
	}

	chanView := fChannel.View()
	if !strings.Contains(chanView, "520 subscribers") {
		t.Errorf("expected '520 subscribers' for channel, got:\n%s", chanView)
	}
	if !strings.Contains(chanView, "Admin approval is required to join this chat") {
		t.Errorf("expected admin approval notice when RequestNeeded=true, got:\n%s", chanView)
	}

	// 3. Joining in progress view
	fJoining := &Foreground{
		windowWidth:  80,
		windowHeight: 24,
		ModalMode:    ModalModeJoinPreview,
		isJoining:    true,
		JoinPreview: &types.ChatInvitePreview{
			Hash:              "joiningHash",
			Title:             "Joining Chat",
			ParticipantsCount: 10,
		},
	}
	joiningView := fJoining.View()
	if !strings.Contains(joiningView, "Joining...") {
		t.Errorf("expected 'Joining...' spinner text when isJoining=true, got:\n%s", joiningView)
	}

	// 4. Join error view
	fError := &Foreground{
		windowWidth:  80,
		windowHeight: 24,
		ModalMode:    ModalModeJoinPreview,
		joinError:    errors.New("user has reached channel limit"),
		JoinPreview: &types.ChatInvitePreview{
			Hash:              "errHash",
			Title:             "Limited Chat",
			ParticipantsCount: 10,
		},
	}
	errView := fError.View()
	if !strings.Contains(errView, "user has reached channel limit") {
		t.Errorf("expected error message when joinError is set, got:\n%s", errView)
	}
}

func TestJoinPreviewModal_KeyboardControls(t *testing.T) {
	newPreviewForeground := func() *Foreground {
		return &Foreground{
			windowWidth:  80,
			windowHeight: 24,
			ModalMode:    ModalModeJoinPreview,
			JoinPreview: &types.ChatInvitePreview{
				Hash:              "testHash123",
				Title:             "Test Chat",
				ParticipantsCount: 10,
			},
		}
	}

	// 1. Esc key should close overlay
	fEsc := newPreviewForeground()
	updatedModel, cmd := fEsc.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatalf("expected non-nil cmd for Esc key")
	}
	msg := cmd()
	if _, ok := msg.(CloseOverlay); !ok {
		t.Errorf("expected CloseOverlay message on Esc, got %T", msg)
	}
	fg := updatedModel.(*Foreground)
	if fg.JoinPreview != nil {
		t.Errorf("expected JoinPreview to be cleared on Esc")
	}

	// 2. 'q' key should close overlay
	fQ := newPreviewForeground()
	_, cmdQ := fQ.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmdQ == nil {
		t.Fatalf("expected non-nil cmd for 'q' key")
	}
	msgQ := cmdQ()
	if _, ok := msgQ.(CloseOverlay); !ok {
		t.Errorf("expected CloseOverlay message on 'q', got %T", msgQ)
	}

	// 3. Enter key initiates join and sets isJoining = true
	fEnter := newPreviewForeground()
	updatedModel, cmdEnter := fEnter.Update(tea.KeyMsg{Type: tea.KeyEnter})
	fgEnter := updatedModel.(*Foreground)
	if !fgEnter.isJoining {
		t.Errorf("expected isJoining to be true after Enter")
	}
	if cmdEnter == nil {
		t.Errorf("expected join cmd batch after Enter")
	}
}

func TestModelUpdate_CheckChatInviteMsg_AlreadyJoined(t *testing.T) {
	ch := types.ChannelInfo{
		ID:           "55555",
		ChannelTitle: "Existing Group",
		IsBroadcast:  false,
	}
	m := Model{
		Mode:     ModeAll,
		Channels: list.New([]list.Item{}, list.NewDefaultDelegate(), 30, 20),
		Groups:   list.New([]list.Item{ch}, list.NewDefaultDelegate(), 30, 20),
		All:      list.New([]list.Item{ch}, list.NewDefaultDelegate(), 30, 20),
		ChatUI:   list.New([]list.Item{}, list.NewDefaultDelegate(), 50, 20),
	}

	msg := types.CheckChatInviteMsg{
		Hash:          "existingHash",
		AlreadyJoined: true,
		Chat:          &ch,
	}

	updatedModel, cmd := m.Update(msg)
	mod := updatedModel.(Model)

	if mod.SelectedGroup.ID != "55555" {
		t.Errorf("expected SelectedGroup to be set to 55555, got %q", mod.SelectedGroup.ID)
	}
	if mod.Mode != ModeGroups {
		t.Errorf("expected mode to switch to ModeGroups, got %v", mod.Mode)
	}
	if cmd == nil {
		t.Errorf("expected navigation command on already joined chat")
	}
}

func TestModelUpdate_CheckChatInviteMsg_UnjoinedPreview(t *testing.T) {
	m := Model{}
	preview := &types.ChatInvitePreview{
		Hash:              "newInviteHash",
		Title:             "New Awesome Group",
		ParticipantsCount: 50,
	}

	msg := types.CheckChatInviteMsg{
		Hash:          "newInviteHash",
		AlreadyJoined: false,
		Preview:       preview,
	}

	_, cmd := m.Update(msg)
	if cmd == nil {
		t.Fatalf("expected OpenModal command for unjoined chat invite")
	}

	openMsg := cmd()
	modalMsg, ok := openMsg.(OpenModalMsg)
	if !ok {
		t.Fatalf("expected OpenModalMsg, got %T", openMsg)
	}
	if modalMsg.ModalMode != ModalModeJoinPreview {
		t.Errorf("expected ModalModeJoinPreview, got %v", modalMsg.ModalMode)
	}
	if modalMsg.JoinPreview != preview {
		t.Errorf("expected JoinPreview to match")
	}
}

func TestModelUpdate_CheckChatInviteMsg_Error(t *testing.T) {
	m := Model{}
	msg := types.CheckChatInviteMsg{
		Hash: "brokenHash",
		Err:  errors.New("INVITE_HASH_EXPIRED"),
	}

	_, cmd := m.Update(msg)
	if cmd == nil {
		t.Fatalf("expected alert command on invite check error")
	}
}

func TestModelUpdate_ImportChatInviteMsg_SuccessAndError(t *testing.T) {
	ch := types.ChannelInfo{
		ID:           "99999",
		ChannelTitle: "Newly Joined Channel",
		IsBroadcast:  true,
	}
	m := Model{
		Mode:     ModeAll,
		Channels: list.New([]list.Item{}, list.NewDefaultDelegate(), 30, 20),
		Groups:   list.New([]list.Item{ch}, list.NewDefaultDelegate(), 30, 20),
		All:      list.New([]list.Item{ch}, list.NewDefaultDelegate(), 30, 20),
		ChatUI:   list.New([]list.Item{}, list.NewDefaultDelegate(), 50, 20),
	}

	// 1. Success
	successMsg := types.ImportChatInviteMsg{
		Hash: "validHash",
		Chat: &ch,
	}

	updatedModel, cmd := m.Update(successMsg)
	mod := updatedModel.(Model)

	if mod.SelectedChannel.ID != "99999" {
		t.Errorf("expected SelectedChannel to be set to 99999, got %q", mod.SelectedChannel.ID)
	}
	if mod.Mode != ModeChannels {
		t.Errorf("expected ModeChannels, got %v", mod.Mode)
	}
	if cmd == nil {
		t.Errorf("expected navigation command on successful join")
	}

	// 2. Error
	errMsg := types.ImportChatInviteMsg{
		Hash: "validHash",
		Err:  errors.New("CHANNELS_TOO_MUCH"),
	}

	_, cmdErr := m.Update(errMsg)
	if cmdErr == nil {
		t.Fatalf("expected command on import chat error")
	}
}

func TestModelUpdate_JoinChannelMsg_SuccessAndError(t *testing.T) {
	ch := types.ChannelInfo{
		ID:           "88888",
		ChannelTitle: "Public Supergroup",
		IsBroadcast:  false,
	}
	m := Model{
		Mode:     ModeAll,
		Channels: list.New([]list.Item{}, list.NewDefaultDelegate(), 30, 20),
		Groups:   list.New([]list.Item{ch}, list.NewDefaultDelegate(), 30, 20),
		All:      list.New([]list.Item{ch}, list.NewDefaultDelegate(), 30, 20),
		ChatUI:   list.New([]list.Item{}, list.NewDefaultDelegate(), 50, 20),
	}

	// 1. Success
	successMsg := types.JoinChannelMsg{
		ChannelID: 88888,
		Chat:      &ch,
	}

	updatedModel, cmd := m.Update(successMsg)
	mod := updatedModel.(Model)

	if mod.SelectedGroup.ID != "88888" {
		t.Errorf("expected SelectedGroup to be set to 88888, got %q", mod.SelectedGroup.ID)
	}
	if mod.Mode != ModeGroups {
		t.Errorf("expected ModeGroups, got %v", mod.Mode)
	}
	if cmd == nil {
		t.Errorf("expected navigation command on successful join")
	}

	// 2. Error
	errMsg := types.JoinChannelMsg{
		ChannelID: 88888,
		Err:       errors.New("USER_BANNED_IN_CHANNEL"),
	}

	_, cmdErr := m.Update(errMsg)
	if cmdErr == nil {
		t.Fatalf("expected command on join channel error")
	}
}

func TestModelUpdate_RouteParsedLinkMsg(t *testing.T) {
	ch := types.ChannelInfo{
		ID:           "12345",
		ChannelTitle: "Target Channel",
		IsBroadcast:  true,
	}
	m := Model{
		Mode:     ModeAll,
		Channels: list.New([]list.Item{ch}, list.NewDefaultDelegate(), 30, 20),
		Groups:   list.New([]list.Item{}, list.NewDefaultDelegate(), 30, 20),
		All:      list.New([]list.Item{ch}, list.NewDefaultDelegate(), 30, 20),
		ChatUI:   list.New([]list.Item{}, list.NewDefaultDelegate(), 50, 20),
	}

	msg := RouteParsedLinkMsg{
		Link: &types.ParsedLink{
			Type:      types.LinkTypeInternalMessage,
			ChannelID: 12345,
			MessageID: 10,
		},
	}

	updatedModel, cmd := m.Update(msg)
	mod := updatedModel.(Model)
	if mod.SelectedChannel.ID != "12345" {
		t.Errorf("expected SelectedChannel to be 12345, got %q", mod.SelectedChannel.ID)
	}
	if cmd == nil {
		t.Errorf("expected navigation command")
	}
}

func TestModelUpdate_HandleEnterKey_InputLinks(t *testing.T) {
	ti := textinput.New()
	m := Model{
		FocusedOn: Input,
		Input:     ti,
	}

	// 1. /open command with public channel link
	m.Input.SetValue("/open https://t.me/durov")
	_, cmd := m.handleEnterKey()
	if cmd == nil {
		t.Fatalf("expected command for /open input")
	}
	msg := cmd()
	openMsg, ok := msg.(types.OpenNewChatWithPeerMsg)
	if !ok {
		t.Fatalf("expected OpenNewChatWithPeerMsg, got %T", msg)
	}
	if openMsg.Chat == nil || openMsg.Chat.Entity != "durov" {
		t.Errorf("unexpected routed link: %+v", openMsg.Chat)
	}

	// 2. Direct public username typed into input
	m.Input.SetValue("@durov")
	_, cmd = m.handleEnterKey()
	if cmd == nil {
		t.Fatalf("expected command for @durov in input")
	}
	msg = cmd()
	openMsg, ok = msg.(types.OpenNewChatWithPeerMsg)
	if !ok {
		t.Fatalf("expected OpenNewChatWithPeerMsg, got %T", msg)
	}
	if openMsg.Chat == nil || openMsg.Chat.Entity != "durov" {
		t.Errorf("unexpected routed link: %+v", openMsg.Chat)
	}
}

func TestModelUpdate_HandleEnterKey_ChatWindowTelegramLink(t *testing.T) {
	msgWithContent := types.FormattedMessage{
		ID:      1,
		Content: "Check out this chat: https://t.me/durov",
	}

	m := Model{
		FocusedOn:            Main,
		Conversations:        []types.FormattedMessage{msgWithContent},
		SelectedMessageIndex: 0,
	}

	_, cmd := m.handleEnterKey()
	if cmd == nil {
		t.Fatalf("expected command when Enter pressed on message with Telegram link")
	}
	msg := cmd()
	openMsg, ok := msg.(types.OpenNewChatWithPeerMsg)
	if !ok {
		t.Fatalf("expected OpenNewChatWithPeerMsg, got %T", msg)
	}
	if openMsg.Chat == nil || openMsg.Chat.Entity != "durov" {
		t.Errorf("unexpected routed link: %+v", openMsg.Chat)
	}
}

func TestUnjoinedChatPrompt_ViewRendering(t *testing.T) {
	d := layoutDimensions{mainWidth: 80, inputHeight: 3}

	// 1. Unjoined broadcast channel
	mChan := Model{
		Mode: ModeChannels,
		SelectedChannel: types.ChannelInfo{
			ID:           "1111",
			ChannelTitle: "software_guy",
			IsBroadcast:  true,
			IsLeft:       true,
		},
	}
	viewChan := prepareInputView(&mChan, d)
	if !strings.Contains(viewChan, "[Enter]") || !strings.Contains(viewChan, "Join") {
		t.Errorf("expected [Enter] Join prompt, got:\n%s", viewChan)
	}
	if strings.Contains(viewChan, "Channel Preview") {
		t.Errorf("expected prompt NOT to contain 'Channel Preview', got:\n%s", viewChan)
	}
	if !strings.Contains(viewChan, "(Tab to focus)") {
		t.Errorf("expected unfocused prompt to contain '(Tab to focus)', got:\n%s", viewChan)
	}

	// Verify focused prompt does not contain '(Tab to focus)'
	mChanFocused := mChan
	mChanFocused.FocusedOn = Input
	viewChanFocused := prepareInputView(&mChanFocused, d)
	if strings.Contains(viewChanFocused, "(Tab to focus)") {
		t.Errorf("expected focused prompt NOT to contain '(Tab to focus)', got:\n%s", viewChanFocused)
	}

	// 2. Unjoined group
	mGroup := Model{
		Mode: ModeGroups,
		SelectedGroup: types.ChannelInfo{
			ID:           "2222",
			ChannelTitle: "freelance_ethio",
			IsBroadcast:  false,
			IsLeft:       true,
		},
	}
	viewGroup := prepareInputView(&mGroup, d)
	if !strings.Contains(viewGroup, "[Enter]") || !strings.Contains(viewGroup, "Join") {
		t.Errorf("expected [Enter] Join prompt, got:\n%s", viewGroup)
	}
	if strings.Contains(viewGroup, "Group Preview") {
		t.Errorf("expected prompt NOT to contain 'Group Preview', got:\n%s", viewGroup)
	}

	// 3. Admin approval required (JoinRequest: true)
	mReq := Model{
		Mode: ModeChannels,
		SelectedChannel: types.ChannelInfo{
			ID:           "3333",
			ChannelTitle: "Private Club",
			IsBroadcast:  true,
			IsLeft:       true,
			JoinRequest:  true,
		},
	}
	viewReq := prepareInputView(&mReq, d)
	if !strings.Contains(viewReq, "Admin approval is required") || !strings.Contains(viewReq, "Request to Join") {
		t.Errorf("expected admin approval join prompt, got:\n%s", viewReq)
	}

	// 4. Joining in progress
	mJoining := mChan
	mJoining.IsJoiningChannel = true
	viewJoining := prepareInputView(&mJoining, d)
	if !strings.Contains(viewJoining, "Joining...") {
		t.Errorf("expected joining indicator in prompt, got:\n%s", viewJoining)
	}

	// 5. Join request sent
	mSent := mReq
	mSent.JoinRequestSent = true
	viewSent := prepareInputView(&mSent, d)
	if !strings.Contains(viewSent, "Join request sent to administrators") {
		t.Errorf("expected join request sent notice in prompt, got:\n%s", viewSent)
	}
}

func TestUnjoinedChatPrompt_EnterKeyJoins(t *testing.T) {
	ch := types.ChannelInfo{
		ID:           "123456",
		ChannelTitle: "software_guy",
		AccessHash:   "987654",
		IsBroadcast:  true,
		IsLeft:       true,
	}

	// 1. Enter on Input when unjoined
	m := Model{
		Mode:            ModeChannels,
		FocusedOn:       Input,
		SelectedChannel: ch,
	}
	newM, _ := m.handleEnterKey()
	mod := newM.(Model)
	if !mod.IsJoiningChannel {
		t.Errorf("expected IsJoiningChannel to be true after pressing Enter on Input")
	}

	// 2. Enter on Main when unjoined
	mMain := Model{
		Mode:            ModeChannels,
		FocusedOn:       Main,
		SelectedChannel: ch,
		Conversations:   []types.FormattedMessage{{ID: 1, Content: "Hello world"}},
	}
	newMMain, _ := mMain.handleEnterKey()
	modMain := newMMain.(Model)
	if !modMain.IsJoiningChannel {
		t.Errorf("expected IsJoiningChannel to be true after pressing Enter on Main")
	}
}

func TestUnjoinedChatPrompt_JoinChannelMsg_SuccessAndApproval(t *testing.T) {
	ch := types.ChannelInfo{
		ID:           "123456",
		ChannelTitle: "software_guy",
		IsBroadcast:  true,
		IsLeft:       true,
	}
	m := Model{
		Mode:             ModeChannels,
		SelectedChannel:  ch,
		Channels:         list.New([]list.Item{ch}, list.NewDefaultDelegate(), 30, 20),
		All:              list.New([]list.Item{ch}, list.NewDefaultDelegate(), 30, 20),
		IsJoiningChannel: true,
	}

	// 1. Normal join success
	succMsg := types.JoinChannelMsg{
		ChannelID: 123456,
		Chat:      &types.ChannelInfo{ID: "123456", ChannelTitle: "software_guy"},
	}
	newM, _ := m.Update(succMsg)
	mod := newM.(Model)
	if mod.IsJoiningChannel {
		t.Errorf("expected IsJoiningChannel to be false")
	}
	if mod.SelectedChannel.IsLeft {
		t.Errorf("expected SelectedChannel.IsLeft to be false after joining")
	}
	if mod.isUnjoinedChat() {
		t.Errorf("expected isUnjoinedChat to return false after joining")
	}

	// 2. Admin approval response (INVITE_REQUEST_SENT)
	mReq := Model{
		Mode:             ModeChannels,
		SelectedChannel:  ch,
		Channels:         list.New([]list.Item{ch}, list.NewDefaultDelegate(), 30, 20),
		All:              list.New([]list.Item{ch}, list.NewDefaultDelegate(), 30, 20),
		IsJoiningChannel: true,
	}
	errMsg := types.JoinChannelMsg{
		ChannelID: 123456,
		Err:       errors.New("INVITE_REQUEST_SENT"),
	}
	newMReq, _ := mReq.Update(errMsg)
	modReq := newMReq.(Model)
	if modReq.IsJoiningChannel {
		t.Errorf("expected IsJoiningChannel to be false")
	}
	if !modReq.JoinRequestSent {
		t.Errorf("expected JoinRequestSent to be true")
	}
}
