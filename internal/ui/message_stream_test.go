package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gotd/td/tg"
	"github.com/kumneger0/cligram/internal/telegram/types"
)

func TestMergeConversations_DeduplicationAndOrder(t *testing.T) {
	m := newTestModel(80, 24)
	baseTime := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)

	m.Conversations = []types.FormattedMessage{
		{ID: 1, Date: baseTime.Add(1 * time.Minute), Content: "First"},
		{ID: 3, Date: baseTime.Add(3 * time.Minute), Content: "Third"},
	}

	newBatch := []types.FormattedMessage{
		{ID: 2, Date: baseTime.Add(2 * time.Minute), Content: "Second"},
		{ID: 3, Date: baseTime.Add(3 * time.Minute), Content: "Third Duplicate"},
		{ID: 4, Date: baseTime.Add(4 * time.Minute), Content: "Fourth"},
	}

	merged := m.mergeConversations(newBatch)

	if len(merged) != 4 {
		t.Fatalf("expected 4 merged messages without duplicates, got %d", len(merged))
	}

	expectedIDs := []int{1, 2, 3, 4}
	for i, id := range expectedIDs {
		if merged[i].ID != id {
			t.Errorf("expected merged[%d].ID = %d, got %d", i, id, merged[i].ID)
		}
	}
}

func TestMergeConversations_250Cap(t *testing.T) {
	m := newTestModel(80, 24)
	baseTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	var lotsOfMessages []types.FormattedMessage
	for i := 1; i <= 300; i++ {
		lotsOfMessages = append(lotsOfMessages, types.FormattedMessage{
			ID:      i,
			Date:    baseTime.Add(time.Duration(i) * time.Minute),
			Content: fmt.Sprintf("Message %d", i),
		})
	}

	merged := m.mergeConversations(lotsOfMessages)

	if len(merged) != 250 {
		t.Fatalf("expected capped at 250 messages, got %d", len(merged))
	}

	// Should preserve the newest 250 messages (IDs 51 to 300)
	if merged[0].ID != 51 {
		t.Errorf("expected oldest retained message ID to be 51, got %d", merged[0].ID)
	}
	if merged[len(merged)-1].ID != 300 {
		t.Errorf("expected newest message ID to be 300, got %d", merged[len(merged)-1].ID)
	}
}

func TestRenderMessageBubble_IncomingAndOutgoing(t *testing.T) {
	m := newTestModel(80, 24)
	m.Mode = ModeUsers
	m.SelectedUser.ReadOutboxMaxID = 100

	outgoingRead := types.FormattedMessage{
		ID:       50,
		Sender:   "you",
		IsFromMe: true,
		Content:  "Hello from me",
		Date:     time.Date(2026, 1, 1, 14, 30, 0, 0, time.UTC),
	}
	outgoingUnread := types.FormattedMessage{
		ID:       150,
		Sender:   "you",
		IsFromMe: true,
		Content:  "Pending message",
		Date:     time.Date(2026, 1, 1, 14, 31, 0, 0, time.UTC),
	}
	incoming := types.FormattedMessage{
		ID:       60,
		Sender:   "Alice",
		IsFromMe: false,
		Content:  "Hello there",
		Date:     time.Date(2026, 1, 1, 14, 32, 0, 0, time.UTC),
	}

	// Outgoing read: contains "You" and "✓✓"
	outReadView := m.renderMessageBubble(outgoingRead, 0, 40, 80, 100)
	if !strings.Contains(outReadView, "You") {
		t.Errorf("expected 'You' in outgoing bubble, got: %s", outReadView)
	}
	if !strings.Contains(outReadView, "✓✓") {
		t.Errorf("expected double check '✓✓' for read outgoing message, got: %s", outReadView)
	}

	// Outgoing unread: contains "You" and single "✓"
	outUnreadView := m.renderMessageBubble(outgoingUnread, 1, 40, 80, 100)
	if !strings.Contains(outUnreadView, "✓") || strings.Contains(outUnreadView, "✓✓") {
		t.Errorf("expected single check '✓' for unread outgoing message, got: %s", outUnreadView)
	}

	// Incoming: contains "Alice" and does not contain read checks
	inView := m.renderMessageBubble(incoming, 2, 40, 80, 100)
	if !strings.Contains(inView, "Alice") {
		t.Errorf("expected sender 'Alice' in incoming bubble, got: %s", inView)
	}
	if strings.Contains(inView, "✓") {
		t.Errorf("incoming bubble should not contain read check marks, got: %s", inView)
	}

	// Selection highlight test
	m.FocusedOn = Main
	m.SelectedMessageIndex = 2
	selectedInView := m.renderMessageBubble(incoming, 2, 40, 80, 100)
	if !strings.Contains(selectedInView, "▎") {
		t.Errorf("expected selection indicator '▎' when card is selected, got: %s", selectedInView)
	}
}

