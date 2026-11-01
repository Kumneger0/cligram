package ntg // nolint:revive

/*
#cgo linux,amd64 LDFLAGS: ${SRCDIR}/lib/linux_amd64/libntgcalls.a -lz -lresolv -ldl -lpthread -lm -lstdc++
#cgo linux,arm64 LDFLAGS: ${SRCDIR}/lib/linux_arm64/libntgcalls.a -lz -lresolv -ldl -lpthread -lm -lstdc++
#cgo darwin,arm64 LDFLAGS: ${SRCDIR}/lib/darwin_arm64/libntgcalls.a -lz -lbz2 -liconv -lc++ -framework Foundation -framework AVFoundation -framework AppKit -framework CoreFoundation -framework Security -framework CoreAudio -framework AudioToolbox -framework CoreVideo -framework CoreMedia -framework CoreGraphics -framework VideoToolbox -framework IOSurface -framework ScreenCaptureKit
#cgo darwin,amd64 LDFLAGS: ${SRCDIR}/lib/darwin_amd64/libntgcalls.a -lz -lbz2 -liconv -lc++ -framework Foundation -framework AVFoundation -framework AppKit -framework CoreFoundation -framework Security -framework CoreAudio -framework AudioToolbox -framework CoreVideo -framework CoreMedia -framework CoreGraphics -framework VideoToolbox -framework IOSurface -framework ScreenCaptureKit
#cgo windows,amd64 LDFLAGS: ${SRCDIR}/lib/windows_amd64/ntgcalls.lib -lws2_32 -lwinmm -lcrypt32

#include <stdint.h>
#include <stdlib.h>
#include <stdbool.h>
#include "ntgcalls.h"

extern void goOnSignaling(ntg_instance* handle, int64_t chatID, uint8_t* data, size_t size, void* userData);
extern void goOnConnectionChange(ntg_instance* handle, int64_t chatID, ntg_connection_info state, void* userData);
extern void goOnFrames(ntg_instance* handle, int64_t chatID, ntg_stream_mode streamMode, ntg_stream_device streamDevice, ntg_frame* frames, size_t size, void* userData);
extern void goOnRemoteSourceChange(ntg_instance* handle, int64_t chatID, ntg_remote_source source, void* userData);
extern void goOnLog(ntg_log_message message, void* userData);
extern void goOnStreamEnd(ntg_instance* handle, int64_t chatID, ntg_stream_type streamType, ntg_stream_device streamDevice, void* userData);
extern void goOnUpgrade(ntg_instance* handle, int64_t chatID, ntg_media_state state, void* userData);
extern void goOnRequestBroadcastTimestamp(ntg_instance* handle, int64_t chatID, void* userData);
extern void goOnRequestBroadcastPart(ntg_instance* handle, int64_t chatID, ntg_segment_part_request segmentPartRequest, void* userData);
*/
import "C"

import (
	"fmt"
	"log/slog"
	"os/exec"
	"unsafe"
)

func parseResult(res C.ntg_result) error {
	if res == C.NTG_OK {
		return nil
	}
	lastErr := C.ntg_last_error()
	if lastErr != nil {
		return fmt.Errorf("ntg error %d: %s", int(res), C.GoString(lastErr))
	}
	return fmt.Errorf("ntg error %d", int(res))
}

func parseBytes(data []byte) (*C.uint8_t, C.size_t) {
	if len(data) > 0 {
		rawBytes := C.CBytes(data)
		return (*C.uint8_t)(rawBytes), C.size_t(len(data))
	}
	return nil, 0
}

func parseStringVector(data **C.char, size C.size_t) []string {
	if data == nil || size == 0 {
		return nil
	}
	result := make([]string, size)
	for i := 0; i < int(size); i++ {
		pointer := *(**C.char)(unsafe.Pointer(uintptr(unsafe.Pointer(data)) + uintptr(i)*unsafe.Sizeof(uintptr(0))))
		if pointer != nil {
			result[i] = C.GoString(pointer)
		}
	}
	return result
}

func parseStringVectorC(data []string) (**C.char, C.size_t) {
	if len(data) > 0 {
		rawData := make([]*C.char, len(data))
		for i, v := range data {
			rawData[i] = C.CString(v)
		}
		return &rawData[0], C.size_t(len(data))
	}
	return nil, 0
}

func parseUint32VectorC(data []uint32) (*C.uint32_t, C.size_t) {
	if len(data) > 0 {
		cData := C.malloc(C.size_t(len(data)) * C.size_t(unsafe.Sizeof(C.uint32_t(0))))
		if cData == nil {
			return nil, 0
		}
		ssrcs := (*C.uint32_t)(cData)
		for i, v := range data {
			*(*C.uint32_t)(unsafe.Pointer(uintptr(unsafe.Pointer(ssrcs)) + uintptr(i)*unsafe.Sizeof(C.uint32_t(0)))) = C.uint32_t(v)
		}
		return ssrcs, C.size_t(len(data))
	}
	return nil, 0
}

func parseSsrcGroups(ssrcGroups []SsrcGroup) *C.ntg_ssrc_group {
	if len(ssrcGroups) > 0 {
		rawGroups := make([]C.ntg_ssrc_group, len(ssrcGroups))
		for i, group := range ssrcGroups {
			ssrcsC, sizeSsrcs := parseUint32VectorC(group.Ssrcs)
			rawGroups[i] = C.ntg_ssrc_group{
				semantics: C.CString(group.Semantics),
				ssrcs:     ssrcsC,
				ssrcs_len: sizeSsrcs,
			}
		}
		return (*C.ntg_ssrc_group)(unsafe.Pointer(&rawGroups[0]))
	}
	return nil
}

func parseDeviceInfoVector(devices *C.ntg_device_info, size C.size_t) []DeviceInfo {
	if devices == nil || size == 0 {
		return nil
	}
	rawDevices := make([]DeviceInfo, size)
	for i := 0; i < int(size); i++ {
		device := *(*C.ntg_device_info)(unsafe.Pointer(uintptr(unsafe.Pointer(devices)) + uintptr(i)*unsafe.Sizeof(C.ntg_device_info{})))
		rawDevices[i] = DeviceInfo{
			Name:     C.GoString(device.name),
			Metadata: C.GoString(device.metadata),
		}
	}
	return rawDevices
}

