package ui

import (
	"math"
	"testing"
	"time"

	"github.com/gotd/td/tg"
	"github.com/kumneger0/cligram/internal/notification"
	"github.com/kumneger0/cligram/internal/telegram/types"
)

func TestNotificationCascade_OutgoingMessage(t *testing.T) {
	user := types.UserInfo{
		PeerID:    "123",
		FirstName: "Alice",
	}
	msg := &tg.Message{
		Out:     true,
		Message: "Hello Alice",
	}

	decision := EvaluateNotificationCascade(user, msg, "999", nil)
	if decision.ShouldNotify {
		t.Errorf("expected outgoing message not to trigger notification")
	}
}

func TestNotificationCascade_FocusSuppression(t *testing.T) {
	user := types.UserInfo{
		PeerID:    "123",
		FirstName: "Alice",
	}
	msg := &tg.Message{
		Message: "Hey there!",
	}

	// Currently focused on Alice (peerID "123")
	decisionFocused := EvaluateNotificationCascade(user, msg, "123", nil)
	if decisionFocused.ShouldNotify {
		t.Errorf("expected notification to be suppressed for currently focused chat")
	}

	// Currently focused on Bob (peerID "456")
	decisionUnfocused := EvaluateNotificationCascade(user, msg, "456", nil)
	if !decisionUnfocused.ShouldNotify {
		t.Errorf("expected notification to fire for unfocused chat")
	}

	// Test channel focus suppression
	channel := types.ChannelInfo{
		ID:           "789",
		ChannelTitle: "News Channel",
		IsBroadcast:  true,
	}
	decisionChannelFocused := EvaluateNotificationCascade(channel, msg, "789", nil)
	if decisionChannelFocused.ShouldNotify {
		t.Errorf("expected notification to be suppressed for currently focused channel")
	}

	decisionChannelUnfocused := EvaluateNotificationCascade(channel, msg, "123", nil)
	if !decisionChannelUnfocused.ShouldNotify {
		t.Errorf("expected notification to fire for unfocused channel")
	}
}

func TestNotificationCascade_SilentMessage(t *testing.T) {
	user := types.UserInfo{
		PeerID:    "123",
		FirstName: "Alice",
	}
	msg := &tg.Message{
		Silent:  true,
		Message: "Silent update",
	}

	decision := EvaluateNotificationCascade(user, msg, "999", nil)
	if decision.ShouldNotify {
		t.Errorf("expected silent message flag to suppress notification")
	}
}

func TestNotificationCascade_PeerNotifySettings(t *testing.T) {
	// 1. Peer with explicit Silent flag
	silentSettings := &tg.PeerNotifySettings{}
	silentSettings.SetSilent(true)

	userSilent := types.UserInfo{
		PeerID:         "101",
		FirstName:      "SilentUser",
		NotifySettings: silentSettings,
	}
	msg := &tg.Message{Message: "Are you there?"}

	decisionSilent := EvaluateNotificationCascade(userSilent, msg, "999", nil)
	if decisionSilent.ShouldNotify {
		t.Errorf("expected peer with Silent=true to suppress notification")
	}

	// 2. Peer with Silent=false
	unmutedSettings := &tg.PeerNotifySettings{}
	unmutedSettings.SetSilent(false)

	userUnmuted := types.UserInfo{
		PeerID:         "102",
		FirstName:      "LoudUser",
		NotifySettings: unmutedSettings,
	}
	decisionUnmuted := EvaluateNotificationCascade(userUnmuted, msg, "999", nil)
	if !decisionUnmuted.ShouldNotify {
		t.Errorf("expected peer with Silent=false to trigger notification")
	}

	// 3. Peer with active MuteUntil timestamp in the future
	futureMuteSettings := &tg.PeerNotifySettings{}
	futureMuteSettings.SetMuteUntil(int(time.Now().Add(1 * time.Hour).Unix()))

	userMutedUntil := types.UserInfo{
		PeerID:         "103",
		FirstName:      "MutedUntilUser",
		NotifySettings: futureMuteSettings,
	}
	decisionMutedUntil := EvaluateNotificationCascade(userMutedUntil, msg, "999", nil)
	if decisionMutedUntil.ShouldNotify {
		t.Errorf("expected active MuteUntil in future to suppress notification")
	}

	// 4. Peer with permanent mute (math.MaxInt32)
	permMuteSettings := &tg.PeerNotifySettings{}
	permMuteSettings.SetMuteUntil(math.MaxInt32)

	userPermMuted := types.UserInfo{
		PeerID:         "104",
		FirstName:      "PermMutedUser",
		NotifySettings: permMuteSettings,
	}
	decisionPermMuted := EvaluateNotificationCascade(userPermMuted, msg, "999", nil)
	if decisionPermMuted.ShouldNotify {
		t.Errorf("expected permanent mute (MaxInt32) to suppress notification")
	}

	// 5. Peer with expired MuteUntil timestamp in the past
	pastMuteSettings := &tg.PeerNotifySettings{}
	pastMuteSettings.SetMuteUntil(int(time.Now().Add(-10 * time.Minute).Unix()))

	userExpiredMute := types.UserInfo{
		PeerID:         "105",
		FirstName:      "ExpiredMuteUser",
		NotifySettings: pastMuteSettings,
	}
	decisionExpiredMute := EvaluateNotificationCascade(userExpiredMute, msg, "999", nil)
	if !decisionExpiredMute.ShouldNotify {
		t.Errorf("expected expired MuteUntil to allow notification")
	}
}

