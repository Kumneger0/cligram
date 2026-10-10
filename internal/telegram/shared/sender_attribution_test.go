package shared_test

import (
	"strconv"
	"testing"

	"github.com/gotd/td/tg"
	"github.com/kumneger0/cligram/internal/telegram/shared"
)

func TestResolveGroupMessageSender_User(t *testing.T) {
	groupID := "100"
	userID := int64(12345)
	users := []tg.UserClass{
		&tg.User{
			ID:        userID,
			FirstName: "Alice",
			LastName:  "Smith",
		},
	}

	msg := &tg.Message{
		ID:     1,
		Out:    false,
		FromID: &tg.PeerUser{UserID: userID},
	}

	res := shared.ResolveGroupMessageSender(msg, groupID, users, nil)
	if res.Sender != "Alice Smith" {
		t.Fatalf("expected sender 'Alice Smith', got '%s'", res.Sender)
	}
	if res.FromID == nil || *res.FromID != strconv.FormatInt(userID, 10) {
		t.Fatalf("expected FromID '%d', got '%v'", userID, res.FromID)
	}
	if res.SenderUserInfo == nil || res.SenderUserInfo.FirstName != "Alice" {
		t.Fatalf("expected SenderUserInfo with FirstName 'Alice', got '%v'", res.SenderUserInfo)
	}
}

func TestResolveGroupMessageSender_LinkedChannel(t *testing.T) {
	groupID := "100"
	channelID := int64(200)
	chats := []tg.ChatClass{
		&tg.Channel{
			ID:    channelID,
			Title: "Tech News Daily",
		},
	}

	msg := &tg.Message{
		ID:     2,
		Out:    false,
		FromID: &tg.PeerChannel{ChannelID: channelID},
	}

	res := shared.ResolveGroupMessageSender(msg, groupID, nil, chats)
	expectedSender := "📢 Tech News Daily"
	if res.Sender != expectedSender {
		t.Fatalf("expected sender '%s', got '%s'", expectedSender, res.Sender)
	}
	if res.FromID == nil || *res.FromID != strconv.FormatInt(channelID, 10) {
		t.Fatalf("expected FromID '%d', got '%v'", channelID, res.FromID)
	}
}

func TestResolveGroupMessageSender_AnonymousAdmin_ChannelIDMatchesGroup(t *testing.T) {
	groupID := "100"
	channelID := int64(100)

	msg := &tg.Message{
		ID:     3,
		Out:    false,
		FromID: &tg.PeerChannel{ChannelID: channelID},
	}

	res := shared.ResolveGroupMessageSender(msg, groupID, nil, nil)
	expectedSender := "🛡️ Anonymous Admin"
	if res.Sender != expectedSender {
		t.Fatalf("expected sender '%s', got '%s'", expectedSender, res.Sender)
	}
}

func TestResolveGroupMessageSender_AnonymousAdmin_NilFromID(t *testing.T) {
	groupID := "100"

	msg := &tg.Message{
		ID:     4,
		Out:    false,
		FromID: nil,
	}

	res := shared.ResolveGroupMessageSender(msg, groupID, nil, nil)
	expectedSender := "🛡️ Anonymous Admin"
	if res.Sender != expectedSender {
		t.Fatalf("expected sender '%s', got '%s'", expectedSender, res.Sender)
	}
}

func TestResolveGroupMessageSender_OutgoingMessage(t *testing.T) {
	groupID := "100"

	msg := &tg.Message{
		ID:  5,
		Out: true,
	}

	res := shared.ResolveGroupMessageSender(msg, groupID, nil, nil)
	if res.Sender != "You" {
		t.Fatalf("expected sender 'You' for outgoing message, got '%s'", res.Sender)
	}
}
