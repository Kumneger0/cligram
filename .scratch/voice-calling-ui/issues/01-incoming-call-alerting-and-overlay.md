# 01: Incoming Call Multi-Channel Alerting & Composited Modal Overlay

**What to build:** Immediate, non-scrolling visual and audio notification when an incoming voice call arrives. Cligram alerts the user via an OS desktop notification with an audible alert chime (`beeep.Alert`) and terminal bell (`\a`). In the TUI, it renders a centered, non-scrolling Incoming Call Overlay modal dialog composited over the background buffer via `bubbletea-overlay` (`overlay.Composite`). Normal chat input typing is safely paused while ringing; pressing `[a]` accepts the call, while `[d]` or `[Esc]` declines and dismisses the dialog.

**Blocked by:** None (can start immediately)

**Status:** resolved

- [x] Receiving `*tg.PhoneCallRequested` triggers `notification.Alert` with caller name and audible chime, plus terminal bell (`\a`).
- [x] Incoming call visual prompt renders as a centered modal box without overflowing the terminal height or causing vertical scrolling.
- [x] `overlay.Composite` is used to flatten the incoming call modal over the background TUI.
- [x] Pressing `[a]` triggers `AcceptCall`.
- [x] Pressing `[d]` or `[Esc]` triggers `DeclineCall` and restores the standard view.
- [x] Keystrokes during incoming call ringing do not leak into active text inputs or send unintended messages.
- [x] Unit tests for `ui.Model` verify incoming call message handling, compositing, key interception, and accept/decline commands.
