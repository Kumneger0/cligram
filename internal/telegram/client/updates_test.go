package client

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/gotd/td/tg"
	"github.com/kumneger0/cligram/internal/notification"
	"github.com/kumneger0/cligram/internal/telegram/types"
)

func TestHandlePhoneCall_IncomingCallAlert(t *testing.T) {
	// 1. Mock sidecar binary finder so test does not depend on host environment
	origFinder := findSidecarBinary
	findSidecarBinary = func() (string, error) {
		return "/usr/local/bin/cligram-voip", nil
	}
	defer func() { findSidecarBinary = origFinder }()

	// 2. Intercept notification.Alert
	var alertTitle, alertMsg string
	origAlertFn := notification.AlertFn
	notification.AlertFn = func(title, msg string) {
		alertTitle = title
		alertMsg = msg
	}
	defer func() { notification.AlertFn = origAlertFn }()

	updateChan := make(chan types.Notification, 10)
	ctx := context.Background()

	req := &tg.PhoneCallRequested{
		ID:         98765,
		AccessHash: 112233,
		AdminID:    424242,
		GAHash:     []byte("test-ga-hash"),
	}

	err := handlePhoneCall(ctx, req, updateChan)
	if err != nil {
		t.Fatalf("unexpected error handling phone call: %v", err)
	}

	// Verify notification.Alert was triggered on the secondary seam
	if !strings.Contains(alertTitle, "Incoming Call") {
		t.Errorf("expected alert title to contain 'Incoming Call', got %q", alertTitle)
	}
	if !strings.Contains(alertMsg, "User 424242") {
		t.Errorf("expected alert message to contain 'User 424242', got %q", alertMsg)
	}

	// Verify update channel received CallStateIncoming
	select {
	case notif := <-updateChan:
		if notif.CallEvent == nil {
			t.Fatalf("expected CallEvent notification")
		}
		if notif.CallEvent.State != types.CallStateIncoming {
			t.Errorf("expected CallStateIncoming, got %v", notif.CallEvent.State)
		}
		if notif.CallEvent.UserID != 424242 {
			t.Errorf("expected UserID 424242, got %d", notif.CallEvent.UserID)
		}
	default:
		t.Fatalf("expected notification on update channel")
	}
}

func TestHandlePhoneCall_MissingSidecarDeclinesCall(t *testing.T) {
	origFinder := findSidecarBinary
	findSidecarBinary = func() (string, error) {
		return "", errors.New("binary not found")
	}
	defer func() { findSidecarBinary = origFinder }()

	var notifyTitle, notifyMsg string
	origNotifyFn := notification.NotifyFn
	notification.NotifyFn = func(title, msg string) {
		notifyTitle = title
		notifyMsg = msg
	}
	defer func() { notification.NotifyFn = origNotifyFn }()

	updateChan := make(chan types.Notification, 10)
	ctx := context.Background()

	req := &tg.PhoneCallRequested{
		ID:         98765,
		AccessHash: 112233,
		AdminID:    424242,
		GAHash:     []byte("test-ga-hash"),
	}

	err := handlePhoneCall(ctx, req, updateChan)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(notifyTitle, "Missed Call") {
		t.Errorf("expected missed call notification, got %q", notifyTitle)
	}
	if !strings.Contains(notifyMsg, "helper not installed") {
		t.Errorf("expected helper missing explanation, got %q", notifyMsg)
	}

	select {
	case notif := <-updateChan:
		if notif.CallEvent == nil || notif.CallEvent.State != types.CallStateBusyMissed {
			t.Errorf("expected CallStateBusyMissed, got %v", notif.CallEvent)
		}
	default:
		t.Fatalf("expected notification on update channel")
	}
}

func TestHandlePhoneCall_OutgoingCallActive(t *testing.T) {
	updateChan := make(chan types.Notification, 10)
	ctx := context.Background()

	peerUserID := int64(888999)
	callID := int64(1234567)

	p2pStatesMu.Lock()
	p2pStates[peerUserID] = &p2pCallState{
		isOutgoing: true,
		inputCall: &tg.InputPhoneCall{
			ID:         callID,
			AccessHash: 998877,
		},
	}
	p2pStatesMu.Unlock()

	defer func() {
		p2pStatesMu.Lock()
		delete(p2pStates, peerUserID)
		p2pStatesMu.Unlock()
	}()

	phoneCall := &tg.PhoneCall{
		ID:            callID,
		AccessHash:    998877,
		AdminID:       111222, // our user ID
		ParticipantID: peerUserID,
		P2PAllowed:    true,
	}

	err := handlePhoneCall(ctx, phoneCall, updateChan)
	if err != nil {
		t.Fatalf("unexpected error handling outgoing PhoneCall: %v", err)
	}

	select {
	case notif := <-updateChan:
		if notif.CallEvent == nil {
			t.Fatalf("expected CallEvent notification")
		}
		if notif.CallEvent.State != types.CallStateActive {
			t.Errorf("expected CallStateActive, got %v", notif.CallEvent.State)
		}
		if notif.CallEvent.UserID != peerUserID {
			t.Errorf("expected UserID %d, got %d", peerUserID, notif.CallEvent.UserID)
		}
		if notif.CallEvent.UserName != fmt.Sprintf("User %d", peerUserID) {
			t.Errorf("expected UserName 'User %d', got %q", peerUserID, notif.CallEvent.UserName)
		}
		if notif.CallEvent.IsRelay {
			t.Errorf("expected IsRelay=false for P2PAllowed=true, got true")
		}
	default:
		t.Fatalf("expected notification on update channel for outgoing PhoneCall")
	}

	p2pStatesMu.Lock()
	state := p2pStates[peerUserID]
	p2pStatesMu.Unlock()
	if state == nil || state.phoneCall != phoneCall {
		t.Errorf("expected p2pState to hold phoneCall object")
	}
}

func TestGlobalUpdateChannel_Binding(t *testing.T) {
	orig := GetGlobalUpdateChannel()
	defer SetGlobalUpdateChannel(orig)

	ch := make(chan types.Notification, 10)
	SetGlobalUpdateChannel(ch)

	if GetGlobalUpdateChannel() != ch {
		t.Errorf("expected GetGlobalUpdateChannel to return bound channel")
	}

	SetGlobalUpdateChannel(nil)
	if GetGlobalUpdateChannel() != nil {
		t.Errorf("expected GetGlobalUpdateChannel to return nil after clearing")
	}
}
