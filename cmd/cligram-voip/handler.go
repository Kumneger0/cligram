package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/kumneger0/cligram/internal/voip"
	"github.com/kumneger0/cligram/ntg"
)

type playbackSession struct {
	fifoPath string
	dumpFile *os.File
	cancel   context.CancelFunc
	cmd      *exec.Cmd
}

type SidecarHandler struct {
	client       *ntg.Client
	encoder      *json.Encoder
	decoder      *json.Decoder
	writeMu      sync.Mutex
	peerGB       map[int64][]byte
	peerGBMu     sync.Mutex
	isP2P        map[int64]bool
	isP2PMu      sync.RWMutex
	playbackMu   sync.Mutex
	playbackSess map[int64]*playbackSession
}

func GetProtocol() ntg.Protocol {
	return ntg.GetProtocol()
}

func NewSidecarHandler(r io.Reader, w io.Writer) *SidecarHandler {
	h := &SidecarHandler{
		client:       ntg.Init(),
		encoder:      json.NewEncoder(w),
		decoder:      json.NewDecoder(r),
		peerGB:       make(map[int64][]byte),
		isP2P:        make(map[int64]bool),
		playbackSess: make(map[int64]*playbackSession),
	}

	h.setupCallbacks()
	return h
}

func (h *SidecarHandler) setupCallbacks() {
	h.client.OnSignal(func(chatID int64, data []byte) {
		slog.Info("emitting signaling data to core", "chatID", chatID, "len", len(data))
		params, _ := json.Marshal(voip.SignalingEvent{
			UserID: chatID,
			Data:   data,
		})
		h.notify("call.on_signaling", params)
	})

	h.client.OnConnectionChange(func(chatID int64, info ntg.NetworkInfo) {
		slog.Info("connection status changed", "chatID", chatID, "state", info.State)
		stateVal := voip.StateConnecting
		switch info.State {
		case ntg.Connected:
			stateVal = voip.StateConnected
		case ntg.Closed, ntg.Failed:
			stateVal = voip.StateFailed
			h.stopPlayback(chatID)
		case ntg.Timeout:
			stateVal = voip.StateTimeout
		}

		h.isP2PMu.RLock()
		isP2P := h.isP2P[chatID]
		h.isP2PMu.RUnlock()

		params, _ := json.Marshal(voip.StateChangeEvent{
			UserID: chatID,
			State:  stateVal,
			IsP2P:  isP2P,
		})
		h.notify("call.on_state_change", params)
	})
}

func (h *SidecarHandler) notify(method string, params json.RawMessage) {
	req := voip.JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
	h.writeMu.Lock()
	err := h.encoder.Encode(req)
	h.writeMu.Unlock()
	if err != nil {
		slog.Error("failed to encode notification", "method", method, "error", err)
	}
}

func (h *SidecarHandler) sendResponse(id *uint64, result any, err *voip.JSONRPCError) {
	resp := voip.JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   err,
	}
	if result != nil {
		resBytes, _ := json.Marshal(result)
		resp.Result = resBytes
	}
	h.writeMu.Lock()
	encErr := h.encoder.Encode(resp)
	h.writeMu.Unlock()
	if encErr != nil {
		slog.Error("failed to encode response", "id", id, "error", encErr)
	}
}

func (h *SidecarHandler) Run() error {
	for {
		var req voip.JSONRPCRequest
		if err := h.decoder.Decode(&req); err != nil {
			return err
		}

		go h.handleRequest(req)
	}
}

