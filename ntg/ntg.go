package ntg // nolint:revive

/*
#cgo linux,amd64 LDFLAGS: ${SRCDIR}/lib/linux_amd64/libntgcalls.a -lz -lresolv -ldl -lpthread -lm -lstdc++
#cgo linux,arm64 LDFLAGS: ${SRCDIR}/lib/linux_arm64/libntgcalls.a -lz -lresolv -ldl -lpthread -lm -lstdc++
#cgo darwin,arm64 LDFLAGS: -L${SRCDIR}/lib/darwin_arm64 -L/usr/local/lib -L/opt/homebrew/lib -lntgcalls -Wl,-rpath,@loader_path/ntg/lib/darwin_arm64 -Wl,-rpath,@executable_path/ntg/lib/darwin_arm64 -Wl,-rpath,@executable_path -Wl,-rpath,/usr/local/lib -Wl,-rpath,/opt/homebrew/lib
#cgo darwin,amd64 LDFLAGS: -L${SRCDIR}/lib/darwin_amd64 -L/usr/local/lib -L/opt/homebrew/lib -lntgcalls -Wl,-rpath,@loader_path/ntg/lib/darwin_amd64 -Wl,-rpath,@executable_path/ntg/lib/darwin_amd64 -Wl,-rpath,@executable_path -Wl,-rpath,/usr/local/lib -Wl,-rpath,/opt/homebrew/lib
#cgo windows,amd64 LDFLAGS: ${SRCDIR}/lib/windows_amd64/ntgcalls.lib -lws2_32 -lwinmm -lcrypt32

#include <stdint.h>
#include <stdlib.h>
#include "ntgcalls.h"

extern void goOnSignaling(ntg_instance* handle, int64_t chatID, uint8_t* data, size_t dataLen, void* userData);
extern void goOnConnectionChange(ntg_instance* handle, int64_t chatID, ntg_connection_info info, void* userData);
extern void goOnLog(ntg_log_message message, void* userData);
*/
import "C"

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"unsafe"
)

type (
	StreamMode      int
	MediaSource     int
	ConnectionState int
	NetworkInfo     struct{ State ConnectionState }
	AuthParams      struct {
		GAOrB          []byte
		KeyFingerprint int64
	}
	Protocol struct {
		MinLayer, MaxLayer   int32
		UDPP2P, UDPReflector bool
		Versions             []string
	}
	RTCServer struct {
		ID                    uint64
		IPv4, IPv6            string
		Username, Password    string
		Port                  uint16
		IsTURN, IsSTUN, IsTCP bool
		PeerTag               []byte
	}
	AudioDescription struct {
		MediaSource  MediaSource
		Input        string
		SampleRate   uint32
		ChannelCount uint8
		KeepOpen     bool
	}
	MediaDescription struct{ Microphone, Speaker *AudioDescription }

	SignalCallback           func(chatID int64, data []byte)
	ConnectionChangeCallback func(chatID int64, info NetworkInfo)
)

const (
	CaptureStream StreamMode = iota
	PlaybackStream
)

const (
	MediaSourceFile MediaSource = 1 << iota
	MediaSourceShell
)

const (
	Connecting ConnectionState = iota
	Connected
	Failed
	Timeout
	Closed
)

type Client struct {
	handle *C.ntg_instance
	mu     sync.RWMutex
	sigCB  []SignalCallback
	connCB []ConnectionChangeCallback
}

func Init() *Client {
	c := &Client{handle: C.ntg_instance_create()}
	ptr := unsafe.Pointer(c)
	C.ntg_on_signaling_data_callback(c.handle, (C.ntg_signaling_data_callback_cb)(unsafe.Pointer(C.goOnSignaling)), ptr)
	C.ntg_on_connection_change_callback(c.handle, (C.ntg_connection_change_callback_cb)(unsafe.Pointer(C.goOnConnectionChange)), ptr)
	C.ntg_set_log_callback((C.ntg_log_cb)(unsafe.Pointer(C.goOnLog)), nil)
	return c
}

func (c *Client) Destroy() {
	if c.handle != nil {
		C.ntg_instance_destroy(c.handle)
		c.handle = nil
	}
}

func (c *Client) OnSignal(cb SignalCallback) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sigCB = append(c.sigCB, cb)
}

func (c *Client) OnConnectionChange(cb ConnectionChangeCallback) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.connCB = append(c.connCB, cb)
}