func parseConnectionState(state C.ntg_connection_state) ConnectionState {
	switch state {
	case C.NTG_CONNECTION_STATE_CONNECTING:
		return Connecting
	case C.NTG_CONNECTION_STATE_CONNECTED:
		return Connected
	case C.NTG_CONNECTION_STATE_TIMEOUT:
		return Timeout
	case C.NTG_CONNECTION_STATE_FAILED:
		return Failed
	case C.NTG_CONNECTION_STATE_CLOSED:
		return Closed
	}
	return Connecting
}

func parseStreamDevice(device C.ntg_stream_device) StreamDevice {
	switch device {
	case C.NTG_STREAM_DEVICE_MICROPHONE:
		return MicrophoneStream
	case C.NTG_STREAM_DEVICE_SPEAKER:
		return SpeakerStream
	case C.NTG_STREAM_DEVICE_CAMERA:
		return CameraStream
	case C.NTG_STREAM_DEVICE_SCREEN:
		return ScreenStream
	}
	return MicrophoneStream
}

func parseStreamStatus(status C.ntg_stream_status) StreamStatus {
	switch status {
	case C.NTG_STREAM_STATUS_ACTIVE:
		return ActiveStream
	case C.NTG_STREAM_STATUS_PAUSED:
		return PausedStream
	case C.NTG_STREAM_STATUS_IDLING:
		return IdlingStream
	}
	return ActiveStream
}

func parseRtcServers(rtcServers []RTCServer) *C.ntg_rtc_server {
	if len(rtcServers) > 0 {
		rawServers := make([]C.ntg_rtc_server, len(rtcServers))
		for i, server := range rtcServers {
			rawServers[i] = C.ntg_rtc_server{
				id:           C.uint64_t(server.ID),
				ipv4:         C.CString(server.IPv4),
				ipv6:         C.CString(server.IPv6),
				username:     C.CString(server.Username),
				password:     C.CString(server.Password),
				port:         C.uint16_t(server.Port),
				turn:         C.bool(server.IsTURN),
				stun:         C.bool(server.IsSTUN),
				tcp:          C.bool(server.IsTCP),
				peer_tag:     nil,
				peer_tag_len: 0,
			}
			if len(server.PeerTag) > 0 {
				peerTagC, peerTagSize := parseBytes(server.PeerTag)
				rawServers[i].peer_tag = peerTagC
				rawServers[i].peer_tag_len = peerTagSize
			}
		}
		return (*C.ntg_rtc_server)(unsafe.Pointer(&rawServers[0]))
	}
	return nil
}

// ---------------------------------------------------------------------------
// Go types that mirror C structs (no C types escape the package)
// ---------------------------------------------------------------------------

type StreamType int

const (
	AudioStream StreamType = iota
	VideoStream
)

type ConnectionMode int

const (
	RtcConnection ConnectionMode = iota
	StreamConnection
	RTMPConnection
)

type ConnectionKind int

const (
	NormalConnection ConnectionKind = iota
	PresentationConnection
)

type ConnectionState int

const (
	Connecting ConnectionState = iota
	Connected
	Failed
	Timeout
	Closed
)

type StreamStatus int

const (
	ActiveStream StreamStatus = iota
	PausedStream
	IdlingStream
)

type StreamMode int

const (
	CaptureStream StreamMode = iota
	PlaybackStream
)

func (sm StreamMode) parseToC() C.ntg_stream_mode {
	switch sm {
	case CaptureStream:
		return C.NTG_STREAM_MODE_CAPTURE
	case PlaybackStream:
		return C.NTG_STREAM_MODE_PLAYBACK
	default:
		return C.NTG_STREAM_MODE_CAPTURE
	}
}

type StreamDevice int

const (
	MicrophoneStream StreamDevice = iota
	SpeakerStream
	CameraStream
	ScreenStream
)

func (sd StreamDevice) ParseToC() C.ntg_stream_device {
	switch sd {
	case MicrophoneStream:
		return C.NTG_STREAM_DEVICE_MICROPHONE
	case SpeakerStream:
		return C.NTG_STREAM_DEVICE_SPEAKER
	case CameraStream:
		return C.NTG_STREAM_DEVICE_CAMERA
	case ScreenStream:
		return C.NTG_STREAM_DEVICE_SCREEN
	default:
		return C.NTG_STREAM_DEVICE_MICROPHONE
	}
}

type MediaSource int

const (
	MediaSourceFile MediaSource = 1 << iota
	MediaSourceShell
	MediaSourceFFmpeg
	MediaSourceDevice
	MediaSourceDesktop
	MediaSourceExternal
)

func (ms MediaSource) parseToC() C.ntg_media_source {
	switch ms {
	case MediaSourceFile:
		return C.NTG_MEDIA_SOURCE_FILE
	case MediaSourceShell:
		return C.NTG_MEDIA_SOURCE_SHELL
	case MediaSourceFFmpeg:
		return C.NTG_MEDIA_SOURCE_FFMPEG
	case MediaSourceDevice:
		return C.NTG_MEDIA_SOURCE_DEVICE
	case MediaSourceDesktop:
		return C.NTG_MEDIA_SOURCE_DESKTOP
	case MediaSourceExternal:
		return C.NTG_MEDIA_SOURCE_EXTERNAL
	default:
		return C.NTG_MEDIA_SOURCE_FILE
	}
}

type MediaSegmentQuality int

const (
	SegmentQualityNone MediaSegmentQuality = iota - 1
	SegmentQualityThumbnail
	SegmentQualityMedium
	SegmentQualityFull
)

type MediaSegmentStatus int

const (
	SegmentStatusNotReady MediaSegmentStatus = iota
	SegmentStatusResyncNeeded
	SegmentStatusSuccess
)

func (mss MediaSegmentStatus) ParseToC() C.ntg_media_segment_part_status {
	switch mss {
	case SegmentStatusNotReady:
		return C.NTG_MEDIA_SEGMENT_PART_STATUS_NOT_READY
	case SegmentStatusResyncNeeded:
		return C.NTG_MEDIA_SEGMENT_PART_STATUS_RESYNC_NEEDED
	case SegmentStatusSuccess:
		return C.NTG_MEDIA_SEGMENT_PART_STATUS_SUCCESS
	default:
		return C.NTG_MEDIA_SEGMENT_PART_STATUS_NOT_READY
	}
}

// MediaState mirrors ntg_media_state.
type MediaState struct {
	Muted              bool
	VideoPaused        bool
	VideoStopped       bool
	PresentationPaused bool
}

// CallInfo holds connection info for a call.
type CallInfo struct {
	Playback, Capture StreamStatus
}

// DeviceInfo mirrors ntg_device_info.
type DeviceInfo struct {
	Name, Metadata string
}

