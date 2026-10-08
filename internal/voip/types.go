package voip

import (
	"encoding/json"
)

// JSON-RPC 2.0 protocol structures

type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *uint64         `json:"id,omitempty"` // nil for notifications
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *uint64         `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *JSONRPCError) Error() string {
	return e.Message
}

// VoIP Call Parameters and Payloads

type CreateCallParams struct {
	UserID     int64  `json:"user_id"`
	IsOutgoing bool   `json:"is_outgoing"`
	DHG        int32  `json:"dh_g"`
	DHP        []byte `json:"dh_p"`
	DHRandom   []byte `json:"dh_random"`
	GAOrB      []byte `json:"g_a_or_b,omitempty"` // nil for outgoing initially
}

type CreateCallResult struct {
	GAOrB           []byte   `json:"g_a_or_b"`
	MinLayer        int32    `json:"min_layer"`
	MaxLayer        int32    `json:"max_layer"`
	UDPP2P          bool     `json:"udp_p2p"`
	UDPReflector    bool     `json:"udp_reflector"`
	LibraryVersions []string `json:"library_versions"`
}

type ExchangeKeysParams struct {
	UserID         int64  `json:"user_id"`
	GAOrB          []byte `json:"g_a_or_b"`
	KeyFingerprint int64  `json:"key_fingerprint"`
}

type ExchangeKeysResult struct {
	GAOrB          []byte `json:"g_a_or_b"`
	KeyFingerprint int64  `json:"key_fingerprint"`
}

type RTCServer struct {
	ID       int64  `json:"id"`
	IP       string `json:"ip"`
	IPv6     string `json:"ipv6"`
	Port     int    `json:"port"`
	PeerTag  []byte `json:"peer_tag"`
	Turn     bool   `json:"turn"`
	Stun     bool   `json:"stun"`
	TCP      bool   `json:"tcp,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type ConnectCallParams struct {
	UserID          int64       `json:"user_id"`
	AuthKey         []byte      `json:"auth_key,omitempty"`
	KeyFingerprint  int64       `json:"key_fingerprint"`
	Servers         []RTCServer `json:"servers"`
	LibraryVersions []string    `json:"library_versions"`
	P2PAllowed      bool        `json:"p2p_allowed"`
}

type SetMuteParams struct {
	UserID int64 `json:"user_id"`
	Muted  bool  `json:"muted"`
}

type StopCallParams struct {
	UserID int64 `json:"user_id"`
}

// Sidecar to Core Asynchronous Event Notifications

type SignalingEvent struct {
	UserID int64  `json:"user_id"`
	Data   []byte `json:"data"`
}

type ConnectionState string

const (
	StateConnecting ConnectionState = "connecting"
	StateConnected  ConnectionState = "connected"
	StateFailed     ConnectionState = "failed"
	StateTimeout    ConnectionState = "timeout"
	StateClosed     ConnectionState = "closed"
)

type StateChangeEvent struct {
	UserID int64           `json:"user_id"`
	State  ConnectionState `json:"state"`
	IsP2P  bool            `json:"is_p2p"`
}

type AudioErrorEvent struct {
	UserID  int64  `json:"user_id"`
	Message string `json:"message"`
}
