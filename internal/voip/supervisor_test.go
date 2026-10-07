package voip_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kumneger0/cligram/internal/voip"
)

func TestSupervisor_FindSidecarBinary_NotFound(t *testing.T) {
	// With an absurd binary name that doesn't exist
	path, err := voip.FindSidecarBinaryCustom("nonexistent-cligram-voip-xyz")
	if !errors.Is(err, voip.ErrHelperNotFound) {
		t.Fatalf("expected ErrHelperNotFound, got err=%v, path=%s", err, path)
	}
}

func TestSupervisor_FindSidecarBinary_Found(t *testing.T) {
	tmpDir := t.TempDir()
	fakeBinary := filepath.Join(tmpDir, "cligram-voip-test")
	if err := os.WriteFile(fakeBinary, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("failed to write fake binary: %v", err)
	}

	// Add tmpDir to PATH
	origPath := os.Getenv("PATH")
	t.Setenv("PATH", tmpDir+string(os.PathListSeparator)+origPath)

	path, err := voip.FindSidecarBinaryCustom("cligram-voip-test")
	if err != nil {
		t.Fatalf("expected to find binary, got err=%v", err)
	}
	if path != fakeBinary {
		t.Fatalf("expected %s, got %s", fakeBinary, path)
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
