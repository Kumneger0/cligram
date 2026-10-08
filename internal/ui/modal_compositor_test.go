package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kumneger0/cligram/internal/telegram/types"
	overlay "github.com/rmhubbert/bubbletea-overlay"
)

type dummyModel struct {
	text string
}

func (d dummyModel) Init() tea.Cmd { return nil }
func (d dummyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if s, ok := msg.(string); ok {
		d.text = s
	}
	return d, nil
}
func (d dummyModel) View() string {
	pad := func(s string) string {
		if len(s) < 80 {
			return s + strings.Repeat(" ", 80-len(s))
		}
		return s
	}
	lines := make([]string, 24)
	for i := 0; i < 23; i++ {
		lines[i] = pad(fmt.Sprintf("line %d", i+1))
	}
	lines[23] = pad("Footer: " + d.text)
	return strings.Join(lines, "\n")
}

type dummyModal struct {
	text string
}

func (d dummyModal) Init() tea.Cmd                           { return nil }
func (d dummyModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return d, nil }
func (d dummyModal) View() string                            { return d.text }

func TestManager_ModalView_CompositesLiveBackground(t *testing.T) {
	bg1 := dummyModel{text: "INITIAL_BG_CHAT_WINDOW"}
	fg := dummyModal{text: "┌────────────┐\n│[MODAL_BOX] │\n└────────────┘"}

	mgr := Manager{
		State:        ModalView,
		WindowWidth:  80,
		WindowHeight: 24,
		Foreground:   fg,
		Background:   bg1,
		Overlay: overlay.New(
			fg,
			bg1,
			overlay.Center,
			overlay.Center,
			0,
			0,
		),
	}

	// Update background model while in ModalView (e.g. background chat receives update)
	mgr.Background = dummyModel{text: "LIVE_BG_UPDATED_CHAT"}

	// When Manager.View() is called, it MUST reflect the updated background state, not INITIAL_BG!
	view := mgr.View()
	if strings.Contains(view, "INITIAL_BG") {
		t.Errorf("stale background bug: view still contains INITIAL_BG:\n%s", view)
	}
	if !strings.Contains(view, "LIVE_BG_UPDATED_CHAT") {
		t.Errorf("expected view to contain LIVE_BG_UPDATED_CHAT from updated background, got:\n%s", view)
	}
	if !strings.Contains(view, "[MODAL_BOX]") {
		t.Errorf("expected view to contain [MODAL_BOX], got:\n%s", view)
	}
}

func TestForeground_SearchCommandPalette_DirectArrowAndEnter(t *testing.T) {
	fg := &Foreground{}
	// Send WindowSizeMsg to initialize input and search list
	fgModel, _ := fg.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	fg = fgModel.(*Foreground)

	// Open search modal
	fgModel, _ = fg.Update(OpenModalMsg{ModalMode: ModalModeSearch})
	fg = fgModel.(*Foreground)

	if fg.focusedOn != SEARCH {
		t.Fatalf("expected focusedOn to be SEARCH initially, got %v", fg.focusedOn)
	}

	// Populate search results
	results := []types.UserInfo{
		{FirstName: "Alice", PeerID: "101"},
		{FirstName: "Bob", PeerID: "102"},
	}
	fgModel, _ = fg.Update(types.SearchUsersMsg{Response: &results})
	fg = fgModel.(*Foreground)

	if len(fg.searchResultCombined.Items()) != 2 {
		t.Fatalf("expected 2 search results, got %d", len(fg.searchResultCombined.Items()))
	}
	if fg.searchResultCombined.Index() != 0 {
		t.Fatalf("expected initial index 0, got %d", fg.searchResultCombined.Index())
	}

	// Press Down arrow while input is focused (WITHOUT pressing Tab)
	fgModel, _ = fg.Update(tea.KeyMsg{Type: tea.KeyDown})
	fg = fgModel.(*Foreground)

	if fg.searchResultCombined.Index() != 1 {
		t.Errorf("expected Down arrow to move list selection to index 1 without needing Tab, got %d", fg.searchResultCombined.Index())
	}

	// Press Enter while input is focused
	fgModel, cmd := fg.Update(tea.KeyMsg{Type: tea.KeyEnter})
	fg = fgModel.(*Foreground)

	if cmd == nil {
		t.Fatalf("expected Enter to trigger command to select user and close modal, got nil cmd")
	}

	// Execute cmd batch to verify CloseOverlay and SelectSearchedUserResult are emitted
	batchMsg := cmd()
	if batchMsg == nil {
		t.Fatalf("expected non-nil batch message from cmd")
	}
}

