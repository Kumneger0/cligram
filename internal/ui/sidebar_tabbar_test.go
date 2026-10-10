package ui

import (
	"context"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	"github.com/kumneger0/cligram/internal/telegram"
	"github.com/kumneger0/cligram/internal/telegram/client"
	"github.com/kumneger0/cligram/internal/telegram/types"
)

func TestSidebarTabBar_Rendering(t *testing.T) {
	m := newTestModel(100, 30)
	m.Mode = ModeAll

	// 5 tabs present
	tabBar := m.renderSidebarTabBar(80)
	if !strings.Contains(tabBar, "[a] All") {
		t.Errorf("expected [a] All in tab bar, got: %s", tabBar)
	}
	if !strings.Contains(tabBar, "[u] Chats") {
		t.Errorf("expected [u] Chats in tab bar, got: %s", tabBar)
	}
	if !strings.Contains(tabBar, "[g] Groups") {
		t.Errorf("expected [g] Groups in tab bar, got: %s", tabBar)
	}
	if !strings.Contains(tabBar, "[c] Channels") {
		t.Errorf("expected [c] Channels in tab bar, got: %s", tabBar)
	}
	if !strings.Contains(tabBar, "[b] Bots") {
		t.Errorf("expected [b] Bots in tab bar, got: %s", tabBar)
	}

	// Active category highlight check: [a] All is rendered with activeTabStyle
	activeAll := activeTabStyle.Render("[a] All")
	if !strings.Contains(tabBar, activeAll) {
		t.Errorf("expected active tab style on [a] All, got: %s", tabBar)
	}

	// Inactive tab check: inactive tab is rendered with inactiveTabStyle
	inactiveChats := inactiveTabStyle.Render("[u] Chats")
	if !strings.Contains(tabBar, inactiveChats) {
		t.Errorf("expected inactive tab style on [u] Chats, got: %s", tabBar)
	}

	// Switch mode to Channels
	m.Mode = ModeChannels
	tabBarChannels := m.renderSidebarTabBar(80)
	activeChannels := activeTabStyle.Render("[c] Channels")
	if !strings.Contains(tabBarChannels, activeChannels) {
		t.Errorf("expected active tab style on [c] Channels when in ModeChannels, got: %s", tabBarChannels)
	}
}

func TestSidebarTabBar_UnreadCountBadges(t *testing.T) {
	m := newTestModel(100, 30)

	// Add users with unread counts
	userItems := []list.Item{
		types.UserInfo{PeerID: "101", FirstName: "Alice", UnreadCount: 3},
		types.UserInfo{PeerID: "102", FirstName: "Bob", UnreadCount: 2},
	}
	m.Users.SetItems(userItems)

	// Add channel with unread
	channelItems := []list.Item{
		types.ChannelInfo{ID: "201", ChannelTitle: "Tech News", UnreadCount: 12, IsBroadcast: true},
	}
	m.Channels.SetItems(channelItems)

	// Populate All list with all unread items
	m.All.SetItems([]list.Item{userItems[0], userItems[1], channelItems[0]})

	// Total unread for All should be 17
	unreadAll := m.categoryUnreadCount(ModeAll)
	if unreadAll != 17 {
		t.Fatalf("expected 17 unread for ModeAll, got %d", unreadAll)
	}

	unreadUsers := m.categoryUnreadCount(ModeUsers)
	if unreadUsers != 5 {
		t.Fatalf("expected 5 unread for ModeUsers, got %d", unreadUsers)
	}

	unreadChannels := m.categoryUnreadCount(ModeChannels)
	if unreadChannels != 12 {
		t.Fatalf("expected 12 unread for ModeChannels, got %d", unreadChannels)
	}

	tabBar := m.renderSidebarTabBar(100)
	if !strings.Contains(tabBar, "[a] All (17)") {
		t.Errorf("expected [a] All (17) in tab bar, got: %s", tabBar)
	}
	if !strings.Contains(tabBar, "[u] Chats (5)") {
		t.Errorf("expected [u] Chats (5) in tab bar, got: %s", tabBar)
	}
	if !strings.Contains(tabBar, "[c] Channels (12)") {
		t.Errorf("expected [c] Channels (12) in tab bar, got: %s", tabBar)
	}

	// Groups has 0 unread: should NOT show (0)
	if strings.Contains(tabBar, "[g] Groups (0)") {
		t.Errorf("expected no (0) badge for zero unread, got: %s", tabBar)
	}
}

