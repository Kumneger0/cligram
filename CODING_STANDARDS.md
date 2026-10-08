# Coding Standards

Engineering conventions and judgement standards for Cligram. Evaluated during code review.

## 1. Concurrency & Bubbletea Architecture

- **Command Closures**: Asynchronous actions in Bubbletea components must be returned as `tea.Cmd` functions (`func() tea.Msg`). Never spawn detached background goroutines (`go func()`) inside command generators or event handlers.
- **Message Dispatch**: All side-effects and background results must be marshaled back into the Bubbletea program loop via typed messages.
- **Goroutine Ownership**: Any long-lived goroutine (e.g. streaming update loops, IPC readers) must have an explicit cancellation channel or context and a well-defined lifecycle tied to its owning struct or supervisor.

## 2. Context Lifecycle & Cancellation

- **Context Propagation**: Always accept `ctx context.Context` as the first parameter for network operations, RPC calls, and signaling handlers.
- **Bounded Operations**: Apply explicit deadlines with `context.WithTimeout` or `context.WithDeadline` for network requests and subprocess handshakes.
- **No Ungrounded Contexts**: Never use `context.TODO()` or `context.Background()` inside non-root call chains.

## 3. Error Handling & Observability

- **Explicit Handling**: Never discard errors silently with blank identifiers (`_ = ...`) in signaling, RPC handlers, or I/O loops.
- **Structured Logging**: Log errors with descriptive context using the repository's logger package (`internal/logger`).
- **User-Facing Feedback**: When an unexpected error interrupts user flow (e.g. sidecar process crash, connection failure), surface actionable feedback via UI notifications or toasts.

## 4. Architectural Boundaries (Cgo Isolation)

- **Pure-Go Core**: All packages under `internal/` and the root binary must build with `CGO_ENABLED=0`. No imports of `C` or Cgo-dependent libraries are allowed in core.
- **Out-of-Process Sidecars**: Native bindings (such as WebRTC or `libntgcalls`) belong strictly inside isolated subcommands under `cmd/` (e.g. `cmd/cligram-voip`).
- **IPC Protocol**: Cross-boundary communication must use structured JSON-RPC over designated file descriptors (such as inherited socketpairs).
