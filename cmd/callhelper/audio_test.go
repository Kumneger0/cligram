package callhelper_test

import (
	"testing"

	"github.com/kumneger0/cligram/cmd/callhelper"
	"github.com/kumneger0/cligram/ntg"
)

func TestAudioToolCandidates(t *testing.T) {
	captureCandidates, playbackCandidates := callhelper.GetAudioCandidates()

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
	captureCmd, playbackCmd, serverName := callhelper.ResolveAudioCommands()
	// In Linux test environments, at least one of PipeWire, PulseAudio, or ALSA is typically installed
	t.Logf("Resolved audio server: %s, capture: %s, playback: %s", serverName, captureCmd, playbackCmd)
}

func TestPrintProtocol(t *testing.T) {
	proto := callhelper.GetProtocol()
	t.Logf("NTG PROTOCOL: %+v", proto)
}

func TestNativeShellStreamSources(t *testing.T) {
	client := ntg.Init()
	defer client.Destroy()

	userID := int64(99999)
	if err := client.CreateP2P(userID); err != nil {
		t.Fatalf("CreateP2P failed: %v", err)
	}

	captureCmd, playbackCmd, _ := callhelper.ResolveAudioCommands()
	if captureCmd == "" {
		captureCmd = "echo test"
	}
	if playbackCmd == "" {
		playbackCmd = "cat > /dev/null"
	}

	capDesc := &ntg.AudioDescription{
		MediaSource:  ntg.MediaSourceShell,
		Input:        captureCmd,
		SampleRate:   48000,
		ChannelCount: 1,
		KeepOpen:     true,
	}
	if err := client.SetStreamSources(userID, ntg.CaptureStream, ntg.MediaDescription{Microphone: capDesc}); err != nil {
		t.Fatalf("SetStreamSources capture failed: %v", err)
	}

	playDesc := &ntg.AudioDescription{
		MediaSource:  ntg.MediaSourceShell,
		Input:        playbackCmd,
		SampleRate:   48000,
		ChannelCount: 1,
		KeepOpen:     true,
	}
	if err := client.SetStreamSources(userID, ntg.PlaybackStream, ntg.MediaDescription{Speaker: playDesc}); err != nil {
		t.Fatalf("SetStreamSources playback failed: %v", err)
	}

	t.Log("SetStreamSources with native shell capture and playback succeeded!")
}

func TestGetEmojisFingerprint(t *testing.T) {
	client := ntg.Init()
	defer client.Destroy()

	userID := int64(99997)
	if err := client.CreateP2P(userID); err != nil {
		t.Fatalf("CreateP2P failed: %v", err)
	}

	emojis, err := client.GetEmojisFingerprint(userID)
	if err != nil {
		t.Logf("GetEmojisFingerprint before key exchange returned err (expected): %v", err)
	} else {
		t.Logf("GetEmojisFingerprint returned: %q", emojis)
	}
}