func TestEliminationOfCursorSnapping_SmartAutoScroll(t *testing.T) {
	m := newTestModel(80, 24)
	m.FocusedOn = Main
	m.renderMessagesViewport(80, 20)

	// Populate initial history
	var initialMsgs [50]types.FormattedMessage
	baseTime := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 30; i++ {
		initialMsgs[i] = types.FormattedMessage{
			ID:      i + 1,
			Date:    baseTime.Add(time.Duration(i) * time.Minute),
			Content: fmt.Sprintf("Line message %d", i+1),
		}
	}

	// Load initial messages (empty history -> should auto-scroll to bottom)
	resModel, _ := m.handleGetMessages(types.GetMessagesMsg{Messages: initialMsgs})
	m = resModel.(Model)

	if m.SelectedMessageIndex != 29 {
		t.Fatalf("expected initial load to select newest message (29), got %d", m.SelectedMessageIndex)
	}

	// Simulate user scrolling up to inspect older messages
	m.viewport.SetYOffset(5)
	m.SelectedMessageIndex = 5

	// Test handleGetMessages when user is scrolled up (!viewport.AtBottom)
	var additionalMsgs [50]types.FormattedMessage
	additionalMsgs[0] = types.FormattedMessage{
		ID:      31,
		Date:    baseTime.Add(31 * time.Minute),
		Content: "New background message 31",
	}

	resModel2, _ := m.handleGetMessages(types.GetMessagesMsg{Messages: additionalMsgs})
	m2 := resModel2.(Model)

	// Since user was scrolled up, scroll position should NOT jump to bottom and selection should not snap to 30
	if m2.SelectedMessageIndex != 5 {
		t.Errorf("expected selection index to remain at 5 when scrolled up, got %d", m2.SelectedMessageIndex)
	}
	if m2.viewport.YOffset != 5 {
		t.Errorf("expected YOffset to remain at 5 when scrolled up, got %d", m2.viewport.YOffset)
	}
}

func TestKeyboardNavigation_SelectionAndScroll(t *testing.T) {
	m := newTestModel(80, 24)
	m.FocusedOn = Main
	baseTime := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)

	for i := 0; i < 15; i++ {
		m.Conversations = append(m.Conversations, types.FormattedMessage{
			ID:      i + 1,
			Date:    baseTime.Add(time.Duration(i) * time.Minute),
			Content: fmt.Sprintf("Card %d", i+1),
		})
	}
	m.renderMessagesViewport(80, 20)
	m.SelectedMessageIndex = 14

	// 'k' or 'up' steps backward
	res, _ := m.handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = res.(Model)
	if m.SelectedMessageIndex != 13 {
		t.Errorf("expected selection index 13 after 'k', got %d", m.SelectedMessageIndex)
	}

	res, _ = m.handleKeyPress(tea.KeyMsg{Type: tea.KeyUp})
	m = res.(Model)
	if m.SelectedMessageIndex != 12 {
		t.Errorf("expected selection index 12 after Up, got %d", m.SelectedMessageIndex)
	}

	// 'j' or 'down' steps forward
	res, _ = m.handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = res.(Model)
	if m.SelectedMessageIndex != 13 {
		t.Errorf("expected selection index 13 after 'j', got %d", m.SelectedMessageIndex)
	}

	res, _ = m.handleKeyPress(tea.KeyMsg{Type: tea.KeyDown})
	m = res.(Model)
	if m.SelectedMessageIndex != 14 {
		t.Errorf("expected selection index 14 after Down, got %d", m.SelectedMessageIndex)
	}

	// Clamping: pressing Down at end does not exceed len-1
	res, _ = m.handleKeyPress(tea.KeyMsg{Type: tea.KeyDown})
	m = res.(Model)
	if m.SelectedMessageIndex != 14 {
		t.Errorf("expected selection index clamped at 14, got %d", m.SelectedMessageIndex)
	}

	// 'home' jumps to 0
	res, _ = m.handleKeyPress(tea.KeyMsg{Type: tea.KeyHome})
	m = res.(Model)
	if m.SelectedMessageIndex != 0 {
		t.Errorf("expected selection index 0 after Home, got %d", m.SelectedMessageIndex)
	}

	// Clamping: pressing Up at 0 does not go below 0
	res, _ = m.handleKeyPress(tea.KeyMsg{Type: tea.KeyUp})
	m = res.(Model)
	if m.SelectedMessageIndex != 0 {
		t.Errorf("expected selection index clamped at 0, got %d", m.SelectedMessageIndex)
	}

	// 'end' jumps to len-1
	res, _ = m.handleKeyPress(tea.KeyMsg{Type: tea.KeyEnd})
	m = res.(Model)
	if m.SelectedMessageIndex != 14 {
		t.Errorf("expected selection index 14 after End, got %d", m.SelectedMessageIndex)
	}
}