func TestSidebarTabBar_ResponsiveWrapping(t *testing.T) {
	m := newTestModel(80, 24)

	// Wide width: fits on single line
	wide := m.renderSidebarTabBar(120)
	if strings.Contains(wide, "\n") {
		t.Errorf("expected single line tab bar for width=120, got: %q", wide)
	}

	// Narrow width: should wrap into multiple rows
	narrow := m.renderSidebarTabBar(30)
	if !strings.Contains(narrow, "\n") {
		t.Errorf("expected multiline tab bar for narrow width=30, got: %q", narrow)
	}
}

func TestThresholdBackfill_CategorySwitch(t *testing.T) {
	oldCligram := telegram.Cligram
	telegram.Cligram = &client.Client{}
	defer func() { telegram.Cligram = oldCligram }()

	m := newTestModel(80, 24)
	m.FocusedOn = SideBar
	m.OffsetDate = 1000
	m.OffsetID = 200

	// Case 1: Category has 0 items (< 10 threshold), should trigger backfill of 100 items
	resChannels, cmdChannels := changeSideBarMode(&m, "c")
	if resChannels.Mode != ModeChannels {
		t.Errorf("expected ModeChannels, got %v", resChannels.Mode)
	}
	if !resChannels.ChannelsLoading {
		t.Errorf("expected ChannelsLoading to be true when count < 10")
	}
	if cmdChannels == nil {
		t.Fatalf("expected backfill fetch command on switch to Channels with 0 items")
	}

	// Case 2: Category has >= 10 items, should NOT trigger backfill
	items10 := make([]list.Item, 10)
	for i := 0; i < 10; i++ {
		items10[i] = types.ChannelInfo{ID: string(rune('1' + i)), ChannelTitle: "Group"}
	}
	m.Groups.SetItems(items10)

	resGroups, cmdGroups := changeSideBarMode(&m, "g")
	if resGroups.Mode != ModeGroups {
		t.Errorf("expected ModeGroups, got %v", resGroups.Mode)
	}
	if resGroups.GroupsLoading {
		t.Errorf("expected GroupsLoading to be false when count >= 10")
	}
	if cmdGroups != nil {
		t.Errorf("expected nil cmd when category has >= 10 items, got non-nil")
	}

	// Case 3: Stream reached the end (OffsetDate == -1 && OffsetID == -1), should NOT fetch even if < 10
	m.OffsetDate = -1
	m.OffsetID = -1
	resBots, cmdBots := changeSideBarMode(&m, "b")
	if resBots.Mode != ModeBots {
		t.Errorf("expected ModeBots, got %v", resBots.Mode)
	}
	if resBots.BotsLoading {
		t.Errorf("expected BotsLoading to remain false when offset is -1")
	}
	if cmdBots != nil {
		t.Errorf("expected nil cmd when offset is -1, got non-nil")
	}
}