func TestForeground_SearchCommandPalette_Rendering(t *testing.T) {
	fg := &Foreground{}
	fgModel, _ := fg.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	fg = fgModel.(*Foreground)
	fgModel, _ = fg.Update(OpenModalMsg{ModalMode: ModalModeSearch})
	fg = fgModel.(*Foreground)

	results := []types.UserInfo{
		{FirstName: "Alice", PeerID: "101"},
		{FirstName: "HelperBot", PeerID: "102", IsBot: true},
	}
	fgModel, _ = fg.Update(types.SearchUsersMsg{Response: &results})
	fg = fgModel.(*Foreground)

	view := fg.View()

	// Verify Command Palette elements
	if !strings.Contains(view, "Search") {
		t.Errorf("expected view to contain Search header, got:\n%s", view)
	}
	// Verify entity badges
	if !strings.Contains(view, "👤") {
		t.Errorf("expected view to contain user icon 👤, got:\n%s", view)
	}
	if !strings.Contains(view, "🤖") {
		t.Errorf("expected view to contain bot icon 🤖, got:\n%s", view)
	}
	// Verify footer keyboard hints
	if !strings.Contains(view, "Navigate") || !strings.Contains(view, "Select") || !strings.Contains(view, "Close") {
		t.Errorf("expected view to contain footer hotkey hints [Navigate, Select, Close], got:\n%s", view)
	}
}

func TestForeground_ForwardQuickPicker_AllDialogTypes(t *testing.T) {
	fg := &Foreground{}
	fgModel, _ := fg.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	fg = fgModel.(*Foreground)

	users := []list.Item{
		types.UserInfo{FirstName: "Alice", PeerID: "101"},
	}
	bots := []list.Item{
		types.UserInfo{FirstName: "HelperBot", PeerID: "102", IsBot: true},
	}
	channels := []list.Item{
		types.ChannelInfo{ChannelTitle: "News Channel", ID: "201", IsBroadcast: true},
	}
	groups := []list.Item{
		types.ChannelInfo{ChannelTitle: "Golang Group", ID: "301", IsBroadcast: false},
	}

	uList := list.New(users, list.NewDefaultDelegate(), 10, 10)
	bList := list.New(bots, list.NewDefaultDelegate(), 10, 10)
	cList := list.New(channels, list.NewDefaultDelegate(), 10, 10)
	gList := list.New(groups, list.NewDefaultDelegate(), 10, 10)

	sampleMsg := types.FormattedMessage{ID: 999, Content: "Hello world"}
	fromPeer := list.Item(types.UserInfo{FirstName: "Me", PeerID: "99"})

	fgModel, _ = fg.Update(OpenModalMsg{
		ModalMode:    ModalModeForwardMessage,
		UsersList:    &uList,
		BotsList:     &bList,
		ChannelsList: &cList,
		GroupsList:   &gList,
		Message:      &sampleMsg,
		FromPeer:     &fromPeer,
	})
	fg = fgModel.(*Foreground)

	if len(fg.allForwardDestinations) != 4 {
		t.Fatalf("expected 4 forward destinations, got %d", len(fg.allForwardDestinations))
	}

	view := fg.View()
	if !strings.Contains(view, "Forward Message") {
		t.Errorf("expected view to contain Forward Message header, got:\n%s", view)
	}
	if !strings.Contains(view, "👤") || !strings.Contains(view, "Alice") {
		t.Errorf("expected user badge and name in forward view, got:\n%s", view)
	}
	if !strings.Contains(view, "🤖") || !strings.Contains(view, "HelperBot") {
		t.Errorf("expected bot badge and name in forward view, got:\n%s", view)
	}
	if !strings.Contains(view, "📢") || !strings.Contains(view, "News Channel") {
		t.Errorf("expected channel badge and name in forward view, got:\n%s", view)
	}
	if !strings.Contains(view, "👥") || !strings.Contains(view, "Golang Group") {
		t.Errorf("expected group badge and name in forward view, got:\n%s", view)
	}
	if !strings.Contains(view, "Navigate") || !strings.Contains(view, "Forward") || !strings.Contains(view, "Cancel") {
		t.Errorf("expected footer hints [Navigate, Forward, Cancel], got:\n%s", view)
	}
}

