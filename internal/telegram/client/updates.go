package client

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/updates"
	"github.com/gotd/td/tg"
	"github.com/kumneger0/cligram/internal/telegram/shared"
	"github.com/kumneger0/cligram/ntg"

	"github.com/kumneger0/cligram/internal/telegram/types"
)

// ---------------------------------------------------------------------------
// P2P call state – managed per-user alongside the update handler
// ---------------------------------------------------------------------------

// p2pCallState tracks the state for an active or pending P2P call.
type p2pCallState struct {
	inputCall   *tg.InputPhoneCall
	isOutgoing  bool
	gAOrB       []byte // g_a_hash (outgoing) or g_a_hash (incoming) initially; later replaced with peer's g_a/g_b
	fingerprint int64
	phoneCall   *tg.PhoneCall // set after PhoneCallObj arrives
	waitData    chan error
}

var (
	// ntgClient is a singleton ntgcalls instance.
	ntgClient     *ntg.Client
	ntgClientOnce sync.Once

	// p2pStates tracks per-user call state, keyed by userID.
	p2pStates   = make(map[int64]*p2pCallState)
	p2pStatesMu sync.Mutex

	// waitConnect is used to wait for the connection to be established.
	waitConnect   = make(map[int64]chan error)
	waitConnectMu sync.Mutex
)

func getNtgClient() *ntg.Client {
	ntgClientOnce.Do(func() {
		ntgClient = ntg.Init()

		// Forward signaling data from ntgcalls to Telegram
		ntgClient.OnSignal(func(chatID int64, data []byte) {
			p2pStatesMu.Lock()
			state := p2pStates[chatID]
			p2pStatesMu.Unlock()

			if state != nil && state.inputCall != nil {
				if Cligram == nil {
					slog.Error("Cligram client is nil during OnSignal")
					return
				}
				_, err := Cligram.API().PhoneSendSignalingData(context.TODO(), &tg.PhoneSendSignalingDataRequest{
					Peer: *state.inputCall,
					Data: data,
				})
				if err != nil {
					slog.Error("failed to send signaling data", "error", err)
				}
			}
		})

		// Handle connection state changes
		ntgClient.OnConnectionChange(func(chatID int64, info ntg.NetworkInfo) {
			waitConnectMu.Lock()
			ch := waitConnect[chatID]
			waitConnectMu.Unlock()

			if ch != nil {
				switch info.State {
				case ntg.Connected:
					ch <- nil
				case ntg.Closed, ntg.Failed:
					ch <- fmt.Errorf("connection failed")
				case ntg.Timeout:
					ch <- fmt.Errorf("connection timeout")
				default:
				}
			}
		})
	})
	return ntgClient
}

// pendingIncomingCalls stores data for calls waiting for user accept/decline.
type pendingIncomingCall struct {
	userID     int64
	callID     int64
	accessHash int64
	gAHash     []byte
}

var (
	pendingIncoming   = make(map[int64]*pendingIncomingCall)
	pendingIncomingMu sync.Mutex
)

