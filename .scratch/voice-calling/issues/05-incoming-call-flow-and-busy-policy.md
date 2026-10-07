# 05: Incoming Voice Call Flow & Call-Waiting Busy Policy

**What to build:** Complete end-to-end incoming call handling. When another Telegram user calls, Cligram renders an incoming call prompt with the caller's name and `[a]` Accept / `[d]` Decline actions. Accepting completes Diffie-Hellman negotiation (`PhoneAcceptCall`) and connects audio via the sidecar. Declining sends `PhoneDiscardCall(Hangup)`. If a call arrives while the user is already on an active call, Cligram automatically rejects it with `PhoneCallDiscardReasonBusy` and logs a missed call notification.

**Blocked by:** 04: Outgoing Voice Call Flow

**Status:** resolved

- [x] Incoming `UpdatePhoneCall(PhoneCallRequested)` triggers an immediate TUI call prompt showing the caller's name.
- [x] Pressing `[a]` accepts the call, runs DH exchange, invokes `call.create` and `call.connect`, and connects two-way audio.
- [x] Pressing `[d]` declines the call, sending `PhoneDiscardCall(Hangup)` and restoring the normal TUI view.
- [x] If an incoming call arrives while `CallOverlay` is already in `CallOverlayActive` state, Cligram automatically responds with `PhoneCallDiscardReasonBusy` without interrupting the active call, and renders a transient missed call alert.
- [x] Integration tests using `net.Pipe` verify incoming call acceptance, user decline, and busy rejection behavior.