func TestSingleKeyActions_OperateOnSelectedMessage(t *testing.T) {
	m := newTestModel(80, 24)
	m.FocusedOn = Main
	m.Mode = ModeUsers
	m.SelectedUser = types.UserInfo{PeerID: "42", FirstName: "Bob"}

	m.Conversations = []types.FormattedMessage{
		{ID: 10, Sender: "Bob", Content: "Message 10", Date: time.Now()},
		{ID: 20, Sender: "you", IsFromMe: true, Content: "Message 20", Date: time.Now()},
	}
	m.renderMessagesViewport(80, 20)

	// Select message 10
	m.SelectedMessageIndex = 0
	if m.SelectedMessage().ID != 10 {
		t.Fatalf("expected SelectedMessage ID 10, got %d", m.SelectedMessage().ID)
	}

	// Reply key 'r'
	mRes, _ := m.handleReplyKey()
	m1 := mRes.(Model)
	if !m1.IsReply || m1.ReplyTo == nil || m1.ReplyTo.ID != 10 {
		t.Errorf("expected reply targeting message ID 10, got %+v", m1.ReplyTo)
	}
	if m1.FocusedOn != Input {
		t.Errorf("expected focus on Input after reply key, got %v", m1.FocusedOn)
	}

	// Select outgoing message 20 and test Edit key 'e'
	m.SelectedMessageIndex = 1
	mRes, _ = m.handleEditKey()
	m2 := mRes.(Model)
	if m2.EditMessage == nil || m2.EditMessage.ID != 20 {
		t.Errorf("expected edit message targeting ID 20, got %+v", m2.EditMessage)
	}
	if m2.Input.Value() != "Message 20" {
		t.Errorf("expected input value 'Message 20', got %q", m2.Input.Value())
	}

	// Forward key 'f' on message 10
	m.SelectedMessageIndex = 0
	_, fCmd := m.handleForwardKey()
	if fCmd == nil {
		t.Fatalf("expected forward command to be produced")
	}
	fMsg := fCmd()
	openModal, ok := fMsg.(OpenModalMsg)
	if !ok || openModal.ModalMode != ModalModeForwardMessage || openModal.Message == nil || openModal.Message.ID != 10 {
		t.Errorf("expected OpenModalMsg for forward with message ID 10, got %+v", fMsg)
	}

	// Delete key 'd' on message 10
	_, dCmd := m.handleDeleteKey()
	if dCmd == nil {
		t.Fatalf("expected delete command to be produced")
	}
	dMsg := dCmd()
	openDeleteModal, ok := dMsg.(OpenModalMsg)
	if !ok || openDeleteModal.ModalMode != ModalModeDeleteMessage || openDeleteModal.Message == nil || openDeleteModal.Message.ID != 10 {
		t.Errorf("expected OpenModalMsg for delete with message ID 10, got %+v", dMsg)
	}
}

