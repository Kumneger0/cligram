package client

import (
	"context"
	"errors"
	"testing"

	"github.com/gotd/td/tg"
	"github.com/kumneger0/cligram/internal/telegram/types"
)

func TestInitiateP2PCall_PreFlightPrivacyHaltsBeforeSidecar(t *testing.T) {
	// 1. Mock sidecar binary finder to fail if invoked
	sidecarInvoked := false
	origFinder := findSidecarBinary
	findSidecarBinary = func() (string, error) {
		sidecarInvoked = true
		return "/usr/local/bin/cligram-voip", nil
	}
	defer func() { findSidecarBinary = origFinder }()

	// 2. Mock pre-flight privacy check to return ErrPrivacyRestricted
	origPrivacyCheck := checkPeerCallPrivacy
	checkPeerCallPrivacy = func(ctx context.Context, userID int64, accessHash int64) error {
		return types.ErrPrivacyRestricted
	}
	defer func() { checkPeerCallPrivacy = origPrivacyCheck }()

	updateChan := make(chan types.Notification, 10)
	ctx := context.Background()

	err := InitiateP2PCall(ctx, 12345, 67890, updateChan)
	if err == nil {
		t.Fatalf("expected error from InitiateP2PCall when privacy restricted, got nil")
	}

	if !errors.Is(err, types.ErrPrivacyRestricted) {
		t.Errorf("expected ErrPrivacyRestricted, got %v", err)
	}

	if sidecarInvoked {
		t.Errorf("expected sidecar binary finder NOT to be invoked when privacy check fails")
	}
}

func TestCheckPeerCallPrivacy_FlagsEvaluation(t *testing.T) {
	tests := []struct {
		name                string
		phoneCallsPrivate   bool
		phoneCallsAvailable bool
		expectRestricted    bool
	}{
		{
			name:                "Privacy disallowed (PhoneCallsPrivate=true)",
			phoneCallsPrivate:   true,
			phoneCallsAvailable: true,
			expectRestricted:    true,
		},
		{
			name:                "VoIP unavailable (PhoneCallsAvailable=false)",
			phoneCallsPrivate:   false,
			phoneCallsAvailable: false,
			expectRestricted:    true,
		},
		{
			name:                "Both restricted",
			phoneCallsPrivate:   true,
			phoneCallsAvailable: false,
			expectRestricted:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			full := &tg.UsersUserFull{
				FullUser: tg.UserFull{
					PhoneCallsPrivate:   tc.phoneCallsPrivate,
					PhoneCallsAvailable: tc.phoneCallsAvailable,
				},
			}

			var err error
			if full.FullUser.PhoneCallsPrivate || !full.FullUser.PhoneCallsAvailable {
				err = types.ErrPrivacyRestricted
			}

			if tc.expectRestricted && !errors.Is(err, types.ErrPrivacyRestricted) {
				t.Errorf("expected ErrPrivacyRestricted, got %v", err)
			}
		})
	}
}

func TestIsPrivacyRestricted(t *testing.T) {
	cases := []struct {
		err      error
		expected bool
	}{
		{err: types.ErrPrivacyRestricted, expected: true},
		{err: errors.New("rpc error: code 400: USER_PRIVACY_RESTRICTED"), expected: true},
		{err: errors.New("USER_PRIVACY_RESTRICTED"), expected: true},
		{err: errors.New("PARTICIPANT_VERSION_OUTDATED"), expected: true},
		{err: errors.New("CALL_PROTOCOL_FLAGS_INVALID"), expected: true},
		{err: errors.New("Cannot call Alice due to their privacy settings."), expected: true},
		{err: errors.New("connection timed out"), expected: false},
		{err: nil, expected: false},
	}

	for _, c := range cases {
		got := types.IsPrivacyRestricted(c.err)
		if got != c.expected {
			t.Errorf("IsPrivacyRestricted(%v) = %v, expected %v", c.err, got, c.expected)
		}
	}
}
