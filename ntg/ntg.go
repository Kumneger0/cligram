package ntg // nolint:revive

/*
#cgo LDFLAGS: -L/usr/local/lib -lntgcalls -ldl -lpthread -lopus -lcrypto -lm

#include <stdint.h>
#include <stdlib.h>
#include "ntgcalls.h"

extern void goOnSignaling(uintptr_t ptr, int64_t userId, uint8_t* data, int size, void* userData);
extern void goOnConnectionChange(uintptr_t ptr, int64_t chatID, ntg_network_info_struct networkInfo, void* userData);
extern void goOnFrames(uintptr_t ptr, int64_t chatID, ntg_stream_mode_enum streamMode, ntg_stream_device_enum streamDevice, ntg_frame_struct* frames, uint64_t size, void* userData);
extern void goOnRemoteSourceChange(uintptr_t ptr, int64_t chatID, ntg_remote_source_struct source, void* userData);
extern void goOnLog(ntg_log_message_struct message);
extern void goOnStreamEnd(uintptr_t ptr, int64_t chatID, ntg_stream_type_enum streamType, ntg_stream_device_enum streamDevice, void* userData);
extern void goOnUpgrade(uintptr_t ptr, int64_t chatID, ntg_media_state_struct state, void* userData);
extern void goOnRequestBroadcastTimestamp(uintptr_t ptr, int64_t chatID, void* userData);
extern void goOnRequestBroadcastPart(uintptr_t ptr, int64_t chatID, ntg_segment_part_request_struct segmentPartRequest, void* userData);

extern void unlockMutex(void*);
*/
import "C"

