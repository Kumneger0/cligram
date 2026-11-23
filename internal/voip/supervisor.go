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

func FindWorkerExecutable() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrHelperNotFound, err)
	}
	return exePath, nil
}

type Supervisor struct {
	customCmd []string
	mu        sync.Mutex
	cmd       *exec.Cmd
	bridge    *JSONRPCBridge
	idleTimer *time.Timer
	onIdle    func()
	onExit    func(error)
}

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

	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		return nil, fmt.Errorf("socketpair: %w", err)
	}

	parentFile := os.NewFile(uintptr(fds[0]), "cligram-call-helper-parent")
	childFile := os.NewFile(uintptr(fds[1]), "cligram-call-helper-child")

	cmd := exec.CommandContext(ctx, binaryPath, args...)
	cmd.ExtraFiles = []*os.File{childFile}

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

	_ = childFile.Close()

	bridge := NewJSONRPCBridge(parentFile)
	bridge.Start()

	s.cmd = cmd
	s.bridge = bridge

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