func TestScrollTriggeredPagination(t *testing.T) {
	oldCligram := telegram.Cligram
	telegram.Cligram = &client.Client{}
	defer func() { telegram.Cligram = oldCligram }()

	m := newTestModel(80, 24)
	m.FocusedOn = SideBar
	m.Mode = ModeAll
	m.OffsetDate = 2000
	m.OffsetID = 100

	// Create 20 items in All
	items := make([]list.Item, 20)
	for i := 0; i < 20; i++ {
		items[i] = types.UserInfo{
			PeerID:    string(rune('A' + i)),
			FirstName: "User",
		}
	}
	m.All.SetItems(items)

	// Case 1: At top of the list (Index = 0), should NOT trigger pagination
	m.All.Select(0)
	mRes, cmd := m.handleListPagination()
	if cmd != nil {
		t.Errorf("expected nil cmd when at index 0 of 20, got non-nil")
	}
	if mRes.OnPagination {
		t.Errorf("expected OnPagination to be false when at index 0")
	}

	// Case 2: Near bottom of the list (Index = 15, which is >= 20-6 = 14)
	m.All.Select(15)
	mRes, cmd = m.handleListPagination()
	if cmd == nil {
		t.Fatalf("expected pagination command when near bottom (index 15 of 20)")
	}
	if !mRes.OnPagination {
		t.Errorf("expected OnPagination to be true after pagination trigger")
	}
	if !mRes.AllLoading {
		t.Errorf("expected AllLoading to be true after pagination trigger")
	}

	// Case 3: Already paginating, subsequent calls should not retrigger
	mRes2, cmd2 := mRes.handleListPagination()
	if cmd2 != nil {
		t.Errorf("expected nil cmd when already on pagination")
	}
	_ = mRes2

	// Case 4: End of list reached (offset is -1)
	m.OnPagination = false
	m.AllLoading = false
	m.OffsetDate = -1
	m.OffsetID = -1
	mResEnd, cmdEnd := m.handleListPagination()
	if cmdEnd != nil {
		t.Errorf("expected nil cmd when end of list reached (offset=-1)")
	}
	if mResEnd.OnPagination {
		t.Errorf("expected OnPagination to remain false when offset is -1")
	}
}

func TestPagination_Deduplication(t *testing.T) {
	m := newTestModel(80, 24)
	initialItems := []list.Item{
		types.UserInfo{PeerID: "user-1", FirstName: "Alice"},
		types.UserInfo{PeerID: "user-2", FirstName: "Bob"},
	}
	m.Users.SetItems(initialItems)

	// Simulate receiving page with 1 duplicate (Bob) and 1 new (Charlie)
	newItems := []types.UserInfo{
		{PeerID: "user-2", FirstName: "Bob"},
		{PeerID: "user-3", FirstName: "Charlie"},
	}

	appendListItems(&m.Users, newItems)

	items := m.Users.Items()
	if len(items) != 3 {
		t.Fatalf("expected 3 items after deduplicated append, got %d", len(items))
	}
	if items[0].FilterValue() != "Alice" || items[1].FilterValue() != "Bob" || items[2].FilterValue() != "Charlie" {
		t.Errorf("unexpected items order/content: %v", items)
	}
}

func TestDialogIngestion_UnifiedStreamPartitioning(t *testing.T) {
	m := newTestModel(80, 24)
	m.SideBarLoading = true

	user := types.UserInfo{PeerID: "u1", FirstName: "User1"}
	bot := types.UserInfo{PeerID: "b1", FirstName: "Bot1", IsBot: true}
	group := types.ChannelInfo{ID: "g1", ChannelTitle: "Group1", IsBroadcast: false}
	channel := types.ChannelInfo{ID: "c1", ChannelTitle: "Channel1", IsBroadcast: true}

	allMsg := types.GetAllChatsResponseMSG{
		Chats: types.AllChats{
			All:          []types.FilterableItem{user, bot, group, channel},
			PrivateChats: []types.UserInfo{user},
			Bots:         []types.UserInfo{bot},
			Groups:       []types.ChannelInfo{group},
			Channels:     []types.ChannelInfo{channel},
			OffsetDate:   500,
			OffsetID:     120,
		},
	}

	resModel, _ := m.Update(allMsg)
	m = resModel.(Model)

	if m.SideBarLoading {
		t.Errorf("expected SideBarLoading to be false")
	}
	if !m.AllLoaded || !m.UsersLoaded || !m.BotsLoaded || !m.ChannelsLoaded || !m.GroupsLoaded {
		t.Errorf("expected all category Loaded flags to be true")
	}
	if m.OffsetDate != 500 || m.OffsetID != 120 {
		t.Errorf("expected global offsets (500, 120), got (%d, %d)", m.OffsetDate, m.OffsetID)
	}
	if len(m.All.Items()) != 4 {
		t.Errorf("expected 4 items in All list, got %d", len(m.All.Items()))
	}
	if len(m.Users.Items()) != 1 {
		t.Errorf("expected 1 item in Users list, got %d", len(m.Users.Items()))
	}
	if len(m.Bots.Items()) != 1 {
		t.Errorf("expected 1 item in Bots list, got %d", len(m.Bots.Items()))
	}
	if len(m.Groups.Items()) != 1 {
		t.Errorf("expected 1 item in Groups list, got %d", len(m.Groups.Items()))
	}
	if len(m.Channels.Items()) != 1 {
		t.Errorf("expected 1 item in Channels list, got %d", len(m.Channels.Items()))
	}
}

