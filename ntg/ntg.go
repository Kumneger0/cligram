package ntg

/*
#cgo LDFLAGS: -L/usr/local/lib -lntgcalls -ldl -lpthread -lopus -lcrypto -lm

#include <stdint.h>
#include <stdlib.h>
#include "ntgcalls.h"

extern void goOnSignaling(uintptr_t ptr, int64_t userId, uint8_t* data, int size, void* userData);
extern void goOnConnectionChange(uintptr_t ptr, int64_t chatID, ntg_network_info_struct networkInfo, void* userData);

// unlockMutex is the promise callback invoked by the C library when
// an async operation completes. It receives a pointer to a Go sync.Mutex
// (stored in ntg_async_struct.userData) and unlocks it so the waiting
// goroutine can proceed.
extern void unlockMutex(void*);
*/
import "C"

import (
	"fmt"
	"sync"
	"unsafe"
)

// ---------------------------------------------------------------------------
// Future – mutex-based async wait (matches the official example)
// ---------------------------------------------------------------------------

// Future wraps the ntg_async_struct pattern. A mutex is locked before the
// C call; the library invokes unlockMutex (via the promise callback) when
// the operation finishes. The Go caller blocks on wait() until that happens.
type Future struct {
	mutex      *sync.Mutex
	errCode    *C.int
	errMessage **C.char
}

func newFuture() *Future {
	f := &Future{
		mutex:      &sync.Mutex{},
		errCode:    new(C.int),
		errMessage: new(*C.char),
	}
	f.mutex.Lock() // will be unlocked by the C callback
	return f
}

func (f *Future) toC() C.ntg_async_struct {
	var s C.ntg_async_struct
	s.userData = unsafe.Pointer(f.mutex)
	s.promise = (C.ntg_async_callback)(unsafe.Pointer(C.unlockMutex))
	s.errorCode = (*C.int)(unsafe.Pointer(f.errCode))
	s.errorMessage = f.errMessage
	return s
}

func (f *Future) wait() {
	f.mutex.Lock() // blocks until unlockMutex is called by C
}

//export unlockMutex
func unlockMutex(p unsafe.Pointer) {
	m := (*sync.Mutex)(p)
	m.Unlock()
}

func parseError(f *Future) error {
	code := int32(*f.errCode)
	if code < 0 {
		var message string
		if *f.errMessage != nil {
			message = C.GoString(*f.errMessage)
		}
		if len(message) == 0 {
			message = fmt.Sprintf("ntg error code: %d", code)
		}
		return fmt.Errorf("%s", message)
	}
	return nil
}

func parseBytes(data []byte) (*C.uint8_t, C.int) {
	if data != nil {
		rawBytes := C.CBytes(data)
		return (*C.uint8_t)(rawBytes), C.int(len(data))
	}
	return nil, 0
}

// ---------------------------------------------------------------------------
// Go types that mirror C structs (no C types escape the package)
// ---------------------------------------------------------------------------

// AuthParams holds the result of ExchangeKeys.
type AuthParams struct {
	GAOrB          []byte
	KeyFingerprint int64
}

// Protocol holds the result of GetProtocol.
type Protocol struct {
	MinLayer     int32
	MaxLayer     int32
	UDPP2P       bool
	UDPReflector bool
	Versions     []string
}

// ConnectionState represents a P2P connection state.
type ConnectionState int

const (
	Connecting ConnectionState = iota
	Connected
	Timeout
	Failed
	Closed
)

// NetworkInfo is a Go-side mirror of ntg_network_info_struct.
type NetworkInfo struct {
	State ConnectionState
}

// RTCServer holds the TURN/STUN connection info for a P2P call.
type RTCServer struct {
	ID       uint64
	IPv4     string
	IPv6     string
	Username string
	Password string
	Port     uint16
	IsTURN   bool
	IsSTUN   bool
	IsTCP    bool
	PeerTag  []byte
}

// Callback types
type SignalCallback func(chatID int64, data []byte)
type ConnectionChangeCallback func(chatID int64, info NetworkInfo)

// ---------------------------------------------------------------------------
// Client
// ---------------------------------------------------------------------------

type Client struct {
	ptr                       C.uintptr_t
	signalCallbacks           []SignalCallback
	connectionChangeCallbacks []ConnectionChangeCallback
}

func Init() *Client {
	instance := &Client{
		ptr: C.ntg_init(),
	}

	// Register C callbacks with a pointer back to this Go instance.
	selfPointer := unsafe.Pointer(instance)
	C.ntg_on_signaling_data(instance.ptr, (C.ntg_signaling_callback)(unsafe.Pointer(C.goOnSignaling)), selfPointer)
	C.ntg_on_connection_change(instance.ptr, (C.ntg_connection_callback)(unsafe.Pointer(C.goOnConnectionChange)), selfPointer)

	return instance
}

func (c *Client) Destroy() {
	C.ntg_destroy(c.ptr)
}