func (h *SidecarHandler) handleRequest(req voip.JSONRPCRequest) {
	switch req.Method {
	case "call.create":
		var params voip.CreateCallParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			h.sendResponse(req.ID, nil, &voip.JSONRPCError{Code: -32602, Message: "invalid params"})
			return
		}

		result, err := h.handleCallCreate(params)
		if err != nil {
			h.sendResponse(req.ID, nil, &voip.JSONRPCError{Code: -32000, Message: err.Error()})
			return
		}
		h.sendResponse(req.ID, result, nil)

	case "call.exchange_keys":
		var params voip.ExchangeKeysParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			h.sendResponse(req.ID, nil, &voip.JSONRPCError{Code: -32602, Message: "invalid params"})
			return
		}

		authParams, err := h.client.ExchangeKeys(params.UserID, params.GAOrB, params.KeyFingerprint)
		if err != nil {
			h.sendResponse(req.ID, nil, &voip.JSONRPCError{Code: -32000, Message: err.Error()})
			return
		}
		h.sendResponse(req.ID, &voip.ExchangeKeysResult{
			GAOrB:          authParams.GAOrB,
			KeyFingerprint: authParams.KeyFingerprint,
		}, nil)

	case "call.connect":
		var params voip.ConnectCallParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			h.sendResponse(req.ID, nil, &voip.JSONRPCError{Code: -32602, Message: "invalid params"})
			return
		}

		h.isP2PMu.Lock()
		h.isP2P[params.UserID] = params.P2PAllowed
		h.isP2PMu.Unlock()

		if err := h.handleCallConnect(params); err != nil {
			h.sendResponse(req.ID, nil, &voip.JSONRPCError{Code: -32000, Message: err.Error()})
			return
		}
		h.sendResponse(req.ID, map[string]string{"status": "ok"}, nil)

	case "call.send_signaling":
		var params voip.SignalingEvent
		if err := json.Unmarshal(req.Params, &params); err != nil {
			h.sendResponse(req.ID, nil, &voip.JSONRPCError{Code: -32602, Message: "invalid params"})
			return
		}

		slog.Info("feeding signaling data into ntgcalls", "userID", params.UserID, "len", len(params.Data))
		if err := h.client.SendSignalingData(params.UserID, params.Data); err != nil {
			slog.Error("failed to send signaling data into ntgcalls", "userID", params.UserID, "error", err)
			h.sendResponse(req.ID, nil, &voip.JSONRPCError{Code: -32000, Message: err.Error()})
			return
		}
		slog.Info("successfully fed signaling data into ntgcalls", "userID", params.UserID)
		h.sendResponse(req.ID, map[string]string{"status": "ok"}, nil)

	case "call.set_mute":
		var params voip.SetMuteParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			h.sendResponse(req.ID, nil, &voip.JSONRPCError{Code: -32602, Message: "invalid params"})
			return
		}

		var err error
		if params.Muted {
			_, err = h.client.Mute(params.UserID)
		} else {
			_, err = h.client.UnMute(params.UserID)
		}
		if err != nil {
			h.sendResponse(req.ID, nil, &voip.JSONRPCError{Code: -32000, Message: err.Error()})
			return
		}
		h.sendResponse(req.ID, map[string]string{"status": "ok"}, nil)

	case "call.stop":
		var params voip.StopCallParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			h.sendResponse(req.ID, nil, &voip.JSONRPCError{Code: -32602, Message: "invalid params"})
			return
		}

		h.stopPlayback(params.UserID)

		if err := h.client.Stop(params.UserID); err != nil {
			slog.Warn("client.Stop failed", "userID", params.UserID, "error", err)
		}
		h.isP2PMu.Lock()
		delete(h.isP2P, params.UserID)
		h.isP2PMu.Unlock()
		h.sendResponse(req.ID, map[string]string{"status": "ok"}, nil)

	default:
		h.sendResponse(req.ID, nil, &voip.JSONRPCError{Code: -32601, Message: "method not found"})
	}
}

func (h *SidecarHandler) handleCallCreate(params voip.CreateCallParams) (*voip.CreateCallResult, error) {
	if err := h.client.CreateP2P(params.UserID); err != nil {
		return nil, fmt.Errorf("create p2p: %w", err)
	}

	gAOrB, err := h.client.InitExchange(
		params.UserID,
		params.DHG,
		params.DHP,
		params.DHRandom,
		params.GAOrB,
	)
	if err != nil {
		return nil, fmt.Errorf("init exchange: %w", err)
	}

	proto := ntg.GetProtocol()
	return &voip.CreateCallResult{
		GAOrB:           gAOrB,
		MinLayer:        proto.MinLayer,
		MaxLayer:        proto.MaxLayer,
		UDPP2P:          proto.UDPP2P,
		UDPReflector:    proto.UDPReflector,
		LibraryVersions: proto.Versions,
	}, nil
}

func (h *SidecarHandler) handleCallConnect(params voip.ConnectCallParams) error {
	// Setup audio pipeline right when call connects, avoiding pipe buffer overflow/stale audio during ringing
	h.setupAudioStreams(params.UserID)

	// Convert voip.RTCServer to ntg.RTCServer
	var ntgServers []ntg.RTCServer
	for _, s := range params.Servers {
		ntgServers = append(ntgServers, ntg.RTCServer{
			ID:       uint64(s.ID),
			IPv4:     s.IP,
			IPv6:     s.IPv6,
			Port:     uint16(s.Port),
			PeerTag:  s.PeerTag,
			IsTURN:   s.Turn,
			IsSTUN:   s.Stun,
			IsTCP:    s.TCP,
			Username: s.Username,
			Password: s.Password,
		})
	}

	libVersions := params.LibraryVersions
	if len(libVersions) == 0 {
		proto := ntg.GetProtocol()
		libVersions = proto.Versions
	}

	err := h.client.ConnectP2P(
		params.UserID,
		ntgServers,
		libVersions,
		params.P2PAllowed,
	)
	if err != nil {
		return fmt.Errorf("connect p2p: %w", err)
	}

	return nil
}

