package main_test

import (
	"testing"

	main "github.com/kumneger0/cligram/cmd/cligram-voip"
)

func TestAudioToolCandidates(t *testing.T) {
	captureCandidates, playbackCandidates := main.GetAudioCandidates()

	if len(captureCandidates) < 3 {
		t.Errorf("expected at least 3 capture candidates (PipeWire, PulseAudio, ALSA), got %d", len(captureCandidates))
	}
	if len(playbackCandidates) < 3 {
		t.Errorf("expected at least 3 playback candidates (PipeWire, PulseAudio, ALSA), got %d", len(playbackCandidates))
	}

	// Verify priority order: PipeWire first, then PulseAudio, then ALSA
	if captureCandidates[0].SoundServer != "PipeWire" {
		t.Errorf("expected PipeWire as first capture candidate, got %s", captureCandidates[0].SoundServer)
	}
	if captureCandidates[1].SoundServer != "PulseAudio" {
		t.Errorf("expected PulseAudio as second capture candidate, got %s", captureCandidates[1].SoundServer)
	}
	if captureCandidates[2].SoundServer != "ALSA" {
		t.Errorf("expected ALSA as third capture candidate, got %s", captureCandidates[2].SoundServer)
	}
}

func TestResolveAudioCommands(t *testing.T) {
	captureCmd, playbackCmd, serverName := main.ResolveAudioCommands()
	// In Linux test environments, at least one of PipeWire, PulseAudio, or ALSA is typically installed
	t.Logf("Resolved audio server: %s, capture: %s, playback: %s", serverName, captureCmd, playbackCmd)
}
