package ui

import (
	"strconv"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	"github.com/kumneger0/cligram/internal/telegram/types"
)

func TestParseTelegramLink_InviteLinks(t *testing.T) {
	testCases := []struct {
		input        string
		expectedHash string
	}{
		{"t.me/+abc123XYZ", "abc123XYZ"},
		{"https://t.me/+abc123XYZ", "abc123XYZ"},
		{"http://t.me/+abc123XYZ", "abc123XYZ"},
		{"telegram.me/+abc123XYZ", "abc123XYZ"},
		{"https://telegram.me/+abc123XYZ", "abc123XYZ"},
		{"www.t.me/+abc123XYZ", "abc123XYZ"},
		{"https://www.t.me/+abc123XYZ", "abc123XYZ"},
		{"https://t.me/joinchat/abc123XYZ", "abc123XYZ"},
		{"t.me/joinchat/abc123XYZ", "abc123XYZ"},
		{"tg://join?invite=abc123XYZ", "abc123XYZ"},
		{"+abc123XYZ", "abc123XYZ"},
		{"https://t.me/+abc123XYZ?start=test#anchor", "abc123XYZ"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			res := ParseTelegramLink(tc.input)
			if res == nil {
				t.Fatalf("expected non-nil result for %q", tc.input)
			}
			if res.Type != types.LinkTypeInvite {
				t.Errorf("expected LinkTypeInvite, got %v", res.Type)
			}
			if res.InviteHash != tc.expectedHash {
				t.Errorf("expected hash %q, got %q", tc.expectedHash, res.InviteHash)
			}
		})
	}
}

func TestParseTelegramLink_InternalChannelLinks(t *testing.T) {
	testCases := []struct {
		input             string
		expectedChannelID int64
		expectedMessageID int
	}{
		{"t.me/c/123456789/42", 123456789, 42},
		{"https://t.me/c/123456789/42", 123456789, 42},
		{"http://telegram.me/c/987654321/999", 987654321, 999},
		{"https://t.me/c/123456789", 123456789, 0},
		{"t.me/c/123456789/", 123456789, 0},
		{"https://t.me/c/123456789/42?thread=123", 123456789, 42},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			res := ParseTelegramLink(tc.input)
			if res == nil {
				t.Fatalf("expected non-nil result for %q", tc.input)
			}
			if res.Type != types.LinkTypeInternalMessage {
				t.Errorf("expected LinkTypeInternalMessage, got %v", res.Type)
			}
			if res.ChannelID != tc.expectedChannelID {
				t.Errorf("expected ChannelID %d, got %d", tc.expectedChannelID, res.ChannelID)
			}
			if res.MessageID != tc.expectedMessageID {
				t.Errorf("expected MessageID %d, got %d", tc.expectedMessageID, res.MessageID)
			}
		})
	}
}

func TestParseTelegramLink_PublicHandlesAndPosts(t *testing.T) {
	testCases := []struct {
		input             string
		expectedUsername  string
		expectedMessageID int
	}{
		{"t.me/durov", "durov", 0},
		{"https://t.me/durov", "durov", 0},
		{"http://telegram.me/durov", "durov", 0},
		{"@durov", "durov", 0},
		{"t.me/durov/42", "durov", 42},
		{"https://t.me/durov/100500", "durov", 100500},
		{"tg://resolve?domain=durov&post=15", "durov", 15},
		{"tg://resolve?domain=telegram", "telegram", 0},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			res := ParseTelegramLink(tc.input)
			if res == nil {
				t.Fatalf("expected non-nil result for %q", tc.input)
			}
			if res.Type != types.LinkTypePublic {
				t.Errorf("expected LinkTypePublic, got %v", res.Type)
			}
			if res.Username != tc.expectedUsername {
				t.Errorf("expected Username %q, got %q", tc.expectedUsername, res.Username)
			}
			if res.MessageID != tc.expectedMessageID {
				t.Errorf("expected MessageID %d, got %d", tc.expectedMessageID, res.MessageID)
			}
		})
	}
}

func TestParseTelegramLink_NonTelegramURLs(t *testing.T) {
	nonTelegram := []string{
		"",
		"   ",
		"https://google.com",
		"http://github.com/gotd/td",
		"just some random text",
		"https://example.com/t.me/fake",
	}

	for _, str := range nonTelegram {
		t.Run(str, func(t *testing.T) {
			res := ParseTelegramLink(str)
			if res != nil {
				t.Errorf("expected nil for non-telegram string %q, got %+v", str, res)
			}
		})
	}
}

func TestExtractTelegramLinks(t *testing.T) {
	text := "Check out https://t.me/+joinGroupNow and also see t.me/durov/42 or message @alice for info!"
	links := ExtractTelegramLinks(text)

	if len(links) != 3 {
		t.Fatalf("expected 3 links extracted, got %d", len(links))
	}

	if links[0].Type != types.LinkTypeInvite || links[0].InviteHash != "joinGroupNow" {
		t.Errorf("unexpected first link: %+v", links[0])
	}
	if links[1].Type != types.LinkTypePublic || links[1].Username != "durov" || links[1].MessageID != 42 {
		t.Errorf("unexpected second link: %+v", links[1])
	}
	if links[2].Type != types.LinkTypePublic || links[2].Username != "alice" {
		t.Errorf("unexpected third link: %+v", links[2])
	}
}

func TestHandleInternalChannelLink_FoundAndNotFound(t *testing.T) {
	ch1 := types.ChannelInfo{
		ID:           "123456789",
		ChannelTitle: "Test Channel",
		IsBroadcast:  true,
	}
	ch2 := types.ChannelInfo{
		ID:           "987654321",
		ChannelTitle: "Test Group",
		IsBroadcast:  false,
	}

	m := Model{
		Mode:     ModeAll,
		Channels: list.New([]list.Item{ch1}, list.NewDefaultDelegate(), 30, 20),
		Groups:   list.New([]list.Item{ch2}, list.NewDefaultDelegate(), 30, 20),
		All:      list.New([]list.Item{ch1, ch2}, list.NewDefaultDelegate(), 30, 20),
		ChatUI:   list.New([]list.Item{}, list.NewDefaultDelegate(), 50, 20),
	}

	// 1. Channel found in Channels
	linkFound := &types.ParsedLink{
		Type:      types.LinkTypeInternalMessage,
		ChannelID: 123456789,
		MessageID: 55,
	}

	cmd := handleInternalChannelLink(&m, linkFound)
	if cmd == nil {
		t.Errorf("expected non-nil cmd on internal link navigation")
	}
	if m.Mode != ModeChannels {
		t.Errorf("expected ModeChannels, got %v", m.Mode)
	}
	if m.SelectedChannel.ID != strconv.FormatInt(linkFound.ChannelID, 10) {
		t.Errorf("expected SelectedChannel ID %q, got %q", "123456789", m.SelectedChannel.ID)
	}

	// 2. Channel not found
	linkNotFound := &types.ParsedLink{
		Type:      types.LinkTypeInternalMessage,
		ChannelID: 999999999,
		MessageID: 1,
	}

	alertCmd := handleInternalChannelLink(&m, linkNotFound)
	if alertCmd == nil {
		t.Errorf("expected alert command when channel is not found")
	}
}
