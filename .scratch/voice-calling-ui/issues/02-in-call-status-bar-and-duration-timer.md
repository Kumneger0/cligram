# 02: In-Call Status Bar, Layout Height Budgeting & Live Duration Timer

**What to build:** Full in-call visual feedback and controls. When a call connects and transitions to active state, a persistent single-line In-Call Status Bar is pinned at the top of the TUI layout: `📞 In call with <Name> (MM:SS) [Relay] [Alt+M: Mute] [Alt+H: Hangup]`. `calculateLayoutDimensions` dynamically deducts 1 row from available height during active calls so the entire UI matches the exact terminal boundary with zero overflow or vertical scrolling. A 1-second interval timer updates the duration in seconds, `Alt+M` toggles mute with a prominent `🔇 MUTED` badge, `Alt+H` hangs up, and the status bar cleanly dismisses upon call termination.

**Blocked by:** 01: Incoming Call Multi-Channel Alerting & Composited Modal Overlay

**Status:** resolved

- [x] Active call displays a persistent single-line status bar at the top with phone icon, peer name, duration timer, Connection Badge, mute status, and hotkey hints.
- [x] Layout height budgeting deducts 1 row from available content height during active calls, ensuring total lines rendered strictly equals terminal height.
- [x] Periodic 1-second `CallTickMsg` updates the duration timer (`MM:SS`).
- [x] Active call notification populates caller name correctly across incoming and outgoing call paths.
- [x] Pressing `Alt+M` sends mute command and dynamically renders `🔇 MUTED` badge.
- [x] Pressing `Alt+H` ends the call and dismisses the status bar, restoring standard layout height.
- [x] Unit tests for `ui.Model` verify layout budgeting, duration progression, mute toggling, and status bar dismissal.
