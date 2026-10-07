# 01: Decouple Cgo & Establish Pure-Go VoIP Seam

**What to build:** Ensure Cligram core compiles as a clean pure-Go application without Cgo dependencies or dynamic linking to C libraries (`libntgcalls`, `libopus`, `libcrypto`). Introduce the abstract `SignalingBridge` interface and mockable in-memory IPC transport seam (`net.Conn` / `io.ReadWriter`) so the entire calling state machine can be tested without Cgo or audio hardware.

**Blocked by:** None (can start immediately)

**Status:** resolved

- [x] Core Cligram builds cleanly with `go build ./...` without requiring `CGO_ENABLED=1` or C shared libraries installed on the system.
- [x] A pure-Go `SignalingBridge` interface defines sidecar communication contracts (`CreateCall`, `ConnectCall`, `SetMute`, `StopCall`, and callbacks for signaling, connection state, and audio errors).
- [x] An in-memory IPC transport seam (`net.Pipe`) is established for unit and integration testing.
- [x] Automated unit tests verify that the pure-Go VoIP core compiles and passes tests cleanly.