// MediaDevices holds microphone, speaker, camera and screen devices list.
type MediaDevices struct {
	Microphone, Speaker, Camera, Screen []DeviceInfo
}

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

// NetworkInfo is a Go-side mirror of ntg_connection_info.
type NetworkInfo struct {
	Kind  ConnectionKind
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

// FrameData mirrors ntg_frame_data.
type FrameData struct {
	AbsoluteCaptureTimestampMs int64
	Width                      uint16
	Height                     uint16
	Rotation                   uint16
}

func (fd *FrameData) ParseToC() C.ntg_frame_data {
	return C.ntg_frame_data{
		absolute_capture_timestamp_ms: C.int64_t(fd.AbsoluteCaptureTimestampMs),
		width:                         C.uint16_t(fd.Width),
		height:                        C.uint16_t(fd.Height),
		rotation:                      C.ntg_video_rotation(fd.Rotation),
	}
}

// Frame mirrors a single ntg_frame.
type Frame struct {
	Ssrc      uint32
	Data      []byte
	FrameData FrameData
}

// RemoteSource mirrors ntg_remote_source.
type RemoteSource struct {
	Ssrc   uint32
	State  StreamStatus
	Device StreamDevice
}

// AudioDescription mirrors ntg_audio_description.
type AudioDescription struct {
	MediaSource  MediaSource
	Input        string
	SampleRate   uint32
	ChannelCount uint8
	KeepOpen     bool
}

func (ad *AudioDescription) parseToC() C.ntg_audio_description {
	var x C.ntg_audio_description
	x.media_source = ad.MediaSource.parseToC()
	x.input = C.CString(ad.Input)
	x.sample_rate = C.uint32_t(ad.SampleRate)
	x.channel_count = C.uint8_t(ad.ChannelCount)
	x.keep_open = C.bool(ad.KeepOpen)
	return x
}

// VideoDescription mirrors ntg_video_description.
type VideoDescription struct {
	MediaSource   MediaSource
	Input         string
	Width, Height int16
	Fps           uint8
	KeepOpen      bool
}

func (vd *VideoDescription) parseToC() C.ntg_video_description {
	var x C.ntg_video_description
	x.media_source = vd.MediaSource.parseToC()
	x.input = C.CString(vd.Input)
	x.width = C.int16_t(vd.Width)
	x.height = C.int16_t(vd.Height)
	x.fps = C.uint8_t(vd.Fps)
	x.keep_open = C.bool(vd.KeepOpen)
	return x
}

// MediaDescription mirrors ntg_media_description.
type MediaDescription struct {
	Microphone *AudioDescription
	Speaker    *AudioDescription
	Camera     *VideoDescription
	Screen     *VideoDescription
}

func (md *MediaDescription) parseToC() C.ntg_media_description {
	var x C.ntg_media_description
	if md.Microphone != nil {
		mic := md.Microphone.parseToC()
		x.microphone = &mic
	}
	if md.Speaker != nil {
		spk := md.Speaker.parseToC()
		x.speaker = &spk
	}
	if md.Camera != nil {
		cam := md.Camera.parseToC()
		x.camera = &cam
	}
	if md.Screen != nil {
		scr := md.Screen.parseToC()
		x.screen = &scr
	}
	return x
}

// SsrcGroup holds details about video sources.
type SsrcGroup struct {
	Semantics string
	Ssrcs     []uint32
}

// SegmentPartRequest mirrors ntg_segment_part_request.
type SegmentPartRequest struct {
	SegmentID     int64
	PartID        int32
	Limit         int32
	Timestamp     int64
	QualityUpdate bool
	ChannelID     int32
	Quality       MediaSegmentQuality
}

// DhConfig mirrors ntg_dh_config.
type DhConfig struct {
	G      int32
	P      []byte
	Random []byte
}

func (dh *DhConfig) parseToC() C.ntg_dh_config {
	var x C.ntg_dh_config
	x.g = C.int32_t(dh.G)
	pC, pSize := parseBytes(dh.P)
	rC, rSize := parseBytes(dh.Random)
	x.p = pC
	x.p_len = pSize
	x.random = rC
	x.random_len = rSize
	return x
}

// Callback types
type SignalCallback func(chatID int64, data []byte)
type ConnectionChangeCallback func(chatID int64, info NetworkInfo)
type FrameCallback func(chatID int64, mode StreamMode, device StreamDevice, frames []Frame)
type RemoteSourceCallback func(chatID int64, source RemoteSource)
type StreamEndCallback func(chatID int64, streamType StreamType, streamDevice StreamDevice)
type UpgradeCallback func(chatID int64, state MediaState)
type BroadcastTimestampCallback func(chatID int64)
type BroadcastPartCallback func(chatID int64, request SegmentPartRequest)

// ---------------------------------------------------------------------------
// Client
// ---------------------------------------------------------------------------

type Client struct {
	handle                      *C.ntg_instance
	signalCallbacks             []SignalCallback
	connectionChangeCallbacks   []ConnectionChangeCallback
	frameCallbacks              []FrameCallback
	remoteSourceCallbacks       []RemoteSourceCallback
	streamEndCallbacks          []StreamEndCallback
	upgradeCallbacks            []UpgradeCallback
	broadcastTimestampCallbacks []BroadcastTimestampCallback
	broadcastPartCallbacks      []BroadcastPartCallback
}

func Init() *Client {
	instance := &Client{
		handle: C.ntg_instance_create(),
	}

	// Register C callbacks with a pointer back to this Go instance.
	selfPointer := unsafe.Pointer(instance)
	C.ntg_on_signaling_data_callback(instance.handle, (C.ntg_signaling_data_callback_cb)(unsafe.Pointer(C.goOnSignaling)), selfPointer)
	C.ntg_on_connection_change_callback(instance.handle, (C.ntg_connection_change_callback_cb)(unsafe.Pointer(C.goOnConnectionChange)), selfPointer)
	C.ntg_on_frames_callback(instance.handle, (C.ntg_frames_callback_cb)(unsafe.Pointer(C.goOnFrames)), selfPointer)
	C.ntg_on_remote_source_change_callback(instance.handle, (C.ntg_remote_source_change_callback_cb)(unsafe.Pointer(C.goOnRemoteSourceChange)), selfPointer)
	C.ntg_on_stream_end_callback(instance.handle, (C.ntg_stream_end_callback_cb)(unsafe.Pointer(C.goOnStreamEnd)), selfPointer)
	C.ntg_on_upgrade_callback(instance.handle, (C.ntg_upgrade_callback_cb)(unsafe.Pointer(C.goOnUpgrade)), selfPointer)
	C.ntg_on_request_broadcast_timestamp_callback(instance.handle, (C.ntg_request_broadcast_timestamp_callback_cb)(unsafe.Pointer(C.goOnRequestBroadcastTimestamp)), selfPointer)
	C.ntg_on_request_broadcast_part_callback(instance.handle, (C.ntg_request_broadcast_part_callback_cb)(unsafe.Pointer(C.goOnRequestBroadcastPart)), selfPointer)

	// Register the library logger to pipe internal ntgcalls WebRTC & playback logs to slog
	C.ntg_set_log_callback((C.ntg_log_cb)(unsafe.Pointer(C.goOnLog)), nil)

	return instance
}

func (c *Client) Destroy() {
	if c.handle != nil {
		C.ntg_instance_destroy(c.handle)
		c.handle = nil
	}
}

// OnSignal registers a callback for signaling data from ntgcalls.
func (c *Client) OnSignal(cb SignalCallback) {
	c.signalCallbacks = append(c.signalCallbacks, cb)
}

// OnConnectionChange registers a callback for connection state changes.
func (c *Client) OnConnectionChange(cb ConnectionChangeCallback) {
	c.connectionChangeCallbacks = append(c.connectionChangeCallbacks, cb)
}

// OnFrame registers a callback for decoded media frames.
func (c *Client) OnFrame(cb FrameCallback) {
	c.frameCallbacks = append(c.frameCallbacks, cb)
}

// OnRemoteSource registers a callback for remote source changes.
func (c *Client) OnRemoteSource(cb RemoteSourceCallback) {
	c.remoteSourceCallbacks = append(c.remoteSourceCallbacks, cb)
}

// OnStreamEnd registers a callback for stream ends.
func (c *Client) OnStreamEnd(cb StreamEndCallback) {
	c.streamEndCallbacks = append(c.streamEndCallbacks, cb)
}

// OnUpgrade registers a callback for media state upgrades.
func (c *Client) OnUpgrade(cb UpgradeCallback) {
	c.upgradeCallbacks = append(c.upgradeCallbacks, cb)
}

// OnRequestBroadcastTimestamp registers a callback for requesting broadcast timestamp.
func (c *Client) OnRequestBroadcastTimestamp(cb BroadcastTimestampCallback) {
	c.broadcastTimestampCallbacks = append(c.broadcastTimestampCallbacks, cb)
}

// OnRequestBroadcastPart registers a callback for requesting broadcast part.
func (c *Client) OnRequestBroadcastPart(cb BroadcastPartCallback) {
	c.broadcastPartCallbacks = append(c.broadcastPartCallbacks, cb)
}

// ---------------------------------------------------------------------------
// Exported C callbacks
// ---------------------------------------------------------------------------

//export goOnSignaling
func goOnSignaling(_ *C.ntg_instance, chatID C.int64_t, data *C.uint8_t, size C.size_t, ptr unsafe.Pointer) {
	self := (*Client)(ptr)
	goData := C.GoBytes(unsafe.Pointer(data), C.int(size))
	for _, cb := range self.signalCallbacks {
		go cb(int64(chatID), goData)
	}
}

//export goOnConnectionChange
func goOnConnectionChange(_ *C.ntg_instance, chatID C.int64_t, state C.ntg_connection_info, ptr unsafe.Pointer) {
	self := (*Client)(ptr)
	var goInfo NetworkInfo
	switch state.kind {
	case C.NTG_CONNECTION_KIND_NORMAL:
		goInfo.Kind = NormalConnection
	case C.NTG_CONNECTION_KIND_PRESENTATION:
		goInfo.Kind = PresentationConnection
	}
	goInfo.State = parseConnectionState(state.state)
	for _, cb := range self.connectionChangeCallbacks {
		go cb(int64(chatID), goInfo)
	}
}

//export goOnRemoteSourceChange
func goOnRemoteSourceChange(_ *C.ntg_instance, chatID C.int64_t, source C.ntg_remote_source, ptr unsafe.Pointer) {
	self := (*Client)(ptr)
	goSource := RemoteSource{
		Ssrc:   uint32(source.ssrc),
		State:  parseStreamStatus(source.state),
		Device: parseStreamDevice(source.device),
	}
	for _, cb := range self.remoteSourceCallbacks {
		go cb(int64(chatID), goSource)
	}
}

//export goOnLog
func goOnLog(msg C.ntg_log_message, _ unsafe.Pointer) {
	file := C.GoString(msg.file)
	line := uint32(msg.line)
	message := C.GoString(msg.message)

	formatted := fmt.Sprintf("(%s:%d) %s", file, line, message)

	switch msg.level {
	case C.NTG_LOG_DEBUG:
		slog.Debug("ntgcalls", "message", formatted)
	case C.NTG_LOG_INFO:
		slog.Info("ntgcalls", "message", formatted)
	case C.NTG_LOG_WARNING:
		slog.Warn("ntgcalls", "message", formatted)
	case C.NTG_LOG_ERROR:
		slog.Error("ntgcalls", "message", formatted)
	default:
		slog.Info("ntgcalls", "message", formatted)
	}
}

//export goOnFrames
func goOnFrames(_ *C.ntg_instance, chatID C.int64_t, streamMode C.ntg_stream_mode, streamDevice C.ntg_stream_device, frames *C.ntg_frame, size C.size_t, ptr unsafe.Pointer) {
	self := (*Client)(ptr)
	goChatID := int64(chatID)

	var goStreamMode StreamMode
	switch streamMode {
	case C.NTG_STREAM_MODE_CAPTURE:
		goStreamMode = CaptureStream
	case C.NTG_STREAM_MODE_PLAYBACK:
		goStreamMode = PlaybackStream
	}

	rawFrames := make([]Frame, size)
	for i := uint64(0); i < uint64(size); i++ {
		rawFrame := *(*C.ntg_frame)(unsafe.Pointer(uintptr(unsafe.Pointer(frames)) + uintptr(i)*unsafe.Sizeof(C.ntg_frame{})))
		rawFrames[i] = Frame{
			Ssrc: uint32(rawFrame.ssrc),
			Data: C.GoBytes(unsafe.Pointer(rawFrame.data), C.int(rawFrame.data_len)),
			FrameData: FrameData{
				AbsoluteCaptureTimestampMs: int64(rawFrame.frame_data.absolute_capture_timestamp_ms),
				Width:                      uint16(rawFrame.frame_data.width),
				Height:                     uint16(rawFrame.frame_data.height),
				Rotation:                   uint16(rawFrame.frame_data.rotation),
			},
		}
	}

	for _, cb := range self.frameCallbacks {
		go cb(goChatID, goStreamMode, parseStreamDevice(streamDevice), rawFrames)
	}
}

//export goOnStreamEnd
func goOnStreamEnd(_ *C.ntg_instance, chatID C.int64_t, streamType C.ntg_stream_type, streamDevice C.ntg_stream_device, ptr unsafe.Pointer) {
	self := (*Client)(ptr)
	goChatID := int64(chatID)
	var goStreamType StreamType
	if streamType == C.NTG_STREAM_TYPE_AUDIO {
		goStreamType = AudioStream
	} else {
		goStreamType = VideoStream
	}
	for _, cb := range self.streamEndCallbacks {
		go cb(goChatID, goStreamType, parseStreamDevice(streamDevice))
	}
}

//export goOnUpgrade
func goOnUpgrade(_ *C.ntg_instance, chatID C.int64_t, state C.ntg_media_state, ptr unsafe.Pointer) {
	self := (*Client)(ptr)
	goChatID := int64(chatID)
	goState := MediaState{
		Muted:              bool(state.muted),
		VideoPaused:        bool(state.video_paused),
		VideoStopped:       bool(state.video_stopped),
		PresentationPaused: bool(state.presentation_paused),
	}
	for _, cb := range self.upgradeCallbacks {
		go cb(goChatID, goState)
	}
}

//export goOnRequestBroadcastTimestamp
func goOnRequestBroadcastTimestamp(_ *C.ntg_instance, chatID C.int64_t, ptr unsafe.Pointer) {
	self := (*Client)(ptr)
	goChatID := int64(chatID)
	for _, cb := range self.broadcastTimestampCallbacks {
		go cb(goChatID)
	}
}

//export goOnRequestBroadcastPart
func goOnRequestBroadcastPart(_ *C.ntg_instance, chatID C.int64_t, segmentPartRequest C.ntg_segment_part_request, ptr unsafe.Pointer) {
	self := (*Client)(ptr)
	goChatID := int64(chatID)
	var goSegmentQuality MediaSegmentQuality
	switch segmentPartRequest.quality {
	case C.NTG_MEDIA_SEGMENT_QUALITY_NONE:
		goSegmentQuality = SegmentQualityNone
	case C.NTG_MEDIA_SEGMENT_QUALITY_THUMBNAIL:
		goSegmentQuality = SegmentQualityThumbnail
	case C.NTG_MEDIA_SEGMENT_QUALITY_MEDIUM:
		goSegmentQuality = SegmentQualityMedium
	case C.NTG_MEDIA_SEGMENT_QUALITY_FULL:
		goSegmentQuality = SegmentQualityFull
	}
	goSegmentPartRequest := SegmentPartRequest{
		SegmentID:     int64(segmentPartRequest.segment_id),
		PartID:        int32(segmentPartRequest.part_id),
		Limit:         int32(segmentPartRequest.limit),
		Timestamp:     int64(segmentPartRequest.timestamp),
		QualityUpdate: bool(segmentPartRequest.quality_update),
		ChannelID:     int32(segmentPartRequest.channel_id),
		Quality:       goSegmentQuality,
	}
	for _, cb := range self.broadcastPartCallbacks {
		go cb(goChatID, goSegmentPartRequest)
	}
}

// ---------------------------------------------------------------------------
// API methods on Client
// ---------------------------------------------------------------------------

func (c *Client) GetState(chatID int64) (MediaState, error) {
	var buffer C.ntg_media_state
	res := C.ntg_get_state(c.handle, C.int64_t(chatID), &buffer)
	if err := parseResult(res); err != nil {
		return MediaState{}, err
	}
	return MediaState{
		Muted:              bool(buffer.muted),
		VideoPaused:        bool(buffer.video_paused),
		VideoStopped:       bool(buffer.video_stopped),
		PresentationPaused: bool(buffer.presentation_paused),
	}, nil
}

func (c *Client) GetConnectionMode(chatID int64) (ConnectionMode, error) {
	var buffer C.ntg_connection_mode
	res := C.ntg_get_connection_mode(c.handle, C.int64_t(chatID), &buffer)
	if err := parseResult(res); err != nil {
		return ConnectionMode(0), err
	}
	switch buffer {
	case C.NTG_CONNECTION_MODE_RTC:
		return RtcConnection, nil
	case C.NTG_CONNECTION_MODE_STREAM:
		return StreamConnection, nil
	case C.NTG_CONNECTION_MODE_RTMP:
		return RTMPConnection, nil
	default:
		return ConnectionMode(0), fmt.Errorf("unknown connection mode")
	}
}

func (c *Client) CreateCall(chatID int64) (string, error) {
	var buffer *C.char
	res := C.ntg_create_call(c.handle, C.int64_t(chatID), &buffer)
	if err := parseResult(res); err != nil {
		return "", err
	}
	defer C.ntg_string_free(buffer)
	return C.GoString(buffer), nil
}

func (c *Client) InitPresentation(chatID int64) (string, error) {
	var buffer *C.char
	res := C.ntg_init_presentation(c.handle, C.int64_t(chatID), &buffer)
	if err := parseResult(res); err != nil {
		return "", err
	}
	defer C.ntg_string_free(buffer)
	return C.GoString(buffer), nil
}

func (c *Client) StopPresentation(chatID int64) error {
	res := C.ntg_stop_presentation(c.handle, C.int64_t(chatID))
	return parseResult(res)
}

func (c *Client) AddIncomingVideo(chatID int64, endpoint string, ssrcGroups []SsrcGroup) (uint32, error) {
	buffer := new(C.uint32_t)
	cEndpoint := C.CString(endpoint)
	defer C.free(unsafe.Pointer(cEndpoint))

	var cGroups *C.ntg_ssrc_group
	if len(ssrcGroups) > 0 {
		cGroups = parseSsrcGroups(ssrcGroups)
	}

	res := C.ntg_add_incoming_video(
		c.handle,
		C.int64_t(chatID),
		0,
		cEndpoint,
		cGroups,
		C.size_t(len(ssrcGroups)),
		buffer,
	)

	// Free ssrcGroups allocations
	if len(ssrcGroups) > 0 && cGroups != nil {
		for i := 0; i < len(ssrcGroups); i++ {
			groupPtr := (*C.ntg_ssrc_group)(unsafe.Pointer(uintptr(unsafe.Pointer(cGroups)) + uintptr(i)*unsafe.Sizeof(C.ntg_ssrc_group{})))
			C.free(unsafe.Pointer(groupPtr.semantics))
			if groupPtr.ssrcs != nil {
				C.free(unsafe.Pointer(groupPtr.ssrcs))
			}
		}
	}

	return uint32(*buffer), parseResult(res)
}

func (c *Client) RemoveIncomingVideo(chatID int64, endpoint string) error {
	cEndpoint := C.CString(endpoint)
	defer C.free(unsafe.Pointer(cEndpoint))
	var out C.bool
	res := C.ntg_remove_incoming_video(c.handle, C.int64_t(chatID), cEndpoint, &out)
	return parseResult(res)
}

func (c *Client) CreateP2P(userID int64) error {
	res := C.ntg_create_p2p_call(c.handle, C.int64_t(userID))
	return parseResult(res)
}

func (c *Client) ConnectP2P(userID int64, servers []RTCServer, versions []string, p2pAllowed bool) error {
	versionsC, sizeVersions := parseStringVectorC(versions)
	defer func() {
		if versionsC != nil {
			for i := 0; i < int(sizeVersions); i++ {
				p := *(**C.char)(unsafe.Pointer(uintptr(unsafe.Pointer(versionsC)) + uintptr(i)*unsafe.Sizeof(uintptr(0))))
				C.free(unsafe.Pointer(p))
			}
		}
	}()

	serversC := parseRtcServers(servers)
	defer func() {
		if serversC != nil {
			for i := 0; i < len(servers); i++ {
				s := (*C.ntg_rtc_server)(unsafe.Pointer(uintptr(unsafe.Pointer(serversC)) + uintptr(i)*unsafe.Sizeof(C.ntg_rtc_server{})))
				C.free(unsafe.Pointer(s.ipv4))
				C.free(unsafe.Pointer(s.ipv6))
				C.free(unsafe.Pointer(s.username))
				C.free(unsafe.Pointer(s.password))
				if s.peer_tag != nil {
					C.free(unsafe.Pointer(s.peer_tag))
				}
			}
		}
	}()

	res := C.ntg_connect_p2p(
		c.handle,
		C.int64_t(userID),
		serversC,
		C.size_t(len(servers)),
		versionsC,
		sizeVersions,
		C.bool(p2pAllowed),
		nil,
	)
	return parseResult(res)
}

func (c *Client) InitExchange(
	userID int64,
	g int32,
	p []byte,
	random []byte,
	gAHash []byte,
) ([]byte, error) {
	dh := DhConfig{G: g, P: p, Random: random}
	dhC := dh.parseToC()
	defer C.free(unsafe.Pointer(dhC.p))
	defer C.free(unsafe.Pointer(dhC.random))

	gAHashC, gAHashSize := parseBytes(gAHash)
	if gAHashC != nil {
		defer C.free(unsafe.Pointer(gAHashC))
	}

	var out *C.uint8_t
	var outSize C.size_t

	res := C.ntg_init_exchange(
		c.handle,
		C.int64_t(userID),
		dhC,
		gAHashC,
		gAHashSize,
		&out,
		&outSize,
	)

	if err := parseResult(res); err != nil {
		return nil, err
	}
	defer C.ntg_bytes_free(unsafe.Pointer(out))

	return C.GoBytes(unsafe.Pointer(out), C.int(outSize)), nil
}

func (c *Client) ExchangeKeys(
	userID int64,
	gAB []byte,
	fingerprint int64,
) (AuthParams, error) {
	var result C.ntg_auth_params

	gABC, gABSize := parseBytes(gAB)
	if gABC != nil {
		defer C.free(unsafe.Pointer(gABC))
	}

	res := C.ntg_exchange_keys(
		c.handle,
		C.int64_t(userID),
		gABC,
		gABSize,
		C.int64_t(fingerprint),
		&result,
	)

	if err := parseResult(res); err != nil {
		return AuthParams{}, err
	}
	defer C.ntg_auth_params_free(&result)

	var gab []byte
	if result.g_a_or_b != nil && result.g_a_or_b_len > 0 {
		gab = C.GoBytes(unsafe.Pointer(result.g_a_or_b), C.int(result.g_a_or_b_len))
	}

	return AuthParams{
		GAOrB:          gab,
		KeyFingerprint: int64(result.key_fingerprint),
	}, nil
}

func (c *Client) SkipExchange(userID int64, encryptionKey []byte, isOutgoing bool) error {
	if len(encryptionKey) == 0 {
		return fmt.Errorf("ntg: encryptionKey is empty")
	}

	encKeyC, encKeySize := parseBytes(encryptionKey)
	if encKeyC != nil {
		defer C.free(unsafe.Pointer(encKeyC))
	}

	res := C.ntg_skip_exchange(
		c.handle,
		C.int64_t(userID),
		encKeyC,
		encKeySize,
		C.bool(isOutgoing),
	)
	return parseResult(res)
}

func (c *Client) SendSignalingData(userID int64, data []byte) error {
	dataC, dataSize := parseBytes(data)
	if dataC != nil {
		defer C.free(unsafe.Pointer(dataC))
	}

	res := C.ntg_send_signaling_data(
		c.handle,
		C.int64_t(userID),
		dataC,
		dataSize,
	)
	return parseResult(res)
}

func GetProtocol() Protocol {
	var buf C.ntg_protocol
	res := C.ntg_get_protocol(&buf)
	if err := parseResult(res); err != nil {
		return Protocol{}
	}
	defer C.ntg_protocol_free(&buf)

	return Protocol{
		MinLayer:     int32(buf.min_layer),
		MaxLayer:     int32(buf.max_layer),
		UDPP2P:       bool(buf.udp_p2p),
		UDPReflector: bool(buf.udp_reflector),
		Versions:     parseStringVector(buf.library_versions, buf.library_versions_len),
	}
}

func (c *Client) Connect(chatID int64, params string, isPresentation bool) error {
	cParams := C.CString(params)
	defer C.free(unsafe.Pointer(cParams))

	res := C.ntg_connect(
		c.handle,
		C.int64_t(chatID),
		cParams,
		C.bool(isPresentation),
	)
	return parseResult(res)
}

func (c *Client) SetStreamSources(chatID int64, streamMode StreamMode, desc MediaDescription) error {
	descC := desc.parseToC()

	defer func() {
		if descC.microphone != nil {
			C.free(unsafe.Pointer(descC.microphone.input))
		}
		if descC.speaker != nil {
			C.free(unsafe.Pointer(descC.speaker.input))
		}
		if descC.camera != nil {
			C.free(unsafe.Pointer(descC.camera.input))
		}
		if descC.screen != nil {
			C.free(unsafe.Pointer(descC.screen.input))
		}
	}()

	res := C.ntg_set_stream_sources(
		c.handle,
		C.int64_t(chatID),
		streamMode.parseToC(),
		descC,
	)
	return parseResult(res)
}

func (c *Client) SendExternalFrame(chatID int64, streamDevice StreamDevice, data []byte, frameData FrameData) error {
	dataC, dataSize := parseBytes(data)
	if dataC != nil {
		defer C.free(unsafe.Pointer(dataC))
	}
	res := C.ntg_send_external_frame(c.handle, C.int64_t(chatID), streamDevice.ParseToC(), dataC, dataSize, frameData.ParseToC())
	return parseResult(res)
}

func (c *Client) SendBroadcastTimestamp(chatID int64, timestamp int64) error {
	res := C.ntg_send_broadcast_timestamp(c.handle, C.int64_t(chatID), C.int64_t(timestamp))
	return parseResult(res)
}

func (c *Client) SendBroadcastPart(chatID int64, segmentID int64, partID int32, status MediaSegmentStatus, qualityUpdate bool, data []byte) error {
	dataC, dataSize := parseBytes(data)
	if dataC != nil {
		defer C.free(unsafe.Pointer(dataC))
	}
	res := C.ntg_send_broadcast_part(c.handle, C.int64_t(chatID), C.int64_t(segmentID), C.int32_t(partID), status.ParseToC(), C.bool(qualityUpdate), dataC, dataSize)
	return parseResult(res)
}

func (c *Client) Pause(chatID int64) (bool, error) {
	var out C.bool
	res := C.ntg_pause(c.handle, C.int64_t(chatID), &out)
	return bool(out), parseResult(res)
}

func (c *Client) Resume(chatID int64) (bool, error) {
	var out C.bool
	res := C.ntg_resume(c.handle, C.int64_t(chatID), &out)
	return bool(out), parseResult(res)
}

func (c *Client) Mute(chatID int64) (bool, error) {
	var out C.bool
	res := C.ntg_mute(c.handle, C.int64_t(chatID), &out)
	return bool(out), parseResult(res)
}

func (c *Client) UnMute(chatID int64) (bool, error) {
	var out C.bool
	res := C.ntg_unmute(c.handle, C.int64_t(chatID), &out)
	return bool(out), parseResult(res)
}

func (c *Client) Stop(chatID int64) error {
	res := C.ntg_stop(c.handle, C.int64_t(chatID))
	return parseResult(res)
}

func (c *Client) Time(chatID int64, streamMode StreamMode) (uint64, error) {
	var buffer C.uint64_t
	res := C.ntg_time(c.handle, C.int64_t(chatID), streamMode.parseToC(), &buffer)
	return uint64(buffer), parseResult(res)
}

func GetMediaDevices() MediaDevices {
	var buffer C.ntg_media_devices
	res := C.ntg_get_media_devices(&buffer)
	if err := parseResult(res); err != nil {
		return MediaDevices{}
	}
	defer C.ntg_media_devices_free(&buffer)

	return MediaDevices{
		Microphone: parseDeviceInfoVector(buffer.microphone, buffer.microphone_len),
		Speaker:    parseDeviceInfoVector(buffer.speaker, buffer.speaker_len),
		Camera:     parseDeviceInfoVector(buffer.camera, buffer.camera_len),
		Screen:     parseDeviceInfoVector(buffer.screen, buffer.screen_len),
	}
}

func (c *Client) CPUUsage() (float64, error) {
	var buffer C.double
	res := C.ntg_cpu_usage(c.handle, &buffer)
	return float64(buffer), parseResult(res)
}

func (c *Client) EnableGLibLoop(enable bool) {
	C.ntg_enable_glib_loop(C.bool(enable))
}

func (c *Client) Calls() map[int64]*CallInfo {
	mapReturn := make(map[int64]*CallInfo)
	var buffer *C.ntg_call_info_entry
	var size C.size_t
	res := C.ntg_calls(c.handle, &buffer, &size)
	if err := parseResult(res); err != nil || buffer == nil {
		return mapReturn
	}
	defer C.ntg_call_info_entry_free(buffer, size)

	for i := 0; i < int(size); i++ {
		rawCall := *(*C.ntg_call_info_entry)(unsafe.Pointer(uintptr(unsafe.Pointer(buffer)) + uintptr(i)*unsafe.Sizeof(C.ntg_call_info_entry{})))
		mapReturn[int64(rawCall.key)] = &CallInfo{
			Playback: parseStreamStatus(rawCall.value.playback),
			Capture:  parseStreamStatus(rawCall.value.capture),
		}
	}
	return mapReturn
}

func Version() string {
	buffer := C.ntg_get_version()
	return C.GoString(buffer)
}

// SetupDefaultAudio configures microphone capture and speaker playback using real devices.
func (c *Client) SetupDefaultAudio(userID int64) error {
	return c.SetupRealAudio(userID)
}

// SetupTestAudio configures microphone capture and speaker playback
// using the best available audio utilities on the system.
func (c *Client) SetupTestAudio(userID int64) error {
	slog.Info("🎙️ Setting up default audio using test URL via ffmpeg")
	urlVideoTest := "https://docs.evostream.com/sample_content/assets/sintel1m720p.mp4"

	_, err := exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("ffmpeg not found, required for test audio streaming")
	}

	mic := AudioDescription{
		MediaSource:  MediaSourceShell,
		SampleRate:   48000,
		ChannelCount: 1,
		KeepOpen:     true,
	}

	baseFFmpeg := "ffmpeg -reconnect 1 -reconnect_at_eof 1 -reconnect_streamed 1 -reconnect_delay_max 2 -i"
	mic.Input = fmt.Sprintf(
		"%s %s -f s16le -ac %d -ar %d -v quiet pipe:1",
		baseFFmpeg,
		urlVideoTest,
		mic.ChannelCount,
		mic.SampleRate,
	)

	descCapture := MediaDescription{
		Microphone: &mic,
	}

	if err := c.SetStreamSources(userID, CaptureStream, descCapture); err != nil {
		return fmt.Errorf("set capture stream: %w", err)
	}

	// Try native device first, fall back to shell playback
	devices := GetMediaDevices()
	if len(devices.Speaker) > 0 {
		slog.Info("🔊 Using native speaker device", "name", devices.Speaker[0].Name, "metadata", devices.Speaker[0].Metadata)
		speaker := AudioDescription{
			MediaSource:  MediaSourceDevice,
			Input:        devices.Speaker[0].Metadata,
			SampleRate:   48000,
			ChannelCount: 1,
			KeepOpen:     true,
		}
		descPlayback := MediaDescription{
			Microphone: &speaker,
		}
		if err := c.SetStreamSources(userID, PlaybackStream, descPlayback); err != nil {
			slog.Warn("🔊 Native device playback failed, falling back to shell", "error", err)
		} else {
			return nil
		}
	}

	// Fallback: use shell-based playback
	var cmdPlayback string
	_, errPlay := exec.LookPath("pw-play")
	_, errPacat := exec.LookPath("pacat")
	_, errAplay := exec.LookPath("aplay")

	if errPlay == nil {
		slog.Info("🔊 PipeWire detected, using pw-play for playback")
		cmdPlayback = "pw-play --format s16 --rate 48000 --channels 1 --latency 20ms -"
	} else if errPacat == nil {
		slog.Info("🔊 PulseAudio detected, using pacat for playback")
		cmdPlayback = "pacat --playback --raw --format=s16le --rate=48000 --channels=1 --latency-msec=40"
	} else if errAplay == nil {
		slog.Info("🔊 ALSA detected, using aplay for playback")
		cmdPlayback = "aplay -f S16_LE -r 48000 -c 1 -t raw -B 20000 -"
	} else {
		slog.Warn("🔊 No playback method available")
	}

	if cmdPlayback != "" {
		speaker := AudioDescription{
			MediaSource:  MediaSourceShell,
			Input:        cmdPlayback,
			SampleRate:   48000,
			ChannelCount: 1,
			KeepOpen:     true,
		}
		descPlayback := MediaDescription{
			Microphone: &speaker,
		}
		if err := c.SetStreamSources(userID, PlaybackStream, descPlayback); err != nil {
			return fmt.Errorf("set playback stream: %w", err)
		}
	}

	return nil
}

