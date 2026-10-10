package ui

import (
	"errors"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kumneger0/cligram/internal/telegram/types"
)

func TestForeground_GroupMembersModal_LoadingAndEmpty(t *testing.T) {
	fg := &Foreground{}
	fgModel, _ := fg.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	fg = fgModel.(*Foreground)

	peer := types.Peer{ID: "-100123456", AccessHash: "98765", ChatType: types.GroupChat}
	fgModel, _ = fg.Update(OpenModalMsg{
		ModalMode:  ModalModeGroupMembers,
		GroupPeer:  &peer,
		GroupTitle: "Golang Developers",
	})
	fg = fgModel.(*Foreground)

	// Verify loading state
	view := fg.View()
	if !strings.Contains(view, "Golang Developers") && !strings.Contains(view, "Group Members") {
		t.Errorf("expected view to contain group title or 'Group Members', got:\n%s", view)
	}
	if !strings.Contains(view, "Loading") {
		t.Errorf("expected view to contain loading indicator, got:\n%s", view)
	}

	// Simulate GetGroupMembersMsg with 0 members
	fgModel, _ = fg.Update(types.GetGroupMembersMsg{
		PeerID:  "-100123456",
		Members: []types.GroupMemberInfo{},
	})
	fg = fgModel.(*Foreground)

	viewEmpty := fg.View()
	if !strings.Contains(viewEmpty, "No members found") && !strings.Contains(viewEmpty, "0") {
		t.Errorf("expected view to indicate empty members, got:\n%s", viewEmpty)
	}
	if !strings.Contains(viewEmpty, "Navigate") || !strings.Contains(viewEmpty, "Close") {
		t.Errorf("expected footer hints in members view, got:\n%s", viewEmpty)
	}
}

func TestForeground_GroupMembersModal_Rendering_WithBadgesAndPresence(t *testing.T) {
	fg := &Foreground{}
	fgModel, _ := fg.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	fg = fgModel.(*Foreground)

	onlineStr := "online"
	recentlyStr := "last seen recently"

	members := []types.GroupMemberInfo{
		{
			User: types.UserInfo{
				FirstName: "Alice",
				LastName:  "Owner",
				Username:  "alice_owner",
				PeerID:    "101",
				IsOnline:  true,
				LastSeen:  &onlineStr,
			},
			Role: types.MemberRoleOwner,
		},
		{
			User: types.UserInfo{
				FirstName: "Bob",
				LastName:  "Admin",
				Username:  "bob_admin",
				PeerID:    "102",
				IsOnline:  false,
				LastSeen:  &recentlyStr,
			},
			Role:        types.MemberRoleAdmin,
			CustomTitle: "Moderator",
		},
		{
			User: types.UserInfo{
				FirstName: "Charlie",
				Username:  "charlie_member",
				PeerID:    "103",
				IsOnline:  false,
			},
			Role: types.MemberRoleMember,
		},
	}

	peer := types.Peer{ID: "-100123456", ChatType: types.GroupChat}
	fgModel, _ = fg.Update(OpenModalMsg{
		ModalMode:  ModalModeGroupMembers,
		GroupPeer:  &peer,
		GroupTitle: "Gophers",
	})
	fg = fgModel.(*Foreground)

	fgModel, _ = fg.Update(types.GetGroupMembersMsg{
		PeerID:  "-100123456",
		Members: members,
	})
	fg = fgModel.(*Foreground)

	view := fg.View()
	if !strings.Contains(view, "[Owner]") {
		t.Errorf("expected view to contain [Owner] badge, got:\n%s", view)
	}
	if !strings.Contains(view, "[Admin]") {
		t.Errorf("expected view to contain [Admin] badge, got:\n%s", view)
	}
	if !strings.Contains(view, "(Moderator)") {
		t.Errorf("expected view to contain (Moderator) custom title, got:\n%s", view)
	}
	if !strings.Contains(view, "Alice") || !strings.Contains(view, "@alice_owner") {
		t.Errorf("expected view to contain Alice and @alice_owner, got:\n%s", view)
	}
	if !strings.Contains(view, "Bob") || !strings.Contains(view, "@bob_admin") {
		t.Errorf("expected view to contain Bob and @bob_admin, got:\n%s", view)
	}
	if !strings.Contains(view, "Charlie") || !strings.Contains(view, "@charlie_member") {
		t.Errorf("expected view to contain Charlie and @charlie_member, got:\n%s", view)
	}
	if !strings.Contains(view, "Online") && !strings.Contains(view, "online") {
		t.Errorf("expected view to show online status for Alice, got:\n%s", view)
	}
	if !strings.Contains(view, "last seen recently") {
		t.Errorf("expected view to show last seen recently for Bob, got:\n%s", view)
	}
}