//export goOnSignaling
func goOnSignaling(_ *C.ntg_instance, chatID C.int64_t, data *C.uint8_t, dataLen C.size_t, ptr unsafe.Pointer) {
	self := (*Client)(ptr)
	bytes := C.GoBytes(unsafe.Pointer(data), C.int(dataLen))
	self.mu.RLock()
	defer self.mu.RUnlock()
	for _, cb := range self.sigCB {
		go cb(int64(chatID), bytes)
	}
}

//export goOnConnectionChange
func goOnConnectionChange(_ *C.ntg_instance, chatID C.int64_t, info C.ntg_connection_info, ptr unsafe.Pointer) {
	self := (*Client)(ptr)
	self.mu.RLock()
	defer self.mu.RUnlock()
	for _, cb := range self.connCB {
		go cb(int64(chatID), NetworkInfo{State: ConnectionState(info.state)})
	}
}

//export goOnLog
func goOnLog(msg C.ntg_log_message, _ unsafe.Pointer) {
	slog.Info("ntgcalls", "msg", fmt.Sprintf("(%s:%d) %s", C.GoString(msg.file), uint32(msg.line), C.GoString(msg.message)))
}

func (c *Client) CreateP2P(userID int64) error {
	return checkResult(C.ntg_create_p2p_call(c.handle, C.int64_t(userID)))
}

func (c *Client) InitExchange(userID int64, g int32, p, random, gAHash []byte) ([]byte, error) {
	cP, pLen, fP := cBytes(p)
	defer fP()
	cRnd, rndLen, fR := cBytes(random)
	defer fR()
	cHash, hashLen, fH := cBytes(gAHash)
	defer fH()

	var out *C.uint8_t
	var outLen C.size_t
	dh := C.ntg_dh_config{g: C.int32_t(g), p: cP, p_len: pLen, random: cRnd, random_len: rndLen}
	if err := checkResult(C.ntg_init_exchange(c.handle, C.int64_t(userID), dh, cHash, hashLen, &out, &outLen)); err != nil {
		return nil, err
	}
	if out == nil || outLen == 0 {
		return nil, nil
	}
	defer C.ntg_bytes_free(unsafe.Pointer(out))
	return C.GoBytes(unsafe.Pointer(out), C.int(outLen)), nil
}

func (c *Client) ExchangeKeys(userID int64, gAB []byte, fingerprint int64) (AuthParams, error) {
	cGAB, gLen, fG := cBytes(gAB)
	defer fG()

	var out C.ntg_auth_params
	if err := checkResult(C.ntg_exchange_keys(c.handle, C.int64_t(userID), cGAB, gLen, C.int64_t(fingerprint), &out)); err != nil {
		return AuthParams{}, err
	}
	defer C.ntg_auth_params_free(&out)

	var gab []byte
	if out.g_a_or_b != nil && out.g_a_or_b_len > 0 {
		gab = C.GoBytes(unsafe.Pointer(out.g_a_or_b), C.int(out.g_a_or_b_len))
	}
	return AuthParams{GAOrB: gab, KeyFingerprint: int64(out.key_fingerprint)}, nil
}

func (c *Client) ConnectP2P(userID int64, servers []RTCServer, versions []string, p2pAllowed bool) error {
	cServers, fS := parseRtcServers(servers)
	defer fS()
	cVersions, fV := parseStrings(versions)
	defer fV()

	var sPtr *C.ntg_rtc_server
	if len(cServers) > 0 {
		sPtr = &cServers[0]
	}
	var vPtr **C.char
	if len(cVersions) > 0 {
		vPtr = &cVersions[0]
	}
	return checkResult(C.ntg_connect_p2p(c.handle, C.int64_t(userID), sPtr, C.size_t(len(servers)), vPtr, C.size_t(len(versions)), C.bool(p2pAllowed), nil))
}

func (c *Client) SetStreamSources(chatID int64, mode StreamMode, desc MediaDescription) error {
	var toFree []unsafe.Pointer
	defer func() {
		for _, p := range toFree {
			C.free(p)
		}
	}()

	sm := C.NTG_STREAM_MODE_CAPTURE
	if mode == PlaybackStream {
		sm = C.NTG_STREAM_MODE_PLAYBACK
	}
	return checkResult(C.ntg_set_stream_sources(c.handle, C.int64_t(chatID), C.ntg_stream_mode(sm),
		C.ntg_media_description{microphone: cAudio(desc.Microphone, &toFree), speaker: cAudio(desc.Speaker, &toFree)}))
}

func (c *Client) SendSignalingData(userID int64, data []byte) error {
	cData, dLen, fD := cBytes(data)
	defer fD()
	return checkResult(C.ntg_send_signaling_data(c.handle, C.int64_t(userID), cData, dLen))
}

