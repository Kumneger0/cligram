# Voice Calling UI & Call Presentation Specification

Status: ready-for-agent

## Problem Statement

While the underlying VoIP media and signaling mechanisms function, Cligram currently gives users no visual feedback during incoming and active voice calls:
1. When a remote Telegram peer calls, Cligram does not render any prompt or notification on the screen, causing incoming calls to go unnoticed and expire without the user knowing. This occurs because call visual elements are rendered above a full-height terminal layout, which immediately causes vertical terminal overflow and forces the call prompt off-screen into scrollback in Bubbletea alt-screen buffers. Furthermore, no audio chime or desktop notification is emitted.
2. When the user is on an active call, there is no persistent indicator showing that the call is in progress, no live timer tracking duration in seconds, and no visible connection privacy badge. The user cannot see how long they have been speaking, whether their microphone is muted, or whether media is flowing directly peer-to-peer or via encrypted reflectors.

## Solution

Provide a robust, overflow-safe TUI presentation for VoIP calling:
1. When an incoming call arrives, display a centered **Incoming Call Overlay** modal dialog composited over the existing TUI buffer using `bubbletea-overlay`, alongside an audible chime and OS desktop notification. The modal clearly shows the caller's identity and actionable key bindings (`[a]` to accept, `[d]` / `[Esc]` to decline), while intercepting key events to prevent accidental keystroke leaking into the chat input.
2. During an active Call Session, render a persistent single-line **In-Call Status Bar** pinned to the top of the viewport displaying peer name, live elapsed time in minutes and seconds (`MM:SS`), dynamic Connection Badge (`[P2P]` vs `[Relay]`), mute status, and hotkey hints (`[Alt+M: Mute/Unmute]`, `[Alt+H: Hangup]`). The application layout dimensions dynamically deduct one line of available height during active calls so the entire UI matches the terminal boundary without vertical scrolling or layout drift.

## User Stories

1. As a Cligram user receiving an incoming Telegram voice call, I want to see a clear, centered modal dialog with the caller's name, so that I immediately know who is calling me.
2. As a Cligram user whose terminal is minimized or unfocused, I want an OS desktop notification and audible bell chime when someone calls, so that I do not miss incoming calls while working in other windows.
3. As a Cligram user with an incoming call, I want to press `[a]` to accept the call immediately, so that two-way voice streaming connects without friction.
4. As a Cligram user with an incoming call, I want to press `[d]` or `[Esc]` to decline the call, so that the call is rejected cleanly and the modal dismisses.
5. As a Cligram user typing a chat message when an incoming call arrives, I want my ongoing typing to be safely suspended, so that typing normal letters does not accidentally trigger hotkeys or get sent as unfinished messages.
6. As a Cligram user on an active call, I want a single-line status bar pinned at the top of my terminal, so that I can glance at call state from any view in the app.
7. As a Cligram user on an active call, I want to see the call duration ticking every second in `MM:SS` format, so that I can accurately monitor call length.
8. As a Cligram user on an active call, I want to see a Connection Badge indicating whether media is direct (`[P2P]`) or relayed (`[Relay]`), so that I know my network privacy level.
9. As a Cligram user on an active call, I want to press `[Alt+M]` to toggle microphone mute, and immediately see a prominent `🔇 MUTED` badge on the status bar when muted, so that I know when I cannot be heard.
10. As a Cligram user on an active call, I want to press `[Alt+H]` from any screen to hang up, so that I can end the call immediately.
11. As a Cligram user actively talking on a call, I want to navigate between chats, groups, and channels without losing the in-call status bar, so that I can look up information during the conversation.
12. As a Cligram user on an active call, I want the chat and sidebar lists to render within the exact terminal height without any scrolling or jitter caused by the status bar, so that the TUI remains crisp and responsive.
13. As a Cligram user whose call terminates or is hung up by the remote peer, I want the status bar to disappear and the layout to restore to standard height automatically, so that the interface cleanly reflects the idle state.

## Implementation Decisions

### Layout Budgeting & Overlay Compositing
- **Incoming Call Overlay Compositing**: The incoming call prompt is treated as a modal overlay composited onto the background view using `bubbletea-overlay` (`overlay.Composite`). This flattens the prompt into the viewport at center coordinates without increasing line counts, guaranteeing that no content scrolls off the terminal screen.
- **In-Call Status Bar Height Budgeting**: When a Call Session is active, layout calculation logic deducts 1 row from available height. The single-line status bar is prepended as part of the primary vertical stack (`callBar` + `sidebar/chat row` + `inputView`), ensuring total lines rendered equals terminal height.
- **Duration Timer & State Synchronization**: A periodic 1-second Bubbletea tick command runs while the call is active, updating elapsed duration computed from the session's start timestamp.

### Multi-Channel Alerting
- **Desktop Alert**: Incoming call signaling invokes the notification service with `beeep.Alert`, simultaneously posting a system notification with caller name and playing the system sound.
- **Terminal Bell**: A terminal bell signal (`\a`) is emitted to trigger terminal emulator alerts.

### Input Isolation & Key Interception
- **Modal Key Isolation**: While the incoming call overlay is visible, normal chat textinput typing is paused. Keypresses of `a` trigger accept, `d` and `esc` trigger decline, and other keys are swallowed to avoid accidental typing.
- **In-Call Global Hotkeys**: `Alt+M` (mute toggle) and `Alt+H` (hangup) are intercepted globally regardless of which UI pane has focus, while preserving alphanumeric typing in text fields.

## Testing Decisions

### Seam Architecture
- **Primary Seam (`ui.Model`)**: All testing is driven through the existing high-level Bubbletea model interface (`Update` and `View`). This tests observable external behavior without mocking internal styling functions:
  - Feeding `types.CallNotification{State: CallStateIncoming}` asserts that `m.View()` contains the centered incoming call dialog and that textinput does not receive subsequent key events.
  - Feeding `tea.KeyMsg{Runes: []rune{'a'}}` verifies that a call acceptance command is returned.
  - Feeding `types.CallNotification{State: CallStateActive}` asserts that `m.View()` contains the top status bar, Connection Badge, and `MM:SS` timer, and that rendered height matches terminal height exactly.
  - Feeding `CallTickMsg` increments elapsed time in `m.View()`.
  - Feeding `Alt+M` and `Alt+H` verifies proper command dispatch and mute badge toggling.
- **Secondary Seam (`notification.Alert`)**: Notification triggers on `*tg.PhoneCallRequested` are verified through unit tests in the client update handler.

### Prior Art
- `internal/ui/call_overlay_test.go` already exercises `ui.Model` with `CallNotification` messages and tests view rendering. The test suite will be expanded to assert exact line heights, compositing behavior, and keyboard suppression.

## Out of Scope
- Group voice chats / video streaming (strictly 1-to-1 audio).
- Custom ringtone audio file playback (desktop alert sound and terminal bell only).
- GUI dialog popups outside the terminal emulator.

## Further Notes
- ADR 0004 (`docs/adr/0004-tui-call-presentation-and-layout-budgeting.md`) documents the architectural rationale for `overlay.Composite` and height budgeting.
- Domain terms `Incoming Call Overlay` and `In-Call Status Bar` are recorded in `GLOSSARY.md`.
