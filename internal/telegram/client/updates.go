package client

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/updates"
	"github.com/gotd/td/tg"
	"github.com/kumneger0/cligram/internal/config"
	"github.com/kumneger0/cligram/internal/telegram/shared"
	"github.com/kumneger0/cligram/internal/telegram/types"
	"github.com/kumneger0/cligram/internal/voip"
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
	voipSupervisor = voip.NewSupervisor("cligram-voip")

	currentBridge   voip.SignalingBridge
	currentBridgeMu sync.Mutex

	// updateChannelGlobal is stored to allow background bridge callbacks to notify the UI.
	updateChannelGlobal   chan types.Notification
	updateChannelGlobalMu sync.Mutex

	// p2pStates tracks per-user call state, keyed by userID.
	p2pStates   = make(map[int64]*p2pCallState)
	p2pStatesMu sync.Mutex

	// waitConnect is used to wait for the connection to be established.
	waitConnect   = make(map[int64]chan error)
	waitConnectMu sync.Mutex

	callMuted   bool
	callMutedMu sync.Mutex

	findSidecarBinary = voip.FindSidecarBinary
)

func getVoipBridge(ctx context.Context) (voip.SignalingBridge, error) {
	currentBridgeMu.Lock()
	defer currentBridgeMu.Unlock()

	voipSupervisor.CancelIdleTimer()

	if currentBridge != nil {
		return currentBridge, nil
	}

	bridge, err := voipSupervisor.Start(ctx)
	if err != nil {
		return nil, err
	}

	// Wire callbacks
	sigCtx, sigCancel := context.WithCancel(context.Background())
	signalingQueue := make(chan voip.SignalingEvent, 128)
	go func() {
		for {
			select {
			case <-sigCtx.Done():
				return
			case event := <-signalingQueue:
				p2pStatesMu.Lock()
				state := p2pStates[event.UserID]
				if state == nil && len(p2pStates) == 1 {
					for uid, s := range p2pStates {
						state = s
						slog.Info("matched outbound signaling to single active call", "eventUserID", event.UserID, "actualUserID", uid)
						break
					}
				}
				p2pStatesMu.Unlock()

				if state == nil {
					slog.Warn("outbound signaling dropped: no active call state found", "userID", event.UserID)
					continue
				}
				if state.inputCall == nil {
					slog.Warn("outbound signaling dropped: call state has nil inputCall", "userID", event.UserID)
					continue
				}
				if Cligram == nil {
					slog.Warn("outbound signaling dropped: Cligram client is nil", "userID", event.UserID)
					continue
				}

				sendCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				slog.Info("sending outbound signaling data to Telegram", "userID", event.UserID, "callID", state.inputCall.ID, "dataLen", len(event.Data))
				_, err := Cligram.API().PhoneSendSignalingData(sendCtx, &tg.PhoneSendSignalingDataRequest{
					Peer: *state.inputCall,
					Data: event.Data,
				})
				cancel()
				if err != nil {
					slog.Error("failed to send signaling data to Telegram", "userID", event.UserID, "callID", state.inputCall.ID, "error", err)
				} else {
					slog.Info("successfully sent signaling data to Telegram", "userID", event.UserID, "callID", state.inputCall.ID)
				}
			}
		}
	}()

	bridge.OnSignaling(func(event voip.SignalingEvent) {
		slog.Info("signaling event received from sidecar", "userID", event.UserID, "dataLen", len(event.Data))
		select {
		case <-sigCtx.Done():
			return
		case signalingQueue <- event:
		default:
			slog.Warn("signaling queue full, dropping outbound packet", "userID", event.UserID)
		}
	})

	bridge.OnStateChange(func(event voip.StateChangeEvent) {
		waitConnectMu.Lock()
		ch := waitConnect[event.UserID]
		waitConnectMu.Unlock()

		if ch != nil {
			switch event.State {
			case "connected":
				select {
				case ch <- nil:
				default:
				}
			case "failed":
				select {
				case ch <- fmt.Errorf("connection failed"):
				default:
				}
			case "timeout":
				select {
				case ch <- fmt.Errorf("connection timeout"):
				default:
				}
			}
		}

		if event.State == "connected" {
			updateChannelGlobalMu.Lock()
			chGlobal := updateChannelGlobal
			updateChannelGlobalMu.Unlock()

			if chGlobal != nil {
				userName := fmt.Sprintf("User %d", event.UserID)
				if Cligram != nil && Cligram.API() != nil {
					userInfo, err := shared.GetUserInfo(sigCtx, *Cligram.API(), event.UserID)
					if err == nil && userInfo != nil {
						userName = userInfo.FirstName
						if userInfo.LastName != "" {
							userName += " " + userInfo.LastName
						}
					}
				}
				select {
				case chGlobal <- types.Notification{
					CallEvent: &types.CallNotification{
						UserID:   event.UserID,
						UserName: userName,
						State:    types.CallStateActive,
						IsRelay:  !event.IsP2P,
					},
				}:
				default:
				}
			}
		}
	})

	bridge.OnAudioError(func(event voip.AudioErrorEvent) {
		slog.Warn("voip audio error reported", "userID", event.UserID, "error", event.Message)
	})

	bridge.OnClose(func(err error) {
		sigCancel()
		slog.Warn("voip bridge connection closed", "error", err)
		currentBridgeMu.Lock()
		currentBridge = nil
		currentBridgeMu.Unlock()

		// Cleanup any active call with Disconnect reason
		p2pStatesMu.Lock()
		uids := make([]int64, 0, len(p2pStates))
		for uid := range p2pStates {
			uids = append(uids, uid)
		}
		p2pStatesMu.Unlock()

		for _, uid := range uids {
			go func(userID int64) {
				p2pStatesMu.Lock()
				state := p2pStates[userID]
				delete(p2pStates, userID)
				p2pStatesMu.Unlock()

				if state != nil && state.inputCall != nil && Cligram != nil {
					_, discErr := Cligram.API().PhoneDiscardCall(context.Background(), &tg.PhoneDiscardCallRequest{
						Peer:   *state.inputCall,
						Reason: &tg.PhoneCallDiscardReasonDisconnect{},
					})
					if discErr != nil {
						slog.Error("failed to discard call on sidecar disconnect", "userID", userID, "error", discErr)
					}
				}

				updateChannelGlobalMu.Lock()
				chGlobal := updateChannelGlobal
				updateChannelGlobalMu.Unlock()

				if chGlobal != nil {
					select {
					case chGlobal <- types.Notification{
						CallEvent: &types.CallNotification{
							UserID: userID,
							State:  types.CallStateEnded,
							Err:    fmt.Errorf("voip helper disconnected unexpectedly"),
						},
					}:
					default:
					}
				}
			}(uid)
		}
	})

	currentBridge = bridge
	return bridge, nil
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

// SetGlobalUpdateChannel explicitly sets the global notification channel used by background voip callbacks.
func SetGlobalUpdateChannel(ch chan types.Notification) {
	updateChannelGlobalMu.Lock()
	updateChannelGlobal = ch
	updateChannelGlobalMu.Unlock()
}

// GetGlobalUpdateChannel returns the current global notification channel.
func GetGlobalUpdateChannel() chan types.Notification {
	updateChannelGlobalMu.Lock()
	defer updateChannelGlobalMu.Unlock()
	return updateChannelGlobal
}

func newUpdateHandler(updateChannel chan types.Notification) telegram.UpdateHandler {
	SetGlobalUpdateChannel(updateChannel)
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
		slog.Info("received incoming signaling data from Telegram", "phoneCallID", update.PhoneCallID, "dataLen", len(update.Data))
		p2pStatesMu.Lock()
		var targetUserID int64
		for userID, state := range p2pStates {
			if state.inputCall != nil && state.inputCall.ID == update.PhoneCallID {
				targetUserID = userID
				break
			}
		}
		if targetUserID == 0 && len(p2pStates) == 1 {
			for userID := range p2pStates {
				targetUserID = userID
				slog.Info("matched incoming signaling data to single active call", "userID", targetUserID, "phoneCallID", update.PhoneCallID)
				break
			}
		}
		p2pStatesMu.Unlock()

		if targetUserID == 0 {
			slog.Warn("dropping incoming signaling data: no active call matches", "phoneCallID", update.PhoneCallID)
			return nil
		}

		currentBridgeMu.Lock()
		bridge := currentBridge
		currentBridgeMu.Unlock()
		if bridge != nil {
			slog.Info("forwarding incoming signaling data to voip bridge", "userID", targetUserID, "dataLen", len(update.Data))
			if err := bridge.SendSignaling(ctx, targetUserID, update.Data); err != nil {
				slog.Error("failed to forward signaling data to bridge", "userID", targetUserID, "error", err)
			}
		} else {
			slog.Warn("voip bridge is nil, cannot forward incoming signaling data", "userID", targetUserID)
		}
		return nil
	})

	dispatcher.OnPhoneCall(func(ctx context.Context, e tg.Entities, update *tg.UpdatePhoneCall) error {
		return handlePhoneCall(ctx, update.PhoneCall, updateChannel)
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
		if msg.Out {
			// For outgoing messages, PeerID is the recipient/destination dialog peer
			if p := msg.GetPeerID(); p != nil {
				peerClass = p
			} else if p, ok := msg.GetFromID(); ok {
				peerClass = p
			}
		} else {
			// For incoming messages in channels or groups, PeerID is the conversation container
			if p := msg.GetPeerID(); p != nil {
				switch p.(type) {
				case *tg.PeerChannel, *tg.PeerChat:
					peerClass = p
				}
			}
			// In private chats, FromID is the sender
			if peerClass == nil {
				if p, ok := msg.GetFromID(); ok {
					peerClass = p
				} else if p := msg.GetPeerID(); p != nil {
					peerClass = p
				}
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
					PeerID:  fromID,
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

// parseRTCServers converts Telegram PhoneConnectionClass to voip.RTCServer.
func parseRTCServers(connections []tg.PhoneConnectionClass) []voip.RTCServer {
	var servers []voip.RTCServer
	for _, conn := range connections {
		switch c := conn.(type) {
		case *tg.PhoneConnectionWebrtc:
			servers = append(servers, voip.RTCServer{
				ID:       c.ID,
				IP:       c.IP,
				IPv6:     c.Ipv6,
				Port:     c.Port,
				Username: c.Username,
				Password: c.Password,
				Turn:     c.Turn,
				Stun:     c.Stun,
			})
		case *tg.PhoneConnection:
			servers = append(servers, voip.RTCServer{
				ID:      c.ID,
				IP:      c.IP,
				IPv6:    c.Ipv6,
				Port:    c.Port,
				PeerTag: c.PeerTag,
				Turn:    false,
				Stun:    false,
				TCP:     c.TCP,
			})
		}
	}
	return servers
}

func buildPhoneCallProtocol(res *voip.CreateCallResult, forceRelay bool) tg.PhoneCallProtocol {
	minLayer := int(res.MinLayer)
	if minLayer == 0 {
		minLayer = 92
	}
	maxLayer := int(res.MaxLayer)
	if maxLayer == 0 {
		maxLayer = 92
	}
	libVersions := res.LibraryVersions
	if len(libVersions) == 0 {
		libVersions = []string{"8.0.0", "9.0.0"}
	}
	udpP2P := res.UDPP2P
	if res.MinLayer == 0 {
		udpP2P = true
	}
	udpReflector := res.UDPReflector
	if res.MinLayer == 0 {
		udpReflector = true
	}

	return tg.PhoneCallProtocol{
		MinLayer:        minLayer,
		MaxLayer:        maxLayer,
		UDPP2P:          udpP2P && !forceRelay,
		UDPReflector:    udpReflector,
		LibraryVersions: libVersions,
	}
}

func InitiateP2PCall(ctx context.Context, userID int64, accessHash int64, updateChannel chan types.Notification) (err error) {
	if updateChannel == nil {
		updateChannel = GetGlobalUpdateChannel()
	}

	bridge, err := getVoipBridge(ctx)
	if err != nil {
		if errors.Is(err, voip.ErrHelperNotFound) {
			slog.Warn("cligram-voip helper not found")
			if updateChannel != nil {
				select {
				case updateChannel <- types.Notification{
					CallEvent: &types.CallNotification{
						UserID: userID,
						State:  types.CallStateMissingHelper,
					},
				}:
				default:
				}
			}
		}
		return err
	}

	// Cleanup on failure if the call wasn't fully established
	defer func() {
		if err != nil {
			p2pStatesMu.Lock()
			delete(p2pStates, userID)
			p2pStatesMu.Unlock()

			waitConnectMu.Lock()
			delete(waitConnect, userID)
			waitConnectMu.Unlock()

			if bridge != nil {
				if stopErr := bridge.StopCall(context.Background(), voip.StopCallParams{UserID: userID}); stopErr != nil {
					slog.Warn("failed to stop call during cleanup", "userID", userID, "error", stopErr)
				}
			}

			if !errors.Is(err, voip.ErrHelperNotFound) {
				if updateChannel != nil {
					select {
					case updateChannel <- types.Notification{
						CallEvent: &types.CallNotification{
							UserID: userID,
							State:  types.CallStateEnded,
						},
					}:
					default:
					}
				}
			}
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

	res, err := bridge.CreateCall(ctx, voip.CreateCallParams{
		UserID:     userID,
		IsOutgoing: true,
		DHG:        int32(dhConfig.G),
		DHP:        dhConfig.P,
		DHRandom:   dhConfig.Random,
	})
	if err != nil {
		return fmt.Errorf("bridge CreateCall: %w", err)
	}

	p2pStatesMu.Lock()
	p2pStates[userID] = &p2pCallState{
		isOutgoing: true,
		gAOrB:      res.GAOrB,
		waitData:   make(chan error, 1),
	}
	p2pStatesMu.Unlock()

	forceRelay := config.GetConfig().Calls.ForceRelay
	protocol := buildPhoneCallProtocol(res, forceRelay)

	reqCallRes, err := Cligram.API().PhoneRequestCall(ctx, &tg.PhoneRequestCallRequest{
		UserID: &tg.InputUser{
			UserID:     userID,
			AccessHash: accessHash,
		},
		RandomID: int(time.Now().UnixNano()),
		GAHash:   res.GAOrB,
		Protocol: protocol,
	})
	if err != nil {
		return fmt.Errorf("phone.requestCall: %w", err)
	}

	if phoneCallWaiting, ok := reqCallRes.PhoneCall.(*tg.PhoneCallWaiting); ok {
		p2pStatesMu.Lock()
		if state, ok := p2pStates[userID]; ok && state != nil {
			state.inputCall = &tg.InputPhoneCall{
				ID:         phoneCallWaiting.ID,
				AccessHash: phoneCallWaiting.AccessHash,
			}
			slog.Info("initialized outgoing call with inputCall from phone.requestCall", "userID", userID, "callID", phoneCallWaiting.ID)
		}
		p2pStatesMu.Unlock()
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

	authParams, err := bridge.ExchangeKeys(ctx, voip.ExchangeKeysParams{
		UserID:         userID,
		GAOrB:          peerGB,
		KeyFingerprint: 0,
	})
	if err != nil {
		return fmt.Errorf("bridge ExchangeKeys: %w", err)
	}

	confirmRes, err := Cligram.API().PhoneConfirmCall(ctx, &tg.PhoneConfirmCallRequest{
		Peer:           *inputCall,
		GA:             authParams.GAOrB,
		KeyFingerprint: authParams.KeyFingerprint,
		Protocol:       protocol,
	})
	if err != nil {
		return fmt.Errorf("phone.confirmCall: %w", err)
	}

	phoneCallObj, ok := confirmRes.PhoneCall.(*tg.PhoneCall)
	if !ok {
		return fmt.Errorf("unexpected phone call type after confirmCall: %T", confirmRes)
	}

	p2pStatesMu.Lock()
	if state, ok := p2pStates[userID]; ok && state != nil {
		state.inputCall = &tg.InputPhoneCall{
			ID:         phoneCallObj.ID,
			AccessHash: phoneCallObj.AccessHash,
		}
		state.phoneCall = phoneCallObj
		slog.Info("updated outgoing call with confirmed inputCall", "userID", userID, "callID", phoneCallObj.ID)
	}
	p2pStatesMu.Unlock()

	// Notify UI and handle confirmed call state immediately
	if updateChannel != nil {
		if err := handlePhoneCall(ctx, phoneCallObj, updateChannel); err != nil {
			slog.Warn("handlePhoneCall failed for confirmed outgoing call", "userID", userID, "error", err)
		}
	}

	// Connect P2P immediately using the connections from confirmCall result
	servers := parseRTCServers(phoneCallObj.Connections)

	waitConnectMu.Lock()
	waitConnect[userID] = make(chan error, 1)
	waitConnectMu.Unlock()

	defer func() {
		waitConnectMu.Lock()
		delete(waitConnect, userID)
		waitConnectMu.Unlock()
	}()

	forceRelay = config.GetConfig().Calls.ForceRelay
	err = bridge.ConnectCall(ctx, voip.ConnectCallParams{
		UserID:          userID,
		Servers:         servers,
		LibraryVersions: phoneCallObj.Protocol.LibraryVersions,
		P2PAllowed:      phoneCallObj.P2PAllowed && !forceRelay,
	})
	if err != nil {
		return fmt.Errorf("bridge ConnectCall: %w", err)
	}

	// Wait for the connection to be established
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

	bridge, err := getVoipBridge(ctx)
	if err != nil {
		return err
	}

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

	res, err := bridge.CreateCall(ctx, voip.CreateCallParams{
		UserID:     userID,
		IsOutgoing: false,
		DHG:        int32(dhConfig.G),
		DHP:        dhConfig.P,
		DHRandom:   dhConfig.Random,
		GAOrB:      pending.gAHash,
	})
	if err != nil {
		return fmt.Errorf("bridge CreateCall: %w", err)
	}

	// Save state
	p2pStatesMu.Lock()
	p2pStates[userID] = &p2pCallState{
		inputCall: &tg.InputPhoneCall{
			ID:         pending.callID,
			AccessHash: pending.accessHash,
		},
		isOutgoing: false,
		gAOrB:      res.GAOrB,
		waitData:   make(chan error, 1),
	}
	p2pStatesMu.Unlock()

	forceRelay := config.GetConfig().Calls.ForceRelay
	protocol := buildPhoneCallProtocol(res, forceRelay)

	acceptRes, err := Cligram.API().PhoneAcceptCall(ctx, &tg.PhoneAcceptCallRequest{
		Peer: tg.InputPhoneCall{
			ID:         pending.callID,
			AccessHash: pending.accessHash,
		},
		GB:       res.GAOrB,
		Protocol: protocol,
	})
	if err != nil {
		return fmt.Errorf("phone.acceptCall: %w", err)
	}

	if acceptRes != nil {
		if waiting, ok := acceptRes.PhoneCall.(*tg.PhoneCallWaiting); ok {
			p2pStatesMu.Lock()
			if state, ok := p2pStates[userID]; ok && state != nil {
				state.inputCall = &tg.InputPhoneCall{
					ID:         waiting.ID,
					AccessHash: waiting.AccessHash,
				}
				slog.Info("updated incoming call with inputCall from phone.acceptCall", "userID", userID, "callID", waiting.ID)
			}
			p2pStatesMu.Unlock()
		}
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
	delete(p2pStates, userID)
	p2pStatesMu.Unlock()

	waitConnectMu.Lock()
	delete(waitConnect, userID)
	waitConnectMu.Unlock()

	currentBridgeMu.Lock()
	bridge := currentBridge
	currentBridgeMu.Unlock()

	if bridge != nil {
		_ = bridge.StopCall(ctx, voip.StopCallParams{UserID: userID})
		p2pStatesMu.Lock()
		activeCount := len(p2pStates)
		p2pStatesMu.Unlock()
		if activeCount == 0 {
			voipSupervisor.ResetIdleTimer(30 * time.Second)
		}
	}

	if state == nil || state.inputCall == nil {
		return nil
	}

	_, err := Cligram.API().PhoneDiscardCall(ctx, &tg.PhoneDiscardCallRequest{
		Peer:   *state.inputCall,
		Reason: &tg.PhoneCallDiscardReasonHangup{},
	})
	return err
}

// ToggleCallMute toggles mute state for an active call. Returns the new muted state.
func ToggleCallMute(userID int64) (bool, error) {
	currentBridgeMu.Lock()
	bridge := currentBridge
	currentBridgeMu.Unlock()

	if bridge == nil {
		return false, fmt.Errorf("no active call bridge")
	}

	callMutedMu.Lock()
	callMuted = !callMuted
	newMuted := callMuted
	callMutedMu.Unlock()

	err := bridge.SetMute(context.Background(), voip.SetMuteParams{UserID: userID, Muted: newMuted})
	return newMuted, err
}
