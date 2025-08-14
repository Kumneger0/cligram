package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"syscall"

	"github.com/gofrs/flock"
	"github.com/kumneger0/cligram/cmd"
	"github.com/kumneger0/cligram/internal/logger"
)

var version = ""

func main() {

	lockFilePath := filepath.Join(os.TempDir(), "cligram.lock")
	fileLock := flock.New(lockFilePath)

	locked, err := fileLock.TryLock()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error trying to acquire lock: %v\n", err)
		os.Exit(1)
	}

	if !locked {
		showAnotherProcessIsRunning(lockFilePath)
		os.Exit(1)
	}

	defer func() {
		if err := fileLock.Unlock(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not unlock file: %v\n", err)
		}
		_ = os.Remove(lockFilePath)
	}()

	pid := os.Getpid()
	if err := os.WriteFile(lockFilePath, []byte(strconv.Itoa(pid)), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not write PID to lock file: %v\n", err)
	}

	logger := logger.Init()
	defer logger.Close()

	if err := cmd.Execute(version); err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}

	if !slices.Contains(os.Args, "upgrade") && version != "" {
		IsUpdateAvailable := cmd.GetNewVersionInfo(version)
		if IsUpdateAvailable.IsUpdateAvailable {
			fmt.Println("An update is available use cligram upgrade to update to latest version")
		}
	}
}

func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	err = process.Signal(syscall.Signal(0))
	return err == nil
}

func showAnotherProcessIsRunning(lockFilePath string) {
	pidBytes, readErr := os.ReadFile(lockFilePath)
	if readErr == nil {
		pid, parseErr := strconv.Atoi(string(pidBytes))
		if parseErr == nil {
			if !isProcessRunning(pid) {
				fmt.Fprintf(os.Stderr, "Another instance of cligram is not running (stale lock file for PID %d).\n", pid)
				fmt.Fprintf(os.Stderr, "Please try removing %s and running again if this persists.\n", lockFilePath)
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "Another instance of cligram is already running (PID: %d).\n", pid)
		} else {
			fmt.Fprintf(os.Stderr, "Another instance of cligram is already running (lock file content unreadable).\n")
		}
	} else {
		fmt.Fprintf(os.Stderr, "Another instance of cligram is already running.\n")
	}
}
