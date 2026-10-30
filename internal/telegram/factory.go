package telegram

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kumneger0/cligram/internal/telegram/client"
	"github.com/kumneger0/cligram/internal/telegram/types"
)

var (
	Cligram             *client.Client
	GlobalAPIID         string
	GlobalAPIHash       string
	ActiveAccount       string
	ActiveSessionCancel context.CancelFunc
	sessionMu           sync.Mutex

	// SwitchAccountHook allows tests to override account switching behavior.
	SwitchAccountHook func(targetAccount string) (types.AccountSwitchedMsg, bool)
)

func SetAPICredentials(apiID, apiHash string) {
	sessionMu.Lock()
	defer sessionMu.Unlock()
	GlobalAPIID = apiID
	GlobalAPIHash = apiHash
}

func GetAPICredentials() (string, string) {
	sessionMu.Lock()
	defer sessionMu.Unlock()
	return GlobalAPIID, GlobalAPIHash
}

func SetActiveAccount(acc string) {
	sessionMu.Lock()
	defer sessionMu.Unlock()
	ActiveAccount = acc
}

func GetActiveAccount() string {
	sessionMu.Lock()
	defer sessionMu.Unlock()
	if ActiveAccount != "" {
		return ActiveAccount
	}
	if Cligram != nil && Cligram.AccountName() != "" {
		return Cligram.AccountName()
	}
	return "account1"
}

func SetSessionCancel(cancel context.CancelFunc) {
	sessionMu.Lock()
	defer sessionMu.Unlock()
	ActiveSessionCancel = cancel
}

func NewClient(ctx context.Context, updateChannel chan types.Notification, telegramAPIID, telegramAPIHash, account string) (*client.Client, error) {
	return client.NewClientFromEnv(ctx, updateChannel, telegramAPIID, telegramAPIHash, account)
}

// SetGlobalUpdateChannel sets the global update channel used for VoIP call notifications.
func SetGlobalUpdateChannel(ch chan types.Notification) {
	client.SetGlobalUpdateChannel(ch)
}

// GetGlobalUpdateChannel returns the global update channel used for VoIP call notifications.
func GetGlobalUpdateChannel() chan types.Notification {
	return client.GetGlobalUpdateChannel()
}

func SwitchAccount(targetAccount string) tea.Cmd {
	return func() tea.Msg {
		if SwitchAccountHook != nil {
			if res, ok := SwitchAccountHook(targetAccount); ok {
				return res
			}
		}

		sessionMu.Lock()
		defer sessionMu.Unlock()

		if ActiveAccount == targetAccount && Cligram != nil {
			user, err := Cligram.GetMe(Cligram.Context())
			if err != nil {
				slog.Error(err.Error())
				return nil
			}
			return types.AccountSwitchedMsg{
				Account: targetAccount,
				User:    user,
				Err:     nil,
			}
		}

		// 1. Cleanly cancel existing session
		if ActiveSessionCancel != nil {
			ActiveSessionCancel()
			ActiveSessionCancel = nil
		}

		// 2. Create new session context
		newCtx, newCancel := context.WithCancel(context.Background())
		updateCh := GetGlobalUpdateChannel()

		newClient, err := NewClient(newCtx, updateCh, GlobalAPIID, GlobalAPIHash, targetAccount)
		if err != nil {
			newCancel()
			return types.AccountSwitchedMsg{
				Account: targetAccount,
				Err:     err,
			}
		}

		// 3. Start client.Run in background and wait for connection
		connected := make(chan struct{})
		runErrCh := make(chan error, 1)

		go func() {
			err := newClient.Client.Run(newCtx, func(runCtx context.Context) error {
				close(connected)
				<-runCtx.Done()
				return runCtx.Err()
			})
			if err != nil && !errors.Is(err, context.Canceled) {
				slog.Error("telegram client run terminated with error", "account", targetAccount, "error", err)
				runErrCh <- err
			}
		}()

		select {
		case <-connected:
		case err := <-runErrCh:
			newCancel()
			return types.AccountSwitchedMsg{
				Account: targetAccount,
				Err:     fmt.Errorf("failed to connect account %s: %w", targetAccount, err),
			}
		case <-time.After(15 * time.Second):
			newCancel()
			return types.AccountSwitchedMsg{
				Account: targetAccount,
				Err:     fmt.Errorf("connection timed out for account %s", targetAccount),
			}
		}

		// 4. Update active client pointers
		Cligram = newClient
		client.Cligram = newClient.Client
		ActiveAccount = targetAccount
		ActiveSessionCancel = newCancel

		// 5. Fetch user profile
		user, err := newClient.GetMe(newCtx)
		if err == nil && user != nil {
			_ = SaveAccountMetadata(targetAccount, types.AccountsOnDeviceInfo{
				AccountName: user.FirstName,
				PhoneNumber: user.Username,
				Path:        targetAccount,
			})
		}

		return types.AccountSwitchedMsg{
			Account: targetAccount,
			User:    user,
			Err:     nil,
		}
	}
}