// OnSignal registers a callback for signaling data from ntgcalls.
func (c *Client) OnSignal(cb SignalCallback) {
	c.signalCallbacks = append(c.signalCallbacks, cb)
}

// OnConnectionChange registers a callback for connection state changes.
func (c *Client) OnConnectionChange(cb ConnectionChangeCallback) {
	c.connectionChangeCallbacks = append(c.connectionChangeCallbacks, cb)
}

//export goOnSignaling
func goOnSignaling(_ C.uintptr_t, chatID C.int64_t, data *C.uint8_t, size C.int, ptr unsafe.Pointer) {
	self := (*Client)(ptr)
	goData := C.GoBytes(unsafe.Pointer(data), size)
	for _, cb := range self.signalCallbacks {
		go cb(int64(chatID), goData)
	}
}

//export goOnConnectionChange
func goOnConnectionChange(_ C.uintptr_t, chatID C.int64_t, networkInfo C.ntg_network_info_struct, ptr unsafe.Pointer) {
	self := (*Client)(ptr)
	var state ConnectionState
	switch networkInfo.state {
	case C.NTG_STATE_CONNECTING:
		state = Connecting
	case C.NTG_STATE_CONNECTED:
		state = Connected
	case C.NTG_STATE_TIMEOUT:
		state = Timeout
	case C.NTG_STATE_FAILED:
		state = Failed
	case C.NTG_STATE_CLOSED:
		state = Closed
	}
	for _, cb := range self.connectionChangeCallbacks {
		go cb(int64(chatID), NetworkInfo{State: state})
	}
}

// ---------------------------------------------------------------------------
// P2P call methods
// ---------------------------------------------------------------------------

func (c *Client) CreateP2P(userID int64) error {
	f := newFuture()
	C.ntg_create_p2p(c.ptr, C.int64_t(userID), f.toC())
	f.wait()
	return parseError(f)
}

func (c *Client) ConnectP2P(userID int64, servers []RTCServer, versions []string, p2pAllowed bool) error {
	var cServersPtr *C.ntg_rtc_server_struct
	var cServers []C.ntg_rtc_server_struct

	if len(servers) > 0 {
		cServers = make([]C.ntg_rtc_server_struct, len(servers))
		for i, s := range servers {
			cServers[i].id = C.uint64_t(s.ID)
			cServers[i].ipv4 = C.CString(s.IPv4)
			cServers[i].ipv6 = C.CString(s.IPv6)
			cServers[i].username = C.CString(s.Username)
			cServers[i].password = C.CString(s.Password)
			cServers[i].port = C.uint16_t(s.Port)
			cServers[i].turn = C.bool(s.IsTURN)
			cServers[i].stun = C.bool(s.IsSTUN)
			cServers[i].tcp = C.bool(s.IsTCP)
			cServers[i].peerTag = nil
			cServers[i].peerTagSize = 0
			if len(s.PeerTag) > 0 {
				peerTagC, peerTagSize := parseBytes(s.PeerTag)
				cServers[i].peerTag = peerTagC
				cServers[i].peerTagSize = peerTagSize
			}
		}
		defer func() {
			for i := range cServers {
				if cServers[i].ipv4 != nil {
					C.free(unsafe.Pointer(cServers[i].ipv4))
				}
				if cServers[i].ipv6 != nil {
					C.free(unsafe.Pointer(cServers[i].ipv6))
				}
				if cServers[i].username != nil {
					C.free(unsafe.Pointer(cServers[i].username))
				}
				if cServers[i].password != nil {
					C.free(unsafe.Pointer(cServers[i].password))
				}
				if cServers[i].peerTag != nil {
					C.free(unsafe.Pointer(cServers[i].peerTag))
				}
			}
		}()
		cServersPtr = &cServers[0]
	}

	var cVersionsPtr **C.char
	var cVersions []*C.char
	if len(versions) > 0 {
		cVersions = make([]*C.char, len(versions))
		for i, v := range versions {
			cVersions[i] = C.CString(v)
		}
		defer func() {
			for _, v := range cVersions {
				C.free(unsafe.Pointer(v))
			}
		}()
		cVersionsPtr = &cVersions[0]
	}

	f := newFuture()
	C.ntg_connect_p2p(
		c.ptr,
		C.int64_t(userID),
		cServersPtr,
		C.int(len(servers)),
		cVersionsPtr,
		C.int(len(versions)),
		C.bool(p2pAllowed),
		f.toC(),
	)
	f.wait()
	return parseError(f)
}