func TestForeground_GroupMembersModal_SearchFiltering(t *testing.T) {
	fg := &Foreground{}
	fgModel, _ := fg.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	fg = fgModel.(*Foreground)

	members := []types.GroupMemberInfo{
		{
			User: types.UserInfo{
				FirstName: "Alice",
				Username:  "alice_owner",
				PeerID:    "101",
			},
			Role: types.MemberRoleOwner,
		},
		{
			User: types.UserInfo{
				FirstName: "Bob",
				Username:  "bob_admin",
				PeerID:    "102",
			},
			Role: types.MemberRoleAdmin,
		},
	}

	peer := types.Peer{ID: "-100123456", ChatType: types.GroupChat}
	fgModel, _ = fg.Update(OpenModalMsg{
		ModalMode:  ModalModeGroupMembers,
		GroupPeer:  &peer,
		GroupTitle: "Gophers",
	})
	fg = fgModel.(*Foreground)

	fgModel, _ = fg.Update(types.GetGroupMembersMsg{
		PeerID:  "-100123456",
		Members: members,
	})
	fg = fgModel.(*Foreground)

	// Filter by "bob"
	fgModel, _ = fg.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b', 'o', 'b'}})
	fg = fgModel.(*Foreground)

	view := fg.View()
	if !strings.Contains(view, "Bob") {
		t.Errorf("expected filtered view to contain Bob, got:\n%s", view)
	}
	if strings.Contains(view, "Alice") {
		t.Errorf("expected filtered view NOT to contain Alice, got:\n%s", view)
	}
}

func TestForeground_GroupMembersModal_NavigationAndSelection(t *testing.T) {
	fg := &Foreground{}
	fgModel, _ := fg.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	fg = fgModel.(*Foreground)

	members := []types.GroupMemberInfo{
		{
			User: types.UserInfo{
				FirstName: "Alice",
				Username:  "alice_owner",
				PeerID:    "101",
			},
			Role: types.MemberRoleOwner,
		},
		{
			User: types.UserInfo{
				FirstName: "Bob",
				Username:  "bob_admin",
				PeerID:    "102",
			},
			Role: types.MemberRoleAdmin,
		},
	}

	peer := types.Peer{ID: "-100123456", ChatType: types.GroupChat}
	fgModel, _ = fg.Update(OpenModalMsg{
		ModalMode:  ModalModeGroupMembers,
		GroupPeer:  &peer,
		GroupTitle: "Gophers",
	})
	fg = fgModel.(*Foreground)

	fgModel, _ = fg.Update(types.GetGroupMembersMsg{
		PeerID:  "-100123456",
		Members: members,
	})
	fg = fgModel.(*Foreground)

	// Navigate down to Bob
	fgModel, _ = fg.Update(tea.KeyMsg{Type: tea.KeyDown})
	fg = fgModel.(*Foreground)

	// Press Enter to select Bob
	fgModel, enterCmd := fg.Update(tea.KeyMsg{Type: tea.KeyEnter})
	fg = fgModel.(*Foreground)

	if enterCmd == nil {
		t.Fatal("expected enterCmd on member selection, got nil")
	}

	// Execute cmds to verify CloseOverlay and SelectSearchedUserResult
	var closed bool
	var selectedUser *types.UserInfo

	rawMsg := enterCmd()
	if bMsg, ok := rawMsg.(tea.BatchMsg); ok {
		for _, cmd := range bMsg {
			if cmd == nil {
				continue
			}
			msg := cmd()
			switch m := msg.(type) {
			case CloseOverlay:
				closed = true
			case SelectSearchedUserResult:
				selectedUser = m.user
			}
		}
	}

	if !closed {
		t.Errorf("expected CloseOverlay to be emitted on member selection")
	}
	if selectedUser == nil || selectedUser.FirstName != "Bob" {
		t.Errorf("expected selected user to be Bob, got %+v", selectedUser)
	}
}

