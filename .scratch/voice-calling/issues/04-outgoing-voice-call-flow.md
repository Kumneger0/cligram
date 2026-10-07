# 04: Outgoing Voice Call Flow

**What to build:** Complete end-to-end outgoing 1-to-1 voice calling. A user initiates a call to a contact from the TUI. Cligram Core runs the Telegram Diffie-Hellman handshake (`MessagesGetDhConfig`, `PhoneRequestCall`), connects with `cligram-voip` via the Signaling Bridge, exchanges signaling packets (`PhoneSendSignalingData` ↔ `call.on_signaling`), confirms the call (`PhoneConfirmCall`), starts audio streaming, and updates the TUI to reflect the active call.

**Blocked by:** 02: Sidecar Supervisor, Missing Helper UX & Fault Recovery, 03: Standalone cligram-voip Daemon & Audio Fallback Chain

**Status:** resolved

- [x] Pressing call on a selected user triggers outgoing call negotiation.
- [x] Core fetches DH config, initializes sidecar session via `call.create`, and sends `phone.requestCall` to Telegram.
- [x] On peer acceptance (`PhoneCallAccepted`), core computes encryption parameters (`ExchangeKeys`), sends `phone.confirmCall`, and invokes `call.connect` on the sidecar.
- [x] Bidirectional WebRTC signaling packets are relayed smoothly between Telegram MTProto and the sidecar.
- [x] TUI overlay displays dialing status and seamlessly transitions to active call state when connected.
- [x] Integration tests using `net.Pipe` verify the full outgoing call handshake and signaling packet relay loop.