func TestFocusCycling_ThreeWay(t *testing.T) {
	m := newTestModel(80, 24)
	m.Mode = ModeUsers
	m.FocusedOn = SideBar

	// Tab: SideBar -> Main
	m1, _ := changeFocusMode(&m, "tab", false)
	if m1.FocusedOn != Main {
		t.Errorf("expected Tab from SideBar to focus Main, got %v", m1.FocusedOn)
	}

	// Tab: Main -> Input
	m2, _ := changeFocusMode(&m1, "tab", false)
	if m2.FocusedOn != Input {
		t.Errorf("expected Tab from Main to focus Input, got %v", m2.FocusedOn)
	}

	// Tab: Input -> SideBar
	m3, _ := changeFocusMode(&m2, "tab", false)
	if m3.FocusedOn != SideBar {
		t.Errorf("expected Tab from Input to focus SideBar, got %v", m3.FocusedOn)
	}

	// Shift+Tab: SideBar -> Input -> Main -> SideBar
	m.FocusedOn = SideBar
	s1, _ := changeFocusMode(&m, "tab", true)
	if s1.FocusedOn != Input {
		t.Errorf("expected Shift+Tab from SideBar to focus Input, got %v", s1.FocusedOn)
	}
	s2, _ := changeFocusMode(&s1, "tab", true)
	if s2.FocusedOn != Main {
		t.Errorf("expected Shift+Tab from Input to focus Main, got %v", s2.FocusedOn)
	}
	s3, _ := changeFocusMode(&s2, "tab", true)
	if s3.FocusedOn != SideBar {
		t.Errorf("expected Shift+Tab from Main to focus SideBar, got %v", s3.FocusedOn)
	}

	// Read-only broadcast channel: skip Input
	m.Mode = ModeChannels
	m.SelectedChannel = types.ChannelInfo{ID: "99", IsBroadcast: true, IsCreator: false}
	m.FocusedOn = SideBar

	ro1, _ := changeFocusMode(&m, "tab", false)
	if ro1.FocusedOn != Main {
		t.Errorf("expected Tab to Main, got %v", ro1.FocusedOn)
	}
	// From Main, should skip Input and go back to SideBar
	ro2, _ := changeFocusMode(&ro1, "tab", false)
	if ro2.FocusedOn != SideBar {
		t.Errorf("expected Tab from Main to skip Input on broadcast channel and focus SideBar, got %v", ro2.FocusedOn)
	}

	// 'i' key should be ignored on read-only broadcast channel
	roMain := m
	roMain.FocusedOn = Main
	res, _ := roMain.handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	mAfterI := res.(Model)
	if mAfterI.FocusedOn != Main {
		t.Errorf("expected 'i' to be ignored on read-only channel, but focused: %v", mAfterI.FocusedOn)
	}

	// Esc transitions: Input -> Main -> SideBar
	writableModel := m
	writableModel.Mode = ModeUsers
	writableModel.FocusedOn = Input
	esc1Res, _ := writableModel.handleKeyPress(tea.KeyMsg{Type: tea.KeyEsc})
	esc1 := esc1Res.(Model)
	if esc1.FocusedOn != Main {
		t.Errorf("expected Esc from Input to focus Main, got %v", esc1.FocusedOn)
	}

	esc2Res, _ := esc1.handleKeyPress(tea.KeyMsg{Type: tea.KeyEsc})
	esc2 := esc2Res.(Model)
	if esc2.FocusedOn != SideBar {
		t.Errorf("expected Esc from Main to focus SideBar, got %v", esc2.FocusedOn)
	}
}

func TestMessageStream_PeerIsolationOnSwitch(t *testing.T) {
	m := newTestModel(80, 24)
	m.Mode = ModeUsers

	userX := types.UserInfo{
		PeerID:    "user-x",
		FirstName: "Alice",
	}
	userY := types.UserInfo{
		PeerID:    "user-y",
		FirstName: "Bob",
	}

	m.Users.SetItems([]list.Item{userX, userY})
	m.All.SetItems([]list.Item{userX, userY})

	// Currently active chat is User Y
	m.ActivePeerID = "user-y"
	m.SelectedUser = userY
	m.Conversations = []types.FormattedMessage{
		{
			ID:      10,
			Content: "Hello Bob",
			Sender:  "Bob",
		},
	}

	// 1. Race condition test: Late GetMessagesMsg for User X arriving while User Y is open
	var lateMessages [50]types.FormattedMessage
	lateMessages[0] = types.FormattedMessage{ID: 50, Content: "Old message with Alice", Sender: "Alice"}
	lateXHistory := types.GetMessagesMsg{
		PeerID:   "user-x",
		Messages: lateMessages,
	}
	resLate, _ := m.handleGetMessages(lateXHistory)
	mAfterLate := resLate.(Model)

	if len(mAfterLate.Conversations) != 1 || mAfterLate.Conversations[0].ID != 10 {
		t.Fatalf("expected late history for user-x to be discarded when user-y is active, but got: %+v", mAfterLate.Conversations)
	}

	// 2. Incoming message for User X while User Y is active
	incomingX := types.NewMessageNotification{
		PeerID: "user-x",
		FromID: "user-x",
		Message: &tg.Message{
			ID:      201,
			Message: "Hey are you there?",
			Out:     false,
		},
	}
	resIncX, _ := m.handleNewMessage(incomingX)
	mAfterIncX := resIncX.(Model)

	// Conversations in view must NOT include User X's message
	if len(mAfterIncX.Conversations) != 1 || mAfterIncX.Conversations[0].ID != 10 {
		t.Fatalf("expected user-x incoming message not to leak into user-y conversations, got: %+v", mAfterIncX.Conversations)
	}

	// User X unread count must be incremented in m.Users and m.All
	userXInList := mAfterIncX.Users.Items()[0].(types.UserInfo)
	if userXInList.UnreadCount != 1 {
		t.Errorf("expected user-x unread count in Users list to be 1, got %d", userXInList.UnreadCount)
	}
	userXInAll := mAfterIncX.All.Items()[0].(types.UserInfo)
	if userXInAll.UnreadCount != 1 {
		t.Errorf("expected user-x unread count in All list to be 1, got %d", userXInAll.UnreadCount)
	}

	// 3. Outgoing message sent to User Y while User Y is active
	outgoingY := types.NewMessageNotification{
		PeerID: "user-y",
		FromID: "user-y",
		Message: &tg.Message{
			ID:      202,
			Message: "Replying to Bob",
			Out:     true,
		},
	}
	resOutY, _ := m.handleNewMessage(outgoingY)
	mAfterOutY := resOutY.(Model)

	if len(mAfterOutY.Conversations) != 2 || mAfterOutY.Conversations[1].ID != 202 {
		t.Fatalf("expected outgoing message to user-y to be appended to active conversations, got: %+v", mAfterOutY.Conversations)
	}
	if mAfterOutY.Conversations[1].Sender != "You" {
		t.Errorf("expected sender to be 'You', got '%s'", mAfterOutY.Conversations[1].Sender)
	}

	// 4. Outgoing message sent to User X (e.g. from another client) arriving while User Y is active
	outgoingX := types.NewMessageNotification{
		PeerID: "user-x",
		FromID: "user-x",
		Message: &tg.Message{
			ID:      203,
			Message: "Sent to Alice elsewhere",
			Out:     true,
		},
	}
	resOutX, _ := mAfterOutY.handleNewMessage(outgoingX)
	mAfterOutX := resOutX.(Model)

	if len(mAfterOutX.Conversations) != 2 {
		t.Fatalf("expected outgoing message to user-x not to be added to user-y conversations, got %d messages", len(mAfterOutX.Conversations))
	}
}

