package types

import (
	"errors"
	"fmt"
	"testing"
)

func TestIsSessionRevoked(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "ErrSessionRevoked direct",
			err:      ErrSessionRevoked,
			expected: true,
		},
		{
			name:     "ErrSessionRevoked wrapped",
			err:      fmt.Errorf("failed to switch: %w", ErrSessionRevoked),
			expected: true,
		},
		{
			name:     "rpc SESSION_REVOKED text",
			err:      errors.New("rpcDoRequest: rpc error code 401: SESSION_REVOKED"),
			expected: true,
		},
		{
			name:     "rpc SESSION_EXPIRED text",
			err:      errors.New("rpc error code 401: SESSION_EXPIRED"),
			expected: true,
		},
		{
			name:     "rpc AUTH_KEY_UNREGISTERED text",
			err:      errors.New("rpc error code 401: AUTH_KEY_UNREGISTERED"),
			expected: true,
		},
		{
			name:     "rpc USER_DEACTIVATED text",
			err:      errors.New("rpc error code 401: USER_DEACTIVATED"),
			expected: true,
		},
		{
			name:     "TelegramError wrapping SESSION_REVOKED",
			err:      NewTelegramError(ErrorCodeAuthFailed, "auth failed", errors.New("401: SESSION_REVOKED")),
			expected: true,
		},
		{
			name:     "unrelated error",
			err:      errors.New("FLOOD_WAIT_10"),
			expected: false,
		},
		{
			name:     "network timeout",
			err:      errors.New("i/o timeout"),
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := IsSessionRevoked(tc.err)
			if actual != tc.expected {
				t.Errorf("IsSessionRevoked(%v) = %v; want %v", tc.err, actual, tc.expected)
			}
		})
	}
}
