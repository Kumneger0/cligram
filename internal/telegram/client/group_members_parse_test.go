package client

import (
	"testing"

	"github.com/gotd/td/tg"
	"github.com/kumneger0/cligram/internal/telegram/types"
)

func TestParseChannelParticipants(t *testing.T) {
	// Test nil input
	if members := parseChannelParticipants(nil); len(members) != 0 {
		t.Fatalf("expected empty members for nil input, got %d", len(members))
	}

	participants := &tg.ChannelsChannelParticipants{
		Users: []tg.UserClass{
			&tg.User{ID: 101, FirstName: "Alice", Username: "alice"},
			&tg.User{ID: 102, FirstName: "Bob", Username: "bob"},
			&tg.User{ID: 103, FirstName: "Charlie", Username: "charlie"},
		},
		Participants: []tg.ChannelParticipantClass{
			&tg.ChannelParticipantCreator{UserID: 101, Rank: "Founder"},
			&tg.ChannelParticipantAdmin{UserID: 102, Rank: "Moderator"},
			&tg.ChannelParticipant{UserID: 103},
		},
	}

	members := parseChannelParticipants(participants)
	if len(members) != 3 {
		t.Fatalf("expected 3 members, got %d", len(members))
	}

	// 1. Owner
	if members[0].User.PeerID != "101" || members[0].User.FirstName != "Alice" {
		t.Errorf("expected Alice for member 0, got %+v", members[0].User)
	}
	if members[0].Role != types.MemberRoleOwner {
		t.Errorf("expected MemberRoleOwner for Alice, got %v", members[0].Role)
	}
	if members[0].CustomTitle != "Founder" {
		t.Errorf("expected CustomTitle Founder, got %q", members[0].CustomTitle)
	}

	// 2. Admin
	if members[1].User.PeerID != "102" || members[1].User.FirstName != "Bob" {
		t.Errorf("expected Bob for member 1, got %+v", members[1].User)
	}
	if members[1].Role != types.MemberRoleAdmin {
		t.Errorf("expected MemberRoleAdmin for Bob, got %v", members[1].Role)
	}
	if members[1].CustomTitle != "Moderator" {
		t.Errorf("expected CustomTitle Moderator, got %q", members[1].CustomTitle)
	}

	// 3. Regular member
	if members[2].User.PeerID != "103" || members[2].User.FirstName != "Charlie" {
		t.Errorf("expected Charlie for member 2, got %+v", members[2].User)
	}
	if members[2].Role != types.MemberRoleMember {
		t.Errorf("expected MemberRoleMember for Charlie, got %v", members[2].Role)
	}
}

func TestParseBasicChatParticipants(t *testing.T) {
	// Test nil input
	if members := parseBasicChatParticipants(nil); len(members) != 0 {
		t.Fatalf("expected empty members for nil input, got %d", len(members))
	}

	full := &tg.MessagesChatFull{
		Users: []tg.UserClass{
			&tg.User{ID: 201, FirstName: "Dave"},
			&tg.User{ID: 202, FirstName: "Eve"},
			&tg.User{ID: 203, FirstName: "Frank"},
		},
		FullChat: &tg.ChatFull{
			Participants: &tg.ChatParticipants{
				Participants: []tg.ChatParticipantClass{
					&tg.ChatParticipantCreator{UserID: 201},
					&tg.ChatParticipantAdmin{UserID: 202},
					&tg.ChatParticipant{UserID: 203},
				},
			},
		},
	}

	members := parseBasicChatParticipants(full)
	if len(members) != 3 {
		t.Fatalf("expected 3 members, got %d", len(members))
	}

	// 1. Creator -> Owner
	if members[0].User.PeerID != "201" || members[0].Role != types.MemberRoleOwner {
		t.Errorf("expected Dave as Owner, got %+v", members[0])
	}

	// 2. Admin -> Admin
	if members[1].User.PeerID != "202" || members[1].Role != types.MemberRoleAdmin {
		t.Errorf("expected Eve as Admin, got %+v", members[1])
	}

	// 3. Member -> Member
	if members[2].User.PeerID != "203" || members[2].Role != types.MemberRoleMember {
		t.Errorf("expected Frank as Member, got %+v", members[2])
	}
}
