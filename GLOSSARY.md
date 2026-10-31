# Cligram

A terminal-based Telegram client written in Go.

## Language

**Call Helper**:
An on-demand child worker process spawned directly from the `cligram` binary (`cligram call-helper`) linked against libntgcalls that manages media streams and WebRTC, communicating with the core client over a local Unix domain socket.
_Avoid_: VoIP sidecar, in-process Cgo, VoIP plugin, native binding

**Audio Pipeline**:
The OS subprocess stream (PipeWire, PulseAudio, or ALSA) piping raw PCM data between the desktop sound server and the call helper.
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
The component in Cligram core that translates Telegram MTProto call events into JSON-RPC messages and forwards them to the call helper.
_Avoid_: Call proxy, socket relay

**Ephemeral Worker**:
A call helper child process spawned on demand upon call initiation via `os.Executable()` and terminated after an idle period following call completion.
_Avoid_: Ephemeral sidecar, daemon service, background worker

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
An opaque WebRTC media signaling payload emitted by the call helper and conveyed across Telegram MTProto via phone.sendSignalingData.
_Avoid_: Control packet, raw payload

**Incoming Call Overlay**:
A modal dialog composited over the TUI viewport upon receiving an incoming Call Session, capturing single-key actions (`[a]` Accept, `[d]` Decline) while preventing accidental chat input keystrokes.
_Avoid_: Call alert, ringing popup

**In-Call Status Bar**:
A persistent single-line bar pinned at the top of the TUI layout throughout an active Call Session, displaying peer identity, live duration timer (`MM:SS`), Connection Badge, and audio controls.
_Avoid_: Call header, active call indicator

**Call Privacy Guard**:
Pre-flight verification of peer privacy flags (`PhoneCallsPrivate`, `PhoneCallsAvailable`) and interception of MTProto privacy errors (`USER_PRIVACY_RESTRICTED`) that halts call initiation prior to allocating call helper resources.
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

**Attachment Preview Chip**:
A styled metadata card rendered above the message input area displaying a staged file attachment (name, size) with single-key detachment (`[Esc]`).
_Avoid_: File badge, attachment label, file box

**Bot Command Autocomplete Overlay**:
An ephemeral card composited directly above the message input bar in bot chats or groups with bots, displaying matched `/command - Description` pairs with `↑`/`↓` selection, `Tab`/`Enter` completion, and `Esc` dismissal.
_Avoid_: Command popup, bot suggestion box, command menu

**Join Preview Modal**:
An interactive modal dialog composited over the live chat view displaying unjoined chat or channel metadata (icon, title, member count, description) and providing single-key action (`[Enter] Join`) to import an invite link or join a public channel.
_Avoid_: Join dialog, channel popup, invite window

**Link Router**:
The in-app URI dispatcher resolving `t.me/+hash` invite links, `t.me/c/<id>/<mid>` private message deep links, and public handles directly within Cligram without opening an external browser.
_Avoid_: URL handler, link opener, web forwarder

**Public Channel Join Prompt**:
A contextual action bar replacing the message input bar when viewing an unjoined public channel or supergroup, displaying single-key join (`[Enter] Join`) or admin approval request (`🔒 Admin approval is required • [Enter] Request to Join`) with real-time feedback.
_Avoid_: Join banner, channel join box, subscription prompt

**Restricted Input Banner**:
An un-focusable status bar replacing the message input area in read-only broadcast channels (`📢 Broadcast Channel (Read-Only)`) or muted groups (`🔇 Muted by admin`), optionally displaying a `[💬 View Comments]` action shortcut when linked discussion comments are available.
_Avoid_: Disabled input box, read-only placeholder, locked text field

**Notification Cascade**:
The tiered evaluation chain (Active Focus Suppression → Silent Flags → Peer NotifySettings → Account-Level Defaults) determining whether an incoming MTProto message triggers an OS desktop alert and whether message preview content is displayed.
_Avoid_: Notification filter, alert pipeline, mute checker

**Reactive Viewport Updates**:
Real-time in-place mutations of active conversation messages (in-place content edits tagged with `(edited)`, immediate message removals on deletion, and per-peer unsent draft restoration) reacting to MTProto events without reloading chat history.
_Avoid_: Viewport reload, chat refresh, message polling

**Service Message**:
An event notification embedded in the conversation stream (e.g. participant joins, leaves, and invite link entries) rendered as a centered, italicized badge distinct from standard user chat bubbles.
_Avoid_: System notification, chat alert, admin event

**Ghost Mode Settings Modal**:
An interactive modal dialog composited over the live chat view displaying AyuGram-style ghost privacy options (`Read Messages`, `Read Stories`, `Send Typing`), supporting arrow/vim navigation, live option toggling, and immediate persistence to disk without requiring a client restart.
_Avoid_: Settings window, preferences dialog, config popup

**Account Switcher Modal**:
An interactive modal dialog composited over the live chat view displaying all configured Telegram accounts on the device with active status badges, hot-swapping active sessions cleanly without quitting, and providing an inline authentication wizard for adding new accounts.
_Avoid_: Account popup, profile selector, multi-login dialog
