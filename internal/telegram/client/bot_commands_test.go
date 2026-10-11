package client

import (
	"context"
	"testing"

	"github.com/kumneger0/cligram/internal/telegram/types"
)

func TestBotCommandsCache(t *testing.T) {
	ClearCachedBotCommands()

	// Empty peer ID should return nil
	c := &Client{}
	cmds, err := c.fetchPeerBotCommands(context.Background(), types.Peer{ID: ""})
	if err != nil {
		t.Fatalf("expected nil err for empty peer, got %v", err)
	}
	if len(cmds) != 0 {
		t.Fatalf("expected empty cmds, got %+v", cmds)
	}

	// Test SetCachedBotCommands and cached hit
	testCmds := []types.BotCommand{
		{Command: "start", Description: "Start the bot"},
		{Command: "settings", Description: "Bot settings"},
	}
	SetCachedBotCommands("123", testCmds)

	cached, err := c.fetchPeerBotCommands(context.Background(), types.Peer{ID: "123"})
	if err != nil {
		t.Fatalf("unexpected err for cached peer: %v", err)
	}
	if len(cached) != 2 {
		t.Fatalf("expected 2 cached commands, got %d", len(cached))
	}
	if cached[0].Command != "start" || cached[1].Command != "settings" {
		t.Errorf("unexpected cached commands: %+v", cached)
	}

	// Test GetPeerBotCommands tea.Cmd execution
	cmd := c.GetPeerBotCommands(context.Background(), types.Peer{ID: "123"})
	msg := cmd()
	peerMsg, ok := msg.(types.PeerBotCommandsMsg)
	if !ok {
		t.Fatalf("expected PeerBotCommandsMsg, got %T", msg)
	}
	if peerMsg.PeerID != "123" {
		t.Errorf("expected PeerID '123', got %q", peerMsg.PeerID)
	}
	if len(peerMsg.Commands) != 2 {
		t.Errorf("expected 2 commands in msg, got %d", len(peerMsg.Commands))
	}

	// Test ClearCachedBotCommands
	ClearCachedBotCommands()
	// With empty client, uncached should return nil, nil
	uncached, err := c.fetchPeerBotCommands(context.Background(), types.Peer{ID: "123"})
	if err != nil {
		t.Errorf("unexpected err: %v", err)
	}
	if len(uncached) != 0 {
		t.Errorf("expected empty cmds after clear cache, got %+v", uncached)
	}
}
