package voip_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kumneger0/cligram/internal/voip"
)

func TestSupervisor_FindWorkerExecutable(t *testing.T) {
	path, err := voip.FindWorkerExecutable()
	if err != nil {
		t.Fatalf("expected to resolve executable, got err=%v", err)
	}
	if path == "" {
		t.Fatal("expected non-empty executable path")
	}
}

func TestSupervisor_IdleTimeout(t *testing.T) {
	sup := voip.NewSupervisor("nonexistent")
	idleFired := make(chan struct{})

	sup.SetIdleTimeoutHandler(func() {
		close(idleFired)
	})

	sup.ResetIdleTimer(50 * time.Millisecond)

	select {
	case <-idleFired:
		// success
	case <-time.After(500 * time.Millisecond):
		t.Fatal("idle timer failed to fire")
	}
}

func TestSupervisor_CrashIsolation(t *testing.T) {
	// Create a dummy worker script that simulates a crash (exits with code 42 immediately)
	tmpDir := t.TempDir()
	crashedWorker := filepath.Join(tmpDir, "crash_worker.sh")
	scriptContent := "#!/bin/sh\nexit 42\n"
	if err := os.WriteFile(crashedWorker, []byte(scriptContent), 0o755); err != nil {
		t.Fatalf("failed to write crash worker: %v", err)
	}

	sup := voip.NewSupervisor(crashedWorker)

	exitReceived := make(chan error, 1)
	sup.OnExit(func(err error) {
		exitReceived <- err
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	bridge, err := sup.Start(ctx)
	if err != nil {
		t.Fatalf("expected Start to launch process without panic, got err=%v", err)
	}
	if bridge == nil {
		t.Fatal("expected non-nil bridge")
	}

	// Verify that worker crash is caught and parent process remains healthy and intact
	select {
	case err := <-exitReceived:
		if err == nil {
			t.Fatal("expected non-nil error indicating process crash")
		}
		// Confirm supervisor is no longer marked running
		if sup.IsRunning() {
			t.Fatal("expected supervisor to not be running after crash")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for worker crash notification")
	}

	// Verify Stop does not panic on crashed process
	if err := sup.Stop(); err != nil {
		t.Fatalf("Stop() after crash failed: %v", err)
	}
}

func TestSupervisor_StartRealCligramCallHelper(t *testing.T) {
	cligramBin, err := filepath.Abs("../../cligram")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(cligramBin); err != nil {
		t.Skip("cligram binary not built yet in repo root, skipping real binary test")
	}

	sup := voip.NewSupervisor(cligramBin, "call-helper")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	bridge, err := sup.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start call-helper with real binary: %v", err)
	}
	defer func() {
		_ = sup.Stop()
	}()

	if !sup.IsRunning() {
		t.Fatal("expected supervisor to report running")
	}

	// Ping or create call to verify IPC communication over socketpair
	p := make([]byte, 256)
	p[0] = 0x7F
	p[255] = 0x01
	random := make([]byte, 256)
	random[0] = 0x42
	res, err := bridge.CreateCall(ctx, voip.CreateCallParams{
		UserID:   12345,
		DHG:      3,
		DHP:      p,
		DHRandom: random,
		GAOrB:    nil,
	})
	if err != nil {
		t.Fatalf("CreateCall RPC failed: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil CreateCallResult")
	}
	t.Logf("CreateCallResult received successfully: minLayer=%d, maxLayer=%d, versions=%v", res.MinLayer, res.MaxLayer, res.LibraryVersions)
}
