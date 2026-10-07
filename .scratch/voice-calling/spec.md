# Voice Calling (VoIP) Specification

Status: resolved

## Problem Statement

Cligram users can communicate through text messages, channels, and media attachments, but cannot initiate or receive 1-to-1 voice calls. When other Telegram users call a Cligram user, the call goes unanswered or is dropped without notice, forcing users to switch to official mobile or desktop Telegram apps. Furthermore, compiling native C-based WebRTC/VoIP libraries directly into the core terminal binary would bloat dependencies, break pure-Go cross-compilation, and make the application prone to fatal crashes from native C audio driver faults.

## Solution

Provide end-to-end encrypted 1-to-1 voice calling directly within the Cligram terminal interface. Cligram coordinates the call lifecycle using Telegram's MTProto Diffie-Hellman protocols while delegating media streaming and hardware audio I/O to a lightweight, ephemeral out-of-process helper daemon (`cligram-voip`). When engaged in a call, users can continue reading and typing messages unimpeded, with controls accessible via non-colliding terminal hotkeys (`Alt+M` for mute, `Alt+H` for hangup) and connection privacy state clearly displayed.

## User Stories

1. As a Cligram user viewing a direct contact, I want to press a hotkey to initiate an outgoing voice call, so that I can speak with them directly from my terminal.
2. As a Cligram user engaged in an outgoing call attempt, I want to see a clear status indicator showing that the call is dialing, so that I know the connection is waiting for the peer to answer.
3. As a Cligram user receiving an incoming call, I want an immediate visual prompt displaying the caller's name and acceptance options, so that I can decide whether to answer or decline.
4. As a Cligram user with an incoming call prompt, I want to press `[a]` to accept the call, so that audio streaming connects immediately.
5. As a Cligram user with an incoming call prompt, I want to press `[d]` to decline the call, so that the caller is notified and my terminal UI returns to normal.
6. As a Cligram user on an active call, I want a persistent single-line status bar showing the call duration, so that I know how long the conversation has lasted.
7. As a Cligram user on an active call, I want to see a Connection Badge indicating whether the call is direct (`[P2P]`) or routed through Telegram reflectors (`[Relay]`), so that I am aware of my network IP privacy status.
8. As a Cligram user on an active call, I want to press `Alt+M` to toggle microphone mute, so that the other party cannot hear me when muted.
9. As a Cligram user with a muted microphone, I want the status bar to show a prominent `🔇 MUTED` indicator, so that I do not speak while accidentally muted.
10. As a Cligram user chatting while on a call, I want to type messages containing the letter 'm' or 'h' without accidentally triggering mute or hanging up, so that my text chat remains completely functional.
11. As a Cligram user on an active call, I want to press `Alt+H` from anywhere in the app to hang up, so that I can terminate the call immediately without switching views.
12. As a Cligram user already on a call when a second call arrives, I want the second call to automatically receive a busy signal via the Call-Waiting Busy Policy, so that my current conversation is not interrupted.
13. As a Cligram user who missed a call while busy or away, I want to see a notification and a synchronized call event in the chat history, so that I know who attempted to call.
14. As a privacy-conscious user, I want to configure `calls.force_relay: true` in my configuration file, so that Cligram never exposes my IP address via direct P2P connections regardless of account defaults.
15. As a user on a system without `cligram-voip` installed, I want an informative modal explaining what helper is missing when I try to call, so that I understand how to install the voice feature without encountering a crash.
16. As a user receiving a call without the helper binary installed, I want the call to automatically decline with a desktop notification, so that the caller is not left hanging indefinitely.
17. As a user running modern Linux with PipeWire, I want microphone capture and speaker playback to stream seamlessly via `pw-record` and `pw-play`, so that audio latency remains low.
18. As a user on a system where PipeWire fails or is unavailable, I want Cligram to automatically fall back to PulseAudio or ALSA, so that voice calls continue to function without manual driver configuration.
19. As a user experiencing unexpected network failure during a call, I want Cligram to cleanly tear down the session and display an informative error banner, so that the UI does not hang in a dead call state.
20. As a user who finishes a call, I want the helper daemon to automatically shut down after an idle period, so that my system resources are conserved.

## Implementation Decisions

### Modular Architecture & Boundary
- **Pure Go Core**: The main Cligram application remains 100% pure Go. It retains full ownership of Telegram MTProto signaling, session state, Diffie-Hellman cryptographic exchanges (`MessagesGetDhConfig`, `PhoneRequestCall`, `PhoneAcceptCall`, `PhoneDiscardCall`), and UI rendering.
- **Ephemeral VoIP Sidecar**: All Cgo bindings to `libntgcalls` and native media stream libraries reside in an independent executable (`cmd/cligram-voip`). The sidecar is stateless with respect to Telegram credentials and owns only WebRTC media transport and local audio pipelines.
- **Lifecycle Management**: Cligram core spawns the sidecar as an on-demand child process upon incoming or outgoing call initiation. If no active call exists for 30 seconds, the sidecar shuts down gracefully.

