package ui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kumneger0/cligram/internal/telegram"
	"github.com/kumneger0/cligram/internal/telegram/client"
	"github.com/kumneger0/cligram/internal/telegram/types"
	overlay "github.com/rmhubbert/bubbletea-overlay"
)

func TestAccountSwitcher_OpenModalAndInit(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	_ = telegram.SaveAccountMetadata("account1", types.AccountsOnDeviceInfo{
		AccountName: "Alice",
		PhoneNumber: "+1111111111",
		Path:        "account1",
	})
	_ = telegram.SaveAccountMetadata("account2", types.AccountsOnDeviceInfo{
		AccountName: "Bob",
		PhoneNumber: "+2222222222",
		Path:        "account2",
	})

	telegram.SetActiveAccount("account1")

	fg := &Foreground{}
	model, _ := fg.Update(OpenModalMsg{
		ModalMode: ModalModeAccountSwitcher,
	})

	updatedFg, ok := model.(*Foreground)
	if !ok {
		t.Fatalf("expected *Foreground, got %T", model)
	}

	if updatedFg.ModalMode != ModalModeAccountSwitcher {
		t.Fatalf("expected ModalMode %s, got %s", ModalModeAccountSwitcher, updatedFg.ModalMode)
	}
	if updatedFg.AccountSwitcherIndex != 0 {
		t.Fatalf("expected AccountSwitcherIndex 0, got %d", updatedFg.AccountSwitcherIndex)
	}
	if updatedFg.ActiveAccount != "account1" {
		t.Fatalf("expected ActiveAccount 'account1', got %s", updatedFg.ActiveAccount)
	}
	if len(updatedFg.Accounts) != 2 {
		t.Fatalf("expected 2 accounts, got %d", len(updatedFg.Accounts))
	}
}

func TestAccountSwitcher_Navigation(t *testing.T) {
	fg := &Foreground{
		ModalMode: ModalModeAccountSwitcher,
		Accounts: []types.AccountsOnDeviceInfo{
			{AccountName: "Alice", Path: "account1"},
			{AccountName: "Bob", Path: "account2"},
		},
		ActiveAccount: "account1",
	}

	// 2 accounts -> indices: 0 (Alice), 1 (Bob), 2 ([+ Add Account])
	if fg.AccountSwitcherIndex != 0 {
		t.Fatalf("initial index should be 0, got %d", fg.AccountSwitcherIndex)
	}

	// Down / j / ctrl+n
	m, _ := fg.Update(tea.KeyMsg{Type: tea.KeyDown})
	fg = m.(*Foreground)
	if fg.AccountSwitcherIndex != 1 {
		t.Errorf("expected index 1 after down, got %d", fg.AccountSwitcherIndex)
	}

	m, _ = fg.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	fg = m.(*Foreground)
	if fg.AccountSwitcherIndex != 2 {
		t.Errorf("expected index 2 after j, got %d", fg.AccountSwitcherIndex)
	}

	// Wrap around to 0
	m, _ = fg.Update(tea.KeyMsg{Type: tea.KeyDown})
	fg = m.(*Foreground)
	if fg.AccountSwitcherIndex != 0 {
		t.Errorf("expected wrap around to index 0, got %d", fg.AccountSwitcherIndex)
	}

	// Up / k / ctrl+p wraps back to 2
	m, _ = fg.Update(tea.KeyMsg{Type: tea.KeyUp})
	fg = m.(*Foreground)
	if fg.AccountSwitcherIndex != 2 {
		t.Errorf("expected wrap around to index 2 on up, got %d", fg.AccountSwitcherIndex)
	}

	m, _ = fg.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	fg = m.(*Foreground)
	if fg.AccountSwitcherIndex != 1 {
		t.Errorf("expected index 1 on k, got %d", fg.AccountSwitcherIndex)
	}
}

func TestAccountSwitcher_RenderView(t *testing.T) {
	fg := &Foreground{
		ModalMode: ModalModeAccountSwitcher,
		Accounts: []types.AccountsOnDeviceInfo{
			{AccountName: "Alice Smith", PhoneNumber: "+14155550001", Path: "account1"},
			{AccountName: "Bob Jones", PhoneNumber: "+14155550002", Path: "account2"},
		},
		ActiveAccount:        "account1",
		AccountSwitcherIndex: 0,
		windowWidth:          80,
	}

	view := fg.View()

	if !strings.Contains(view, "Telegram Accounts") {
		t.Errorf("expected header 'Telegram Accounts' in view:\n%s", view)
	}
	if !strings.Contains(view, "Alice Smith") {
		t.Errorf("expected 'Alice Smith' in view:\n%s", view)
	}
	if !strings.Contains(view, "+14155550001") {
		t.Errorf("expected phone '+14155550001' in view:\n%s", view)
	}
	if !strings.Contains(view, "[Active]") {
		t.Errorf("expected '[Active]' badge in view:\n%s", view)
	}
	if !strings.Contains(view, "(account1)") {
		t.Errorf("expected '(account1)' in view:\n%s", view)
	}
	if !strings.Contains(view, "Bob Jones") {
		t.Errorf("expected 'Bob Jones' in view:\n%s", view)
	}
	if !strings.Contains(view, "+14155550002") {
		t.Errorf("expected phone '+14155550002' in view:\n%s", view)
	}
	if !strings.Contains(view, "[+ Add Account]") {
		t.Errorf("expected '[+ Add Account]' in view:\n%s", view)
	}
}

