package telegram

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
	"github.com/kumneger0/cligram/internal/telegram/client"
	"github.com/kumneger0/cligram/internal/telegram/shared"
	"github.com/kumneger0/cligram/internal/telegram/types"
)

type AuthSession struct {
	Client     *client.Client
	AccountDir string
	ctx        context.Context
	cancel     context.CancelFunc
	mu         sync.Mutex
}

var (
	StartAuthSessionHook func(accountDir string) (*AuthSession, error)
	SendAuthCodeHook     func(session *AuthSession, phone string) tea.Cmd
	SignInCodeHook       func(session *AuthSession, phone, code, phoneCodeHash string) tea.Cmd
	AuthPasswordHook     func(session *AuthSession, password string) tea.Cmd
)

func StartAuthSession(accountDir string) (*AuthSession, error) {
	if StartAuthSessionHook != nil {
		return StartAuthSessionHook(accountDir)
	}

	apiID, apiHash := GetAPICredentials()
	if apiID == "" || apiHash == "" {
		return nil, errors.New("telegram API credentials not initialized")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cligram, err := NewClient(ctx, nil, apiID, apiHash, accountDir)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create client for auth: %w", err)
	}

	session := &AuthSession{
		Client:     cligram,
		AccountDir: accountDir,
		ctx:        ctx,
		cancel:     cancel,
	}

	connected := make(chan struct{})
	errCh := make(chan error, 1)

	go func() {
		err := cligram.Client.Run(ctx, func(cCtx context.Context) error {
			close(connected)
			<-cCtx.Done()
			return cCtx.Err()
		})
		if err != nil && !errors.Is(err, context.Canceled) {
			slog.Debug("auth session run finished", "error", err)
			errCh <- err
		}
	}()

	select {
	case <-connected:
		return session, nil
	case err := <-errCh:
		cancel()
		return nil, fmt.Errorf("auth connection failed: %w", err)
	case <-time.After(15 * time.Second):
		cancel()
		return nil, errors.New("auth connection timed out")
	}
}

func CancelAuthSession(session *AuthSession) {
	if session == nil {
		return
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.cancel != nil {
		session.cancel()
		session.cancel = nil
	}
}

func SendAuthCodeCmd(session *AuthSession, phone string) tea.Cmd {
	if SendAuthCodeHook != nil {
		return SendAuthCodeHook(session, phone)
	}

	return func() tea.Msg {
		if session == nil || session.Client == nil {
			return types.AuthCodeSentMsg{Err: errors.New("invalid auth session")}
		}

		sentCodeClass, err := session.Client.Client.Auth().SendCode(session.ctx, phone, auth.SendCodeOptions{})
		if err != nil {
			return types.AuthCodeSentMsg{Err: err}
		}

		authCode, ok := sentCodeClass.(*tg.AuthSentCode)
		if !ok {
			return types.AuthCodeSentMsg{Err: errors.New("unexpected response from Telegram while sending code")}
		}

		return types.AuthCodeSentMsg{
			PhoneCodeHash: authCode.PhoneCodeHash,
			Err:           nil,
		}
	}
}

func SignInCodeCmd(session *AuthSession, phone, code, phoneCodeHash string) tea.Cmd {
	if SignInCodeHook != nil {
		return SignInCodeHook(session, phone, code, phoneCodeHash)
	}

	return func() tea.Msg {
		if session == nil || session.Client == nil {
			return types.AuthSignInMsg{Err: errors.New("invalid auth session")}
		}

		authAuthorization, err := session.Client.Client.Auth().SignIn(session.ctx, phone, code, phoneCodeHash)
		if err != nil {
			if errors.Is(err, auth.ErrPasswordAuthNeeded) {
				return types.AuthSignInMsg{
					PasswordNeeded: true,
					Err:            nil,
				}
			}
			return types.AuthSignInMsg{Err: err}
		}

		if authAuthorization != nil {
			self, err := session.Client.Client.Self(session.ctx)
			var userInfo *types.UserInfo
			if err == nil && self != nil {
				userInfo = shared.ConvertTGUserToUserInfo(self)
				_ = SaveAccountMetadata(session.AccountDir, types.AccountsOnDeviceInfo{
					AccountName: self.FirstName,
					PhoneNumber: self.Phone,
					Path:        session.AccountDir,
				})
			}
			return types.AuthSignInMsg{
				PasswordNeeded: false,
				User:           userInfo,
				Err:            nil,
			}
		}

		return types.AuthSignInMsg{Err: errors.New("sign in failed with no authorization response")}
	}
}

func AuthPasswordCmd(session *AuthSession, password string) tea.Cmd {
	if AuthPasswordHook != nil {
		return AuthPasswordHook(session, password)
	}

	return func() tea.Msg {
		if session == nil || session.Client == nil {
			return types.AuthPasswordMsg{Err: errors.New("invalid auth session")}
		}

		authorization, err := session.Client.Client.Auth().Password(session.ctx, password)
		if err != nil {
			return types.AuthPasswordMsg{Err: err}
		}

		if authorization != nil {
			self, err := session.Client.Client.Self(session.ctx)
			var userInfo *types.UserInfo
			if err == nil && self != nil {
				userInfo = shared.ConvertTGUserToUserInfo(self)
				_ = SaveAccountMetadata(session.AccountDir, types.AccountsOnDeviceInfo{
					AccountName: self.FirstName,
					PhoneNumber: self.Phone,
					Path:        session.AccountDir,
				})
			}
			return types.AuthPasswordMsg{
				User: userInfo,
				Err:  nil,
			}
		}

		return types.AuthPasswordMsg{Err: errors.New("password authentication failed")}
	}
}
