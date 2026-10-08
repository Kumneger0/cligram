# Cligram

A terminal-based Telegram client written in Go.

## Language

**VoIP Sidecar**:
A standalone helper daemon linked against libntgcalls that manages media streams and WebRTC, communicating with the core client over a local Unix domain socket.
_Avoid_: In-process Cgo, VoIP plugin, native binding

**Audio Pipeline**:
The OS subprocess stream (PipeWire, PulseAudio, or ALSA) piping raw PCM data between the desktop sound server and the VoIP sidecar.
_Avoid_: Sound driver, audio engine

**Call Relay**:
Routing voice call media through Telegram's encrypted reflector servers to keep peer IP addresses anonymous.
_Avoid_: Proxy call, TURN call

**Direct P2P Call**:
A peer-to-peer media connection established directly between callers' IP addresses.
_Avoid_: Raw call, peer connection

**Call Session**:
The complete lifecycle of a single voice conversation, encompassing Diffie-Hellman key exchange, media negotiation, active streaming, and teardown.
_Avoid_: Call instance, phone connection

**Signaling Bridge**:
The component in Cligram core that translates Telegram MTProto call events into JSON-RPC messages and forwards them to the VoIP sidecar.
_Avoid_: Call proxy, socket relay

**Ephemeral Sidecar**:
A helper process spawned on demand upon call initiation and terminated after an idle period following call completion.
_Avoid_: Daemon service, background worker

**Connection Badge**:
A TUI status indicator reflecting whether voice call media is currently transported via direct peer-to-peer (`[P2P]`) or routed through Telegram reflectors (`[Relay]`).
_Avoid_: Network tag, call mode

**Call-Waiting Busy Policy**:
The automated rejection of incoming calls with a busy response when the client is already engaged in an active voice call session.
_Avoid_: Call holding, call queuing

**Audio Fallback Chain**:
The prioritized runtime sequence of sound server backends (PipeWire → PulseAudio → ALSA) traversed if the primary audio capture or playback tool fails.
_Avoid_: Audio retry, driver cascade

**Signaling Packet**:
An opaque WebRTC media signaling payload emitted by the VoIP sidecar and conveyed across Telegram MTProto via phone.sendSignalingData.
_Avoid_: Control packet, raw payload

**Incoming Call Overlay**:
A modal dialog composited over the TUI viewport upon receiving an incoming Call Session, capturing single-key actions (`[a]` Accept, `[d]` Decline) while preventing accidental chat input keystrokes.
_Avoid_: Call alert, ringing popup

**In-Call Status Bar**:
A persistent single-line bar pinned at the top of the TUI layout throughout an active Call Session, displaying peer identity, live duration timer (`MM:SS`), Connection Badge, and audio controls.
_Avoid_: Call header, active call indicator