import (
	"fmt"
	"log/slog"
	"os/exec"
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

func parseBool(f *Future) (bool, error) {
	return *f.errCode == 0, parseError(f)
}

func parseBytes(data []byte) (*C.uint8_t, C.int) {
	if len(data) > 0 {
		rawBytes := C.CBytes(data)
		return (*C.uint8_t)(rawBytes), C.int(len(data))
	}
	return nil, 0
}

func parseStringVector(data unsafe.Pointer, size C.int) []string {
	result := make([]string, size)
	for i := 0; i < int(size); i++ {
		pointer := *(**C.char)(unsafe.Pointer(uintptr(data) + uintptr(i)*unsafe.Sizeof(uintptr(0))))
		result[i] = C.GoString(pointer)
		C.free(unsafe.Pointer(pointer))
	}
	defer C.free(data)
	return result
}

func parseStringVectorC(data []string) (**C.char, C.int) {
	if len(data) > 0 {
		rawData := make([]*C.char, len(data))
		for i, v := range data {
			rawData[i] = C.CString(v)
		}
		return &rawData[0], C.int(len(data))
	}
	return nil, 0
}

func parseUint32VectorC(data []uint32) (*C.uint32_t, C.int) {
	if len(data) > 0 {
		cData := C.malloc(C.size_t(len(data)) * C.size_t(unsafe.Sizeof(C.uint32_t(0))))
		if cData == nil {
			return nil, 0
		}
		ssrcs := (*C.uint32_t)(cData)
		for i, v := range data {
			*(*C.uint32_t)(unsafe.Pointer(uintptr(unsafe.Pointer(ssrcs)) + uintptr(i)*unsafe.Sizeof(C.uint32_t(0)))) = C.uint32_t(v)
		}
		return ssrcs, C.int(len(data))
	}
	return nil, 0
}

func parseSsrcGroups(ssrcGroups []SsrcGroup) *C.ntg_ssrc_group_struct {
	if len(ssrcGroups) > 0 {
		rawGroups := make([]C.ntg_ssrc_group_struct, len(ssrcGroups))
		for i, group := range ssrcGroups {
			ssrcsC, sizeSsrcs := parseUint32VectorC(group.Ssrcs)
			rawGroups[i] = C.ntg_ssrc_group_struct{
				semantics: C.CString(group.Semantics),
				ssrcs:     ssrcsC,
				sizeSsrcs: sizeSsrcs,
			}
		}
		return (*C.ntg_ssrc_group_struct)(unsafe.Pointer(&rawGroups[0]))
	}
	return nil
}

func parseDeviceInfoVector(devices unsafe.Pointer, size C.int) []DeviceInfo {
	rawDevices := make([]DeviceInfo, size)
	for i := 0; i < int(size); i++ {
		device := *(*C.ntg_device_info_struct)(unsafe.Pointer(uintptr(devices) + uintptr(i)*unsafe.Sizeof(C.ntg_device_info_struct{})))
		rawDevices[i] = DeviceInfo{
			Name:     C.GoString(device.name),
			Metadata: C.GoString(device.metadata),
		}
		C.free(unsafe.Pointer(device.name))
		C.free(unsafe.Pointer(device.metadata))
	}
	defer C.free(devices)
	return rawDevices
}

func parseConnectionState(state C.ntg_connection_state_enum) ConnectionState {
	switch state {
	case C.NTG_STATE_CONNECTING:
		return Connecting
	case C.NTG_STATE_CONNECTED:
		return Connected
	case C.NTG_STATE_TIMEOUT:
		return Timeout
	case C.NTG_STATE_FAILED:
		return Failed
	case C.NTG_STATE_CLOSED:
		return Closed
	}
	return Connecting
}

func parseStreamDevice(device C.ntg_stream_device_enum) StreamDevice {
	switch device {
	case C.NTG_STREAM_MICROPHONE:
		return MicrophoneStream
	case C.NTG_STREAM_SPEAKER:
		return SpeakerStream
	case C.NTG_STREAM_CAMERA:
		return CameraStream
	case C.NTG_STREAM_SCREEN:
		return ScreenStream
	}
	return MicrophoneStream
}

func parseStreamStatus(status C.ntg_stream_status_enum) StreamStatus {
	switch status {
	case C.NTG_ACTIVE:
		return ActiveStream
	case C.NTG_PAUSED:
		return PausedStream
	case C.NTG_IDLING:
		return IdlingStream
	}
	return ActiveStream
}

func parseRtcServers(rtcServers []RTCServer) *C.ntg_rtc_server_struct {
	if len(rtcServers) > 0 {
		rawServers := make([]C.ntg_rtc_server_struct, len(rtcServers))
		for i, server := range rtcServers {
			rawServers[i] = C.ntg_rtc_server_struct{
				id:          C.uint64_t(server.ID),
				ipv4:        C.CString(server.IPv4),
				ipv6:        C.CString(server.IPv6),
				username:    C.CString(server.Username),
				password:    C.CString(server.Password),
				port:        C.uint16_t(server.Port),
				turn:        C.bool(server.IsTURN),
				stun:        C.bool(server.IsSTUN),
				tcp:         C.bool(server.IsTCP),
				peerTag:     nil,
				peerTagSize: 0,
			}
			if len(server.PeerTag) > 0 {
				peerTagC, peerTagSize := parseBytes(server.PeerTag)
				rawServers[i].peerTag = peerTagC
				rawServers[i].peerTagSize = peerTagSize
			}
		}
		return (*C.ntg_rtc_server_struct)(unsafe.Pointer(&rawServers[0]))
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

func (sm StreamMode) parseToC() C.ntg_stream_mode_enum {
	switch sm {
	case CaptureStream:
		return C.NTG_STREAM_CAPTURE
	case PlaybackStream:
		return C.NTG_STREAM_PLAYBACK
	default:
		return C.NTG_STREAM_CAPTURE
	}
}

type StreamDevice int

const (
	MicrophoneStream StreamDevice = iota
	SpeakerStream
	CameraStream
	ScreenStream
)

func (sd StreamDevice) ParseToC() C.ntg_stream_device_enum {
	switch sd {
	case MicrophoneStream:
		return C.NTG_STREAM_MICROPHONE
	case SpeakerStream:
		return C.NTG_STREAM_SPEAKER
	case CameraStream:
		return C.NTG_STREAM_CAMERA
	case ScreenStream:
		return C.NTG_STREAM_SCREEN
	default:
		return C.NTG_STREAM_MICROPHONE
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

func (ms MediaSource) parseToC() C.ntg_media_source_enum {
	switch ms {
	case MediaSourceFile:
		return C.NTG_FILE
	case MediaSourceShell:
		return C.NTG_SHELL
	case MediaSourceFFmpeg:
		return C.NTG_FFMPEG
	case MediaSourceDevice:
		return C.NTG_DEVICE
	case MediaSourceDesktop:
		return C.NTG_DESKTOP
	case MediaSourceExternal:
		return C.NTG_EXTERNAL
	default:
		return C.NTG_FILE
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

func (mss MediaSegmentStatus) ParseToC() C.ntg_media_segment_status_enum {
	switch mss {
	case SegmentStatusNotReady:
		return C.NTG_MEDIA_SEGMENT_NOT_READY
	case SegmentStatusResyncNeeded:
		return C.NTG_MEDIA_SEGMENT_RESYNC_NEEDED
	case SegmentStatusSuccess:
		return C.NTG_MEDIA_SEGMENT_SUCCESS
	default:
		return C.NTG_MEDIA_SEGMENT_NOT_READY
	}
}

// MediaState mirrors ntg_media_state_struct.
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

// DeviceInfo mirrors ntg_device_info_struct.
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

// NetworkInfo is a Go-side mirror of ntg_network_info_struct.
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

// FrameData mirrors ntg_frame_data_struct.
type FrameData struct {
	AbsoluteCaptureTimestampMs int64
	Width                      uint16
	Height                     uint16
	Rotation                   uint16
}

func (fd *FrameData) ParseToC() C.ntg_frame_data_struct {
	return C.ntg_frame_data_struct{
		absoluteCaptureTimestampMs: C.int64_t(fd.AbsoluteCaptureTimestampMs),
		width:                      C.uint16_t(fd.Width),
		height:                     C.uint16_t(fd.Height),
		rotation:                   C.uint16_t(fd.Rotation),
	}
}

// Frame mirrors a single ntg_frame_struct.
type Frame struct {
	Ssrc      uint32
	Data      []byte
	FrameData FrameData
}

// RemoteSource mirrors ntg_remote_source_struct.
type RemoteSource struct {
	Ssrc   uint32
	State  StreamStatus
	Device StreamDevice
}

// AudioDescription mirrors ntg_audio_description_struct.
type AudioDescription struct {
	MediaSource  MediaSource
	Input        string
	SampleRate   uint32
	ChannelCount uint8
	KeepOpen     bool
}

func (ad *AudioDescription) parseToC() C.ntg_audio_description_struct {
	var x C.ntg_audio_description_struct
	x.mediaSource = ad.MediaSource.parseToC()
	x.input = C.CString(ad.Input)
	x.sampleRate = C.uint32_t(ad.SampleRate)
	x.channelCount = C.uint8_t(ad.ChannelCount)
	x.keepOpen = C.bool(ad.KeepOpen)
	return x
}

// VideoDescription mirrors ntg_video_description_struct.
type VideoDescription struct {
	MediaSource   MediaSource
	Input         string
	Width, Height int16
	Fps           uint8
	KeepOpen      bool
}

func (vd *VideoDescription) parseToC() C.ntg_video_description_struct {
	var x C.ntg_video_description_struct
	x.mediaSource = vd.MediaSource.parseToC()
	x.input = C.CString(vd.Input)
	x.width = C.int16_t(vd.Width)
	x.height = C.int16_t(vd.Height)
	x.fps = C.uint8_t(vd.Fps)
	x.keepOpen = C.bool(vd.KeepOpen)
	return x
}

// MediaDescription mirrors ntg_media_description_struct.
type MediaDescription struct {
	Microphone *AudioDescription
	Speaker    *AudioDescription
	Camera     *VideoDescription
	Screen     *VideoDescription
}

func (md *MediaDescription) parseToC() C.ntg_media_description_struct {
	var x C.ntg_media_description_struct
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

// SegmentPartRequest mirrors ntg_segment_part_request_struct.
type SegmentPartRequest struct {
	SegmentID     int64
	PartID        int32
	Limit         int32
	Timestamp     int64
	QualityUpdate bool
	ChannelID     int32
	Quality       MediaSegmentQuality
}

// DhConfig mirrors ntg_dh_config_struct.
type DhConfig struct {
	G      int32
	P      []byte
	Random []byte
}

func (dh *DhConfig) parseToC() C.ntg_dh_config_struct {
	var x C.ntg_dh_config_struct
	x.g = C.int32_t(dh.G)
	pC, pSize := parseBytes(dh.P)
	rC, rSize := parseBytes(dh.Random)
	x.p = pC
	x.sizeP = pSize
	x.random = rC
	x.sizeRandom = rSize
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
	ptr                         C.uintptr_t
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
		ptr: C.ntg_init(),
	}

	// Register C callbacks with a pointer back to this Go instance.
	selfPointer := unsafe.Pointer(instance)
	C.ntg_on_signaling_data(instance.ptr, (C.ntg_signaling_callback)(unsafe.Pointer(C.goOnSignaling)), selfPointer)
	C.ntg_on_connection_change(instance.ptr, (C.ntg_connection_callback)(unsafe.Pointer(C.goOnConnectionChange)), selfPointer)
	C.ntg_on_frames(instance.ptr, (C.ntg_frame_callback)(unsafe.Pointer(C.goOnFrames)), selfPointer)
	C.ntg_on_remote_source_change(instance.ptr, (C.ntg_remote_source_callback)(unsafe.Pointer(C.goOnRemoteSourceChange)), selfPointer)
	C.ntg_on_stream_end(instance.ptr, (C.ntg_stream_callback)(unsafe.Pointer(C.goOnStreamEnd)), selfPointer)
	C.ntg_on_upgrade(instance.ptr, (C.ntg_upgrade_callback)(unsafe.Pointer(C.goOnUpgrade)), selfPointer)
	C.ntg_on_request_broadcast_timestamp(instance.ptr, (C.ntg_broadcast_timestamp_callback)(unsafe.Pointer(C.goOnRequestBroadcastTimestamp)), selfPointer)
	C.ntg_on_request_broadcast_part(instance.ptr, (C.ntg_broadcast_part_callback)(unsafe.Pointer(C.goOnRequestBroadcastPart)), selfPointer)

	// Register the library logger to pipe internal ntgcalls WebRTC & playback logs to slog
	C.ntg_register_logger((C.ntg_log_message_callback)(unsafe.Pointer(C.goOnLog)))

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
	var goInfo NetworkInfo
	switch networkInfo.kind {
	case C.NTG_KIND_NORMAL:
		goInfo.Kind = NormalConnection
	case C.NTG_KIND_PRESENTATION:
		goInfo.Kind = PresentationConnection
	}
	goInfo.State = parseConnectionState(networkInfo.state)
	for _, cb := range self.connectionChangeCallbacks {
		go cb(int64(chatID), goInfo)
	}
}

//export goOnRemoteSourceChange
func goOnRemoteSourceChange(_ C.uintptr_t, chatID C.int64_t, source C.ntg_remote_source_struct, ptr unsafe.Pointer) {
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
func goOnLog(msg C.ntg_log_message_struct) {
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
func goOnFrames(_ C.uintptr_t, chatID C.int64_t, streamMode C.ntg_stream_mode_enum, streamDevice C.ntg_stream_device_enum, frames *C.ntg_frame_struct, size C.uint64_t, ptr unsafe.Pointer) {
	self := (*Client)(ptr)
	goChatID := int64(chatID)

	var goStreamMode StreamMode
	switch streamMode {
	case C.NTG_STREAM_CAPTURE:
		goStreamMode = CaptureStream
	case C.NTG_STREAM_PLAYBACK:
		goStreamMode = PlaybackStream
	}

	rawFrames := make([]Frame, size)
	for i := uint64(0); i < uint64(size); i++ {
		rawFrame := *(*C.ntg_frame_struct)(unsafe.Pointer(uintptr(unsafe.Pointer(frames)) + uintptr(i)*unsafe.Sizeof(C.ntg_frame_struct{})))
		rawFrames[i] = Frame{
			Ssrc: uint32(rawFrame.ssrc),
			Data: C.GoBytes(unsafe.Pointer(rawFrame.data), rawFrame.sizeData),
			FrameData: FrameData{
				AbsoluteCaptureTimestampMs: int64(rawFrame.frameData.absoluteCaptureTimestampMs),
				Width:                      uint16(rawFrame.frameData.width),
				Height:                     uint16(rawFrame.frameData.height),
				Rotation:                   uint16(rawFrame.frameData.rotation),
			},
		}
	}

	for _, cb := range self.frameCallbacks {
		go cb(goChatID, goStreamMode, parseStreamDevice(streamDevice), rawFrames)
	}
}

//export goOnStreamEnd
func goOnStreamEnd(_ C.uintptr_t, chatID C.int64_t, streamType C.ntg_stream_type_enum, streamDevice C.ntg_stream_device_enum, ptr unsafe.Pointer) {
	self := (*Client)(ptr)
	goChatID := int64(chatID)
	var goStreamType StreamType
	if streamType == C.NTG_STREAM_AUDIO {
		goStreamType = AudioStream
	} else {
		goStreamType = VideoStream
	}
	for _, cb := range self.streamEndCallbacks {
		go cb(goChatID, goStreamType, parseStreamDevice(streamDevice))
	}
}

//export goOnUpgrade
func goOnUpgrade(_ C.uintptr_t, chatID C.int64_t, state C.ntg_media_state_struct, ptr unsafe.Pointer) {
	self := (*Client)(ptr)
	goChatID := int64(chatID)
	goState := MediaState{
		Muted:              bool(state.muted),
		VideoPaused:        bool(state.videoPaused),
		VideoStopped:       bool(state.videoStopped),
		PresentationPaused: bool(state.presentationPaused),
	}
	for _, cb := range self.upgradeCallbacks {
		go cb(goChatID, goState)
	}
}

//export goOnRequestBroadcastTimestamp
func goOnRequestBroadcastTimestamp(_ C.uintptr_t, chatID C.int64_t, ptr unsafe.Pointer) {
	self := (*Client)(ptr)
	goChatID := int64(chatID)
	for _, cb := range self.broadcastTimestampCallbacks {
		go cb(goChatID)
	}
}

//export goOnRequestBroadcastPart
func goOnRequestBroadcastPart(_ C.uintptr_t, chatID C.int64_t, segmentPartRequest C.ntg_segment_part_request_struct, ptr unsafe.Pointer) {
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
		SegmentID:     int64(segmentPartRequest.segmentId),
		PartID:        int32(segmentPartRequest.partId),
		Limit:         int32(segmentPartRequest.limit),
		Timestamp:     int64(segmentPartRequest.timestamp),
		QualityUpdate: bool(segmentPartRequest.qualityUpdate),
		ChannelID:     int32(segmentPartRequest.channelId),
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
	f := newFuture()
	var buffer C.ntg_media_state_struct
	C.ntg_get_state(c.ptr, C.int64_t(chatID), &buffer, f.toC())
	f.wait()
	err := parseError(f)
	if err != nil {
		return MediaState{}, err
	}
	return MediaState{
		Muted:              bool(buffer.muted),
		VideoPaused:        bool(buffer.videoPaused),
		VideoStopped:       bool(buffer.videoStopped),
		PresentationPaused: bool(buffer.presentationPaused),
	}, nil
}

func (c *Client) GetConnectionMode(chatID int64) (ConnectionMode, error) {
	f := newFuture()
	var buffer C.ntg_connection_mode_enum
	C.ntg_get_connection_mode(c.ptr, C.int64_t(chatID), &buffer, f.toC())
	f.wait()
	err := parseError(f)
	if err != nil {
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
	f := newFuture()
	C.ntg_create(c.ptr, C.int64_t(chatID), &buffer, f.toC())
	f.wait()
	if err := parseError(f); err != nil {
		return "", err
	}
	defer C.free(unsafe.Pointer(buffer))
	return C.GoString(buffer), nil
}

func (c *Client) InitPresentation(chatID int64) (string, error) {
	var buffer *C.char
	f := newFuture()
	C.ntg_init_presentation(c.ptr, C.int64_t(chatID), &buffer, f.toC())
	f.wait()
	if err := parseError(f); err != nil {
		return "", err
	}
	defer C.free(unsafe.Pointer(buffer))
	return C.GoString(buffer), nil
}

func (c *Client) StopPresentation(chatID int64) error {
	f := newFuture()
	C.ntg_stop_presentation(c.ptr, C.int64_t(chatID), f.toC())
	f.wait()
	return parseError(f)
}

func (c *Client) AddIncomingVideo(chatID int64, endpoint string, ssrcGroups []SsrcGroup) (uint32, error) {
	buffer := new(C.uint32_t)
	f := newFuture()
	cEndpoint := C.CString(endpoint)
	defer C.free(unsafe.Pointer(cEndpoint))

	var cGroups *C.ntg_ssrc_group_struct
	if len(ssrcGroups) > 0 {
		cGroups = parseSsrcGroups(ssrcGroups)
	}

	C.ntg_add_incoming_video(
		c.ptr,
		C.int64_t(chatID),
		cEndpoint,
		cGroups,
		C.int(len(ssrcGroups)),
		buffer,
		f.toC(),
	)
	f.wait()

	// Free ssrcGroups allocations
	if len(ssrcGroups) > 0 && cGroups != nil {
		for i := 0; i < len(ssrcGroups); i++ {
			groupPtr := (*C.ntg_ssrc_group_struct)(unsafe.Pointer(uintptr(unsafe.Pointer(cGroups)) + uintptr(i)*unsafe.Sizeof(C.ntg_ssrc_group_struct{})))
			C.free(unsafe.Pointer(groupPtr.semantics))
			if groupPtr.ssrcs != nil {
				C.free(unsafe.Pointer(groupPtr.ssrcs))
			}
		}
	}

	return uint32(*buffer), parseError(f)
}

func (c *Client) RemoveIncomingVideo(chatID int64, endpoint string) error {
	cEndpoint := C.CString(endpoint)
	defer C.free(unsafe.Pointer(cEndpoint))
	f := newFuture()
	C.ntg_remove_incoming_video(c.ptr, C.int64_t(chatID), cEndpoint, f.toC())
	f.wait()
	return parseError(f)
}

func (c *Client) CreateP2P(userID int64) error {
	f := newFuture()
	C.ntg_create_p2p(c.ptr, C.int64_t(userID), f.toC())
	f.wait()
	return parseError(f)
}

func (c *Client) ConnectP2P(userID int64, servers []RTCServer, versions []string, p2pAllowed bool) error {
	f := newFuture()
	versionsC, sizeVersions := parseStringVectorC(versions)
	C.ntg_connect_p2p(
		c.ptr,
		C.int64_t(userID),
		parseRtcServers(servers),
		C.int(len(servers)),
		versionsC,
		C.int(sizeVersions),
		C.bool(p2pAllowed),
		f.toC(),
	)
	f.wait()
	return parseError(f)
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

	gAHashC, gAHashSize := parseBytes(gAHash)

	var out *C.uint8_t
	var outSize C.int

	f := newFuture()
	C.ntg_init_exchange(
		c.ptr,
		C.int64_t(userID),
		&dhC,
		gAHashC,
		gAHashSize,
		&out,
		&outSize,
		f.toC(),
	)
	f.wait()

	defer C.free(unsafe.Pointer(dhC.p))
	defer C.free(unsafe.Pointer(dhC.random))
	if gAHashC != nil {
		defer C.free(unsafe.Pointer(gAHashC))
	}
	defer C.free(unsafe.Pointer(out))

	if err := parseError(f); err != nil {
		return nil, err
	}

	return C.GoBytes(unsafe.Pointer(out), outSize), nil
}

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

	var gab []byte
	if result.g_a_or_b != nil && result.sizeGAB > 0 {
		gab = C.GoBytes(unsafe.Pointer(result.g_a_or_b), result.sizeGAB)
		defer C.free(unsafe.Pointer(result.g_a_or_b))
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

func GetProtocol() Protocol {
	var buf C.ntg_protocol_struct
	C.ntg_get_protocol(&buf)

	return Protocol{
		MinLayer:     int32(buf.minLayer),
		MaxLayer:     int32(buf.maxLayer),
		UDPP2P:       bool(buf.udpP2P),
		UDPReflector: bool(buf.udpReflector),
		Versions:     parseStringVector(unsafe.Pointer(buf.libraryVersions), buf.libraryVersionsSize),
	}
}

func (c *Client) Connect(chatID int64, params string, isPresentation bool) error {
	cParams := C.CString(params)
	defer C.free(unsafe.Pointer(cParams))

	f := newFuture()
	C.ntg_connect(
		c.ptr,
		C.int64_t(chatID),
		cParams,
		C.bool(isPresentation),
		f.toC(),
	)
	f.wait()
	return parseError(f)
}

func (c *Client) SetStreamSources(chatID int64, streamMode StreamMode, desc MediaDescription) error {
	f := newFuture()
	descC := desc.parseToC()

	// Defer frees of all C.CString inputs inside nested structures
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

	C.ntg_set_stream_sources(
		c.ptr,
		C.int64_t(chatID),
		streamMode.parseToC(),
		descC,
		f.toC(),
	)
	f.wait()
	return parseError(f)
}

func (c *Client) SendExternalFrame(chatID int64, streamDevice StreamDevice, data []byte, frameData FrameData) error {
	f := newFuture()
	dataC, dataSize := parseBytes(data)
	if dataC != nil {
		defer C.free(unsafe.Pointer(dataC))
	}
	C.ntg_send_external_frame(c.ptr, C.int64_t(chatID), streamDevice.ParseToC(), dataC, dataSize, frameData.ParseToC(), f.toC())
	f.wait()
	return parseError(f)
}

func (c *Client) SendBroadcastTimestamp(chatID int64, timestamp int64) error {
	f := newFuture()
	C.ntg_send_broadcast_timestamp(c.ptr, C.int64_t(chatID), C.int64_t(timestamp), f.toC())
	f.wait()
	return parseError(f)
}

func (c *Client) SendBroadcastPart(chatID int64, segmentID int64, partID int32, status MediaSegmentStatus, qualityUpdate bool, data []byte) error {
	f := newFuture()
	dataC, dataSize := parseBytes(data)
	if dataC != nil {
		defer C.free(unsafe.Pointer(dataC))
	}
	C.ntg_send_broadcast_part(c.ptr, C.int64_t(chatID), C.int64_t(segmentID), C.int32_t(partID), status.ParseToC(), C.bool(qualityUpdate), dataC, dataSize, f.toC())
	f.wait()
	return parseError(f)
}

func (c *Client) Pause(chatID int64) (bool, error) {
	f := newFuture()
	C.ntg_pause(c.ptr, C.int64_t(chatID), f.toC())
	f.wait()
	return parseBool(f)
}

func (c *Client) Resume(chatID int64) (bool, error) {
	f := newFuture()
	C.ntg_resume(c.ptr, C.int64_t(chatID), f.toC())
	f.wait()
	return parseBool(f)
}

func (c *Client) Mute(chatID int64) (bool, error) {
	f := newFuture()
	C.ntg_mute(c.ptr, C.int64_t(chatID), f.toC())
	f.wait()
	return parseBool(f)
}

func (c *Client) UnMute(chatID int64) (bool, error) {
	f := newFuture()
	C.ntg_unmute(c.ptr, C.int64_t(chatID), f.toC())
	f.wait()
	return parseBool(f)
}

func (c *Client) Stop(chatID int64) error {
	f := newFuture()
	C.ntg_stop(c.ptr, C.int64_t(chatID), f.toC())
	f.wait()
	return parseError(f)
}

func (c *Client) Time(chatID int64, streamMode StreamMode) (uint64, error) {
	f := newFuture()
	var buffer C.int64_t
	C.ntg_time(c.ptr, C.int64_t(chatID), streamMode.parseToC(), &buffer, f.toC())
	f.wait()
	return uint64(buffer), parseError(f)
}

func GetMediaDevices() MediaDevices {
	var buffer C.ntg_media_devices_struct
	C.ntg_get_media_devices(&buffer)
	return MediaDevices{
		Microphone: parseDeviceInfoVector(unsafe.Pointer(buffer.microphone), buffer.sizeMicrophone),
		Speaker:    parseDeviceInfoVector(unsafe.Pointer(buffer.speaker), buffer.sizeSpeaker),
		Camera:     parseDeviceInfoVector(unsafe.Pointer(buffer.camera), buffer.sizeCamera),
		Screen:     parseDeviceInfoVector(unsafe.Pointer(buffer.screen), buffer.sizeScreen),
	}
}

func (c *Client) CPUUsage() (float64, error) {
	f := newFuture()
	var buffer C.double
	C.ntg_cpu_usage(c.ptr, &buffer, f.toC())
	f.wait()
	return float64(buffer), parseError(f)
}

func (c *Client) EnableGLibLoop(enable bool) {
	C.ntg_enable_g_lib_loop(C.bool(enable))
}

func (c *Client) Calls() map[int64]*CallInfo {
	mapReturn := make(map[int64]*CallInfo)
	f := newFuture()
	var buffer *C.ntg_call_info_struct
	var size C.int
	C.ntg_calls(c.ptr, &buffer, &size, f.toC())
	f.wait()
	for i := 0; i < int(size); i++ {
		rawCall := *(*C.ntg_call_info_struct)(unsafe.Pointer(uintptr(unsafe.Pointer(buffer)) + uintptr(i)*unsafe.Sizeof(C.ntg_call_info_struct{})))
		mapReturn[int64(rawCall.chatId)] = &CallInfo{
			Playback: parseStreamStatus(rawCall.playback),
			Capture:  parseStreamStatus(rawCall.capture),
		}
	}
	if buffer != nil {
		defer C.free(unsafe.Pointer(buffer))
	}
	return mapReturn
}

func Version() string {
	var buffer *C.char
	C.ntg_get_version(&buffer)
	defer C.free(unsafe.Pointer(buffer))
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
			Speaker: &speaker,
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
			Speaker: &speaker,
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
			Speaker: speaker,
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
			speaker = &AudioDescription{
				MediaSource:  MediaSourceShell,
				Input:        cmdPlayback,
				SampleRate:   48000,
				ChannelCount: 1,
				KeepOpen:     true,
			}
			descPlayback := MediaDescription{
				Speaker: speaker,
			}
			if err := c.SetStreamSources(userID, PlaybackStream, descPlayback); err != nil {
				return fmt.Errorf("set playback stream fallback: %w", err)
			}
		}
	}

	return nil
}