func newUpdateHandler(updateChannel chan types.Notification) telegram.UpdateHandler {
	dispatcher := tg.NewUpdateDispatcher()
	dispatcher.OnNewChannelMessage(func(ctx context.Context, e tg.Entities, update *tg.UpdateNewChannelMessage) error {
		msg, ok := update.Message.(*tg.Message)
		if !ok {
			return nil
		}

		if peerClass, ok := msg.GetFromID(); ok {
			slog.Debug("received channel message", "peer", peerClass)
			return nil
		}
		return nil
	})

	dispatcher.OnGroupCall(func(ctx context.Context, e tg.Entities, update *tg.UpdateGroupCall) error {
		return nil
	})

	// Handle signaling data updates from Telegram
	dispatcher.OnPhoneCallSignalingData(func(ctx context.Context, e tg.Entities, update *tg.UpdatePhoneCallSignalingData) error {
		// Look up which userID this call belongs to
		p2pStatesMu.Lock()
		for userID, state := range p2pStates {
			if state.inputCall != nil && state.inputCall.ID == update.PhoneCallID {
				p2pStatesMu.Unlock()
				client := getNtgClient()
				err := client.SendSignalingData(userID, update.Data)
				if err != nil {
					slog.Error("failed to send signaling data to ntgcalls", "userID", userID, "error", err)
				}
				return nil
			}
		}
		p2pStatesMu.Unlock()
		return nil
	})

	dispatcher.OnPhoneCall(func(ctx context.Context, e tg.Entities, update *tg.UpdatePhoneCall) error {
		switch u := update.PhoneCall.(type) {
		case *tg.PhoneCallRequested:
			// ---- INCOMING CALL ----
			userID := u.AdminID
			slog.Info("incoming call request", "userID", userID)

			// Store pending call data for user to accept/decline
			pendingIncomingMu.Lock()
			pendingIncoming[userID] = &pendingIncomingCall{
				userID:     userID,
				callID:     u.ID,
				accessHash: u.AccessHash,
				gAHash:     u.GAHash,
			}
			pendingIncomingMu.Unlock()

			// Resolve caller name
			userName := fmt.Sprintf("User %d", userID)
			userInfo, err := shared.GetUserInfo(ctx, *Cligram.API(), userID)
			if err == nil && userInfo != nil {
				userName = userInfo.FirstName
				if userInfo.LastName != "" {
					userName += " " + userInfo.LastName
				}
			}

			// Notify the UI about the incoming call
			select {
			case updateChannel <- types.Notification{
				CallEvent: &types.CallNotification{
					UserID:   userID,
					UserName: userName,
					State:    types.CallStateIncoming,
				},
			}:
			default:
				slog.Warn("update channel is full, dropping incoming call notification")
			}

		case *tg.PhoneCallAccepted:
			// ---- OUTGOING CALL: peer accepted ----
			// This means WE initiated the call (phone.requestCall) and the peer accepted.
			userID := u.ParticipantID
			slog.Info("call accepted by peer", "userID", userID)

			p2pStatesMu.Lock()
			state := p2pStates[userID]
			p2pStatesMu.Unlock()

			if state == nil {
				slog.Warn("PhoneCallAccepted but no p2pState", "userID", userID)
				return nil
			}

			// Update state with the peer's g_b and the InputPhoneCall
			p2pStatesMu.Lock()
			state.gAOrB = u.GB
			state.inputCall = &tg.InputPhoneCall{
				ID:         u.ID,
				AccessHash: u.AccessHash,
			}
			p2pStatesMu.Unlock()

			// Signal the waiting goroutine
			state.waitData <- nil

		case *tg.PhoneCall:
			// ---- CALL ACTIVE (both sides) ----
			slog.Info("call is active")

			// Determine which userID this belongs to
			var userID int64
			p2pStatesMu.Lock()
			for uid, state := range p2pStates {
				if state.inputCall != nil && state.inputCall.ID == u.ID {
					userID = uid
					break
				}
			}
			state := p2pStates[userID]
			p2pStatesMu.Unlock()

			if state == nil {
				// Could be the outgoing side's PhoneCall - try AdminID
				userID = u.AdminID
				p2pStatesMu.Lock()
				state = p2pStates[userID]
				p2pStatesMu.Unlock()
			}

			if state == nil {
				slog.Warn("PhoneCallObj but no p2pState found")
				return nil
			}

			if !state.isOutgoing {
				p2pStatesMu.Lock()
				state.gAOrB = u.GAOrB
				state.fingerprint = u.KeyFingerprint
				state.phoneCall = u
				p2pStatesMu.Unlock()

				client := getNtgClient()

				_, err := client.ExchangeKeys(userID, u.GAOrB, u.KeyFingerprint)
				if err != nil {
					slog.Error("ntg ExchangeKeys (incoming) failed", "error", err)
					return err
				}

				servers := parseRTCServers(u.Connections)
				err = client.ConnectP2P(
					userID,
					servers,
					u.Protocol.LibraryVersions,
					u.P2PAllowed,
				)
				if err != nil {
					slog.Error("ntg ConnectP2P (incoming) failed", "error", err)
					return err
				}
			} else {
				p2pStatesMu.Lock()
				state.phoneCall = u
				p2pStatesMu.Unlock()
			}

			// Notify UI that call is now active
			select {
			case updateChannel <- types.Notification{
				CallEvent: &types.CallNotification{
					UserID: userID,
					State:  types.CallStateActive,
				},
			}:
			default:
				slog.Warn("update channel is full, dropping active call notification")
			}

		case *tg.PhoneCallDiscarded:
			slog.Info("call ended", "duration", u.Duration)

			var discardedUserID int64
			p2pStatesMu.Lock()
			for uid, state := range p2pStates {
				if state.inputCall != nil && state.inputCall.ID == u.ID {
					discardedUserID = uid
					// Signal any waiting goroutines
					select {
					case state.waitData <- fmt.Errorf("call discarded"):
					default:
					}
					delete(p2pStates, uid)

					waitConnectMu.Lock()
					delete(waitConnect, uid)
					waitConnectMu.Unlock()

					client := getNtgClient()
					_ = client.Stop(uid)
					break
				}
			}
			p2pStatesMu.Unlock()

			// Also clean up any pending incoming call
			pendingIncomingMu.Lock()
			for uid := range pendingIncoming {
				if discardedUserID == 0 {
					discardedUserID = uid
				}
				delete(pendingIncoming, uid)
			}
			pendingIncomingMu.Unlock()

			// Notify UI that call has ended
			select {
			case updateChannel <- types.Notification{
				CallEvent: &types.CallNotification{
					UserID:   discardedUserID,
					State:    types.CallStateEnded,
					Duration: u.Duration,
				},
			}:
			default:
				slog.Warn("update channel is full, dropping ended call notification")
			}

		case *tg.PhoneCallWaiting:
			slog.Info("call waiting", "participantID", u.ParticipantID)

			userID := u.ParticipantID

			p2pStatesMu.Lock()
			if state, ok := p2pStates[userID]; ok {
				state.inputCall = &tg.InputPhoneCall{
					ID:         u.ID,
					AccessHash: u.AccessHash,
				}
			}
			p2pStatesMu.Unlock()

		default:
			fmt.Printf("Unknown call state: %#v\n", update.PhoneCall)
		}

		return nil
	})

	dispatcher.OnReadHistoryOutbox(func(ctx context.Context, e tg.Entities, u *tg.UpdateReadHistoryOutbox) error {
		peerClass := u.GetPeer()
		var PeerType types.ChatType
		var peerID string
		switch peer := peerClass.(type) {
		case *tg.PeerUser:
			peerID = strconv.FormatInt(peer.UserID, 10)
			PeerType = types.UserChat
		case *tg.PeerChannel:
			peerID = strconv.FormatInt(peer.ChannelID, 10)
			PeerType = types.ChannelChat
		case *tg.PeerChat:
			peerID = strconv.FormatInt(peer.ChatID, 10)
			PeerType = types.GroupChat
		default:
			slog.Warn("unknown peer type", "peer", peerClass)
			return nil
		}

		maxID := u.GetMaxID()
		notification := types.Notification{
			ReadHistoryOutbox: &types.ReadHistoryOutboxNotification{
				PeerID:   peerID,
				MaxID:    maxID,
				PeerType: PeerType,
			},
		}

		select {
		case updateChannel <- notification:
		default:
			slog.Warn("update channel is full, dropping read history outbox notification")
		}
		return nil
	})

	dispatcher.OnNewMessage(func(ctx context.Context, e tg.Entities, update *tg.UpdateNewMessage) error {
		msg, ok := update.Message.(*tg.Message)
		if !ok {
			return nil
		}
		var peerClass tg.PeerClass
		if p, ok := msg.GetFromID(); ok {
			peerClass = p
		} else if msg.Out {
			if p := msg.GetPeerID(); p != nil {
				peerClass = p
			}
		}

		if peerClass == nil {
			if p := msg.GetPeerID(); p != nil {
				peerClass = p
			}
		}

		if peerClass != nil {
			var fromID string
			switch peer := peerClass.(type) {
			case *tg.PeerUser:
				fromID = strconv.FormatInt(peer.UserID, 10)
			case *tg.PeerChannel:
				fromID = strconv.FormatInt(peer.ChannelID, 10)
			case *tg.PeerChat:
				fromID = strconv.FormatInt(peer.ChatID, 10)
			default:
				slog.Warn("unknown peer type", "peer", peerClass)
				return nil
			}

			notification := types.Notification{
				NewMessage: &types.NewMessageNotification{
					ID:      msg.GetID(),
					FromID:  fromID,
					Message: msg,
				},
			}

			select {
			case updateChannel <- notification:
			default:
				slog.Warn("update channel is full, dropping message")
			}
		} else {
			slog.Error("could not determine peer from message")
		}
		return nil
	})

	dispatcher.OnUserTyping(func(ctx context.Context, e tg.Entities, update *tg.UpdateUserTyping) error {
		userID := update.UserID
		userInfo, err := shared.GetUserInfo(ctx, *Cligram.API(), userID)
		if err != nil {
			return types.NewTelegramError(types.ErrorCodeUserNotFound, err.Error(), nil)
		}

		userInfo.IsOnline = true
		userInfo.IsTyping = true
		notification := types.Notification{
			UserTyping: &types.UserTypingNotification{
				User: *userInfo,
			},
		}

		select {
		case updateChannel <- notification:
		default:
			slog.Warn("update channel is full, dropping typing notification")
		}
		return nil
	})

	dispatcher.OnUserStatus(func(ctx context.Context, e tg.Entities, update *tg.UpdateUserStatus) error {
		userID := update.UserID
		userInfo, err := shared.GetUserInfo(ctx, *Cligram.API(), userID)
		if err != nil {
			return types.NewTelegramError(types.ErrorCodeUserNotFound, err.Error(), nil)
		}

		if userInfo == nil {
			return types.NewUserNotFoundError(userID)
		}

		var lastSeen time.Time
		var isOnline bool

		switch status := update.Status.(type) {
		case *tg.UserStatusOffline:
			lastSeen = time.Unix(int64(status.WasOnline), 0)
			isOnline = false
		case *tg.UserStatusOnline:
			lastSeen = time.Now()
			isOnline = true
		default:
			lastSeen = time.Time{}
			isOnline = false
		}

		userInfo.IsOnline = isOnline

		notification := types.Notification{
			UserStatus: &types.UserStatusNotification{
				UserInfo: *userInfo,
				Status: types.UserStatus{
					IsOnline: isOnline,
					LastSeen: lastSeen,
				},
			},
		}

		select {
		case updateChannel <- notification:
		default:
			slog.Warn("update channel is full, dropping status notification")
		}
		return nil
	})

	return updates.New(updates.Config{
		Handler: dispatcher,
	})
}