func TestNotificationCascade_AccountDefaultsFallback(t *testing.T) {
	// Account-level defaults:
	// - Users: unmuted
	// - Chats: muted via MuteUntil
	// - Broadcasts: muted via Silent
	userDefault := &tg.PeerNotifySettings{}
	userDefault.SetSilent(false)

	chatDefault := &tg.PeerNotifySettings{}
	chatDefault.SetMuteUntil(int(time.Now().Add(2 * time.Hour).Unix()))

	broadcastDefault := &tg.PeerNotifySettings{}
	broadcastDefault.SetSilent(true)

	accountDefaults := &types.AccountNotifySettings{
		Users:      userDefault,
		Chats:      chatDefault,
		Broadcasts: broadcastDefault,
	}

	msg := &tg.Message{Message: "Testing account defaults"}

	// 1. Private chat user with unset settings -> falls back to Users default (unmuted)
	userNoSettings := types.UserInfo{
		PeerID:         "201",
		FirstName:      "DefaultUser",
		NotifySettings: nil,
	}
	decisionUser := EvaluateNotificationCascade(userNoSettings, msg, "999", accountDefaults)
	if !decisionUser.ShouldNotify {
		t.Errorf("expected unset user to inherit unmuted account default for Users")
	}

	// 2. Group with unset settings -> falls back to Chats default (muted)
	groupNoSettings := types.ChannelInfo{
		ID:             "301",
		ChannelTitle:   "DefaultGroup",
		IsBroadcast:    false,
		NotifySettings: nil,
	}
	decisionGroup := EvaluateNotificationCascade(groupNoSettings, msg, "999", accountDefaults)
	if decisionGroup.ShouldNotify {
		t.Errorf("expected unset group to inherit muted account default for Chats")
	}

	// 3. Channel with unset settings -> falls back to Broadcasts default (muted)
	channelNoSettings := types.ChannelInfo{
		ID:             "401",
		ChannelTitle:   "DefaultChannel",
		IsBroadcast:    true,
		NotifySettings: nil,
	}
	decisionChannel := EvaluateNotificationCascade(channelNoSettings, msg, "999", accountDefaults)
	if decisionChannel.ShouldNotify {
		t.Errorf("expected unset channel to inherit silent account default for Broadcasts")
	}

	// 4. Peer with explicit settings overrides account defaults
	channelExplicitUnmuted := types.ChannelInfo{
		ID:           "402",
		ChannelTitle: "ImportantChannel",
		IsBroadcast:  true,
		NotifySettings: func() *tg.PeerNotifySettings {
			s := &tg.PeerNotifySettings{}
			s.SetSilent(false)
			s.SetMuteUntil(0)
			return s
		}(),
	}
	decisionExplicit := EvaluateNotificationCascade(channelExplicitUnmuted, msg, "999", accountDefaults)
	if !decisionExplicit.ShouldNotify {
		t.Errorf("expected peer explicit unmuted setting to override muted account default")
	}
}

func TestNotificationCascade_PreviewResolution(t *testing.T) {
	// Account default has ShowPreviews = false
	userDefault := &tg.PeerNotifySettings{}
	userDefault.SetShowPreviews(false)
	accountDefaults := &types.AccountNotifySettings{
		Users: userDefault,
	}

	msg := &tg.Message{Message: "Secret preview text"}

	// 1. User without settings inherits account default (ShowPreviews = false)
	userInherit := types.UserInfo{
		PeerID:         "501",
		FirstName:      "Alice",
		NotifySettings: nil,
	}
	decisionInherit := EvaluateNotificationCascade(userInherit, msg, "999", accountDefaults)
	if !decisionInherit.ShouldNotify {
		t.Fatalf("expected decision to notify")
	}
	if decisionInherit.ShowPreview {
		t.Errorf("expected ShowPreview to be false from account default")
	}
	title, content := FormatNotificationContent(userInherit, msg, decisionInherit.ShowPreview)
	if title != "Alice" || content != "Sent you a new message" {
		t.Errorf("expected generic message content, got title=%q, content=%q", title, content)
	}

	// 2. User with explicit ShowPreviews = true overrides account default
	peerPreviewsOn := &tg.PeerNotifySettings{}
	peerPreviewsOn.SetShowPreviews(true)
	userOverride := types.UserInfo{
		PeerID:         "502",
		FirstName:      "Bob",
		NotifySettings: peerPreviewsOn,
	}
	decisionOverride := EvaluateNotificationCascade(userOverride, msg, "999", accountDefaults)
	if !decisionOverride.ShouldNotify {
		t.Fatalf("expected decision to notify")
	}
	if !decisionOverride.ShowPreview {
		t.Errorf("expected ShowPreview to be true from peer override")
	}
	title2, content2 := FormatNotificationContent(userOverride, msg, decisionOverride.ShowPreview)
	if title2 != "Bob" || content2 != "Secret preview text" {
		t.Errorf("expected actual message preview, got title=%q, content=%q", title2, content2)
	}
}

