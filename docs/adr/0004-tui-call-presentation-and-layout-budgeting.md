# TUI Call Presentation and Layout Budgeting

In terminal applications using Bubbletea alt-screen buffers, rendering content whose line count exceeds terminal window height causes terminal vertical scrolling, pushing top lines off-screen into scrollback. To guarantee that VoIP call notifications and in-call controls are always visible without distorting the layout or scrolling off-screen:

1. **Incoming Call Overlay**: Rendered as a modal dialogue and composited over the existing TUI buffer via `github.com/rmhubbert/bubbletea-overlay` (`overlay.Composite`). This centers the dialog, halts accidental text input in the chat box, and preserves background content without expanding total line count.
2. **In-Call Status Bar**: Rendered as a persistent single-line bar pinned at the very top of the layout during an active call. When active, `calculateLayoutDimensions` dynamically deducts 1 row from available height so the combined vertical stack (`callBar` + `sidebar/chat row` + `inputView`) matches terminal height exactly.
3. **Multi-Channel Alerting**: Incoming calls dispatch desktop notifications and audible bell signals via `internal/notification` so the user is alerted even if the terminal window is blurred or minimized.
