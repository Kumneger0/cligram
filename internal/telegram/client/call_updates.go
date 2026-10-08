package client

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/gotd/td/tg"
	"github.com/kumneger0/cligram/internal/config"
	"github.com/kumneger0/cligram/internal/notification"
	"github.com/kumneger0/cligram/internal/telegram/shared"
	"github.com/kumneger0/cligram/internal/telegram/types"
	"github.com/kumneger0/cligram/internal/voip"
)

func handlePhoneCall(ctx context.Context, phoneCall tg.PhoneCallClass, updateChannel chan types.Notification) error {
	switch u := phoneCall.(type) {
	case *tg.PhoneCallRequested:
		// ---- INCOMING CALL ----
		userID := u.AdminID
		slog.Info("incoming call request", "userID", userID)

		// Resolve caller name
		userName := fmt.Sprintf("User %d", userID)
		if Cligram != nil && Cligram.API() != nil {
			userInfo, err := shared.GetUserInfo(ctx, *Cligram.API(), userID)
			if err == nil && userInfo != nil {
				userName = userInfo.FirstName
				if userInfo.LastName != "" {
					userName += " " + userInfo.LastName
				}
			}
		}

		// 1. Call-Waiting Busy Policy: Check if already in an active call
		p2pStatesMu.Lock()
		inCall := len(p2pStates) > 0
		p2pStatesMu.Unlock()

		if inCall {
			slog.Info("incoming call while already in call, sending busy discard", "callerID", userID)
			if Cligram != nil && Cligram.API() != nil {
				if _, err := Cligram.API().PhoneDiscardCall(ctx, &tg.PhoneDiscardCallRequest{
					Peer: tg.InputPhoneCall{
						ID:         u.ID,
						AccessHash: u.AccessHash,
					},
					Reason: &tg.PhoneCallDiscardReasonBusy{},
				}); err != nil {
					slog.Error("failed to discard busy call", "callerID", userID, "error", err)
				}
			}
			select {
			case updateChannel <- types.Notification{
				CallEvent: &types.CallNotification{
					UserID:   userID,
					UserName: userName,
					State:    types.CallStateBusyMissed,
				},
			}:
			default:
			}
			return nil
		}

		// 2. Check if cligram-voip helper binary is installed
		if _, err := findSidecarBinary(); err != nil {
			slog.Warn("incoming call received but cligram-voip helper is missing", "caller", userName)
			if Cligram != nil && Cligram.API() != nil {
				if _, err := Cligram.API().PhoneDiscardCall(ctx, &tg.PhoneDiscardCallRequest{
					Peer: tg.InputPhoneCall{
						ID:         u.ID,
						AccessHash: u.AccessHash,
					},
					Reason: &tg.PhoneCallDiscardReasonBusy{},
				}); err != nil {
					slog.Error("failed to discard call on missing helper", "callerID", userID, "error", err)
				}
			}
			notification.Notify("Missed Call", fmt.Sprintf("Incoming call from %s declined (cligram-voip helper not installed)", userName))
			select {
			case updateChannel <- types.Notification{
				CallEvent: &types.CallNotification{
					UserID:   userID,
					UserName: userName,
					State:    types.CallStateBusyMissed,
				},
			}:
			default:
			}
			return nil
		}

		// Store pending call data for user to accept/decline
		pendingIncomingMu.Lock()
		pendingIncoming[userID] = &pendingIncomingCall{
			userID:     userID,
			callID:     u.ID,
			accessHash: u.AccessHash,
			gAHash:     u.GAHash,
		}
		pendingIncomingMu.Unlock()

		// Alert via desktop notification and audible chime
		notification.Alert("📞 Incoming Call", fmt.Sprintf("Incoming call from %s", userName))

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
		slog.Info("call is active", "callID", u.ID)

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
			userID = u.AdminID
			p2pStatesMu.Lock()
			state = p2pStates[userID]
			p2pStatesMu.Unlock()
		}

		if state == nil && len(p2pStates) == 1 {
			p2pStatesMu.Lock()
			for uid, s := range p2pStates {
				userID = uid
				state = s
				slog.Info("matched PhoneCall update to single active call", "userID", userID, "callID", u.ID)
				break
			}
			p2pStatesMu.Unlock()
		}

		if state == nil {
			slog.Warn("PhoneCallObj but no p2pState found", "callID", u.ID)
			return nil
		}

		p2pStatesMu.Lock()
		state.inputCall = &tg.InputPhoneCall{
			ID:         u.ID,
			AccessHash: u.AccessHash,
		}
		state.phoneCall = u
		p2pStatesMu.Unlock()

		if !state.isOutgoing {
			p2pStatesMu.Lock()
			state.gAOrB = u.GAOrB
			state.fingerprint = u.KeyFingerprint
			p2pStatesMu.Unlock()

			currentBridgeMu.Lock()
			bridge := currentBridge
			currentBridgeMu.Unlock()

			if bridge != nil {
				_, err := bridge.ExchangeKeys(ctx, voip.ExchangeKeysParams{
					UserID:         userID,
					GAOrB:          u.GAOrB,
					KeyFingerprint: u.KeyFingerprint,
				})
				if err != nil {
					slog.Error("bridge ExchangeKeys (incoming) failed", "error", err)
					return err
				}

				forceRelay := config.GetConfig().Calls.ForceRelay
				servers := parseRTCServers(u.Connections)
				err = bridge.ConnectCall(ctx, voip.ConnectCallParams{
					UserID:          userID,
					Servers:         servers,
					LibraryVersions: u.Protocol.LibraryVersions,
					P2PAllowed:      u.P2PAllowed && !forceRelay,
				})
				if err != nil {
					slog.Error("bridge ConnectCall (incoming) failed", "error", err)
					return err
				}
			}
		}

		// Notify UI that call is now active
		select {
		case updateChannel <- types.Notification{
			CallEvent: &types.CallNotification{
				UserID:  userID,
				State:   types.CallStateActive,
				IsRelay: !u.P2PAllowed,
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
				select {
				case state.waitData <- fmt.Errorf("call discarded"):
				default:
				}
				delete(p2pStates, uid)

				waitConnectMu.Lock()
				delete(waitConnect, uid)
				waitConnectMu.Unlock()

				currentBridgeMu.Lock()
				bridge := currentBridge
				currentBridgeMu.Unlock()

				if bridge != nil {
					_ = bridge.StopCall(ctx, voip.StopCallParams{UserID: uid})
					if len(p2pStates) == 0 {
						voipSupervisor.ResetIdleTimer(30 * time.Second)
					}
				}
				break
			}
		}
		p2pStatesMu.Unlock()

		pendingIncomingMu.Lock()
		for uid := range pendingIncoming {
			if discardedUserID == 0 {
				discardedUserID = uid
			}
			delete(pendingIncoming, uid)
		}
		pendingIncomingMu.Unlock()

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
		fmt.Printf("Unknown call state: %#v\n", phoneCall)
	}

	return nil
}