func TestAccountSwitcher_DismissKeys(t *testing.T) {
	dismissKeys := []tea.KeyMsg{
		{Type: tea.KeyEsc},
		{Type: tea.KeyCtrlA},
		{Type: tea.KeyRunes, Runes: []rune{'q'}},
	}

	for _, k := range dismissKeys {
		fg := &Foreground{
			ModalMode: ModalModeAccountSwitcher,
			Accounts: []types.AccountsOnDeviceInfo{
				{AccountName: "Alice", Path: "account1"},
			},
			ActiveAccount: "account1",
		}
		_, cmd := fg.Update(k)
		if cmd == nil {
			t.Fatalf("expected close command on key %s, got nil", k.String())
		}
		msg := cmd()
		if _, ok := msg.(CloseOverlay); !ok {
			t.Errorf("expected CloseOverlay message on key %s, got %T", k.String(), msg)
		}
	}
}

func TestAccountSwitcher_Manager_CtrlA_Toggle(t *testing.T) {
	bg := dummyModel{text: "BACKGROUND_CHAT"}
	fg := &Foreground{}

	mgr := Manager{
		State:        MainView,
		WindowWidth:  80,
		WindowHeight: 24,
		Foreground:   fg,
		Background:   bg,
		Overlay: overlay.New(
			fg,
			bg,
			overlay.Center,
			overlay.Center,
			0,
			0,
		),
	}

	// 1. In MainView: Ctrl+A opens modal
	m, cmd := mgr.Update(tea.KeyMsg{Type: tea.KeyCtrlA})
	mgr = m.(Manager)
	if mgr.State != ModalView {
		t.Fatalf("expected State to be ModalView after Ctrl+A, got %v", mgr.State)
	}
	if cmd == nil {
		t.Fatalf("expected cmd emitting OpenModalMsg, got nil")
	}
	openMsg := cmd()
	if om, ok := openMsg.(OpenModalMsg); !ok || om.ModalMode != ModalModeAccountSwitcher {
		t.Errorf("expected OpenModalMsg with ModalModeAccountSwitcher, got %+v", openMsg)
	}

	// Set foreground modal mode to account switcher
	fg.ModalMode = ModalModeAccountSwitcher
	mgr.Foreground = fg

	// 2. In ModalView (AccountSwitcher): Ctrl+A closes modal
	m, _ = mgr.Update(tea.KeyMsg{Type: tea.KeyCtrlA})
	mgr = m.(Manager)
	if mgr.State != MainView {
		t.Fatalf("expected State to return to MainView after Ctrl+A in modal, got %v", mgr.State)
	}
}