// parseRTCServers converts Telegram PhoneConnectionClass to ntg.RTCServer.
func parseRTCServers(connections []tg.PhoneConnectionClass) []ntg.RTCServer {
	var servers []ntg.RTCServer
	for _, conn := range connections {
		switch c := conn.(type) {
		case *tg.PhoneConnectionWebrtc:
			servers = append(servers, ntg.RTCServer{
				ID:       uint64(c.ID),
				IPv4:     c.IP,
				IPv6:     c.Ipv6,
				Port:     uint16(c.Port),
				Username: c.Username,
				Password: c.Password,
				IsTURN:   c.Turn,
				IsSTUN:   c.Stun,
				IsTCP:    false,
			})
		case *tg.PhoneConnection:
			servers = append(servers, ntg.RTCServer{
				ID:      uint64(c.ID),
				IPv4:    c.IP,
				IPv6:    c.Ipv6,
				Port:    uint16(c.Port),
				PeerTag: c.PeerTag,
				IsTCP:   c.TCP,
			})
		}
	}
	return servers
}

func InitiateP2PCall(ctx context.Context, userID int64, accessHash int64) (err error) {
	client := getNtgClient()

	// Cleanup on failure if the call wasn't fully established
	defer func() {
		if err != nil {
			p2pStatesMu.Lock()
			delete(p2pStates, userID)
			p2pStatesMu.Unlock()

			waitConnectMu.Lock()
			delete(waitConnect, userID)
			waitConnectMu.Unlock()

			_ = client.Stop(userID)
		}
	}()

	dhRaw, err := Cligram.API().MessagesGetDhConfig(ctx, &tg.MessagesGetDhConfigRequest{
		Version:      0,
		RandomLength: 256,
	})
	if err != nil {
		return fmt.Errorf("MessagesGetDhConfig: %w", err)
	}
	dhConfig, ok := dhRaw.(*tg.MessagesDhConfig)
	if !ok {
		return fmt.Errorf("unexpected DH config type: %T", dhRaw)
	}

	if err := client.CreateP2P(userID); err != nil {
		return fmt.Errorf("ntg CreateP2P: %w", err)
	}

	// 2.5 Set up real audio streams
	if err := client.SetupRealAudio(userID); err != nil {
		return fmt.Errorf("ntg SetupRealAudio: %w", err)
	}

	// 3. InitExchange (no gAHash for outgoing calls → ntgcalls returns g_a_hash)
	gAHash, err := client.InitExchange(
		userID,
		int32(dhConfig.G),
		dhConfig.P,
		dhConfig.Random,
		nil,
	)
	if err != nil {
		return fmt.Errorf("ntg InitExchange: %w", err)
	}

	p2pStatesMu.Lock()
	p2pStates[userID] = &p2pCallState{
		isOutgoing: true,
		gAOrB:      gAHash,
		waitData:   make(chan error, 1),
	}
	p2pStatesMu.Unlock()

	protocol := ntg.GetProtocol()
	_, err = Cligram.API().PhoneRequestCall(ctx, &tg.PhoneRequestCallRequest{
		UserID: &tg.InputUser{
			UserID:     userID,
			AccessHash: accessHash,
		},
		RandomID: int(time.Now().UnixNano()),
		GAHash:   gAHash,
		Protocol: tg.PhoneCallProtocol{
			MinLayer:        int(protocol.MinLayer),
			MaxLayer:        int(protocol.MaxLayer),
			LibraryVersions: protocol.Versions,
			UDPP2P:          protocol.UDPP2P,
			UDPReflector:    protocol.UDPReflector,
		},
	})
	if err != nil {
		return fmt.Errorf("phone.requestCall: %w", err)
	}

	p2pStatesMu.Lock()
	state := p2pStates[userID]
	p2pStatesMu.Unlock()

	select {
	case err := <-state.waitData:
		if err != nil {
			return err
		}
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(30 * time.Second):
		return fmt.Errorf("timed out waiting for call acceptance")
	}

	p2pStatesMu.Lock()
	peerGB := state.gAOrB
	inputCall := state.inputCall
	p2pStatesMu.Unlock()

	authParams, err := client.ExchangeKeys(userID, peerGB, 0)
	if err != nil {
		return fmt.Errorf("ntg ExchangeKeys: %w", err)
	}

	confirmRes, err := Cligram.API().PhoneConfirmCall(ctx, &tg.PhoneConfirmCallRequest{
		Peer:           *inputCall,
		GA:             authParams.GAOrB,
		KeyFingerprint: authParams.KeyFingerprint,
		Protocol: tg.PhoneCallProtocol{
			MinLayer:        int(protocol.MinLayer),
			MaxLayer:        int(protocol.MaxLayer),
			LibraryVersions: protocol.Versions,
			UDPP2P:          protocol.UDPP2P,
			UDPReflector:    protocol.UDPReflector,
		},
	})
	if err != nil {
		return fmt.Errorf("phone.confirmCall: %w", err)
	}

	phoneCallObj, ok := confirmRes.PhoneCall.(*tg.PhoneCall)
	if !ok {
		return fmt.Errorf("unexpected phone call type after confirmCall: %T", confirmRes)
	}

	// 9. Connect P2P immediately using the connections from confirmCall result
	servers := parseRTCServers(phoneCallObj.Connections)

	waitConnectMu.Lock()
	waitConnect[userID] = make(chan error, 1)
	waitConnectMu.Unlock()

	// Ensure waitConnect is cleaned up eventually
	defer func() {
		waitConnectMu.Lock()
		delete(waitConnect, userID)
		waitConnectMu.Unlock()
	}()

	err = client.ConnectP2P(
		userID,
		servers,
		phoneCallObj.Protocol.LibraryVersions,
		phoneCallObj.P2PAllowed,
	)
	if err != nil {
		return fmt.Errorf("ntg ConnectP2P: %w", err)
	}

	// 10. Wait for the connection to be established
	waitConnectMu.Lock()
	ch := waitConnect[userID]
	waitConnectMu.Unlock()

	if ch != nil {
		select {
		case err = <-ch:
			if err != nil {
				return err
			}
			slog.Info("P2P connected successfully")
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(30 * time.Second):
			return fmt.Errorf("timed out waiting for P2P connection")
		}
	}

	return nil
}

