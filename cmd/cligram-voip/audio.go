package main

import (
	"log/slog"
	"os/exec"
)

type AudioToolCandidate struct {
	SoundServer string
	Binary      string
	Command     string
}

// GetAudioCandidates returns the prioritized list of capture and playback tools.
func GetAudioCandidates() ([]AudioToolCandidate, []AudioToolCandidate) {
	captureCandidates := []AudioToolCandidate{
		{
			SoundServer: "PipeWire",
			Binary:      "pw-record",
			Command:     "pw-record --format s16 --rate 48000 --channels 1 -",
		},
		{
			SoundServer: "PulseAudio",
			Binary:      "parec",
			Command:     "parec --format=s16le --rate=48000 --channels=1 --raw",
		},
		{
			SoundServer: "ALSA",
			Binary:      "arecord",
			Command:     "arecord -f S16_LE -r 48000 -c 1 -t raw -",
		},
	}

	playbackCandidates := []AudioToolCandidate{
		{
			SoundServer: "PipeWire",
			Binary:      "pw-play",
			Command:     "pw-play --format s16 --rate 48000 --channels 1 --latency 20ms -",
		},
		{
			SoundServer: "PulseAudio",
			Binary:      "pacat",
			Command:     "pacat --playback --raw --format=s16le --rate=48000 --channels=1 --latency-msec=40",
		},
		{
			SoundServer: "ALSA",
			Binary:      "aplay",
			Command:     "aplay -f S16_LE -r 48000 -c 1 -t raw -B 20000 -",
		},
	}

	return captureCandidates, playbackCandidates
}

// ProbeAudioTool checks if an audio tool is available and executable on the system.
func ProbeAudioTool(binary string) bool {
	path, err := exec.LookPath(binary)
	return err == nil && path != ""
}

// ResolveAudioCommands traverses the Audio Fallback Chain and returns the available capture and playback commands.
func ResolveAudioCommands() (string, string, string) {
	captures, playbacks := GetAudioCandidates()

	var selectedCapture string
	var selectedPlayback string
	var selectedServer string

	for i := 0; i < len(captures); i++ {
		c := captures[i]
		p := playbacks[i]

		if ProbeAudioTool(c.Binary) && ProbeAudioTool(p.Binary) {
			slog.Info("🎙️ Found audio pair", "server", c.SoundServer, "capture", c.Binary, "playback", p.Binary)
			return c.Command, p.Command, c.SoundServer
		}
	}

	// If no matching pair, resolve independently
	for _, c := range captures {
		if _, err := exec.LookPath(c.Binary); err == nil {
			selectedCapture = c.Command
			selectedServer = c.SoundServer
			break
		}
	}
	for _, p := range playbacks {
		if _, err := exec.LookPath(p.Binary); err == nil {
			selectedPlayback = p.Command
			if selectedServer == "" {
				selectedServer = p.SoundServer
			}
			break
		}
	}

	return selectedCapture, selectedPlayback, selectedServer
}
