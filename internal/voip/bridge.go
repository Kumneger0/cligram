package voip

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
)

// SignalingBridge defines the interface for communicating with the VoIP sidecar.
type SignalingBridge interface {
	Start()
	Close() error
	CreateCall(ctx context.Context, params CreateCallParams) (*CreateCallResult, error)
	ExchangeKeys(ctx context.Context, params ExchangeKeysParams) (*ExchangeKeysResult, error)
	ConnectCall(ctx context.Context, params ConnectCallParams) error
	SendSignaling(ctx context.Context, userID int64, data []byte) error
	SetMute(ctx context.Context, params SetMuteParams) error
	StopCall(ctx context.Context, params StopCallParams) error
	OnSignaling(handler func(SignalingEvent))
	OnStateChange(handler func(StateChangeEvent))
	OnAudioError(handler func(AudioErrorEvent))
	OnClose(handler func(error))
}

// JSONRPCBridge implements SignalingBridge over an io.ReadWriteCloser stream.
type JSONRPCBridge struct {
	conn      io.ReadWriteCloser
	encoder   *json.Encoder
	decoder   *json.Decoder
	writeMu   sync.Mutex
	requestID uint64

	pendingMu sync.Mutex
	pending   map[uint64]chan *JSONRPCResponse

	onSignaling   func(SignalingEvent)
	onStateChange func(StateChangeEvent)
	onAudioError  func(AudioErrorEvent)
	onClose       func(error)

	closed   chan struct{}
	closeErr error
	once     sync.Once
}

// NewJSONRPCBridge creates a new bridge wrapping an underlying stream.
func NewJSONRPCBridge(conn io.ReadWriteCloser) *JSONRPCBridge {
	return &JSONRPCBridge{
		conn:    conn,
		encoder: json.NewEncoder(conn),
		decoder: json.NewDecoder(conn),
		pending: make(map[uint64]chan *JSONRPCResponse),
		closed:  make(chan struct{}),
	}
}

// Start begins the background read pump.
func (b *JSONRPCBridge) Start() {
	go b.readLoop()
}

// Close closes the underlying transport and fails pending requests.
func (b *JSONRPCBridge) Close() error {
	var err error
	b.once.Do(func() {
		close(b.closed)
		err = b.conn.Close()
		b.pendingMu.Lock()
		for id, ch := range b.pending {
			close(ch)
			delete(b.pending, id)
		}
		b.pendingMu.Unlock()
	})
	return err
}

func (b *JSONRPCBridge) OnSignaling(h func(SignalingEvent))     { b.onSignaling = h }
func (b *JSONRPCBridge) OnStateChange(h func(StateChangeEvent)) { b.onStateChange = h }
func (b *JSONRPCBridge) OnAudioError(h func(AudioErrorEvent))   { b.onAudioError = h }
func (b *JSONRPCBridge) OnClose(h func(error))                  { b.onClose = h }

func (b *JSONRPCBridge) CreateCall(ctx context.Context, params CreateCallParams) (*CreateCallResult, error) {
	var result CreateCallResult
	err := b.call(ctx, "call.create", params, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (b *JSONRPCBridge) ExchangeKeys(ctx context.Context, params ExchangeKeysParams) (*ExchangeKeysResult, error) {
	var result ExchangeKeysResult
	err := b.call(ctx, "call.exchange_keys", params, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (b *JSONRPCBridge) ConnectCall(ctx context.Context, params ConnectCallParams) error {
	return b.call(ctx, "call.connect", params, nil)
}

func (b *JSONRPCBridge) SendSignaling(ctx context.Context, userID int64, data []byte) error {
	return b.call(ctx, "call.send_signaling", SignalingEvent{UserID: userID, Data: data}, nil)
}

func (b *JSONRPCBridge) SetMute(ctx context.Context, params SetMuteParams) error {
	return b.call(ctx, "call.set_mute", params, nil)
}

func (b *JSONRPCBridge) StopCall(ctx context.Context, params StopCallParams) error {
	return b.call(ctx, "call.stop", params, nil)
}

func (b *JSONRPCBridge) call(ctx context.Context, method string, params any, result any) error {
	select {
	case <-b.closed:
		return errors.New("bridge closed")
	default:
	}

	id := atomic.AddUint64(&b.requestID, 1)

	paramsBytes, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("marshal params: %w", err)
	}

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  method,
		Params:  paramsBytes,
	}

	respChan := make(chan *JSONRPCResponse, 1)
	b.pendingMu.Lock()
	b.pending[id] = respChan
	b.pendingMu.Unlock()

	defer func() {
		b.pendingMu.Lock()
		delete(b.pending, id)
		b.pendingMu.Unlock()
	}()

	b.writeMu.Lock()
	err = b.encoder.Encode(req)
	b.writeMu.Unlock()
	if err != nil {
		return fmt.Errorf("write request: %w", err)
	}

	select {
	case resp, ok := <-respChan:
		if !ok || resp == nil {
			return errors.New("bridge closed")
		}
		if resp.Error != nil {
			return resp.Error
		}
		if result != nil && resp.Result != nil {
			if err := json.Unmarshal(resp.Result, result); err != nil {
				return fmt.Errorf("unmarshal result: %w", err)
			}
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-b.closed:
		return errors.New("bridge closed")
	}
}

func (b *JSONRPCBridge) readLoop() {
	for {
		var raw json.RawMessage
		if err := b.decoder.Decode(&raw); err != nil {
			b.closeErr = err
			_ = b.Close()
			if b.onClose != nil {
				b.onClose(err)
			}
			return
		}

		// Try as response first
		var resp JSONRPCResponse
		if err := json.Unmarshal(raw, &resp); err == nil && resp.ID != nil {
			b.pendingMu.Lock()
			ch, ok := b.pending[*resp.ID]
			b.pendingMu.Unlock()
			if ok {
				ch <- &resp
			}
			continue
		}

		// Try as notification / request
		var req JSONRPCRequest
		if err := json.Unmarshal(raw, &req); err == nil {
			b.handleNotification(req)
		}
	}
}

func (b *JSONRPCBridge) handleNotification(req JSONRPCRequest) {
	switch req.Method {
	case "call.on_signaling":
		if b.onSignaling != nil {
			var event SignalingEvent
			if err := json.Unmarshal(req.Params, &event); err == nil {
				b.onSignaling(event)
			}
		}
	case "call.on_state_change":
		if b.onStateChange != nil {
			var event StateChangeEvent
			if err := json.Unmarshal(req.Params, &event); err == nil {
				b.onStateChange(event)
			}
		}
	case "call.on_audio_error":
		if b.onAudioError != nil {
			var event AudioErrorEvent
			if err := json.Unmarshal(req.Params, &event); err == nil {
				b.onAudioError(event)
			}
		}
	}
}