func TestStartupInitialIngestion(t *testing.T) {
	oldCligram := telegram.Cligram
	telegram.Cligram = &client.Client{}
	defer func() { telegram.Cligram = oldCligram }()

	m := newTestModel(80, 24)

	cmd := m.Init()
	if cmd == nil {
		t.Fatalf("expected Init() to return a batch command")
	}

	// Init dispatches a batch containing GetUserChatsCmd(..., 100)
	batchMsg := cmd()
	if batchMsg == nil {
		t.Logf("batchMsg produced")
	}
	_ = context.Background()
}

func TestSidebarTabBar_LoadingSpinner(t *testing.T) {
	m := newTestModel(80, 24)
	m.ensureSidebarSpinner()
	m.ChannelsLoading = true

	tabBar := m.renderSidebarTabBar(80)
	spinFrame := m.SidebarSpinner.View()
	if !strings.Contains(tabBar, spinFrame) {
		t.Errorf("expected spinner frame %q in tab bar when channels are loading, got: %s", spinFrame, tabBar)
	}
}

func TestSidebar_LoadingContent(t *testing.T) {
	m := newTestModel(80, 24)
	m.Mode = ModeChannels
	m.ChannelsLoading = true

	d := layoutDimensions{
		sidebarWidth:  24,
		mainWidth:     56,
		contentHeight: 20,
		inputHeight:   4,
	}

	content := prepareSidebarContent(&m, d)
	if !strings.Contains(content, "Loading") {
		t.Errorf("expected 'Loading' in sidebar loading content, got: %s", content)
	}
}

func TestCustomDelegate_Rendering_ListUI(t *testing.T) {
	m := newTestModel(80, 24)
	m.FocusedOn = SideBar
	m.Users.SetWidth(30)
	m.Channels.SetWidth(30)

	delegate := CustomDelegate{Model: &m}
	user := types.UserInfo{
		PeerID:      "101",
		FirstName:   "Alice",
		IsOnline:    true,
		UnreadCount: 4,
	}

	m.Users.SetItems([]list.Item{user})
	m.Users.Select(0)

	var buf strings.Builder
	delegate.Render(&buf, m.Users, 0, user)
	rendered := buf.String()

	// Should have active indicator '▎'
	if !strings.Contains(rendered, "▎") {
		t.Errorf("expected left indicator '▎' for selected focused item, got: %s", rendered)
	}
	// Should have online indicator '🟢'
	if !strings.Contains(rendered, "🟢") {
		t.Errorf("expected online icon '🟢', got: %s", rendered)
	}
	// Should have title
	if !strings.Contains(rendered, "Alice") {
		t.Errorf("expected 'Alice' in rendered output, got: %s", rendered)
	}
	// Should have unread badge '4'
	if !strings.Contains(rendered, "4") {
		t.Errorf("expected unread count '4' in rendered output, got: %s", rendered)
	}

	// Test Bot icon
	bot := types.UserInfo{
		PeerID:    "102",
		FirstName: "ChatGPTBot",
		IsBot:     true,
	}
	buf.Reset()
	delegate.Render(&buf, m.Users, 1, bot)
	if !strings.Contains(buf.String(), "🤖") {
		t.Errorf("expected bot icon '🤖', got: %s", buf.String())
	}

	// Test Channel broadcast icon
	channel := types.ChannelInfo{
		ID:           "201",
		ChannelTitle: "News",
		IsBroadcast:  true,
	}
	buf.Reset()
	delegate.Render(&buf, m.Channels, 1, channel)
	if !strings.Contains(buf.String(), "📢") {
		t.Errorf("expected broadcast icon '📢', got: %s", buf.String())
	}
}

