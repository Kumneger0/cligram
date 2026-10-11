package client

import (
	"context"
	"testing"

	"github.com/gotd/td/tg"
	"github.com/kumneger0/cligram/internal/telegram/types"
)

func TestClientInviteMethods_Validation(t *testing.T) {
	c := &Client{}

	// 1. CheckChatInvite with empty hash
	cmd := c.CheckChatInvite(context.Background(), "")
	msg := cmd()
	checkMsg, ok := msg.(types.CheckChatInviteMsg)
	if !ok {
		t.Fatalf("expected CheckChatInviteMsg, got %T", msg)
	}
	if checkMsg.Err == nil {
		t.Errorf("expected error for empty invite hash")
	}

	// 2. ImportChatInvite with empty hash
	cmd = c.ImportChatInvite(context.Background(), "")
	msg = cmd()
	importMsg, ok := msg.(types.ImportChatInviteMsg)
	if !ok {
		t.Fatalf("expected ImportChatInviteMsg, got %T", msg)
	}
	if importMsg.Err == nil {
		t.Errorf("expected error for empty invite hash")
	}

	// 3. JoinChannel with 0 channel ID
	cmd = c.JoinChannel(context.Background(), 0, 0)
	msg = cmd()
	joinMsg, ok := msg.(types.JoinChannelMsg)
	if !ok {
		t.Fatalf("expected JoinChannelMsg, got %T", msg)
	}
	if joinMsg.Err == nil {
		t.Errorf("expected error for 0 channel ID")
	}
}

func TestExtractChatFromUpdates(t *testing.T) {
	// Nil updates
	if res := extractChatFromUpdates(nil); res != nil {
		t.Errorf("expected nil for nil updates, got %+v", res)
	}

	// *tg.Updates with a Channel
	ch := &tg.Channel{
		ID:                12345,
		Title:             "Updates Channel",
		AccessHash:        67890,
		Broadcast:         true,
		ParticipantsCount: 500,
	}
	updates := &tg.Updates{
		Chats: []tg.ChatClass{ch},
	}
	res := extractChatFromUpdates(updates)
	if res == nil {
		t.Fatalf("expected non-nil ChannelInfo from updates")
	}
	if res.ID != "12345" || res.ChannelTitle != "Updates Channel" || !res.IsBroadcast {
		t.Errorf("unexpected extracted channel info: %+v", res)
	}

	// *tg.UpdatesCombined with a Chat
	chat := &tg.Chat{
		ID:                54321,
		Title:             "Updates Group",
		ParticipantsCount: 15,
	}
	combined := &tg.UpdatesCombined{
		Chats: []tg.ChatClass{chat},
	}
	resGroup := extractChatFromUpdates(combined)
	if resGroup == nil {
		t.Fatalf("expected non-nil ChannelInfo from combined updates")
	}
	if resGroup.ID != "54321" || resGroup.ChannelTitle != "Updates Group" || resGroup.IsBroadcast {
		t.Errorf("unexpected extracted group info: %+v", resGroup)
	}
}