func TestForeground_GroupMembersModal_FetchErrorHandling(t *testing.T) {
	fg := &Foreground{}
	fgModel, _ := fg.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	fg = fgModel.(*Foreground)

	peer := types.Peer{ID: "-100123456", ChatType: types.GroupChat}
	fgModel, _ = fg.Update(OpenModalMsg{
		ModalMode:  ModalModeGroupMembers,
		GroupPeer:  &peer,
		GroupTitle: "Secret Group",
	})
	fg = fgModel.(*Foreground)

	// Simulate error
	fgModel, _ = fg.Update(types.GetGroupMembersMsg{
		PeerID: "-100123456",
		Err:    errors.New("CHAT_ADMIN_REQUIRED"),
	})
	fg = fgModel.(*Foreground)

	view := fg.View()
	if !strings.Contains(view, "Failed to fetch members") || !strings.Contains(view, "CHAT_ADMIN_REQUIRED") {
		t.Errorf("expected view to show error message, got:\n%s", view)
	}
}

func TestModel_KeyM_GroupMemberRoster(t *testing.T) {
	m := newTestModel(80, 24)
	m.Mode = ModeGroups
	m.FocusedOn = Main
	group := types.ChannelInfo{
		ID:           "2001",
		ChannelTitle: "Dev Chat",
		AccessHash:   "55555",
	}
	m.SelectedGroup = group
	m.Groups.SetItems([]list.Item{group})
	m.Groups.Select(0)

	// Press 'm' while focused on Main in group
	_, cmd := m.handleMemberRosterKey()
	if cmd == nil {
		t.Fatal("expected cmd when pressing 'm' in group chat, got nil")
	}

	var foundOpenModal bool
	var modalMsg OpenModalMsg

	raw := cmd()
	if bMsg, ok := raw.(tea.BatchMsg); ok {
		for _, c := range bMsg {
			if c == nil {
				continue
			}
			msg := c()
			if om, ok := msg.(OpenModalMsg); ok {
				foundOpenModal = true
				modalMsg = om
			}
		}
	} else if om, ok := raw.(OpenModalMsg); ok {
		foundOpenModal = true
		modalMsg = om
	}

	if !foundOpenModal {
		t.Fatalf("expected OpenModalMsg to be produced, got %+v", raw)
	}
	if modalMsg.ModalMode != ModalModeGroupMembers {
		t.Errorf("expected ModalModeGroupMembers, got %v", modalMsg.ModalMode)
	}
	if modalMsg.GroupTitle != "Dev Chat" {
		t.Errorf("expected GroupTitle 'Dev Chat', got %q", modalMsg.GroupTitle)
	}
	if modalMsg.GroupPeer == nil || modalMsg.GroupPeer.ID != "2001" {
		t.Errorf("expected GroupPeer ID '2001', got %+v", modalMsg.GroupPeer)
	}

	// Press 'm' when focused on Input -> should return nil cmd
	m.FocusedOn = Input
	_, inputCmd := m.handleMemberRosterKey()
	if inputCmd != nil {
		t.Errorf("expected nil cmd when focused on Input, got %+v", inputCmd)
	}

	// Press 'm' on a private User chat -> should return nil cmd
	m.Mode = ModeUsers
	m.FocusedOn = Main
	m.SelectedUser = types.UserInfo{PeerID: "3001", FirstName: "Dave"}
	m.Users.SetItems([]list.Item{m.SelectedUser})
	m.Users.Select(0)
	_, userCmd := m.handleMemberRosterKey()
	if userCmd != nil {
		t.Errorf("expected nil cmd on 1-on-1 user chat, got %+v", userCmd)
	}
}