func TestVariableHeightMessageRendering(t *testing.T) {
	m := newTestModel(80, 24)
	m.Mode = ModeUsers

	shortMsg := types.FormattedMessage{
		ID:      1,
		Sender:  "Alice",
		Content: "Hi",
		Date:    time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
	}
	multiLineMsg := types.FormattedMessage{
		ID:      2,
		Sender:  "Alice",
		Content: "This is a very long message that is guaranteed to wrap across multiple lines in the chat bubble UI because it exceeds the standard bubble width. Here is line two and line three of the multiline message content.",
		Date:    time.Date(2026, 1, 1, 10, 1, 0, 0, time.UTC),
	}
	replyMsg := types.FormattedMessage{
		ID:      3,
		Sender:  "Bob",
		Content: "Replying to earlier context",
		Date:    time.Date(2026, 1, 1, 10, 2, 0, 0, time.UTC),
		ReplyTo: &types.FormattedMessage{
			ID:      1,
			Sender:  "Alice",
			Content: "Hi",
		},
	}

	shortCard := m.renderMessageBubble(shortMsg, 0, 40, 80, 0)
	multiCard := m.renderMessageBubble(multiLineMsg, 1, 40, 80, 0)
	replyCard := m.renderMessageBubble(replyMsg, 2, 40, 80, 0)

	shortLines := len(strings.Split(shortCard, "\n"))
	multiLines := len(strings.Split(multiCard, "\n"))
	replyLines := len(strings.Split(replyCard, "\n"))

	if multiLines <= shortLines {
		t.Errorf("expected multiline message to have more lines than short message, got multi=%d, short=%d", multiLines, shortLines)
	}
	if replyLines <= shortLines {
		t.Errorf("expected reply message to have more lines than short message, got reply=%d, short=%d", replyLines, shortLines)
	}
	if !strings.Contains(replyCard, "Hi") {
		t.Errorf("expected reply bubble to include quoted content 'Hi', got:\n%s", replyCard)
	}
}

func TestScrollingBoundaries_AndClamping(t *testing.T) {
	m := newTestModel(80, 20)
	m.Mode = ModeUsers
	m.viewport.Height = 10

	var messages []types.FormattedMessage
	baseTime := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 15; i++ {
		messages = append(messages, types.FormattedMessage{
			ID:      i + 1,
			Content: fmt.Sprintf("Message item %d with multi-line detail\nSecond line of item %d", i+1, i+1),
			Date:    baseTime.Add(time.Duration(i) * time.Minute),
		})
	}
	m.Conversations = messages
	m.renderMessagesViewport(80, 10)

	// Boundary clamping in SelectedMessage
	m.SelectedMessageIndex = -10
	if m.SelectedMessage().ID != 1 {
		t.Errorf("expected clamped to index 0 (ID 1), got %d", m.SelectedMessage().ID)
	}
	m.SelectedMessageIndex = 100
	if m.SelectedMessage().ID != 15 {
		t.Errorf("expected clamped to index 14 (ID 15), got %d", m.SelectedMessage().ID)
	}

	// Scroll into view when target card is below viewport
	m.viewport.SetYOffset(0)
	m.SelectedMessageIndex = 12
	m.scrollSelectedMessageIntoView()
	target := m.messageLineRanges[12]
	if m.viewport.YOffset+m.viewport.Height <= target.endLine {
		t.Errorf("expected target card 12 end line (%d) to be within visible viewport [YOffset %d, %d]",
			target.endLine, m.viewport.YOffset, m.viewport.YOffset+m.viewport.Height)
	}

	// Scroll into view when target card is above viewport
	m.SelectedMessageIndex = 1
	m.scrollSelectedMessageIntoView()
	target1 := m.messageLineRanges[1]
	if m.viewport.YOffset > target1.startLine {
		t.Errorf("expected viewport to scroll up to target card start line %d, got YOffset %d",
			target1.startLine, m.viewport.YOffset)
	}
}

