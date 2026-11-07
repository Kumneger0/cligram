package notification

import (
	"bytes"
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestRingtone_Lifecycle(t *testing.T) {
	var playCount atomic.Int32
	SetRingtonePlayer(func(ctx context.Context, rt RingtoneType) {
		playCount.Add(1)
	})
	defer ResetRingtonePlayer()

	if IsRingtoneActive() {
		t.Errorf("expected ringtone not active initially")
	}

	StartRingtone(RingtoneIncoming)
	if !IsRingtoneActive() {
		t.Errorf("expected ringtone active after StartRingtone")
	}

	// Give a moment for the initial play to execute
	time.Sleep(50 * time.Millisecond)
	if playCount.Load() == 0 {
		t.Errorf("expected ringtone player to be invoked at least once")
	}

	StopRingtone()
	if IsRingtoneActive() {
		t.Errorf("expected ringtone not active after StopRingtone")
	}
}

func TestRingtone_WavGeneration(t *testing.T) {
	outgoing := generateOutgoingRing(8000)
	if len(outgoing) < 44 {
		t.Fatalf("expected outgoing wav size >= 44, got %d", len(outgoing))
	}
	if !bytes.HasPrefix(outgoing, []byte("RIFF")) {
		t.Errorf("expected RIFF header in outgoing wav")
	}

	incoming := generateIncomingChime(8000)
	if len(incoming) < 44 {
		t.Fatalf("expected incoming wav size >= 44, got %d", len(incoming))
	}
	if !bytes.HasPrefix(incoming, []byte("RIFF")) {
		t.Errorf("expected RIFF header in incoming wav")
	}
}