func TestModel_LiveNotifySettingsUpdate(t *testing.T) {
	user := types.UserInfo{
		PeerID:    "601",
		FirstName: "Carol",
	}

	m := Model{
		SelectedUser: user,
	}

	// 1. Initially Carol has no NotifySettings
	if m.SelectedUser.NotifySettings != nil {
		t.Fatalf("expected initial NotifySettings to be nil")
	}

	// 2. Send peer-level NotifySettingsUpdateNotification muting Carol
	mutedSettings := &tg.PeerNotifySettings{}
	mutedSettings.SetSilent(true)

	updatedModel, _ := m.Update(types.NotifySettingsUpdateNotification{
		PeerID:         "601",
		SourceType:     "peer",
		NotifySettings: mutedSettings,
	})
	m = updatedModel.(Model)

	if m.SelectedUser.NotifySettings == nil {
		t.Fatalf("expected SelectedUser.NotifySettings to be updated")
	}
	if silent, ok := m.SelectedUser.NotifySettings.GetSilent(); !ok || !silent {
		t.Errorf("expected SelectedUser to be muted after update")
	}

	// 3. Send account-level default update for Users
	newUsersDefault := &tg.PeerNotifySettings{}
	newUsersDefault.SetMuteUntil(int(time.Now().Add(5 * time.Hour).Unix()))

	updatedModel2, _ := m.Update(types.NotifySettingsUpdateNotification{
		SourceType:     "users",
		NotifySettings: newUsersDefault,
	})
	m = updatedModel2.(Model)

	if m.AccountNotifySettings == nil || m.AccountNotifySettings.Users == nil {
		t.Fatalf("expected AccountNotifySettings.Users to be updated")
	}
	if muteUntil, ok := m.AccountNotifySettings.Users.GetMuteUntil(); !ok || muteUntil == 0 {
		t.Errorf("expected AccountNotifySettings.Users to have MuteUntil set")
	}
}

func TestModel_SendNewMessageNotification_EndToEnd(t *testing.T) {
	var notifiedTitle, notifiedContent string
	notificationCount := 0

	origNotifyFn := notification.NotifyFn
	notification.NotifyFn = func(title, message string) {
		notificationCount++
		notifiedTitle = title
		notifiedContent = message
	}
	defer func() {
		notification.NotifyFn = origNotifyFn
	}()

	user := types.UserInfo{
		PeerID:    "701",
		FirstName: "Dan",
		LastName:  "Smith",
	}

	m := &Model{
		ActivePeerID: "999", // Currently focused on chat 999
	}

	msg := &tg.Message{
		Message: "Hello Dan here!",
	}

	// 1. Message arrives from Dan while viewing chat 999 -> should notify
	m.sendNewMessageNotification(user, msg)
	if notificationCount != 1 {
		t.Errorf("expected 1 notification, got %d", notificationCount)
	}
	if notifiedTitle != "Dan Smith" || notifiedContent != "Hello Dan here!" {
		t.Errorf("unexpected notification content: title=%q, content=%q", notifiedTitle, notifiedContent)
	}

	// 2. User switches focus to Dan (ActivePeerID = "701") -> notification should be suppressed
	m.ActivePeerID = "701"
	m.sendNewMessageNotification(user, msg)
	if notificationCount != 1 {
		t.Errorf("expected notification to be suppressed for focused chat, count is %d", notificationCount)
	}

	// 3. User switches away from Dan (ActivePeerID = "999"), but Dan is muted
	m.ActivePeerID = "999"
	mutedSettings := &tg.PeerNotifySettings{}
	mutedSettings.SetSilent(true)
	user.NotifySettings = mutedSettings

	m.sendNewMessageNotification(user, msg)
	if notificationCount != 1 {
		t.Errorf("expected notification to be suppressed for muted peer, count is %d", notificationCount)
	}
}