func TestModel_KeyM_ViaUpdate(t *testing.T) {
	m := newTestModel(80, 24)
	m.Mode = ModeGroups
	m.FocusedOn = Main
	group := types.ChannelInfo{
		ID:           "2002",
		ChannelTitle: "Gophers Group",
		AccessHash:   "66666",
	}
	m.SelectedGroup = group
	m.Groups.SetItems([]list.Item{group})
	m.Groups.Select(0)

	// Send key 'm' through Model.Update
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	if cmd == nil {
		t.Fatal("expected cmd when pressing 'm' via Update, got nil")
	}

	var foundOpenModal bool
	raw := cmd()
	if bMsg, ok := raw.(tea.BatchMsg); ok {
		for _, c := range bMsg {
			if c == nil {
				continue
			}
			msg := c()
			if om, ok := msg.(OpenModalMsg); ok && om.ModalMode == ModalModeGroupMembers {
				foundOpenModal = true
			}
		}
	} else if om, ok := raw.(OpenModalMsg); ok && om.ModalMode == ModalModeGroupMembers {
		foundOpenModal = true
	}

	if !foundOpenModal {
		t.Errorf("expected OpenModalMsg with ModalModeGroupMembers via Model.Update")
	}
}

func TestManager_GroupMemberRoster_Integration(t *testing.T) {
	bg := newTestModel(80, 24)
	bg.Mode = ModeGroups
	bg.FocusedOn = Main
	group := types.ChannelInfo{
		ID:           "5001",
		ChannelTitle: "Dev Team",
		AccessHash:   "99999",
	}
	bg.SelectedGroup = group
	bg.Groups.SetItems([]list.Item{group})
	bg.Groups.Select(0)

	fg := &Foreground{}
	mgr := Manager{
		State:        MainView,
		WindowWidth:  80,
		WindowHeight: 24,
		Foreground:   fg,
		Background:   bg,
	}

	// 1. Press 'm' on Manager while in MainView
	mgrModel, cmd := mgr.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	mgr = mgrModel.(Manager)

	if cmd == nil {
		t.Fatal("expected cmd on pressing 'm' in Manager")
	}

	// Execute cmd from background to trigger OpenModalMsg
	var openModalMsg OpenModalMsg
	raw := cmd()
	if bMsg, ok := raw.(tea.BatchMsg); ok {
		for _, c := range bMsg {
			if c == nil {
				continue
			}
			msg := c()
			if om, ok := msg.(OpenModalMsg); ok {
				openModalMsg = om
			}
		}
	} else if om, ok := raw.(OpenModalMsg); ok {
		openModalMsg = om
	}

	if openModalMsg.ModalMode != ModalModeGroupMembers {
		t.Fatalf("expected OpenModalMsg with ModalModeGroupMembers, got %+v", openModalMsg)
	}

	// Feed OpenModalMsg to Manager
	mgrModel, _ = mgr.Update(openModalMsg)
	mgr = mgrModel.(Manager)

	if mgr.State != ModalView {
		t.Errorf("expected Manager state to be ModalView, got %v", mgr.State)
	}

	// Feed GetGroupMembersMsg to Manager
	members := []types.GroupMemberInfo{
		{
			User: types.UserInfo{
				FirstName: "Elena",
				Username:  "elena_lead",
				PeerID:    "6001",
				IsOnline:  true,
			},
			Role: types.MemberRoleOwner,
		},
	}
	mgrModel, _ = mgr.Update(types.GetGroupMembersMsg{
		PeerID:  "5001",
		Members: members,
	})
	mgr = mgrModel.(Manager)

	// Verify view contains Elena
	view := mgr.View()
	if !strings.Contains(view, "Elena") || !strings.Contains(view, "[Owner]") {
		t.Errorf("expected modal composite to show Elena and [Owner], got:\n%s", view)
	}

	// Press Enter to select member
	mgrModel, enterCmd := mgr.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mgr = mgrModel.(Manager)

	if enterCmd == nil {
		t.Fatal("expected enterCmd from member selection")
	}

	// Execute batch commands (CloseOverlay and SelectSearchedUserResult)
	var closeMsg CloseOverlay
	var selectResult SelectSearchedUserResult
	rawEnter := enterCmd()
	if bMsg, ok := rawEnter.(tea.BatchMsg); ok {
		for _, c := range bMsg {
			if c == nil {
				continue
			}
			msg := c()
			switch m := msg.(type) {
			case CloseOverlay:
				closeMsg = m
			case SelectSearchedUserResult:
				selectResult = m
			}
		}
	}

	// Feed CloseOverlay to Manager
	mgrModel, _ = mgr.Update(closeMsg)
	mgr = mgrModel.(Manager)
	if mgr.State != MainView {
		t.Errorf("expected Manager to return to MainView after CloseOverlay, got %v", mgr.State)
	}

	// Feed SelectSearchedUserResult to Manager -> background switches to user chat
	mgrModel, _ = mgr.Update(selectResult)
	mgr = mgrModel.(Manager)

	bgModel := mgr.Background.(Model)
	if bgModel.SelectedUser.PeerID != "6001" {
		t.Errorf("expected background selected user to be 6001 (Elena), got %+v", bgModel.SelectedUser)
	}
	if bgModel.Mode != ModeUsers {
		t.Errorf("expected background mode to switch to ModeUsers, got %v", bgModel.Mode)
	}
}