func TestForeground_ForwardQuickPicker_InstantFiltering(t *testing.T) {
	fg := &Foreground{}
	fgModel, _ := fg.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	fg = fgModel.(*Foreground)

	users := []list.Item{
		types.UserInfo{FirstName: "Alice", PeerID: "101"},
		types.UserInfo{FirstName: "Bob", PeerID: "102"},
	}
	channels := []list.Item{
		types.ChannelInfo{ChannelTitle: "Golang Group", ID: "301"},
	}
	uList := list.New(users, list.NewDefaultDelegate(), 10, 10)
	cList := list.New(channels, list.NewDefaultDelegate(), 10, 10)

	sampleMsg := types.FormattedMessage{ID: 999, Content: "Hello world"}
	fromPeer := list.Item(types.UserInfo{FirstName: "Me", PeerID: "99"})

	fgModel, _ = fg.Update(OpenModalMsg{
		ModalMode:    ModalModeForwardMessage,
		UsersList:    &uList,
		ChannelsList: &cList,
		Message:      &sampleMsg,
		FromPeer:     &fromPeer,
	})
	fg = fgModel.(*Foreground)

	// Filter by typing 'g' then 'o'
	fgModel, _ = fg.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	fgModel, _ = fgModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	fg = fgModel.(*Foreground)

	if len(fg.forwardDestinations.Items()) != 1 {
		t.Fatalf("expected 1 filtered item for query 'Go', got %d", len(fg.forwardDestinations.Items()))
	}
	firstItem := fg.forwardDestinations.Items()[0].(ForwardDestinationItem)
	if firstItem.Name != "Golang Group" {
		t.Errorf("expected filtered item to be Golang Group, got %s", firstItem.Name)
	}

	// Press Enter to forward
	fgModel, cmd := fg.Update(tea.KeyMsg{Type: tea.KeyEnter})
	fg = fgModel.(*Foreground)

	if cmd == nil {
		t.Fatalf("expected non-nil cmd on Enter")
	}
	batchMsg := cmd()
	if batchMsg == nil {
		t.Fatalf("expected non-nil batch msg on Enter")
	}
}

func TestManager_SearchAllowsTypingQ(t *testing.T) {
	fg := &Foreground{}
	fgModel, _ := fg.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	fg = fgModel.(*Foreground)
	fgModel, _ = fg.Update(OpenModalMsg{ModalMode: ModalModeSearch})
	fg = fgModel.(*Foreground)

	bg := dummyModel{text: "Background"}
	mgr := Manager{
		State:        ModalView,
		WindowWidth:  80,
		WindowHeight: 24,
		Foreground:   fg,
		Background:   bg,
	}

	// Typing 'q' in ModalView must NOT close modal or quit app
	mgrModel, _ := mgr.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	mgr = mgrModel.(Manager)

	if mgr.State != ModalView {
		t.Errorf("expected Manager state to remain ModalView after typing 'q', got %v", mgr.State)
	}

	activeFg := mgr.Foreground.(*Foreground)
	if !strings.Contains(activeFg.input.Value(), "q") {
		t.Errorf("expected search input to receive 'q', got %q", activeFg.input.Value())
	}
}
