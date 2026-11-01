package voip

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

var ErrHelperNotFound = errors.New("call-helper worker executable not found")

// FindWorkerExecutable resolves the current cligram executable path.
func FindWorkerExecutable() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrHelperNotFound, err)
	}
	return exePath, nil
}

// Supervisor manages the lifecycle of the out-of-process voice call-helper worker.
type Supervisor struct {
	customCmd []string

	mu        sync.Mutex
	cmd       *exec.Cmd
	bridge    *JSONRPCBridge
	idleTimer *time.Timer
	onIdle    func()
	onExit    func(error)
}

// NewSupervisor creates a supervisor. Optional customCmd can be supplied (mainly for tests)
// to override the executable and arguments. When omitted, it launches os.Executable() with argument "call-helper".
func NewSupervisor(customCmd ...string) *Supervisor {
	return &Supervisor{
		customCmd: customCmd,
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

// Start spawns the call-helper worker if not already running, establishes the anonymous socketpair on FD 3, and returns the bridge.
func (s *Supervisor) Start(ctx context.Context) (*JSONRPCBridge, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cancelIdleTimerLocked()

	if s.bridge != nil {
		return s.bridge, nil
	}

	var binaryPath string
	var args []string

	if len(s.customCmd) > 0 && s.customCmd[0] != "" {
		binaryPath = s.customCmd[0]
		if len(s.customCmd) > 1 {
			args = s.customCmd[1:]
		}
	} else {
		var err error
		binaryPath, err = FindWorkerExecutable()
		if err != nil {
			return nil, err
		}
		args = []string{"call-helper"}
	}

	// Create anonymous socketpair
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		return nil, fmt.Errorf("socketpair: %w", err)
	}

	parentFile := os.NewFile(uintptr(fds[0]), "cligram-call-helper-parent")
	childFile := os.NewFile(uintptr(fds[1]), "cligram-call-helper-child")

	cmd := exec.CommandContext(ctx, binaryPath, args...)
	cmd.ExtraFiles = []*os.File{childFile} // In child, ExtraFiles start at FD 3

	// Redirect worker stdout and stderr to /tmp/cligram-call-helper.log for observability
	if logFile, err := os.OpenFile("/tmp/cligram-call-helper.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err == nil {
		cmd.Stdout = logFile
		cmd.Stderr = logFile
		defer logFile.Close()
	}

	if err := cmd.Start(); err != nil {
		_ = parentFile.Close()
		_ = childFile.Close()
		return nil, fmt.Errorf("start call-helper worker: %w", err)
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

// Stop terminates the running worker process cleanly.
func (s *Supervisor) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cancelIdleTimerLocked()

	if s.bridge != nil {
		_ = s.bridge.Close()
		s.bridge = nil
	}

	if s.cmd != nil && s.cmd.Process != nil {
		slog.Info("terminating cligram call-helper worker")
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

// ResetIdleTimer schedules graceful shutdown of the worker after duration d.
func (s *Supervisor) ResetIdleTimer(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cancelIdleTimerLocked()
	s.idleTimer = time.AfterFunc(d, func() {
		slog.Info("cligram call-helper idle timeout reached, stopping worker")
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