func TestForeground_GroupMembersModal_TabToggleAndJKNavigation(t *testing.T) {
	fg := &Foreground{}
	fgModel, _ := fg.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	fg = fgModel.(*Foreground)

	members := []types.GroupMemberInfo{
		{User: types.UserInfo{FirstName: "Alice", PeerID: "101"}, Role: types.MemberRoleOwner},
		{User: types.UserInfo{FirstName: "Bob", PeerID: "102"}, Role: types.MemberRoleAdmin},
		{User: types.UserInfo{FirstName: "Charlie", PeerID: "103"}, Role: types.MemberRoleMember},
	}

	peer := types.Peer{ID: "-100123456", ChatType: types.GroupChat}
	fgModel, _ = fg.Update(OpenModalMsg{
		ModalMode:  ModalModeGroupMembers,
		GroupPeer:  &peer,
		GroupTitle: "Gophers",
	})
	fgModel, _ = fgModel.Update(types.GetGroupMembersMsg{
		PeerID:  "-100123456",
		Members: members,
	})
	fg = fgModel.(*Foreground)

	if fg.focusedOn != SEARCH {
		t.Fatalf("expected initial focus to be SEARCH, got %v", fg.focusedOn)
	}

	// Press Tab -> switches focus to LIST
	fgModel, _ = fg.Update(tea.KeyMsg{Type: tea.KeyTab})
	fg = fgModel.(*Foreground)
	if fg.focusedOn != LIST {
		t.Fatalf("expected focus to switch to LIST after Tab, got %v", fg.focusedOn)
	}

	// Navigate with 'j' (down to Bob at index 1)
	fgModel, _ = fg.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	fg = fgModel.(*Foreground)
	if fg.membersList.Index() != 1 {
		t.Errorf("expected index 1 after pressing 'j', got %d", fg.membersList.Index())
	}

	// Navigate with 'j' (down to Charlie at index 2)
	fgModel, _ = fg.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	fg = fgModel.(*Foreground)
	if fg.membersList.Index() != 2 {
		t.Errorf("expected index 2 after pressing 'j', got %d", fg.membersList.Index())
	}

	// Navigate with 'k' (up to Bob at index 1)
	fgModel, _ = fg.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	fg = fgModel.(*Foreground)
	if fg.membersList.Index() != 1 {
		t.Errorf("expected index 1 after pressing 'k', got %d", fg.membersList.Index())
	}

	// Press Tab -> switches focus back to SEARCH
	fgModel, _ = fg.Update(tea.KeyMsg{Type: tea.KeyTab})
	fg = fgModel.(*Foreground)
	if fg.focusedOn != SEARCH {
		t.Fatalf("expected focus to switch back to SEARCH after Tab, got %v", fg.focusedOn)
	}
}

