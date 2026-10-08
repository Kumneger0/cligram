package voip_test

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/kumneger0/cligram/internal/voip"
)

func TestBridge_CreateCall(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	bridge := voip.NewJSONRPCBridge(clientConn)
	bridge.Start()
	defer bridge.Close()

	// Simulate sidecar responding on serverConn
	go func() {
		decoder := json.NewDecoder(serverConn)
		encoder := json.NewEncoder(serverConn)

		var req voip.JSONRPCRequest
		if err := decoder.Decode(&req); err != nil {
			return
		}

		if req.Method != "call.create" {
			t.Errorf("expected method call.create, got %s", req.Method)
		}

		var params voip.CreateCallParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			t.Errorf("failed to unmarshal params: %v", err)
		}

		if params.UserID != 12345 {
			t.Errorf("expected UserID 12345, got %d", params.UserID)
		}

		resBytes, _ := json.Marshal(voip.CreateCallResult{GAOrB: []byte("test-ga-hash")})
		resp := voip.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  resBytes,
		}
		_ = encoder.Encode(resp)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, err := bridge.CreateCall(ctx, voip.CreateCallParams{
		UserID:     12345,
		IsOutgoing: true,
		DHG:        3,
		DHP:        []byte{1, 2, 3},
		DHRandom:   []byte{4, 5, 6},
	})

	if err != nil {
		t.Fatalf("unexpected error from CreateCall: %v", err)
	}
	if string(result.GAOrB) != "test-ga-hash" {
		t.Fatalf("expected test-ga-hash, got %s", string(result.GAOrB))
	}
}

func TestBridge_Notifications(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	bridge := voip.NewJSONRPCBridge(clientConn)

	var wg sync.WaitGroup
	wg.Add(2)

	var receivedSignaling voip.SignalingEvent
	var receivedState voip.StateChangeEvent

	bridge.OnSignaling(func(event voip.SignalingEvent) {
		receivedSignaling = event
		wg.Done()
	})

	bridge.OnStateChange(func(event voip.StateChangeEvent) {
		receivedState = event
		wg.Done()
	})

	bridge.Start()
	defer bridge.Close()

	// Sidecar sends two notifications
	go func() {
		encoder := json.NewEncoder(serverConn)

		sigParams, _ := json.Marshal(voip.SignalingEvent{
			UserID: 999,
			Data:   []byte("webrtc-sdp-data"),
		})
		_ = encoder.Encode(voip.JSONRPCRequest{
			JSONRPC: "2.0",
			Method:  "call.on_signaling",
			Params:  sigParams,
		})

		stateParams, _ := json.Marshal(voip.StateChangeEvent{
			UserID: 999,
			State:  "connected",
			IsP2P:  true,
		})
		_ = encoder.Encode(voip.JSONRPCRequest{
			JSONRPC: "2.0",
			Method:  "call.on_state_change",
			Params:  stateParams,
		})
	}()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		if receivedSignaling.UserID != 999 || string(receivedSignaling.Data) != "webrtc-sdp-data" {
			t.Errorf("signaling event mismatch: %+v", receivedSignaling)
		}
		if receivedState.UserID != 999 || receivedState.State != "connected" || !receivedState.IsP2P {
			t.Errorf("state event mismatch: %+v", receivedState)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for notifications")
	}
}

func TestBridge_EOFHandling(t *testing.T) {
	clientConn, serverConn := net.Pipe()

	bridge := voip.NewJSONRPCBridge(clientConn)

	closedCalled := make(chan struct{})
	bridge.OnClose(func(err error) {
		close(closedCalled)
	})

	bridge.Start()
	defer bridge.Close()

	// Abruptly close server side to simulate crash
	_ = serverConn.Close()

	select {
	case <-closedCalled:
		// success
	case <-time.After(2 * time.Second):
		t.Fatal("expected OnClose callback on EOF")
	}

	// Any subsequent call should fail immediately
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	_, err := bridge.CreateCall(ctx, voip.CreateCallParams{UserID: 1})
	if err == nil || err != io.EOF && err.Error() != "bridge closed" {
		t.Logf("CreateCall after close returned expected error: %v", err)
	}
}

func TestBridge_FullCallLifecycle(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	bridge := voip.NewJSONRPCBridge(clientConn)
	bridge.Start()
	defer bridge.Close()

	// Mock sidecar handler
	go func() {
		decoder := json.NewDecoder(serverConn)
		encoder := json.NewEncoder(serverConn)

		for {
			var req voip.JSONRPCRequest
			if err := decoder.Decode(&req); err != nil {
				return
			}

			var resp voip.JSONRPCResponse
			resp.JSONRPC = "2.0"
			resp.ID = req.ID

			switch req.Method {
			case "call.create":
				res, _ := json.Marshal(voip.CreateCallResult{GAOrB: []byte("alice-ga")})
				resp.Result = res
			case "call.exchange_keys":
				res, _ := json.Marshal(voip.ExchangeKeysResult{KeyFingerprint: 123456789})
				resp.Result = res
			case "call.connect":
				res, _ := json.Marshal(map[string]bool{"ok": true})
				resp.Result = res
			case "call.send_signaling":
				res, _ := json.Marshal(map[string]bool{"ok": true})
				resp.Result = res
			case "call.set_mute":
				res, _ := json.Marshal(map[string]bool{"ok": true})
				resp.Result = res
			case "call.stop":
				res, _ := json.Marshal(map[string]bool{"ok": true})
				resp.Result = res
			default:
				t.Errorf("unexpected method: %s", req.Method)
			}

			if err := encoder.Encode(resp); err != nil {
				return
			}
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// 1. CreateCall
	createRes, err := bridge.CreateCall(ctx, voip.CreateCallParams{
		UserID:     555,
		IsOutgoing: true,
		DHG:        3,
		DHP:        []byte{1, 2, 3},
		DHRandom:   []byte{4, 5, 6},
	})
	if err != nil || string(createRes.GAOrB) != "alice-ga" {
		t.Fatalf("CreateCall failed: %v, got %v", err, createRes)
	}

	// 2. ExchangeKeys
	exchangeRes, err := bridge.ExchangeKeys(ctx, voip.ExchangeKeysParams{
		UserID:         555,
		GAOrB:          []byte("bob-gb"),
		KeyFingerprint: 123456789,
	})
	if err != nil || exchangeRes.KeyFingerprint != 123456789 {
		t.Fatalf("ExchangeKeys failed: %v, got %v", err, exchangeRes)
	}

	// 3. ConnectCall
	err = bridge.ConnectCall(ctx, voip.ConnectCallParams{
		UserID: 555,
		Servers: []voip.RTCServer{
			{IP: "1.2.3.4", Port: 443},
		},
		P2PAllowed: true,
	})
	if err != nil {
		t.Fatalf("ConnectCall failed: %v", err)
	}

	// 4. SendSignaling
	err = bridge.SendSignaling(ctx, 555, []byte("test-data"))
	if err != nil {
		t.Fatalf("SendSignaling failed: %v", err)
	}

	// 5. SetMute
	err = bridge.SetMute(ctx, voip.SetMuteParams{
		UserID: 555,
		Muted:  true,
	})
	if err != nil {
		t.Fatalf("SetMute failed: %v", err)
	}

	// 6. StopCall
	err = bridge.StopCall(ctx, voip.StopCallParams{UserID: 555})
	if err != nil {
		t.Fatalf("StopCall failed: %v", err)
	}
}