// AcceptIncomingCall accepts a pending incoming P2P call.
func AcceptIncomingCall(ctx context.Context, userID int64) error {
	pendingIncomingMu.Lock()
	pending := pendingIncoming[userID]
	delete(pendingIncoming, userID)
	pendingIncomingMu.Unlock()

	if pending == nil {
		return fmt.Errorf("no pending incoming call from user %d", userID)
	}

	client := getNtgClient()

	// Get DH config from Telegram
	dhRaw, err := Cligram.API().MessagesGetDhConfig(ctx, &tg.MessagesGetDhConfigRequest{
		Version:      0,
		RandomLength: 256,
	})
	if err != nil {
		return fmt.Errorf("MessagesGetDhConfig: %w", err)
	}
	dhConfig, ok := dhRaw.(*tg.MessagesDhConfig)
	if !ok {
		return fmt.Errorf("unexpected DH config type: %T", dhRaw)
	}

	// 1. CreateP2P
	if err := client.CreateP2P(userID); err != nil {
		return fmt.Errorf("ntg CreateP2P: %w", err)
	}

	// 2. Set up real audio streams
	if err := client.SetupRealAudio(userID); err != nil {
		return fmt.Errorf("ntg SetupRealAudio: %w", err)
	}

	// 3. InitExchange with the caller's gAHash
	gB, err := client.InitExchange(
		userID,
		int32(dhConfig.G),
		dhConfig.P,
		dhConfig.Random,
		pending.gAHash,
	)
	if err != nil {
		return fmt.Errorf("ntg InitExchange: %w", err)
	}

	// Save state
	p2pStatesMu.Lock()
	p2pStates[userID] = &p2pCallState{
		inputCall: &tg.InputPhoneCall{
			ID:         pending.callID,
			AccessHash: pending.accessHash,
		},
		isOutgoing: false,
		gAOrB:      gB,
		waitData:   make(chan error, 1),
	}
	p2pStatesMu.Unlock()

	// 4. Accept the call via Telegram API
	protocol := ntg.GetProtocol()
	_, err = Cligram.API().PhoneAcceptCall(ctx, &tg.PhoneAcceptCallRequest{
		Peer: tg.InputPhoneCall{
			ID:         pending.callID,
			AccessHash: pending.accessHash,
		},
		GB: gB,
		Protocol: tg.PhoneCallProtocol{
			MinLayer:        int(protocol.MinLayer),
			MaxLayer:        int(protocol.MaxLayer),
			LibraryVersions: protocol.Versions,
			UDPP2P:          protocol.UDPP2P,
			UDPReflector:    protocol.UDPReflector,
		},
	})
	if err != nil {
		return fmt.Errorf("phone.acceptCall: %w", err)
	}

	return nil
}