// SetupRealAudio configures real microphone capture and speaker playback
func (c *Client) SetupRealAudio(userID int64) error {
	slog.Info("🎙️ Setting up real audio")

	devices := GetMediaDevices()

	// Microphone setup
	var mic *AudioDescription
	if len(devices.Microphone) > 0 {
		slog.Info("🎙️ Using native microphone device", "name", devices.Microphone[0].Name, "metadata", devices.Microphone[0].Metadata)
		mic = &AudioDescription{
			MediaSource:  MediaSourceDevice,
			Input:        devices.Microphone[0].Metadata,
			SampleRate:   48000,
			ChannelCount: 1,
			KeepOpen:     true,
		}
		descCapture := MediaDescription{
			Microphone: mic,
		}
		if err := c.SetStreamSources(userID, CaptureStream, descCapture); err != nil {
			slog.Warn("🎙️ Native device capture failed, falling back to shell", "error", err)
			mic = nil // reset so we fallback
		}
	}

	if mic == nil {
		var cmdCapture string
		_, errRecord := exec.LookPath("pw-record")
		_, errParec := exec.LookPath("parec")
		_, errArecord := exec.LookPath("arecord")

		if errRecord == nil {
			slog.Info("🎙️ PipeWire detected, using pw-record for capture")
			cmdCapture = "pw-record --format s16 --rate 48000 --channels 1 -"
		} else if errParec == nil {
			slog.Info("🎙️ PulseAudio detected, using parec for capture")
			cmdCapture = "parec --format=s16le --rate=48000 --channels=1 --raw"
		} else if errArecord == nil {
			slog.Info("🎙️ ALSA detected, using arecord for capture")
			cmdCapture = "arecord -f S16_LE -r 48000 -c 1 -t raw -"
		} else {
			slog.Warn("🎙️ No capture method available")
		}

		if cmdCapture != "" {
			mic = &AudioDescription{
				MediaSource:  MediaSourceShell,
				Input:        cmdCapture,
				SampleRate:   48000,
				ChannelCount: 1,
				KeepOpen:     true,
			}
			descCapture := MediaDescription{
				Microphone: mic,
			}
			if err := c.SetStreamSources(userID, CaptureStream, descCapture); err != nil {
				return fmt.Errorf("set capture stream fallback: %w", err)
			}
		}
	}

	// Speaker setup
	var speaker *AudioDescription
	if len(devices.Speaker) > 0 {
		slog.Info("🔊 Using native speaker device", "name", devices.Speaker[0].Name, "metadata", devices.Speaker[0].Metadata)
		speaker = &AudioDescription{
			MediaSource:  MediaSourceDevice,
			Input:        devices.Speaker[0].Metadata,
			SampleRate:   48000,
			ChannelCount: 1,
			KeepOpen:     true,
		}
		descPlayback := MediaDescription{
			Microphone: speaker,
		}
		if err := c.SetStreamSources(userID, PlaybackStream, descPlayback); err != nil {
			slog.Warn("🔊 Native device playback failed, falling back to shell", "error", err)
			speaker = nil // reset so we fallback
		}
	}

	if speaker == nil {
		var cmdPlayback string
		_, errPlay := exec.LookPath("pw-play")
		_, errPacat := exec.LookPath("pacat")
		_, errAplay := exec.LookPath("aplay")

		if errPlay == nil {
			slog.Info("🔊 PipeWire detected, using pw-play for playback")
			cmdPlayback = "pw-play --format s16 --rate 48000 --channels 1 --latency 20ms -"
		} else if errPacat == nil {
			slog.Info("🔊 PulseAudio detected, using pacat for playback")
			cmdPlayback = "pacat --playback --raw --format=s16le --rate=48000 --channels=1 --latency-msec=40"
		} else if errAplay == nil {
			slog.Info("🔊 ALSA detected, using aplay for playback")
			cmdPlayback = "aplay -f S16_LE -r 48000 -c 1 -t raw -B 20000 -"
		} else {
			slog.Warn("🔊 No playback method available")
		}

		if cmdPlayback != "" {
			speaker := AudioDescription{
				MediaSource:  MediaSourceShell,
				Input:        cmdPlayback,
				SampleRate:   48000,
				ChannelCount: 1,
				KeepOpen:     true,
			}
			descPlayback := MediaDescription{
				Microphone: &speaker,
			}
			if err := c.SetStreamSources(userID, PlaybackStream, descPlayback); err != nil {
				return fmt.Errorf("set playback stream fallback: %w", err)
			}
		}
	}

	return nil
}