func TestForeground_GroupMembersModal_Pagination(t *testing.T) {
	fg := &Foreground{}
	fgModel, _ := fg.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	fg = fgModel.(*Foreground)

	peer := types.Peer{ID: "-100123456", AccessHash: "98765", ChatType: types.ChannelChat}
	fgModel, _ = fg.Update(OpenModalMsg{
		ModalMode:  ModalModeGroupMembers,
		GroupPeer:  &peer,
		GroupTitle: "Mega Group",
	})
	fg = fgModel.(*Foreground)

	if !fg.hasMoreMembers {
		t.Fatal("expected hasMoreMembers to be true initially")
	}
	if fg.membersOffset != 0 {
		t.Fatalf("expected initial membersOffset 0, got %d", fg.membersOffset)
	}

	// 1. First batch: 200 members
	firstBatch := make([]types.GroupMemberInfo, 200)
	for i := 0; i < 200; i++ {
		firstBatch[i] = types.GroupMemberInfo{
			User: types.UserInfo{
				FirstName: "User",
				PeerID:    string(rune('A' + (i % 26))),
			},
			Role: types.MemberRoleMember,
		}
	}

	fgModel, _ = fg.Update(types.GetGroupMembersMsg{
		PeerID:  "-100123456",
		Offset:  0,
		Members: firstBatch,
	})
	fg = fgModel.(*Foreground)

	if len(fg.allMembers) != 200 {
		t.Fatalf("expected 200 members in allMembers, got %d", len(fg.allMembers))
	}
	if fg.membersOffset != 200 {
		t.Fatalf("expected membersOffset 200, got %d", fg.membersOffset)
	}
	if !fg.hasMoreMembers {
		t.Fatal("expected hasMoreMembers to be true when 200 items received")
	}

	// 2. Tab to LIST, move to near end (e.g., select index 196)
	fgModel, _ = fg.Update(tea.KeyMsg{Type: tea.KeyTab})
	fg = fgModel.(*Foreground)
	fg.membersList.Select(196)

	// Press 'j' to navigate down -> should trigger pagination
	fgModel, _ = fg.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	fg = fgModel.(*Foreground)

	if !fg.isPaginatingMembers {
		t.Fatal("expected isPaginatingMembers to be true after scrolling near bottom")
	}

	// Verify View reflects loading state
	viewLoadingMore := fg.View()
	if !strings.Contains(viewLoadingMore, "Loading") {
		t.Errorf("expected view to contain 'Loading', got:\n%s", viewLoadingMore)
	}

	// 3. Second batch: 50 members (final page, < 200)
	secondBatch := make([]types.GroupMemberInfo, 50)
	for i := 0; i < 50; i++ {
		secondBatch[i] = types.GroupMemberInfo{
			User: types.UserInfo{
				FirstName: "ExtraUser",
				PeerID:    string(rune('a' + (i % 26))),
			},
			Role: types.MemberRoleMember,
		}
	}

	fgModel, _ = fg.Update(types.GetGroupMembersMsg{
		PeerID:  "-100123456",
		Offset:  200,
		Members: secondBatch,
	})
	fg = fgModel.(*Foreground)

	if len(fg.allMembers) != 250 {
		t.Fatalf("expected 250 total members after pagination, got %d", len(fg.allMembers))
	}
	if fg.membersOffset != 250 {
		t.Fatalf("expected membersOffset 250, got %d", fg.membersOffset)
	}
	if fg.hasMoreMembers {
		t.Fatal("expected hasMoreMembers to be false when batch size < 200")
	}
	if fg.isPaginatingMembers {
		t.Fatal("expected isPaginatingMembers to be false after batch completes")
	}

	// Verify view shows updated count 250
	viewAfter := fg.View()
	if !strings.Contains(viewAfter, "250") {
		t.Errorf("expected view to show 250 members, got:\n%s", viewAfter)
	}

	// 4. Test error handling during pagination (offset > 0)
	fgModel, _ = fg.Update(types.GetGroupMembersMsg{
		PeerID: "-100123456",
		Offset: 250,
		Err:    errors.New("network timeout"),
	})
	fg = fgModel.(*Foreground)

	// Existing 250 members should NOT be wiped out
	if len(fg.allMembers) != 250 {
		t.Fatalf("expected allMembers to retain 250 items on pagination error, got %d", len(fg.allMembers))
	}
	if fg.groupMembersError == nil {
		t.Fatal("expected groupMembersError to be populated for pagination errors")
	}
	viewErr := fg.View()
	if !strings.Contains(viewErr, "failed to load more members") {
		t.Errorf("expected view to display pagination error message, got:\n%s", viewErr)
	}
}