func (h *SidecarHandler) setupStream(userID int64, streamMode ntg.StreamMode, command string) {
	if command == "" {
		return
	}
	desc := &ntg.AudioDescription{
		MediaSource:  ntg.MediaSourceShell,
		Input:        command,
		SampleRate:   48000,
		ChannelCount: 1,
		KeepOpen:     true,
	}
	media := ntg.MediaDescription{
		Microphone: desc,
	}
	if err := h.client.SetStreamSources(userID, streamMode, media); err != nil {
		slog.Warn("failed to set stream source", "mode", streamMode, "error", err)
		params, _ := json.Marshal(voip.AudioErrorEvent{
			UserID:  userID,
			Message: fmt.Sprintf("stream %d failed: %v", streamMode, err),
		})
		h.notify("call.on_audio_error", params)
	}
}

func (h *SidecarHandler) stopPlayback(userID int64) {
	h.playbackMu.Lock()
	defer h.playbackMu.Unlock()
	if sess, ok := h.playbackSess[userID]; ok {
		slog.Info("stopping playback pipeline", "userID", userID)
		sess.cancel()
		if sess.dumpFile != nil {
			_ = sess.dumpFile.Close()
		}
		_ = os.Remove(sess.fifoPath)
		delete(h.playbackSess, userID)
	}
}

func (h *SidecarHandler) setupAudioStreams(userID int64) {
	captureCmd, playbackCmd, soundServer := ResolveAudioCommands()
	slog.Info("🎙️ Setting up audio pipeline with TeeReader", "server", soundServer, "userID", userID)

	// 1. Capture stream (microphone) via subprocess shell command
	h.setupStream(userID, ntg.CaptureStream, captureCmd)

	// 2. Playback stream: stream to FIFO, tee to /tmp/cligram_incoming.raw, and pipe to player
	fifoPath := filepath.Join(os.TempDir(), fmt.Sprintf("cligram_playback_%d.fifo", userID))
	_ = os.Remove(fifoPath)
	if err := exec.Command("mkfifo", fifoPath).Run(); err != nil {
		slog.Error("failed to create playback FIFO", "path", fifoPath, "error", err)
		h.setupStream(userID, ntg.PlaybackStream, playbackCmd)
		return
	}

	rawDumpPath := "/tmp/cligram_incoming.raw"
	dumpFile, err := os.Create(rawDumpPath)
	if err != nil {
		slog.Warn("failed to create incoming audio dump file", "path", rawDumpPath, "error", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	sess := &playbackSession{
		fifoPath: fifoPath,
		dumpFile: dumpFile,
		cancel:   cancel,
	}

	h.playbackMu.Lock()
	if old, exists := h.playbackSess[userID]; exists {
		old.cancel()
		if old.dumpFile != nil {
			_ = old.dumpFile.Close()
		}
		_ = os.Remove(old.fifoPath)
	}
	h.playbackSess[userID] = sess
	h.playbackMu.Unlock()

	// Launch background reader and playback process
	go func() {
		fifoReader, err := os.OpenFile(fifoPath, os.O_RDONLY, 0)
		if err != nil {
			slog.Warn("failed to open playback FIFO for reading", "error", err)
			return
		}
		defer fifoReader.Close()

		var r io.Reader = fifoReader
		if dumpFile != nil {
			r = io.TeeReader(fifoReader, dumpFile)
		}

		args := strings.Fields(playbackCmd)
		if len(args) == 0 {
			return
		}

		slog.Info("🔊 Spawning audio player with TeeReader", "binary", args[0], "dumpFile", rawDumpPath)
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		cmd.Stdin = r
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		sess.cmd = cmd

		if err := cmd.Start(); err != nil {
			slog.Error("failed to start playback command", "cmd", playbackCmd, "error", err)
			return
		}
		_ = cmd.Wait()
	}()

	// Configure libntgcalls to stream playback into the FIFO
	speakerDesc := &ntg.AudioDescription{
		MediaSource:  ntg.MediaSourceFile,
		Input:        fifoPath,
		SampleRate:   48000,
		ChannelCount: 1,
		KeepOpen:     true,
	}
	media := ntg.MediaDescription{
		Microphone: speakerDesc,
	}
	if err := h.client.SetStreamSources(userID, ntg.PlaybackStream, media); err != nil {
		slog.Error("failed to set playback stream sources", "error", err)
		params, _ := json.Marshal(voip.AudioErrorEvent{
			UserID:  userID,
			Message: fmt.Sprintf("stream %d failed: %v", ntg.PlaybackStream, err),
		})
		h.notify("call.on_audio_error", params)
	}
}
