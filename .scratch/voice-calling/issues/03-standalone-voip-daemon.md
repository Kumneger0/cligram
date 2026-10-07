# 03: Standalone cligram-voip Daemon & Audio Fallback Chain

**What to build:** The standalone `cmd/cligram-voip` sidecar binary linked with `libntgcalls` that reads/writes JSON-RPC 2.0 messages over FD 3. It supervises local audio capture and playback subprocesses with a dynamic Audio Fallback Chain (`pw-record`/`pw-play` ➡️ `parec`/`pacat` ➡️ `arecord`/`aplay`), falling back to the next sound server if a tool fails within 1 second of startup.

**Blocked by:** 01: Decouple Cgo & Establish Pure-Go VoIP Seam

**Status:** resolved

- [x] `cmd/cligram-voip` compiles into an executable binary using `make build-voip`.
- [x] Listens on inherited file descriptor 3 for JSON-RPC 2.0 requests (`call.create`, `call.connect`, `call.set_mute`, `call.stop`).
- [x] Connects `ntgcalls` WebRTC signaling and emits asynchronous notifications (`call.on_signaling`, `call.on_state_change`, `call.on_audio_error`).
- [x] Implements the Audio Fallback Chain: probes PipeWire, PulseAudio, and ALSA; automatically cascades to the next tool if a subprocess fails to start or exits immediately.
- [x] Supports a local loopback diagnostic mode (`cligram-voip --loopback`) verifying mic capture and speaker playback without network connections.
- [x] Unit and integration tests verify JSON-RPC request dispatch and audio tool detection logic.