func TestAppendListItems_DeduplicationByID(t *testing.T) {
	l := list.New([]list.Item{}, list.DefaultDelegate{}, 20, 20)

	// Two different users with identical first name "Alex"
	user1 := types.UserInfo{PeerID: "101", FirstName: "Alex"}
	user2 := types.UserInfo{PeerID: "102", FirstName: "Alex"}
	// User with empty first name
	user3 := types.UserInfo{PeerID: "103", FirstName: ""}
	// Channel with title "Updates"
	ch := types.ChannelInfo{ID: "500", ChannelTitle: "Updates", IsBroadcast: true}
	// Group with identical title "Updates"
	gr := types.ChannelInfo{ID: "600", ChannelTitle: "Updates", IsBroadcast: false}

	// First batch
	appendListItems(&l, []types.UserInfo{user1, user2, user3})
	if len(l.Items()) != 3 {
		t.Fatalf("expected 3 items, got %d", len(l.Items()))
	}

	// Appending duplicate of user1 should not increase item count
	appendListItems(&l, []types.UserInfo{user1})
	if len(l.Items()) != 3 {
		t.Errorf("expected duplicate user1 to be ignored, got %d items", len(l.Items()))
	}

	// Appending distinct channel and group with same title to an All list
	lAll := list.New([]list.Item{}, list.DefaultDelegate{}, 20, 20)
	appendListItems(&lAll, []types.FilterableItem{ch, gr})
	if len(lAll.Items()) != 2 {
		t.Errorf("expected both channel and group with same title to be kept, got %d items", len(lAll.Items()))
	}
}

func TestSidebarTabBar_LoadingWithUnreadBadge(t *testing.T) {
	m := newTestModel(80, 24)
	m.ensureSidebarSpinner()
	m.ChannelsLoading = true
	m.Channels.SetItems([]list.Item{
		types.ChannelInfo{ID: "1", ChannelTitle: "Announcements", UnreadCount: 5, IsBroadcast: true},
	})

	tabBar := m.renderSidebarTabBar(80)
	spinFrame := m.SidebarSpinner.View()
	// Should contain both unread count "(5)" and spinner frame
	if !strings.Contains(tabBar, "(5)") {
		t.Errorf("expected '(5)' in tab bar while loading, got: %s", tabBar)
	}
	if !strings.Contains(tabBar, spinFrame) {
		t.Errorf("expected spinner frame %q in tab bar alongside unread count, got: %s", spinFrame, tabBar)
	}
}

func TestSidebarTabBar_DynamicBadgeUpdates(t *testing.T) {
	m := newTestModel(80, 24)
	user := types.UserInfo{PeerID: "1", FirstName: "User1", UnreadCount: 0}
	m.Users.SetItems([]list.Item{user})

	tabBarBefore := m.renderSidebarTabBar(80)
	if strings.Contains(tabBarBefore, "Chats (") {
		t.Errorf("expected no unread count initially, got: %s", tabBarBefore)
	}

	// Dynamic update: simulate new unread messages
	user.UnreadCount = 7
	m.Users.SetItem(0, user)

	tabBarAfter := m.renderSidebarTabBar(80)
	if !strings.Contains(tabBarAfter, "Chats (7)") {
		t.Errorf("expected updated 'Chats (7)' badge, got: %s", tabBarAfter)
	}
}

func TestHandleListPagination_ShortList(t *testing.T) {
	oldCligram := telegram.Cligram
	telegram.Cligram = &client.Client{}
	defer func() { telegram.Cligram = oldCligram }()

	m := newTestModel(80, 24)
	m.Mode = ModeUsers
	m.OffsetDate = 5000
	m.OffsetID = 100
	m.Users.SetItems([]list.Item{
		types.UserInfo{PeerID: "1", FirstName: "A"},
		types.UserInfo{PeerID: "2", FirstName: "B"},
		types.UserInfo{PeerID: "3", FirstName: "C"},
	})
	m.Users.Select(2) // At end of 3-item list

	mRes, cmd := m.handleListPagination()
	if !mRes.OnPagination {
		t.Errorf("expected OnPagination to be true for short list scrolled to end")
	}
	if !mRes.UsersLoading {
		t.Errorf("expected UsersLoading to be true during short list pagination")
	}
	if cmd == nil {
		t.Errorf("expected pagination cmd to be produced")
	}
}
