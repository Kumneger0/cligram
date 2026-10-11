package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/gotd/td/tg"
	"github.com/kumneger0/cligram/internal/telegram/types"
)

func TestReactiveViewport_EditMessageInPlace(t *testing.T) {
	m := newTestModel(80, 24)
	m.ActivePeerID = "123"
	peerID := "123"

	origMsg := types.FormattedMessage{
		ID:       10,
		PeerID:   &peerID,
		Sender:   "Alice",
		Content:  "Original text",
		Date:     time.Now().Add(-5 * time.Minute),
		IsEdited: false,
	}
	m.Conversations = []types.FormattedMessage{origMsg}
	m.updateConversations()

	initialBubble := m.renderMessageBubble(m.Conversations[0], 0, 40, 80, 0)
	if strings.Contains(initialBubble, "(edited)") {
		t.Fatalf("expected initial bubble not to contain (edited), got: %s", initialBubble)
	}

	editTime := time.Now()
	editNotification := types.EditMessageNotification{
		ID:     10,
		PeerID: "123",
		Message: &tg.Message{
			ID:      10,
			Message: "Updated in-place text",
			Out:     false,
		},
		EditDate: editTime,
	}

	updatedModel, _ := m.Update(editNotification)
	m = updatedModel.(Model)

	if len(m.Conversations) != 1 {
		t.Fatalf("expected 1 conversation message, got %d", len(m.Conversations))
	}
	if m.Conversations[0].Content != "Updated in-place text" {
		t.Errorf("expected content 'Updated in-place text', got %q", m.Conversations[0].Content)
	}
	if !m.Conversations[0].IsEdited {
		t.Errorf("expected IsEdited to be true")
	}
	if m.Conversations[0].EditDate == nil {
		t.Errorf("expected EditDate to not be nil")
	}

	editedBubble := m.renderMessageBubble(m.Conversations[0], 0, 40, 80, 0)
	if !strings.Contains(editedBubble, "(edited)") {
		t.Errorf("expected edited bubble to contain '(edited)', got: %s", editedBubble)
	}
	if !strings.Contains(editedBubble, "Updated in-place text") {
		t.Errorf("expected edited bubble to contain 'Updated in-place text', got: %s", editedBubble)
	}
}

func TestReactiveViewport_EditOutgoingMessageLocal(t *testing.T) {
	m := newTestModel(80, 24)
	peerID := "123"
	m.ActivePeerID = peerID
	m.Mode = ModeUsers
	m.Users.SetItems([]list.Item{types.UserInfo{PeerID: peerID, FirstName: "Alice"}})
	m.Users.Select(0)

	origMsg := types.FormattedMessage{
		ID:       20,
		PeerID:   &peerID,
		Sender:   "You",
		IsFromMe: true,
		Content:  "Original my message",
		Date:     time.Now().Add(-2 * time.Minute),
	}
	m.Conversations = []types.FormattedMessage{origMsg}
	m.SelectedMessageIndex = 0
	m.updateConversations()

	// Simulate pressing 'e' to edit message 20
	msgCopy := m.Conversations[0]
	m.EditMessage = &msgCopy
	m.Input.SetValue("Edited my message")

	// Submit edit by pressing Enter (sendMessage)
	mUpdated, _ := sendMessage(&m)

	if len(mUpdated.Conversations) != 1 {
		t.Fatalf("expected 1 conversation message, got %d", len(mUpdated.Conversations))
	}
	if mUpdated.Conversations[0].Content != "Edited my message" {
		t.Errorf("expected updated content 'Edited my message', got %q", mUpdated.Conversations[0].Content)
	}
	if !mUpdated.Conversations[0].IsEdited {
		t.Errorf("expected IsEdited to be true")
	}
	if mUpdated.Conversations[0].EditDate == nil {
		t.Errorf("expected EditDate to not be nil")
	}
	if mUpdated.EditMessage != nil {
		t.Errorf("expected EditMessage to be reset to nil")
	}

	bubble := mUpdated.renderMessageBubble(mUpdated.Conversations[0], 0, 40, 80, 0)
	if !strings.Contains(bubble, "(edited)") {
		t.Errorf("expected rendered bubble to show '(edited)', got: %s", bubble)
	}
	if !strings.Contains(bubble, "Edited my message") {
		t.Errorf("expected rendered bubble to show updated text, got: %s", bubble)
	}
}

