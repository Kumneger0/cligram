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

**Call Privacy Guard**:
Pre-flight verification of peer privacy flags (`PhoneCallsPrivate`, `PhoneCallsAvailable`) and interception of MTProto privacy errors (`USER_PRIVACY_RESTRICTED`) that halts call initiation prior to allocating VoIP sidecar resources.
_Avoid_: Call permission check, privacy blocker

**Live Modal Compositor**:
The frame-by-frame overlay rendering architecture that composites dialog modals directly onto the current background view rather than caching stale framebuffer state.
_Avoid_: Stale overlay, modal snapshot

**Message Stream**:
The scrollable TUI viewport rendering conversation history as styled message cards with active message selection, independent of the sidebar dialog list.
_Avoid_: Chat list, message table, conversation list

**Sidebar Tab Bar**:
The category selector at the top of the sidebar partitioning dialogs into Direct Chats, Groups, Channels, and Bots with unread counters and lazy on-demand fetching.
_Avoid_: Folder bar, category menu

**Media Attachment**:
A rich payload (photo, video, audio, voice note, or document) associated with a message, presented in the terminal as an actionable metadata badge.
_Avoid_: File blob, attachment object

**Sender Attribution**:
The explicit resolution of message author identities (`msg.FromID`) into individual participant display names, channel tags (`📢 ChannelTitle`), or administrative badges (`🛡️ Anonymous Admin`) rendered with deterministic user colors in group chats.
_Avoid_: User tag, author label, sender string

**Member Roster**:
An interactive modal list displaying participants of a group or supergroup, supporting search, presence indicators, and direct chat initiation.
_Avoid_: Participant list, member table, group users

**Message Coalescing**:
The visual grouping of consecutive messages sent by the same author within a 5-minute sliding window by omitting redundant author handles and tightening vertical spacing between bubbles.
_Avoid_: Message grouping, message merging, bubble combining