// DeclineIncomingCall declines a pending incoming P2P call.
func DeclineIncomingCall(ctx context.Context, userID int64) error {
	pendingIncomingMu.Lock()
	pending := pendingIncoming[userID]
	delete(pendingIncoming, userID)
	pendingIncomingMu.Unlock()

	if pending == nil {
		return fmt.Errorf("no pending incoming call from user %d", userID)
	}

	_, err := Cligram.API().PhoneDiscardCall(ctx, &tg.PhoneDiscardCallRequest{
		Peer: tg.InputPhoneCall{
			ID:         pending.callID,
			AccessHash: pending.accessHash,
		},
		Reason: &tg.PhoneCallDiscardReasonHangup{},
	})
	return err
}

// HangupCall terminates an active P2P call.
func HangupCall(ctx context.Context, userID int64) error {
	p2pStatesMu.Lock()
	state := p2pStates[userID]
	p2pStatesMu.Unlock()

	if state == nil || state.inputCall == nil {
		return fmt.Errorf("no active call with user %d", userID)
	}

	client := getNtgClient()
	_ = client.Stop(userID)

	_, err := Cligram.API().PhoneDiscardCall(ctx, &tg.PhoneDiscardCallRequest{
		Peer:   *state.inputCall,
		Reason: &tg.PhoneCallDiscardReasonHangup{},
	})

	p2pStatesMu.Lock()
	delete(p2pStates, userID)
	p2pStatesMu.Unlock()

	waitConnectMu.Lock()
	delete(waitConnect, userID)
	waitConnectMu.Unlock()

	return err
}

// ToggleCallMute toggles mute state for an active call. Returns the new muted state.
func ToggleCallMute(userID int64) (bool, error) {
	client := getNtgClient()
	state, err := client.GetState(userID)
	if err != nil {
		return false, err
	}
	if state.Muted {
		_, err = client.UnMute(userID)
		return false, err
	}
	_, err = client.Mute(userID)
	return true, err
}