func TestAccountSwitcher_SelectAccount_HotSwap(t *testing.T) {
	origHook := telegram.SwitchAccountHook
	defer func() { telegram.SwitchAccountHook = origHook }()

	switchedTo := ""
	telegram.SwitchAccountHook = func(targetAccount string) (types.AccountSwitchedMsg, bool) {
		switchedTo = targetAccount
		return types.AccountSwitchedMsg{
			Account: targetAccount,
			User: &types.UserInfo{
				FirstName: "Bob",
				Username:  "bob_telegram",
			},
			Err: nil,
		}, true
	}

	fg := &Foreground{
		ModalMode: ModalModeAccountSwitcher,
		Accounts: []types.AccountsOnDeviceInfo{
			{AccountName: "Alice", Path: "account1"},
			{AccountName: "Bob", Path: "account2"},
		},
		ActiveAccount:        "account1",
		AccountSwitcherIndex: 1, // Highlighting Bob
	}

	// Press Enter to switch to Bob
	m, cmd := fg.Update(tea.KeyMsg{Type: tea.KeyEnter})
	fg = m.(*Foreground)
	if !fg.IsSwitchingAccount {
		t.Errorf("expected IsSwitchingAccount to be true")
	}
	if cmd == nil {
		t.Fatalf("expected switch command, got nil")
	}

	// Execute batch command to get AccountSwitchedMsg
	switchMsg := cmd()
	// BubbleTea batch returns a slice or msg
	var accountMsg types.AccountSwitchedMsg
	switch v := switchMsg.(type) {
	case types.AccountSwitchedMsg:
		accountMsg = v
	case tea.BatchMsg:
		for _, c := range v {
			if c != nil {
				res := c()
				if am, ok := res.(types.AccountSwitchedMsg); ok {
					accountMsg = am
				}
			}
		}
	}

	if switchedTo != "account2" {
		t.Errorf("expected SwitchAccountHook called with 'account2', got %s", switchedTo)
	}

	// Feed AccountSwitchedMsg to Foreground -> should close overlay
	m, closeCmd := fg.Update(accountMsg)
	fg = m.(*Foreground)
	if fg.ActiveAccount != "account2" {
		t.Errorf("expected active account updated to 'account2', got %s", fg.ActiveAccount)
	}
	if closeCmd == nil {
		t.Fatalf("expected close command after account switched, got nil")
	}
	if _, ok := closeCmd().(CloseOverlay); !ok {
		t.Errorf("expected CloseOverlay msg, got %T", closeCmd())
	}

	// Feed AccountSwitchedMsg to Model -> should reset conversation and sidebar state
	oldCligram := telegram.Cligram
	telegram.Cligram = &client.Client{}
	defer func() { telegram.Cligram = oldCligram }()

	model := Model{
		Conversations: []types.FormattedMessage{
			{ID: 1, Content: "Old message from Alice"},
		},
		ActivePeerID: "peer123",
		CurrentUser: &types.UserInfo{
			FirstName: "Alice",
		},
	}

	updatedModel, _ := model.Update(accountMsg)
	resModel := updatedModel.(Model)

	if len(resModel.Conversations) != 0 {
		t.Errorf("expected conversations cleared after account switch, got %d", len(resModel.Conversations))
	}
	if resModel.ActivePeerID != "" {
		t.Errorf("expected ActivePeerID cleared, got %s", resModel.ActivePeerID)
	}
	if resModel.CurrentUser == nil || resModel.CurrentUser.FirstName != "Bob" {
		t.Errorf("expected CurrentUser updated to Bob, got %+v", resModel.CurrentUser)
	}
	if !resModel.SideBarLoading {
		t.Errorf("expected SideBarLoading to be true after account switch")
	}
}