func (c *Client) Mute(chatID int64) (bool, error) {
	var out C.bool
	return bool(out), checkResult(C.ntg_mute(c.handle, C.int64_t(chatID), &out))
}

func (c *Client) UnMute(chatID int64) (bool, error) {
	var out C.bool
	return bool(out), checkResult(C.ntg_unmute(c.handle, C.int64_t(chatID), &out))
}

func (c *Client) Stop(chatID int64) error {
	return checkResult(C.ntg_stop(c.handle, C.int64_t(chatID)))
}

func (c *Client) GetEmojisFingerprint(chatID int64) (string, error) {
	var out *C.char
	if err := checkResult(C.ntg_get_emojis_fingerprint(c.handle, C.int64_t(chatID), &out)); err != nil || out == nil {
		return "", err
	}
	defer C.ntg_string_free(out)
	return C.GoString(out), nil
}

func GetProtocol() Protocol {
	var proto C.ntg_protocol
	if C.ntg_get_protocol(&proto) != C.NTG_OK {
		return Protocol{MinLayer: 92, MaxLayer: 92, UDPP2P: true, UDPReflector: true, Versions: []string{"8.0.0", "9.0.0"}}
	}
	defer C.ntg_protocol_free(&proto)

	var versions []string
	if proto.library_versions != nil {
		for _, s := range unsafe.Slice(proto.library_versions, int(proto.library_versions_len)) {
			versions = append(versions, C.GoString(s))
		}
	}
	return Protocol{MinLayer: int32(proto.min_layer), MaxLayer: int32(proto.max_layer), UDPP2P: bool(proto.udp_p2p), UDPReflector: bool(proto.udp_reflector), Versions: versions}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func checkResult(res C.ntg_result) error {
	if res == C.NTG_OK {
		return nil
	}
	if errStr := C.ntg_last_error(); errStr != nil {
		return errors.New(C.GoString(errStr))
	}
	return fmt.Errorf("ntg error: %d", int(res))
}

func cBytes(b []byte) (*C.uint8_t, C.size_t, func()) {
	if len(b) == 0 {
		return nil, 0, func() {}
	}
	p := C.CBytes(b)
	return (*C.uint8_t)(p), C.size_t(len(b)), func() { C.free(p) }
}

func cStr(s string, toFree *[]unsafe.Pointer) *C.char {
	if s == "" {
		return nil
	}
	cs := C.CString(s)
	*toFree = append(*toFree, unsafe.Pointer(cs))
	return cs
}

func cAudio(desc *AudioDescription, toFree *[]unsafe.Pointer) *C.ntg_audio_description {
	if desc == nil {
		return nil
	}
	src := C.NTG_MEDIA_SOURCE_FILE
	if desc.MediaSource == MediaSourceShell {
		src = C.NTG_MEDIA_SOURCE_SHELL
	}
	return &C.ntg_audio_description{
		media_source:  C.ntg_media_source(src),
		sample_rate:   C.uint32_t(desc.SampleRate),
		channel_count: C.uint8_t(desc.ChannelCount),
		keep_open:     C.bool(desc.KeepOpen),
		input:         cStr(desc.Input, toFree),
	}
}

func parseStrings(data []string) ([]*C.char, func()) {
	var cStrs []*C.char
	for _, s := range data {
		cStrs = append(cStrs, C.CString(s))
	}
	return cStrs, func() {
		for _, s := range cStrs {
			C.free(unsafe.Pointer(s))
		}
	}
}

func parseRtcServers(servers []RTCServer) ([]C.ntg_rtc_server, func()) {
	cServers := make([]C.ntg_rtc_server, len(servers))
	var toFree []unsafe.Pointer
	for i, s := range servers {
		cServers[i] = C.ntg_rtc_server{
			id:       C.uint64_t(s.ID),
			port:     C.uint16_t(s.Port),
			turn:     C.bool(s.IsTURN),
			stun:     C.bool(s.IsSTUN),
			tcp:      C.bool(s.IsTCP),
			ipv4:     cStr(s.IPv4, &toFree),
			ipv6:     cStr(s.IPv6, &toFree),
			username: cStr(s.Username, &toFree),
			password: cStr(s.Password, &toFree),
		}
		if len(s.PeerTag) > 0 {
			tag, tagLen, _ := cBytes(s.PeerTag)
			cServers[i].peer_tag, cServers[i].peer_tag_len = tag, tagLen
			toFree = append(toFree, unsafe.Pointer(tag))
		}
	}
	return cServers, func() {
		for _, p := range toFree {
			C.free(p)
		}
	}
}
