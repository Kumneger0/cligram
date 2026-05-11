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
	})
	return ntgClient
}

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
		//TODO: implement group cal updates
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
				_ = client.SendSignalingData(userID, update.Data)
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
			fmt.Println("📞 Incoming call request from user", u.AdminID)

			userID := u.AdminID

			// Get DH config from Telegram
			dhRaw, err := Cligram.API().MessagesGetDhConfig(ctx, &tg.MessagesGetDhConfigRequest{
				Version:      0,
				RandomLength: 256,
			})
			if err != nil {
				fmt.Println("MessagesGetDhConfig failed:", err)
				return err
			}
			dhConfig, ok := dhRaw.(*tg.MessagesDhConfig)
			if !ok {
				return fmt.Errorf("unexpected DH config type: %T", dhRaw)
			}

			client := getNtgClient()

			// 1. CreateP2P
			if err := client.CreateP2P(userID); err != nil {
				fmt.Println("ntg CreateP2P failed:", err)
				return err
			}

			// 1.5 Set up default audio streams
			if err := client.SetupDefaultAudio(userID); err != nil {
				fmt.Println("ntg SetupDefaultAudio failed:", err)
				return err
			}

			// 2. InitExchange with the caller's gAHash
			gB, err := client.InitExchange(
				userID,
				int32(dhConfig.G),
				dhConfig.P,
				dhConfig.Random,
				u.GAHash, // gAHash from the caller
			)
			if err != nil {
				fmt.Println("ntg InitExchange failed:", err)
				return err
			}

			// Save state
			p2pStatesMu.Lock()
			p2pStates[userID] = &p2pCallState{
				inputCall: &tg.InputPhoneCall{
					ID:         u.ID,
					AccessHash: u.AccessHash,
				},
				isOutgoing: false,
				gAOrB:      gB,
				waitData:   make(chan error, 1),
			}
			p2pStatesMu.Unlock()

			// 3. Accept the call
			protocol := ntg.GetProtocol()
			_, err = Cligram.API().PhoneAcceptCall(ctx, &tg.PhoneAcceptCallRequest{
				Peer: tg.InputPhoneCall{
					ID:         u.ID,
					AccessHash: u.AccessHash,
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
				fmt.Println("phone.acceptCall failed:", err)
				return err
			}

			// The PhoneCallObj update will arrive later with g_a and fingerprint

		case *tg.PhoneCallAccepted:
			// ---- OUTGOING CALL: peer accepted ----
			// This means WE initiated the call (phone.requestCall) and the peer accepted.
			userID := u.ParticipantID
			fmt.Println("📞 Call accepted by user", userID)

			p2pStatesMu.Lock()
			state := p2pStates[userID]
			p2pStatesMu.Unlock()

			if state == nil {
				fmt.Println("WARNING: PhoneCallAccepted but no p2pState for user", userID)
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
			fmt.Println("🔥 Call is active")

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
				fmt.Println("WARNING: PhoneCallObj but no p2pState found")
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
					fmt.Println("ntg ExchangeKeys (incoming) failed:", err)
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
					fmt.Println("ntg ConnectP2P (incoming) failed:", err)
					return err
				}
			} else {
				p2pStatesMu.Lock()
				state.phoneCall = u
				p2pStatesMu.Unlock()
			}

		case *tg.PhoneCallDiscarded:
			fmt.Println("❌ Call ended, duration:", u.Duration)

			p2pStatesMu.Lock()
			for uid, state := range p2pStates {
				if state.inputCall != nil && state.inputCall.ID == u.ID {
					// Signal any waiting goroutines
					select {
					case state.waitData <- fmt.Errorf("call discarded"):
					default:
					}
					delete(p2pStates, uid)
					client := getNtgClient()
					_ = client.Stop(uid)
					break
				}
			}
			p2pStatesMu.Unlock()

		case *tg.PhoneCallWaiting:
			fmt.Println("⏳ Call waiting...")

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

	// Register ntgcalls callbacks
	client := getNtgClient()

	// Forward signaling data from ntgcalls to Telegram
	client.OnSignal(func(chatID int64, data []byte) {
		p2pStatesMu.Lock()
		state := p2pStates[chatID]
		p2pStatesMu.Unlock()
		if state != nil && state.inputCall != nil {
			_, _ = Cligram.API().PhoneSendSignalingData(context.Background(), &tg.PhoneSendSignalingDataRequest{
				Peer: *state.inputCall,
				Data: data,
			})
		}
	})

	// Handle connection state changes
	client.OnConnectionChange(func(chatID int64, info ntg.NetworkInfo) {
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

func InitiateP2PCall(ctx context.Context, userID int64, accessHash int64) error {
	client := getNtgClient()

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

	// 2.5 Set up default audio streams
	if err := client.SetupDefaultAudio(userID); err != nil {
		return fmt.Errorf("ntg SetupDefaultAudio: %w", err)
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
	case <-time.After(30 * time.Second):
		return fmt.Errorf("timed out waiting for call acceptance")
	}

	p2pStatesMu.Lock()
	peerGB := state.gAOrB
	inputCall := state.inputCall
	p2pStatesMu.Unlock()

	authParams, err := client.ExchangeKeys(userID, peerGB, 0)
	if err != nil {
		fmt.Println("ntg ExchangeKeys failed", err)
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
		fmt.Println("phone.confirmCall failed", err)
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

	err = client.ConnectP2P(
		userID,
		servers,
		phoneCallObj.Protocol.LibraryVersions,
		phoneCallObj.P2PAllowed,
	)
	if err != nil {
		fmt.Println("ntg ConnectP2P failed", err)
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
				fmt.Println("P2P connection failed", err)
				return err
			}
			fmt.Println("P2P connected successfully")
		case <-time.After(30 * time.Second):
			return fmt.Errorf("timed out waiting for P2P connection")
		}
	}

	return nil
}
