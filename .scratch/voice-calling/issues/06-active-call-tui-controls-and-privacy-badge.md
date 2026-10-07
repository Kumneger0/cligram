# 06: Active Call TUI Controls & Connection Privacy Badge

**What to build:** Full in-call TUI user experience. A persistent single-line status bar at the top displays the peer name, live call duration timer, dynamic `[P2P]` vs `[Relay]` Connection Badge (respecting cloud setting and local `calls.force_relay: true` config), `Alt+M` for instant mute with red `🔇 MUTED` badge, and `Alt+H` for immediate hangup. Normal alphanumeric typing in the chat textarea is verified to be completely unaffected by call hotkeys.

**Blocked by:** 05: Incoming Voice Call Flow & Call-Waiting Busy Policy

**Status:** resolved

- [x] Active call displays a single-line status bar at the top: `📞 With: <User> (00:00) [Relay] [Alt+M: Mute] [Alt+H: Hangup]`.
- [x] A tick command updates the call duration timer once per second.
- [x] Pressing `Alt+M` sends `call.set_mute` to the sidecar and dynamically toggles the red `🔇 MUTED` indicator.
- [x] Pressing `Alt+H` terminates the call via `PhoneDiscardCall` and `call.stop`, resetting the status bar to `None`.
- [x] Typing normal characters ('m', 'h', etc.) in the chat message input field while a call is active does not trigger mute or hangup.
- [x] Connection Badge displays `[P2P]` when media is connected directly or `[Relay]` when connected via Telegram reflectors.
- [x] If `calls.force_relay: true` is configured in `config.yaml`, P2P connections are disabled and media always routes via relays.
- [x] Unit tests for `internal/ui` verify key interception, timer updates, and status bar rendering.