func TestReadReceipts_ModeAll(t *testing.T) {
	m := newTestModel(80, 24)
	m.Mode = ModeAll

	user := types.UserInfo{
		PeerID:          "user-all-1",
		FirstName:       "Diana",
		ReadOutboxMaxID: 100,
	}
	m.All.SetItems([]list.Item{user})
	m.All.Select(0)

	if m.ActiveReadOutboxMaxID() != 100 {
		t.Fatalf("expected ActiveReadOutboxMaxID in ModeAll to be 100, got %d", m.ActiveReadOutboxMaxID())
	}

	m.Conversations = []types.FormattedMessage{
		{
			ID:       50,
			Content:  "Read outgoing message",
			Sender:   "You",
			IsFromMe: true,
			Date:     time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
		},
		{
			ID:       150,
			Content:  "Unread outgoing message",
			Sender:   "You",
			IsFromMe: true,
			Date:     time.Date(2026, 1, 1, 12, 1, 0, 0, time.UTC),
		},
	}

	view := m.renderMessagesViewport(80, 20)
	if !strings.Contains(view, "✓✓") {
		t.Errorf("expected double checkmark '✓✓' for read message in ModeAll, got view:\n%s", view)
	}
	if !strings.Contains(view, "✓") {
		t.Errorf("expected checkmark '✓' for unread message in ModeAll, got view:\n%s", view)
	}
}

func TestWindowResize_PreservesScrollOffset(t *testing.T) {
	m := newTestModel(80, 24)
	m.Mode = ModeUsers

	var messages []types.FormattedMessage
	baseTime := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 30; i++ {
		messages = append(messages, types.FormattedMessage{
			ID:      i + 1,
			Content: fmt.Sprintf("Message item %d with plenty of text to allow scrollback", i+1),
			Date:    baseTime.Add(time.Duration(i) * time.Minute),
		})
	}
	m.Conversations = messages
	m.renderMessagesViewport(80, 20)

	m.viewport.SetYOffset(15)

	res, _ := m.handleWindowSize(tea.WindowSizeMsg{Width: 100, Height: 35})
	mAfter := res.(Model)

	if mAfter.viewport.YOffset != 15 {
		t.Errorf("expected resize to preserve scroll offset 15, got %d", mAfter.viewport.YOffset)
	}
	if mAfter.viewport.Width <= 0 || mAfter.viewport.Height <= 0 {
		t.Errorf("expected valid viewport dimensions after resize, got %dx%d", mAfter.viewport.Width, mAfter.viewport.Height)
	}
}

func TestMainFocus_KeysDoNotTriggerViewportPageScroll(t *testing.T) {
	m := newTestModel(80, 24)
	m.Mode = ModeUsers
	m.FocusedOn = Main
	m.Conversations = []types.FormattedMessage{
		{ID: 1, Content: "Msg 1"},
		{ID: 2, Content: "Msg 2"},
	}
	m.SelectedMessageIndex = 0
	m.renderMessagesViewport(80, 20)
	m.viewport.SetYOffset(0)

	// Press 'd' (delete message)
	dRes, dCmd := m.handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	mAfterD := dRes.(Model)
	if dCmd == nil {
		t.Fatalf("expected delete modal cmd from 'd'")
	}
	if mAfterD.viewport.YOffset != 0 {
		t.Errorf("expected 'd' not to scroll viewport half-page, got YOffset %d", mAfterD.viewport.YOffset)
	}

	// Press 'f' (forward message)
	fRes, fCmd := m.handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	mAfterF := fRes.(Model)
	if fCmd == nil {
		t.Fatalf("expected forward modal cmd from 'f'")
	}
	if mAfterF.viewport.YOffset != 0 {
		t.Errorf("expected 'f' not to scroll viewport page-down, got YOffset %d", mAfterF.viewport.YOffset)
	}
}

func TestHandleWindowSize_NoPanicOnUninitializedForumTopics(t *testing.T) {
	m := newTestModel(80, 24)
	m.SelectedGroupForumTopics = list.Model{}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("handleWindowSize panicked on uninitialized SelectedGroupForumTopics: %v", r)
		}
	}()

	res, _ := m.handleWindowSize(tea.WindowSizeMsg{Width: 100, Height: 35})
	if res == nil {
		t.Fatalf("expected non-nil model response from handleWindowSize")
	}
}

