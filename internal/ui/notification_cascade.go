package ui

import (
	"fmt"
	"time"

	"github.com/gotd/td/tg"
	"github.com/kumneger0/cligram/internal/telegram/types"
)

// NotificationDecision represents the outcome of evaluating the notification cascade.
type NotificationDecision struct {
	ShouldNotify bool
	ShowPreview  bool
}

// EvaluateNotificationCascade determines whether an incoming message should trigger a desktop alert
// based on chat focus, silent flags, peer NotifySettings, and account-level defaults.
func EvaluateNotificationCascade(
	peer any,
	message *tg.Message,
	currentPeerID string,
	accountSettings *types.AccountNotifySettings,
) NotificationDecision {
	if message == nil || message.Out {
		return NotificationDecision{ShouldNotify: false, ShowPreview: false}
	}

	peerID, sourceType, peerSettings := extractPeerNotifyInfo(peer)
	if peerID == "" {
		return NotificationDecision{ShouldNotify: false, ShowPreview: false}
	}

	if currentPeerID != "" && peerID == currentPeerID {
		return NotificationDecision{ShouldNotify: false, ShowPreview: false}
	}

	if message.Silent {
		return NotificationDecision{ShouldNotify: false, ShowPreview: false}
	}

	hasPeerDecision := false
	peerMuted := false

	if peerSettings != nil {
		if silent, ok := peerSettings.GetSilent(); ok {
			hasPeerDecision = true
			if silent {
				peerMuted = true
			}
		}

		if muteUntil, ok := peerSettings.GetMuteUntil(); ok {
			hasPeerDecision = true
			nowUnix := time.Now().Unix()
			if muteUntil > 0 && int64(muteUntil) > nowUnix {
				peerMuted = true
			} else if muteUntil == 0 || int64(muteUntil) <= nowUnix {
				if !peerMuted {
					peerMuted = false
				}
			}
		}
	}

	if hasPeerDecision {
		if peerMuted {
			return NotificationDecision{ShouldNotify: false, ShowPreview: false}
		}
		showPreview := resolveShowPreview(peerSettings, getCategoryDefault(sourceType, accountSettings))
		return NotificationDecision{ShouldNotify: true, ShowPreview: showPreview}
	}

	categoryDefault := getCategoryDefault(sourceType, accountSettings)
	if categoryDefault != nil {
		if silent, ok := categoryDefault.GetSilent(); ok && silent {
			return NotificationDecision{ShouldNotify: false, ShowPreview: false}
		}
		if muteUntil, ok := categoryDefault.GetMuteUntil(); ok {
			nowUnix := time.Now().Unix()
			if muteUntil > 0 && int64(muteUntil) > nowUnix {
				return NotificationDecision{ShouldNotify: false, ShowPreview: false}
			}
		}
	}

	showPreview := resolveShowPreview(peerSettings, categoryDefault)
	return NotificationDecision{ShouldNotify: true, ShowPreview: showPreview}
}

// extractPeerNotifyInfo retrieves the ID, source type ("users", "chats", "broadcasts"), and NotifySettings from a peer.
func extractPeerNotifyInfo(peer any) (id string, sourceType string, notifySettings *tg.PeerNotifySettings) {
	switch p := peer.(type) {
	case types.UserInfo:
		return p.PeerID, "users", p.NotifySettings
	case *types.UserInfo:
		if p != nil {
			return p.PeerID, "users", p.NotifySettings
		}
	case types.ChannelInfo:
		if p.IsBroadcast {
			return p.ID, "broadcasts", p.NotifySettings
		}
		return p.ID, "chats", p.NotifySettings
	case *types.ChannelInfo:
		if p != nil {
			if p.IsBroadcast {
				return p.ID, "broadcasts", p.NotifySettings
			}
			return p.ID, "chats", p.NotifySettings
		}
	}
	return "", "", nil
}

// getCategoryDefault returns the account default notify settings for the given source type.
func getCategoryDefault(sourceType string, accountSettings *types.AccountNotifySettings) *tg.PeerNotifySettings {
	if accountSettings == nil {
		return nil
	}
	switch sourceType {
	case "users":
		return accountSettings.Users
	case "chats":
		return accountSettings.Chats
	case "broadcasts":
		return accountSettings.Broadcasts
	default:
		return nil
	}
}

// resolveShowPreview evaluates preview preference cascading from peer to account defaults.
func resolveShowPreview(peerSettings *tg.PeerNotifySettings, defaultSettings *tg.PeerNotifySettings) bool {
	if peerSettings != nil {
		if sp, ok := peerSettings.GetShowPreviews(); ok {
			return sp
		}
	}
	if defaultSettings != nil {
		if sp, ok := defaultSettings.GetShowPreviews(); ok {
			return sp
		}
	}
	return true
}

// FormatNotificationContent constructs the notification title and body according to preview rules.
func FormatNotificationContent(peer any, message *tg.Message, showPreview bool) (title string, content string) {
	switch v := peer.(type) {
	case types.UserInfo:
		title = v.FirstName
		if v.LastName != "" {
			title = fmt.Sprintf("%s %s", v.FirstName, v.LastName)
		}
		if showPreview && message != nil {
			content = message.Message
		} else {
			content = "Sent you a new message"
		}
	case *types.UserInfo:
		title = v.FirstName
		if v.LastName != "" {
			title = fmt.Sprintf("%s %s", v.FirstName, v.LastName)
		}
		if showPreview && message != nil {
			content = message.Message
		} else {
			content = "Sent you a new message"
		}
	case types.ChannelInfo:
		title = v.ChannelTitle
		if showPreview && message != nil {
			content = message.Message
		} else {
			content = "New message"
		}
	case *types.ChannelInfo:
		title = v.ChannelTitle
		if showPreview && message != nil {
			content = message.Message
		} else {
			content = "New message"
		}
	}
	return title, content
}