func TestAccountSwitcher_AuthWizard_Transitions(t *testing.T) {
	origStart := telegram.StartAuthSessionHook
	origSend := telegram.SendAuthCodeHook
	origSign := telegram.SignInCodeHook
	origPass := telegram.AuthPasswordHook
	defer func() {
		telegram.StartAuthSessionHook = origStart
		telegram.SendAuthCodeHook = origSend
		telegram.SignInCodeHook = origSign
		telegram.AuthPasswordHook = origPass
	}()

	telegram.StartAuthSessionHook = func(accountDir string) (*telegram.AuthSession, error) {
		return &telegram.AuthSession{AccountDir: accountDir}, nil
	}
	telegram.SendAuthCodeHook = func(session *telegram.AuthSession, phone string) tea.Cmd {
		return func() tea.Msg {
			return types.AuthCodeSentMsg{PhoneCodeHash: "mock_hash_123"}
		}
	}
	telegram.SignInCodeHook = func(session *telegram.AuthSession, phone, code, phoneCodeHash string) tea.Cmd {
		return func() tea.Msg {
			if code == "12345" {
				return types.AuthSignInMsg{PasswordNeeded: true}
			}
			return types.AuthSignInMsg{Err: errors.New("invalid code")}
		}
	}
	telegram.AuthPasswordHook = func(session *telegram.AuthSession, password string) tea.Cmd {
		return func() tea.Msg {
			if password == "secret2fa" {
				return types.AuthPasswordMsg{
					User: &types.UserInfo{FirstName: "Charlie", Username: "charlie_tg"},
				}
			}
			return types.AuthPasswordMsg{Err: errors.New("invalid password")}
		}
	}

	fg := &Foreground{
		ModalMode: ModalModeAccountSwitcher,
		Accounts: []types.AccountsOnDeviceInfo{
			{AccountName: "Alice", Path: "account1"},
		},
		ActiveAccount:        "account1",
		AccountSwitcherIndex: 1, // Hovering on [+ Add Account]
		windowWidth:          80,
	}

	// 1. Press Enter on [+ Add Account] -> enters Step 0 (Phone Number)
	m, _ := fg.Update(tea.KeyMsg{Type: tea.KeyEnter})
	fg = m.(*Foreground)
	if !fg.InAuthWizard {
		t.Fatalf("expected InAuthWizard true")
	}
	if fg.AuthWizardStep != 0 {
		t.Fatalf("expected AuthWizardStep 0, got %d", fg.AuthWizardStep)
	}
	view := fg.View()
	if !strings.Contains(view, "Step 1/3: Phone Number") {
		t.Errorf("expected view to contain Step 1/3, got:\n%s", view)
	}

	// 2. Type phone number "+14155559999" and submit
	fg.authPhoneInput.SetValue("+14155559999")
	m, cmd := fg.Update(tea.KeyMsg{Type: tea.KeyEnter})
	fg = m.(*Foreground)
	if !fg.authLoading {
		t.Errorf("expected authLoading true while sending code")
	}
	if cmd == nil {
		t.Fatalf("expected SendAuthCodeCmd, got nil")
	}

	// Feed AuthCodeSentMsg -> moves to Step 1 (Login Code)
	m, _ = fg.Update(types.AuthCodeSentMsg{PhoneCodeHash: "mock_hash_123"})
	fg = m.(*Foreground)
	if fg.AuthWizardStep != 1 {
		t.Fatalf("expected AuthWizardStep 1, got %d", fg.AuthWizardStep)
	}
	if fg.authPhoneCodeHash != "mock_hash_123" {
		t.Fatalf("expected hash 'mock_hash_123', got %s", fg.authPhoneCodeHash)
	}
	view = fg.View()
	if !strings.Contains(view, "Step 2/3: Login Code") {
		t.Errorf("expected view to contain Step 2/3, got:\n%s", view)
	}

	// 3. Type code "12345" and submit
	fg.authCodeInput.SetValue("12345")
	m, cmd = fg.Update(tea.KeyMsg{Type: tea.KeyEnter})
	fg = m.(*Foreground)
	if cmd == nil {
		t.Fatalf("expected SignInCodeCmd, got nil")
	}

	// Feed AuthSignInMsg with PasswordNeeded = true -> moves to Step 2 (2FA Password)
	m, _ = fg.Update(types.AuthSignInMsg{PasswordNeeded: true})
	fg = m.(*Foreground)
	if fg.AuthWizardStep != 2 {
		t.Fatalf("expected AuthWizardStep 2 for 2FA, got %d", fg.AuthWizardStep)
	}
	view = fg.View()
	if !strings.Contains(view, "Step 3/3: Two-Factor Password (2FA)") {
		t.Errorf("expected view to contain Step 3/3, got:\n%s", view)
	}

	// 4. Type password "secret2fa" and submit
	fg.authPasswordInput.SetValue("secret2fa")
	m, cmd = fg.Update(tea.KeyMsg{Type: tea.KeyEnter})
	fg = m.(*Foreground)
	if cmd == nil {
		t.Fatalf("expected AuthPasswordCmd, got nil")
	}

	// Feed AuthPasswordMsg success -> triggers session switch!
	m, switchCmd := fg.Update(types.AuthPasswordMsg{
		User: &types.UserInfo{FirstName: "Charlie"},
	})
	fg = m.(*Foreground)
	if fg.InAuthWizard {
		t.Errorf("expected InAuthWizard to be false after successful login")
	}
	if !fg.IsSwitchingAccount {
		t.Errorf("expected IsSwitchingAccount to be true")
	}
	if switchCmd == nil {
		t.Fatalf("expected switch command, got nil")
	}
}

func TestAccountSwitcher_AuthWizard_CancelViaEsc(t *testing.T) {
	fg := &Foreground{
		ModalMode: ModalModeAccountSwitcher,
		Accounts: []types.AccountsOnDeviceInfo{
			{AccountName: "Alice", Path: "account1"},
		},
		ActiveAccount:        "account1",
		AccountSwitcherIndex: 1,
	}

	// Start wizard
	fg.startAuthWizard()
	if !fg.InAuthWizard {
		t.Fatalf("expected InAuthWizard true")
	}

	// Press Esc -> aborts wizard and returns to accounts list
	m, cmd := fg.Update(tea.KeyMsg{Type: tea.KeyEsc})
	fg = m.(*Foreground)
	if fg.InAuthWizard {
		t.Errorf("expected InAuthWizard to be false after Esc")
	}
	if cmd != nil {
		// Should NOT emit CloseOverlay, should stay in modal
		msg := cmd()
		if _, ok := msg.(CloseOverlay); ok {
			t.Errorf("Esc in wizard should NOT close overlay, should return to accounts list")
		}
	}

	// Subsequent Esc closes overlay
	_, closeCmd := fg.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if closeCmd == nil {
		t.Fatalf("expected CloseOverlay cmd on second Esc, got nil")
	}
	if _, ok := closeCmd().(CloseOverlay); !ok {
		t.Errorf("expected CloseOverlay, got %T", closeCmd())
	}
}