func TestReactiveViewport_DeleteMessagesImmediate(t *testing.T) {
	m := newTestModel(80, 24)
	m.ActivePeerID = "123"
	peerID := "123"

	m.Conversations = []types.FormattedMessage{
		{ID: 1, PeerID: &peerID, Content: "Msg 1", Date: time.Now()},
		{ID: 2, PeerID: &peerID, Content: "Msg 2", Date: time.Now()},
		{ID: 3, PeerID: &peerID, Content: "Msg 3", Date: time.Now()},
	}
	m.SelectedMessageIndex = 2
	m.updateConversations()

	delNotification := types.DeleteMessagesNotification{
		ChannelID: 0,
		Messages:  []int{2},
	}
	updatedModel, _ := m.Update(delNotification)
	m = updatedModel.(Model)

	if len(m.Conversations) != 2 {
		t.Fatalf("expected 2 messages remaining, got %d", len(m.Conversations))
	}
	if m.Conversations[0].ID != 1 || m.Conversations[1].ID != 3 {
		t.Errorf("expected message IDs [1, 3], got [%d, %d]", m.Conversations[0].ID, m.Conversations[1].ID)
	}
	if m.SelectedMessageIndex != 1 {
		t.Errorf("expected SelectedMessageIndex 1, got %d", m.SelectedMessageIndex)
	}

	delAllNotification := types.DeleteMessagesNotification{
		ChannelID: 0,
		Messages:  []int{1, 3},
	}
	updatedModel, _ = m.Update(delAllNotification)
	m = updatedModel.(Model)

	if len(m.Conversations) != 0 {
		t.Fatalf("expected 0 messages remaining, got %d", len(m.Conversations))
	}

	viewportOutput := m.renderMessagesViewport(80, 24)
	if !strings.Contains(viewportOutput, "No messages yet") {
		t.Errorf("expected viewport to show 'No messages yet', got: %s", viewportOutput)
	}
}

func TestReactiveViewport_ServiceMessages(t *testing.T) {
	m := newTestModel(80, 24)
	m.ActivePeerID = "456"

	serviceMsg := types.FormattedMessage{
		ID:        50,
		Sender:    "Bob",
		Content:   "👤 Bob joined the group via invite link",
		IsService: true,
		Date:      time.Now(),
	}

	bubble := m.renderMessageBubble(serviceMsg, 0, 40, 80, 0)
	if !strings.Contains(bubble, "👤 Bob joined the group via invite link") {
		t.Errorf("expected bubble to contain service message text, got: %s", bubble)
	}

	normalMsg := types.FormattedMessage{
		ID:      51,
		Sender:  "Bob",
		Content: "Hello everyone",
		Date:    time.Now(),
	}
	if shouldCoalesceMessages(serviceMsg, normalMsg) {
		t.Errorf("expected shouldCoalesceMessages to be false when one is service message")
	}
	if shouldCoalesceMessages(normalMsg, serviceMsg) {
		t.Errorf("expected shouldCoalesceMessages to be false when prev is service message")
	}

	newMsgNotification := types.NewMessageNotification{
		ID:        50,
		PeerID:    "456",
		Formatted: &serviceMsg,
	}
	updatedModel, _ := m.Update(newMsgNotification)
	m = updatedModel.(Model)

	if len(m.Conversations) != 1 {
		t.Fatalf("expected 1 conversation message, got %d", len(m.Conversations))
	}
	if !m.Conversations[0].IsService {
		t.Errorf("expected conversation message to have IsService true")
	}
	if m.Conversations[0].Content != "👤 Bob joined the group via invite link" {
		t.Errorf("expected content to match service message, got %q", m.Conversations[0].Content)
	}
}

func TestReactiveViewport_DraftsPreservationAndClear(t *testing.T) {
	m := newTestModel(80, 24)
	m.Mode = ModeUsers
	m.ActivePeerID = "user1"
	m.SelectedUser = types.UserInfo{PeerID: "user1", FirstName: "Alice"}
	m.Users.SetItems([]list.Item{
		types.UserInfo{PeerID: "user1", FirstName: "Alice"},
		types.UserInfo{PeerID: "user2", FirstName: "Bob"},
	})

	m.Users.Select(0)
	m.Input.SetValue("Draft for Alice")

	m.Users.Select(1)
	mUpdated, _ := handleUserChange(&m, nil, nil)
	m = mUpdated

	if m.ActivePeerID != "user2" {
		t.Fatalf("expected ActivePeerID 'user2', got %q", m.ActivePeerID)
	}
	if m.Drafts["user1"] != "Draft for Alice" {
		t.Errorf("expected Drafts[user1] to be 'Draft for Alice', got %q", m.Drafts["user1"])
	}
	if m.Input.Value() != "" {
		t.Errorf("expected Input for user2 to be empty, got %q", m.Input.Value())
	}

	m.Input.SetValue("Draft for Bob")

	m.Users.Select(0)
	mUpdated, _ = handleUserChange(&m, nil, nil)
	m = mUpdated

	if m.ActivePeerID != "user1" {
		t.Fatalf("expected ActivePeerID 'user1', got %q", m.ActivePeerID)
	}
	if m.Drafts["user2"] != "Draft for Bob" {
		t.Errorf("expected Drafts[user2] to be 'Draft for Bob', got %q", m.Drafts["user2"])
	}
	if m.Input.Value() != "Draft for Alice" {
		t.Errorf("expected restored draft for Alice to be 'Draft for Alice', got %q", m.Input.Value())
	}

	mSent, _ := sendMessage(&m)
	m = mSent

	if m.Input.Value() != "" {
		t.Errorf("expected input to be reset after sendMessage, got %q", m.Input.Value())
	}
	if _, exists := m.Drafts["user1"]; exists {
		t.Errorf("expected Drafts[user1] to be cleared after sending message")
	}
}
