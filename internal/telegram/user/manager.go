package user

import (
	"context"
	"log/slog"
	"time"

	"github.com/gotd/td/tg"

	"github.com/kumneger0/cligram/internal/telegram/shared"
	"github.com/kumneger0/cligram/internal/telegram/types"
)

type Manager struct {
	client APIClient
}

type APIClient interface {
	GetAPI() *tg.Client
	Context() context.Context
}

func NewManager(client APIClient) types.UserManager {
	return &Manager{
		client: client,
	}
}

func (m *Manager) GetUserInfo(ctx context.Context, userID int64) (*types.UserInfo, error) {
	inputUser := &tg.InputUser{
		UserID: userID,
	}

	userClasses, err := m.client.GetAPI().UsersGetUsers(ctx, []tg.InputUserClass{inputUser})
	if err != nil {
		return nil, types.NewUserNotFoundError(userID)
	}

	if len(userClasses) == 0 {
		return nil, types.NewUserNotFoundError(userID)
	}

	tgUser, ok := userClasses[0].(*tg.User)
	if !ok {
		return nil, types.NewUserNotFoundError(userID)
	}

	return shared.ConvertTGUserToUserInfo(tgUser), nil
}

func (m *Manager) GetUserStatus(ctx context.Context, userID int64) (*types.UserStatus, error) {
	inputUser := &tg.InputUser{
		UserID: userID,
	}
	userClasses, err := m.client.GetAPI().UsersGetUsers(ctx, []tg.InputUserClass{inputUser})
	if err != nil {
		return nil, types.NewUserNotFoundError(userID)
	}
	if len(userClasses) == 0 {
		return nil, types.NewUserNotFoundError(userID)
	}
	tgUser, ok := userClasses[0].(*tg.User)
	if !ok {
		return nil, types.NewUserNotFoundError(userID)
	}

	var lastSeen time.Time
	isOnline := false

	if tgUser.Status != nil {
		switch s := tgUser.Status.(type) {
		case *tg.UserStatusOnline:
			isOnline = true
			lastSeen = time.Time{}
		case *tg.UserStatusOffline:
			isOnline = false
			lastSeen = time.Unix(int64(s.WasOnline), 0)
		case *tg.UserStatusRecently, *tg.UserStatusLastWeek, *tg.UserStatusLastMonth:
			isOnline = false
			lastSeen = time.Time{}
		}
	}

	return &types.UserStatus{
		IsOnline: isOnline,
		LastSeen: lastSeen,
	}, nil
}

func (m *Manager) SearchUsers(ctx context.Context, query string) ([]types.UserInfo, error) {
	searchRequest := &tg.ContactsSearchRequest{
		Q:     query,
		Limit: 50,
	}

	searchResult, err := m.client.GetAPI().ContactsSearch(ctx, searchRequest)
	if err != nil {
		slog.Error("failed to search users", "error", err, "query", query)
		return nil, types.NewTelegramError(types.ErrorCodeUserNotFound, "failed to search users", err)
	}

	var users []types.UserInfo
	for _, userClass := range searchResult.Users {
		if tgUser, ok := userClass.(*tg.User); ok {
			users = append(users, *shared.ConvertTGUserToUserInfo(tgUser))
		}
	}

	return users, nil
}
