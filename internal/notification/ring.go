package notification

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

type RingtoneType int

const (
	RingtoneIncoming RingtoneType = iota
	RingtoneOutgoing
)

var (
	ringMu       sync.Mutex
	ringCancel   context.CancelFunc
	ringActive   bool
	ringPlayerFn = defaultPlayRingtoneOnce

	outgoingWavOnce sync.Once
	outgoingWavPath string

	incomingWavOnce sync.Once
	incomingWavPath string
)

// SetRingtonePlayer allows tests to intercept or disable audio execution.
func SetRingtonePlayer(fn func(ctx context.Context, rt RingtoneType)) {
	ringMu.Lock()
	defer ringMu.Unlock()
	ringPlayerFn = fn
}

// ResetRingtonePlayer resets the ringtone player to default.
func ResetRingtonePlayer() {
	ringMu.Lock()
	defer ringMu.Unlock()
	ringPlayerFn = defaultPlayRingtoneOnce
}

// StartRingtone begins looping the ringing sound in the background until StopRingtone is called.
func StartRingtone(rt RingtoneType) {
	ringMu.Lock()
	defer ringMu.Unlock()

	if ringCancel != nil {
		ringCancel()
	}

	ctx, cancel := context.WithCancel(context.Background())
	ringCancel = cancel
	ringActive = true

	go func(ctx context.Context, rt RingtoneType) {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			ringMu.Lock()
			fn := ringPlayerFn
			ringMu.Unlock()

			if fn != nil {
				fn(ctx, rt)
			}

			interval := 2500 * time.Millisecond
			if rt == RingtoneIncoming {
				interval = 2000 * time.Millisecond
			}

			select {
			case <-ctx.Done():
				return
			case <-time.After(interval):
			}
		}
	}(ctx, rt)
}

// StopRingtone immediately halts any running ringtone playback and cancels the loop.
func StopRingtone() {
	ringMu.Lock()
	defer ringMu.Unlock()

	if ringCancel != nil {
		ringCancel()
		ringCancel = nil
	}
	ringActive = false
}

// IsRingtoneActive reports whether a ringtone is currently active.
func IsRingtoneActive() bool {
	ringMu.Lock()
	defer ringMu.Unlock()
	return ringActive
}

func defaultPlayRingtoneOnce(ctx context.Context, rt RingtoneType) {
	// 1. On Linux, prefer desktop sound theme if canberra-gtk-play is available
	if runtime.GOOS == "linux" {
		themeID := "phone-outgoing-calling"
		if rt == RingtoneIncoming {
			themeID = "phone-incoming-call"
		}
		if _, err := exec.LookPath("canberra-gtk-play"); err == nil {
			cmd := exec.CommandContext(ctx, "canberra-gtk-play", "-i", themeID)
			if err := cmd.Run(); err == nil {
				return
			}
		}
	}

	// 2. Play synthesized WAV tone via system audio player
	wavPath := getRingtoneWav(rt)
	if wavPath != "" {
		if playWav(ctx, wavPath) {
			return
		}
	}

	// 3. Fallback: terminal bell
	select {
	case <-ctx.Done():
	default:
		fmt.Print("\a")
	}
}

func playWav(ctx context.Context, wavPath string) bool {
	var candidates []string
	if runtime.GOOS == "darwin" {
		candidates = []string{"afplay"}
	} else {
		candidates = []string{"pw-play", "paplay", "aplay"}
	}

	for _, bin := range candidates {
		if path, err := exec.LookPath(bin); err == nil && path != "" {
			var cmd *exec.Cmd
			if bin == "aplay" {
				cmd = exec.CommandContext(ctx, bin, "-q", wavPath)
			} else {
				cmd = exec.CommandContext(ctx, bin, wavPath)
			}
			if err := cmd.Run(); err == nil {
				return true
			}
		}
	}
	return false
}

func getRingtoneWav(rt RingtoneType) string {
	if rt == RingtoneIncoming {
		incomingWavOnce.Do(func() {
			data := generateIncomingChime(8000)
			p := filepath.Join(os.TempDir(), "cligram-incoming-ring.wav")
			if err := os.WriteFile(p, data, 0o644); err == nil {
				incomingWavPath = p
			}
		})
		return incomingWavPath
	}

	outgoingWavOnce.Do(func() {
		data := generateOutgoingRing(8000)
		p := filepath.Join(os.TempDir(), "cligram-outgoing-ring.wav")
		if err := os.WriteFile(p, data, 0o644); err == nil {
			outgoingWavPath = p
		}
	})
	return outgoingWavPath
}

func generateOutgoingRing(sampleRate int) []byte {
	durationSec := 1.2
	numSamples := int(float64(sampleRate) * durationSec)
	buf := new(bytes.Buffer)

	writeWavHeader(buf, sampleRate, numSamples)

	fadeSec := 0.04
	for i := range numSamples {
		t := float64(i) / float64(sampleRate)
		env := 1.0
		if t < fadeSec {
			env = t / fadeSec
		} else if t > durationSec-fadeSec {
			env = (durationSec - t) / fadeSec
		}

		s := 0.5*math.Sin(2*math.Pi*440*t) + 0.5*math.Sin(2*math.Pi*480*t)
		val := int16(s * env * 20000)
		_ = binary.Write(buf, binary.LittleEndian, val)
	}

	return buf.Bytes()
}

func generateIncomingChime(sampleRate int) []byte {
	notes := []float64{523.25, 659.25, 783.99, 1046.50} // C5, E5, G5, C6
	noteDuration := 0.22
	totalDuration := float64(len(notes)) * noteDuration
	numSamples := int(float64(sampleRate) * totalDuration)

	buf := new(bytes.Buffer)
	writeWavHeader(buf, sampleRate, numSamples)

	samplesPerNote := int(float64(sampleRate) * noteDuration)
	for _, freq := range notes {
		for i := range samplesPerNote {
			t := float64(i) / float64(sampleRate)
			env := math.Exp(-t * 6.0)
			val := int16(0.7 * math.Sin(2*math.Pi*freq*t) * env * 24000)
			_ = binary.Write(buf, binary.LittleEndian, val)
		}
	}

	return buf.Bytes()
}

func writeWavHeader(buf *bytes.Buffer, sampleRate, numSamples int) {
	buf.WriteString("RIFF")
	_ = binary.Write(buf, binary.LittleEndian, uint32(36+numSamples*2))
	buf.WriteString("WAVEfmt ")
	_ = binary.Write(buf, binary.LittleEndian, uint32(16))
	_ = binary.Write(buf, binary.LittleEndian, uint16(1)) // PCM
	_ = binary.Write(buf, binary.LittleEndian, uint16(1)) // Mono
	_ = binary.Write(buf, binary.LittleEndian, uint32(sampleRate))
	_ = binary.Write(buf, binary.LittleEndian, uint32(sampleRate*2))
	_ = binary.Write(buf, binary.LittleEndian, uint16(2))
	_ = binary.Write(buf, binary.LittleEndian, uint16(16))
	buf.WriteString("data")
	_ = binary.Write(buf, binary.LittleEndian, uint32(numSamples*2))
}
