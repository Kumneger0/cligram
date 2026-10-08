package main_test

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"testing"

	main "github.com/kumneger0/cligram/cmd/cligram-voip"
	"github.com/kumneger0/cligram/ntg"
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

func TestPrintProtocol(t *testing.T) {
	proto := main.GetProtocol()
	t.Logf("NTG PROTOCOL: %+v", proto)
}

func TestMediaDevices(t *testing.T) {
	devices := ntg.GetMediaDevices()
	t.Logf("Mics: %d", len(devices.Microphone))
	for i, m := range devices.Microphone {
		t.Logf("  Mic %d: name=%s, metadata=%s", i, m.Name, m.Metadata)
	}
	t.Logf("Speakers: %d", len(devices.Speaker))
	for i, s := range devices.Speaker {
		t.Logf("  Speaker %d: name=%s, metadata=%s", i, s.Name, s.Metadata)
	}
}

func TestSetupAudio(t *testing.T) {
	client := ntg.Init()
	defer client.Destroy()

	userID := int64(99999)
	if err := client.CreateP2P(userID); err != nil {
		t.Fatalf("CreateP2P failed: %v", err)
	}

	if err := client.SetupRealAudio(userID); err != nil {
		t.Fatalf("SetupRealAudio failed: %v", err)
	}
	t.Log("SetupRealAudio succeeded!")
}

func TestTeeReaderPlayback(t *testing.T) {
	client := ntg.Init()
	defer client.Destroy()

	userID := int64(99998)
	if err := client.CreateP2P(userID); err != nil {
		t.Fatalf("CreateP2P failed: %v", err)
	}

	fifoPath := "/tmp/test_cligram_playback.fifo"
	_ = os.Remove(fifoPath)
	if err := exec.Command("mkfifo", fifoPath).Run(); err != nil {
		t.Fatalf("mkfifo failed: %v", err)
	}
	defer os.Remove(fifoPath)

	// Open reader in background to prevent FIFO open blocking
	readStarted := make(chan struct{})
	go func() {
		close(readStarted)
		f, err := os.OpenFile(fifoPath, os.O_RDONLY, 0)
		if err == nil {
			defer f.Close()
			buf := make([]byte, 1024)
			for {
				if _, err := f.Read(buf); err != nil {
					break
				}
			}
		}
	}()

	<-readStarted

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

	if err := client.SetStreamSources(userID, ntg.PlaybackStream, media); err != nil {
		t.Fatalf("SetStreamSources with MediaSourceFile failed: %v", err)
	}
	t.Log("SetStreamSources with FIFO succeeded!")
}

func TestPipewirePlayWithTeeReader(t *testing.T) {
	data := make([]byte, 48000*2/10) // 100ms
	src := bytes.NewReader(data)
	dumpFile, err := os.CreateTemp("", "test_dump_*.raw")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(dumpFile.Name())
	defer dumpFile.Close()

	tee := io.TeeReader(src, dumpFile)
	cmd := exec.Command("pw-play", "--format", "s16", "--rate", "48000", "--channels", "1", "--latency", "20ms", "-")
	cmd.Stdin = tee
	if err := cmd.Run(); err != nil {
		t.Logf("pw-play returned: %v", err)
	}
	info, _ := dumpFile.Stat()
	if info.Size() != int64(len(data)) {
		t.Fatalf("expected %d bytes in dump, got %d", len(data), info.Size())
	}
	t.Logf("TeeReader successfully wrote %d bytes to dumpFile while pw-play ran!", info.Size())
}