func TestForeground_GroupMembersModal_TotalMembersDisplay(t *testing.T) {
	fg := &Foreground{}
	fgModel, _ := fg.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	fg = fgModel.(*Foreground)

	peer := types.Peer{ID: "-100123456", ChatType: types.ChannelChat}
	fgModel, _ = fg.Update(OpenModalMsg{
		ModalMode:    ModalModeGroupMembers,
		GroupPeer:    &peer,
		GroupTitle:   "Super Devs",
		TotalMembers: 10904,
	})
	fg = fgModel.(*Foreground)

	// Even before members arrive, total member count (10904) is displayed
	view := fg.View()
	if !strings.Contains(view, "10904") {
		t.Errorf("expected view to display total members count (10904), got:\n%s", view)
	}

	// First batch of 200 arrives with MTProto filtered participants TotalCount 10851
	batch := make([]types.GroupMemberInfo, 200)
	for i := 0; i < 200; i++ {
		batch[i] = types.GroupMemberInfo{
			User: types.UserInfo{FirstName: "User", PeerID: "1"},
			Role: types.MemberRoleMember,
		}
	}
	fgModel, _ = fg.Update(types.GetGroupMembersMsg{
		PeerID:     "-100123456",
		Offset:     0,
		Members:    batch,
		TotalCount: 10851, // Telegram ChannelsGetParticipants excludes deleted accounts
	})
	fg = fgModel.(*Foreground)

	// Header preserves official total members (10904) rather than downgrading to 10851
	viewAfterBatch := fg.View()
	if !strings.Contains(viewAfterBatch, "10904") {
		t.Errorf("expected view to preserve 10904 total members, got:\n%s", viewAfterBatch)
	}
	if strings.Contains(viewAfterBatch, "10851") {
		t.Errorf("view should not downgrade official count to 10851, got:\n%s", viewAfterBatch)
	}

	// Remote search returns 2 matches with TotalCount 2; totalMembers should not be overwritten
	fgModel, _ = fg.Update(types.SearchGroupMembersMsg{
		PeerID:     "-100123456",
		Query:      "",
		Members:    batch[:2],
		TotalCount: 2,
	})
	fg = fgModel.(*Foreground)
	if fg.totalMembers != 10904 {
		t.Errorf("expected totalMembers to remain 10904 after search msg, got %d", fg.totalMembers)
	}

	// Fallback case: if TotalMembers was unknown (0), it populates from GetGroupMembersMsg
	fg2 := &Foreground{}
	fgModel2, _ := fg2.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	fg2 = fgModel2.(*Foreground)
	fgModel2, _ = fg2.Update(OpenModalMsg{
		ModalMode:    ModalModeGroupMembers,
		GroupPeer:    &peer,
		GroupTitle:   "Super Devs",
		TotalMembers: 0,
	})
	fg2 = fgModel2.(*Foreground)
	fgModel2, _ = fg2.Update(types.GetGroupMembersMsg{
		PeerID:     "-100123456",
		Offset:     0,
		Members:    batch[:10],
		TotalCount: 500,
	})
	fg2 = fgModel2.(*Foreground)
	if fg2.totalMembers != 500 {
		t.Errorf("expected totalMembers to be populated as 500 when previously 0, got %d", fg2.totalMembers)
	}
}