func TestReplyFeature_ModeAllAndPeers(t *testing.T) {
	m := newTestModel(80, 24)
	m.FocusedOn = Main
	m.Mode = ModeAll

	user := types.UserInfo{PeerID: "123", FirstName: "Alice"}
	m.All.SetItems([]list.Item{user})
	m.ActivePeerID = "123"
	m.Conversations = []types.FormattedMessage{
		{ID: 101, Sender: "Alice", Content: "Hello there", Date: time.Now()},
		{ID: 102, Sender: "you", IsFromMe: true, Content: "Hi Alice", Date: time.Now()},
	}
	m.SelectedMessageIndex = 0

	// 1. Reply to Alice's message in ModeAll
	res, _ := m.handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	mReply := res.(Model)
	if !mReply.IsReply {
		t.Fatalf("expected IsReply=true in ModeAll with user chat")
	}
	if mReply.ReplyTo == nil || mReply.ReplyTo.ID != 101 {
		t.Fatalf("expected ReplyTo targeting message 101, got %+v", mReply.ReplyTo)
	}
	if mReply.FocusedOn != Input {
		t.Fatalf("expected FocusedOn=Input after 'r', got %v", mReply.FocusedOn)
	}

	// 2. Cancel reply on Esc
	resEsc, _ := mReply.handleKeyPress(tea.KeyMsg{Type: tea.KeyEsc})
	mCanceled := resEsc.(Model)
	if mCanceled.IsReply {
		t.Errorf("expected IsReply=false after Esc, got true")
	}
	if mCanceled.ReplyTo != nil {
		t.Errorf("expected ReplyTo=nil after Esc, got %+v", mCanceled.ReplyTo)
	}
	if mCanceled.FocusedOn != Main {
		t.Errorf("expected FocusedOn=Main after Esc from Input, got %v", mCanceled.FocusedOn)
	}

	// 3. Test read-only broadcast channel in ModeAll
	readOnlyCh := types.ChannelInfo{ID: "ch456", ChannelTitle: "Announcements", IsBroadcast: true, IsCreator: false}
	mROModel := newTestModel(80, 24)
	mROModel.FocusedOn = Main
	mROModel.Mode = ModeAll
	mROModel.All.SetItems([]list.Item{readOnlyCh})
	mROModel.ActivePeerID = "ch456"
	mROModel.Conversations = []types.FormattedMessage{
		{ID: 301, Sender: "Admin", Content: "Broadcast", Date: time.Now()},
	}
	mROModel.SelectedMessageIndex = 0

	resRO, _ := mROModel.handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	mRO := resRO.(Model)
	if mRO.IsReply {
		t.Errorf("expected IsReply=false in read-only broadcast channel, got true")
	}

	// 4. Test reply from SideBar focus
	mSide := newTestModel(80, 24)
	mSide.FocusedOn = SideBar
	mSide.Mode = ModeAll
	mSide.All.SetItems([]list.Item{user})
	mSide.ActivePeerID = "123"
	mSide.Conversations = []types.FormattedMessage{
		{ID: 201, Sender: "Alice", Content: "Msg 1", Date: time.Now()},
		{ID: 202, Sender: "Alice", Content: "Msg 2", Date: time.Now()},
	}
	mSide.SelectedMessageIndex = -1

	resSide, _ := mSide.handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	mSideReply := resSide.(Model)
	if !mSideReply.IsReply {
		t.Fatalf("expected IsReply=true from SideBar with active conversation")
	}
	if mSideReply.ReplyTo == nil || mSideReply.ReplyTo.ID != 202 {
		t.Fatalf("expected reply targeting latest message 202, got %+v", mSideReply.ReplyTo)
	}
	if mSideReply.FocusedOn != Input {
		t.Fatalf("expected FocusedOn=Input after reply from SideBar, got %v", mSideReply.FocusedOn)
	}
}

func TestPrepareInputView_ReplyBanner(t *testing.T) {
	m := newTestModel(80, 24)
	m.IsReply = true
	m.ReplyTo = &types.FormattedMessage{
		ID:      50,
		Sender:  "Charlie",
		Content: "This is a long message to test snippet rendering in the reply context bar",
	}

	view := prepareInputView(&m, layoutDimensions{inputHeight: 3})
	if !strings.Contains(view, "Replying to Charlie") {
		t.Errorf("expected input view to contain 'Replying to Charlie', got:\n%s", view)
	}
	if !strings.Contains(view, "Esc to cancel") {
		t.Errorf("expected input view to contain 'Esc to cancel', got:\n%s", view)
	}
}

