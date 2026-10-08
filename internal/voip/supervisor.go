package voip

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

var ErrHelperNotFound = errors.New("cligram-voip helper binary not found")

// FindSidecarBinary searches for the cligram-voip executable in standard locations.
func FindSidecarBinary() (string, error) {
	return FindSidecarBinaryCustom("cligram-voip")
}

// FindSidecarBinaryCustom searches for a specific binary name in:
// 1. Next to current executable
// 2. Current working directory (for 'go run .')
// 3. PATH
// 4. Standard user and system bin locations (~/.local/bin, /usr/local/bin, /usr/bin)
func FindSidecarBinaryCustom(binaryName string) (string, error) {
	// 1. Check next to current executable
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		candidate := filepath.Join(exeDir, binaryName)
		if fi, err := os.Stat(candidate); err == nil && !fi.IsDir() && fi.Mode()&0o111 != 0 {
			return candidate, nil
		}
	}

	// 2. Check current working directory (e.g. when run with 'go run .')
	if cwd, err := os.Getwd(); err == nil {
		candidate := filepath.Join(cwd, binaryName)
		if fi, err := os.Stat(candidate); err == nil && !fi.IsDir() && fi.Mode()&0o111 != 0 {
			return candidate, nil
		}
	}

	// 3. Check PATH
	if p, err := exec.LookPath(binaryName); err == nil {
		return p, nil
	}

	// 4. Check ~/.local/bin
	if homeDir, err := os.UserHomeDir(); err == nil {
		candidate := filepath.Join(homeDir, ".local", "bin", binaryName)
		if fi, err := os.Stat(candidate); err == nil && !fi.IsDir() && fi.Mode()&0o111 != 0 {
			return candidate, nil
		}
	}

	// 5. Check standard system locations
	for _, dir := range []string{"/usr/local/bin", "/usr/bin"} {
		candidate := filepath.Join(dir, binaryName)
		if fi, err := os.Stat(candidate); err == nil && !fi.IsDir() && fi.Mode()&0o111 != 0 {
			return candidate, nil
		}
	}

	return "", ErrHelperNotFound
}

// Supervisor manages the lifecycle of the out-of-process VoIP sidecar.
type Supervisor struct {
	binaryName string

	mu        sync.Mutex
	cmd       *exec.Cmd
	bridge    *JSONRPCBridge
	idleTimer *time.Timer
	onIdle    func()
	onExit    func(error)
}

// NewSupervisor creates a supervisor targeting binaryName (defaults to "cligram-voip").
func NewSupervisor(binaryName string) *Supervisor {
	if binaryName == "" {
		binaryName = "cligram-voip"
	}
	return &Supervisor{
		binaryName: binaryName,
	}
}

func (s *Supervisor) SetIdleTimeoutHandler(h func()) {
	s.mu.Lock()
	s.onIdle = h
	s.mu.Unlock()
}

func (s *Supervisor) OnExit(h func(error)) {
	s.mu.Lock()
	s.onExit = h
	s.mu.Unlock()
}

// Start spawns the sidecar if not already running, establishes the anonymous socketpair on FD 3, and returns the bridge.
func (s *Supervisor) Start(ctx context.Context) (SignalingBridge, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cancelIdleTimerLocked()

	if s.bridge != nil {
		return s.bridge, nil
	}

	binaryPath, err := FindSidecarBinaryCustom(s.binaryName)
	if err != nil {
		return nil, err
	}

	// Create anonymous socketpair
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		return nil, fmt.Errorf("socketpair: %w", err)
	}

	parentFile := os.NewFile(uintptr(fds[0]), "cligram-voip-parent")
	childFile := os.NewFile(uintptr(fds[1]), "cligram-voip-child")

	cmd := exec.CommandContext(ctx, binaryPath)
	cmd.ExtraFiles = []*os.File{childFile} // In child, ExtraFiles start at FD 3

	// Redirect sidecar stdout and stderr to /tmp/cligram-voip.log for observability
	if logFile, err := os.OpenFile("/tmp/cligram-voip.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		cmd.Stdout = logFile
		cmd.Stderr = logFile
		defer logFile.Close()
	}

	if err := cmd.Start(); err != nil {
		_ = parentFile.Close()
		_ = childFile.Close()
		return nil, fmt.Errorf("start sidecar: %w", err)
	}

	// Child holds its own file descriptor; close child end in parent
	_ = childFile.Close()

	bridge := NewJSONRPCBridge(parentFile)
	bridge.Start()

	s.cmd = cmd
	s.bridge = bridge

	// Monitor child process exit
	go func(c *exec.Cmd, b *JSONRPCBridge) {
		waitErr := c.Wait()
		_ = b.Close()

		s.mu.Lock()
		if s.cmd == c {
			s.cmd = nil
			s.bridge = nil
		}
		exitHandler := s.onExit
		s.mu.Unlock()

		if exitHandler != nil {
			exitHandler(waitErr)
		}
	}(cmd, bridge)

	return bridge, nil
}

// Stop terminates the running sidecar process cleanly.
func (s *Supervisor) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cancelIdleTimerLocked()

	if s.bridge != nil {
		_ = s.bridge.Close()
		s.bridge = nil
	}

	if s.cmd != nil && s.cmd.Process != nil {
		slog.Info("terminating cligram-voip sidecar")
		_ = s.cmd.Process.Signal(syscall.SIGTERM)
		s.cmd = nil
	}

	return nil
}

func (s *Supervisor) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.bridge != nil
}

// ResetIdleTimer schedules graceful shutdown of the sidecar after duration d.
func (s *Supervisor) ResetIdleTimer(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cancelIdleTimerLocked()
	s.idleTimer = time.AfterFunc(d, func() {
		slog.Info("cligram-voip idle timeout reached, stopping sidecar")
		_ = s.Stop()

		s.mu.Lock()
		onIdle := s.onIdle
		s.mu.Unlock()

		if onIdle != nil {
			onIdle()
		}
	})
}

// CancelIdleTimer cancels any active idle shutdown countdown.
func (s *Supervisor) CancelIdleTimer() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cancelIdleTimerLocked()
}

func (s *Supervisor) cancelIdleTimerLocked() {
	if s.idleTimer != nil {
		s.idleTimer.Stop()
		s.idleTimer = nil
	}
}