func TestForeground_GroupMembersModal_RemoteSearch(t *testing.T) {
	fg := &Foreground{}
	fgModel, _ := fg.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	fg = fgModel.(*Foreground)

	peer := types.Peer{ID: "-100123456", ChatType: types.ChannelChat}
	fgModel, _ = fg.Update(OpenModalMsg{
		ModalMode:    ModalModeGroupMembers,
		GroupPeer:    &peer,
		GroupTitle:   "Super Devs",
		TotalMembers: 5000,
	})
	fg = fgModel.(*Foreground)

	// Initial batch: Alice and Bob only (Zoe is NOT in the initial 200 batch)
	initialMembers := []types.GroupMemberInfo{
		{User: types.UserInfo{FirstName: "Alice", PeerID: "101"}, Role: types.MemberRoleOwner},
		{User: types.UserInfo{FirstName: "Bob", PeerID: "102"}, Role: types.MemberRoleAdmin},
	}
	fgModel, _ = fg.Update(types.GetGroupMembersMsg{
		PeerID:     "-100123456",
		Offset:     0,
		Members:    initialMembers,
		TotalCount: 5000,
	})
	fg = fgModel.(*Foreground)

	// User types "Zoe"
	for _, r := range "Zoe" {
		fgModel, _ = fg.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		fg = fgModel.(*Foreground)
	}

	// In-memory filter shows 0 found initially
	if len(fg.membersList.Items()) != 0 {
		t.Fatalf("expected 0 in-memory items for 'Zoe', got %d", len(fg.membersList.Items()))
	}

	// Trigger search tick for the latest sequence
	fgModel, _ = fg.Update(MemberSearchTickMsg{
		Peer:  peer,
		Query: "Zoe",
		Seq:   fg.membersSearchSeq,
	})
	fg = fgModel.(*Foreground)
	if !fg.isSearchingMembers {
		t.Fatal("expected isSearchingMembers to be true after search tick")
	}

	// Remote Telegram search returns Zoe from the full 5000 member group
	zoeMember := types.GroupMemberInfo{
		User: types.UserInfo{
			FirstName: "Zoe",
			LastName:  "Engineer",
			Username:  "zoe_lead",
			PeerID:    "9999",
		},
		Role: types.MemberRoleMember,
	}
	fgModel, _ = fg.Update(types.SearchGroupMembersMsg{
		PeerID:     "-100123456",
		Query:      "Zoe",
		Members:    []types.GroupMemberInfo{zoeMember},
		TotalCount: 5000,
	})
	fg = fgModel.(*Foreground)

	if fg.isSearchingMembers {
		t.Fatal("expected isSearchingMembers to be false after search completes")
	}
	if len(fg.membersList.Items()) != 1 {
		t.Fatalf("expected 1 item in membersList after remote search, got %d", len(fg.membersList.Items()))
	}

	// View shows Zoe and total members context
	viewZoe := fg.View()
	if !strings.Contains(viewZoe, "Zoe") || !strings.Contains(viewZoe, "@zoe_lead") {
		t.Errorf("expected view to contain Zoe and @zoe_lead, got:\n%s", viewZoe)
	}
	if !strings.Contains(viewZoe, "5000") {
		t.Errorf("expected view to show total members (5000), got:\n%s", viewZoe)
	}
}