### IPC Transport & Protocol
- **Transport**: Inter-process communication uses an anonymous UNIX stream `socketpair` created by Cligram core and passed to the child process via file descriptor inheritance on FD 3 (`exec.Cmd.ExtraFiles`). No named socket files are created on disk.
- **Protocol**: JSON-RPC 2.0 over newline-delimited stream.
- **Core-to-Sidecar Requests**:
  - `call.create(userId, isOutgoing, dhConfig)`: Initializes media session and returns Diffie-Hellman parameters.
  - `call.connect(userId, authParams, servers, p2pAllowed)`: Connects WebRTC/relay transport and starts audio streaming.
  - `call.set_mute(userId, muted)`: Controls microphone capture stream state.
  - `call.stop(userId)`: Terminates streams and stops audio subprocesses.
- **Sidecar-to-Core Notifications**:
  - `call.on_signaling(userId, data)`: Forwards WebRTC signaling packets to Core for MTProto transmission via `PhoneSendSignalingData`.
  - `call.on_state_change(userId, state, isP2P)`: Emits connection status (`connecting`, `connected`, `disconnected`).
  - `call.on_audio_error(userId, message)`: Reports audio driver or device failures.

### Audio Pipeline & Fallback Chain
- **Supervision**: Audio capture and playback subprocesses are spawned and supervised directly by the VoIP sidecar. Raw PCM audio never crosses the IPC socket.
- **Dynamic Fallback**: The sidecar evaluates sound servers in priority order:
  1. PipeWire: `pw-record` and `pw-play` (format s16, rate 48000, 1 channel).
  2. PulseAudio: `parec` and `pacat` (format s16le, rate 48000, 1 channel).
  3. ALSA: `arecord` and `aplay` (format S16_LE, rate 48000, 1 channel).
  If a launched tool exits with an error within 1 second of startup, the sidecar automatically falls back to the next tool in the chain.

### TUI Presentation & Keybinding Safety
- **Collision-Free Controls**:
  - `Alt+M`: Toggle microphone mute.
  - `Alt+H`: Hang up active call.
  - Regular typing keys (`m`, `h`, `Enter`) remain completely isolated for chat input.
- **Active Call Status Bar**: Rendered as a persistent single-line bar at the top of the terminal viewport:
  `📞 In call with <User> (02:45)  [Relay]  [Alt+M: Mute]  [Alt+H: Hangup]`
  (Dynamically displays `🔇 MUTED` in bold red when muted, and updates the Connection Badge between `[P2P]` and `[Relay]`).
- **Call-Waiting Busy Policy**: Incoming calls during an active call trigger immediate `PhoneCallDiscardReasonBusy` server rejection and a non-intrusive notification.
- **Crash Supervisor**: Kernel `io.EOF` on the IPC socket reader immediately triggers `PhoneCallDiscardReasonDisconnect`, clears the UI overlay to `None`, and renders an error banner.

## Testing Decisions

### Test Quality & Boundaries
- Tests must strictly evaluate observable external behavior at module boundaries, avoiding coupling to internal private state.
- Pure Go tests must run fast without requiring Cgo, physical audio hardware, or external network connections.

### Seams Under Test
1. **The IPC Stream Seam (`net.Conn` / `io.ReadWriter`)**:
   - Primary integration seam. By driving the core's `SignalingBridge` with `net.Pipe()`, automated tests simulate the sidecar daemon.
   - Tests verify: outgoing call initialization, MTProto DH handshake sequencing, incoming call acceptance, mute toggling, signaling packet forwarding, and immediate graceful teardown upon sidecar crash (`io.EOF`).
2. **The TUI Message Seam (`Model.Update`)**:
   - Tests Bubble Tea model updates and view output when receiving call notifications.
   - Tests verify: `Alt+M` toggles mute without interfering with textarea text, `Alt+H` emits hangup command, and `CallOverlay` renders appropriate status text for each state (`None`, `Incoming`, `Active`).
3. **The Sidecar JSON-RPC Seam**:
   - Tests the sidecar JSON-RPC parser and dispatcher against simulated core commands.

### Prior Art
- Existing message list handlers and RPC client update pumps in `internal/ui/` and `internal/telegram/client/`.

## Out of Scope
- Group voice chats / Telegram voice rooms (channels and group megagroups).
- Video calling or screen sharing.
- Software audio equalizer, noise suppression filters, or custom audio effects.
- Multi-call holding or conferencing (automatic busy response is enforced).
- Native GUI call notification popups (relies on existing system notification bridge).

## Further Notes
- Architectural decisions are formally recorded in:
  - `docs/adr/0001-voip-sidecar-process.md`
  - `docs/adr/0002-signaling-and-media-separation.md`
  - `docs/adr/0003-anonymous-socketpair-ipc.md`
- Terminology adheres to `GLOSSARY.md`.
