# 02: Sidecar Supervisor, Missing Helper UX & Fault Recovery

**What to build:** An on-demand process supervisor in Cligram Core that discovers, spawns, and monitors the ephemeral `cligram-voip` sidecar over an anonymous `socketpair` (FD 3). If the sidecar is not installed, Cligram displays an actionable TUI modal on outgoing call attempts and auto-declines incoming calls with a desktop notification. If the sidecar process crashes, Cligram instantly traps `io.EOF`, issues an MTProto disconnect cleanup, resets the TUI overlay, and shows a red status error banner.

**Blocked by:** 01: Decouple Cgo & Establish Pure-Go VoIP Seam

**Status:** resolved

- [x] Cligram core searches `$PATH`, `~/.local/bin`, and the directory alongside the `cligram` binary for `cligram-voip`.
- [x] If `cligram-voip` is missing, attempting an outgoing call renders an actionable TUI modal explaining how to install the helper, and incoming calls are automatically declined with a user notification.
- [x] When a call begins, Cligram spawns `cligram-voip` as a child process with one end of an anonymous `socketpair` passed on FD 3 (`exec.Cmd.ExtraFiles`).
- [x] Cligram supervisor detects child exit or socket `io.EOF`, immediately dispatches `PhoneDiscardCall(Disconnect)`, resets the TUI call overlay to `None`, and renders an error banner.
- [x] The sidecar is gracefully terminated after 30 seconds of post-call idle time.
- [x] Unit tests using mock processes or closed socket streams verify supervisor startup, missing helper UX, and EOF crash recovery.