func TestMessageStream_MediaAttachment_Rendering(t *testing.T) {
	m := newTestModel(80, 24)
	m.Conversations = []types.FormattedMessage{
		{
			ID:      201,
			Sender:  "Alice",
			Content: "Vacation snapshot",
			Date:    time.Now(),
			MediaAttachment: &types.MediaAttachment{
				Type:     types.MediaTypePhoto,
				Width:    1920,
				Height:   1080,
				FileSize: 204800,
				Status:   types.MediaStatusIdle,
			},
		},
		{
			ID:      202,
			Sender:  "Bob",
			Content: "Quarterly review",
			Date:    time.Now(),
			MediaAttachment: &types.MediaAttachment{
				Type:     types.MediaTypeDocument,
				FileName: "review.pdf",
				FileSize: 1048576,
				Status:   types.MediaStatusDownloading,
			},
		},
		{
			ID:      203,
			Sender:  "Dave",
			Content: "Classic track",
			Date:    time.Now(),
			MediaAttachment: &types.MediaAttachment{
				Type:      types.MediaTypeAudio,
				Title:     "Bohemian Rhapsody",
				Performer: "Queen",
				Duration:  354,
				FileSize:  8000000,
				Status:    types.MediaStatusIdle,
			},
		},
	}

	bubble0 := m.renderMessageBubble(m.Conversations[0], 0, 60, 80, 0)
	if !strings.Contains(bubble0, "📷 Photo") {
		t.Errorf("expected bubble0 to contain '📷 Photo', got:\n%s", bubble0)
	}
	if !strings.Contains(bubble0, "1920x1080") {
		t.Errorf("expected bubble0 to contain '1920x1080', got:\n%s", bubble0)
	}
	if !strings.Contains(bubble0, "Vacation snapshot") {
		t.Errorf("expected bubble0 to contain caption 'Vacation snapshot', got:\n%s", bubble0)
	}

	bubble1 := m.renderMessageBubble(m.Conversations[1], 1, 60, 80, 0)
	if !strings.Contains(bubble1, "review.pdf") {
		t.Errorf("expected bubble1 to contain 'review.pdf', got:\n%s", bubble1)
	}
	if !strings.Contains(bubble1, "Downloading") {
		t.Errorf("expected bubble1 to contain downloading indicator, got:\n%s", bubble1)
	}

	bubble2 := m.renderMessageBubble(m.Conversations[2], 2, 60, 80, 0)
	if !strings.Contains(bubble2, "Queen - Bohemian Rhapsody") {
		t.Errorf("expected bubble2 to contain 'Queen - Bohemian Rhapsody', got:\n%s", bubble2)
	}
	if !strings.Contains(bubble2, "5:54") {
		t.Errorf("expected bubble2 to contain duration '5:54', got:\n%s", bubble2)
	}
}

func TestMessageStream_MediaAttachment_EnterDownloads(t *testing.T) {
	m := newTestModel(80, 24)
	m.FocusedOn = Main
	m.Conversations = []types.FormattedMessage{
		{
			ID:      301,
			Sender:  "Alice",
			Content: "Photo",
			Date:    time.Now(),
			MediaAttachment: &types.MediaAttachment{
				Type:     types.MediaTypePhoto,
				FileName: "sample.jpg",
				FileSize: 50000,
				Status:   types.MediaStatusIdle,
			},
		},
	}
	m.SelectedMessageIndex = 0

	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updatedModel.(Model)

	if cmd == nil {
		t.Fatalf("expected non-nil cmd when pressing Enter on media attachment")
	}
	if m.Conversations[0].MediaAttachment.Status != types.MediaStatusDownloading {
		t.Errorf("expected status MediaStatusDownloading, got %v", m.Conversations[0].MediaAttachment.Status)
	}

	completeMsg := types.DownloadMediaCompleteMsg{
		MessageID: 301,
		LocalPath: "/tmp/sample.jpg",
		Err:       nil,
	}
	updatedAfterComplete, _ := m.Update(completeMsg)
	m = updatedAfterComplete.(Model)

	if m.Conversations[0].MediaAttachment.Status != types.MediaStatusDownloaded {
		t.Errorf("expected status MediaStatusDownloaded, got %v", m.Conversations[0].MediaAttachment.Status)
	}
	if m.Conversations[0].MediaAttachment.LocalPath != "/tmp/sample.jpg" {
		t.Errorf("expected LocalPath '/tmp/sample.jpg', got %q", m.Conversations[0].MediaAttachment.LocalPath)
	}

	// Pressing Enter again on already-downloaded media should NOT transition back to downloading
	updatedReopen, reopenCmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updatedReopen.(Model)
	if m.Conversations[0].MediaAttachment.Status != types.MediaStatusDownloaded {
		t.Errorf("expected status to stay MediaStatusDownloaded, got %v", m.Conversations[0].MediaAttachment.Status)
	}
	if reopenCmd == nil {
		t.Errorf("expected alert toast command on instant re-open")
	}
}