// InitExchange performs the Diffie-Hellman exchange with ntgcalls.
// ntgcalls generates its own DH keys internally and returns g_a_or_b bytes.
//
//   - g      : generator (from Telegram's MessagesGetDhConfig)
//   - p      : safe prime bytes
//   - random : server-provided random bytes from MessagesGetDhConfig
//   - gAHash : SHA-256(g^a mod p) received from the peer (nil for outgoing calls)
func (c *Client) InitExchange(
	userID int64,
	g int32,
	p []byte,
	random []byte,
	gAHash []byte,
) ([]byte, error) {
	pC, pSize := parseBytes(p)
	rC, rSize := parseBytes(random)

	dh := C.ntg_dh_config_struct{
		g:          C.int32_t(g),
		p:          pC,
		sizeP:      pSize,
		random:     rC,
		sizeRandom: rSize,
	}

	gAHashC, gAHashSize := parseBytes(gAHash)

	var out *C.uint8_t
	var outSize C.int

	f := newFuture()
	C.ntg_init_exchange(
		c.ptr,
		C.int64_t(userID),
		&dh,
		gAHashC,
		gAHashSize,
		&out,
		&outSize,
		f.toC(),
	)
	f.wait()

	defer C.free(unsafe.Pointer(pC))
	defer C.free(unsafe.Pointer(rC))
	defer C.free(unsafe.Pointer(gAHashC))
	defer C.free(unsafe.Pointer(out))

	if err := parseError(f); err != nil {
		return nil, err
	}

	return C.GoBytes(unsafe.Pointer(out), outSize), nil
}

// ExchangeKeys completes the key exchange.
// gAB is the peer's g_a or g_b value. fingerprint is the key fingerprint
// (0 when the caller is the outgoing party, since the library computes it).
func (c *Client) ExchangeKeys(
	userID int64,
	gAB []byte,
	fingerprint int64,
) (AuthParams, error) {
	f := newFuture()
	var result C.ntg_auth_params_struct

	gABC, gABSize := parseBytes(gAB)

	C.ntg_exchange_keys(
		c.ptr,
		C.int64_t(userID),
		gABC,
		gABSize,
		C.int64_t(fingerprint),
		&result,
		f.toC(),
	)
	f.wait()

	if err := parseError(f); err != nil {
		return AuthParams{}, err
	}

	return AuthParams{
		GAOrB:          C.GoBytes(unsafe.Pointer(result.g_a_or_b), result.sizeGAB),
		KeyFingerprint: int64(result.key_fingerprint),
	}, nil
}

// SkipExchange provides the pre-computed encryption key directly to ntgcalls,
// bypassing the library's own DH exchange.
func (c *Client) SkipExchange(userID int64, encryptionKey []byte, isOutgoing bool) error {
	if len(encryptionKey) == 0 {
		return fmt.Errorf("ntg: encryptionKey is empty")
	}

	f := newFuture()
	encKeyC, encKeySize := parseBytes(encryptionKey)

	C.ntg_skip_exchange(
		c.ptr,
		C.int64_t(userID),
		encKeyC,
		encKeySize,
		C.bool(isOutgoing),
		f.toC(),
	)
	f.wait()
	return parseError(f)
}

// SendSignalingData forwards WebRTC signaling data to the peer.
func (c *Client) SendSignalingData(userID int64, data []byte) error {
	f := newFuture()
	dataC, dataSize := parseBytes(data)

	C.ntg_send_signaling_data(
		c.ptr,
		C.int64_t(userID),
		dataC,
		dataSize,
		f.toC(),
	)
	f.wait()
	return parseError(f)
}

// GetProtocol returns the protocol information from ntgcalls.
func GetProtocol() Protocol {
	var buf C.ntg_protocol_struct
	C.ntg_get_protocol(&buf)

	versions := make([]string, buf.libraryVersionsSize)
	for i := 0; i < int(buf.libraryVersionsSize); i++ {
		ptr := *(**C.char)(unsafe.Pointer(uintptr(unsafe.Pointer(buf.libraryVersions)) + uintptr(i)*unsafe.Sizeof(uintptr(0))))
		versions[i] = C.GoString(ptr)
		C.free(unsafe.Pointer(ptr))
	}
	C.free(unsafe.Pointer(buf.libraryVersions))

	return Protocol{
		MinLayer:     int32(buf.minLayer),
		MaxLayer:     int32(buf.maxLayer),
		UDPP2P:       bool(buf.udpP2P),
		UDPReflector: bool(buf.udpReflector),
		Versions:     versions,
	}
}

func (c *Client) Connect(chatID int64, params string) error {
	cParams := C.CString(params)
	defer C.free(unsafe.Pointer(cParams))

	f := newFuture()
	C.ntg_connect(
		c.ptr,
		C.int64_t(chatID),
		cParams,
		false,
		f.toC(),
	)
	f.wait()
	return parseError(f)
}

func (c *Client) SetStreamSources(
	chatID int64,
	mic, speaker *C.ntg_audio_description_struct,
) error {
	var desc C.ntg_media_description_struct
	desc.microphone = mic
	desc.speaker = speaker

	f := newFuture()
	C.ntg_set_stream_sources(
		c.ptr,
		C.int64_t(chatID),
		C.NTG_STREAM_CAPTURE,
		desc,
		f.toC(),
	)
	f.wait()
	return parseError(f)
}

func (c *Client) Stop(chatID int64) error {
	f := newFuture()
	C.ntg_stop(c.ptr, C.int64_t(chatID), f.toC())
	f.wait()
	return parseError(f)
}
