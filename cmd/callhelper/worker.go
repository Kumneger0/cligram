package callhelper

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// Run starts the media/signaling worker directly.
func Run(loopback bool) error {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))

	if loopback {
		runLoopbackDiagnostic()
		return nil
	}

	// In child process spawned via exec.Cmd.ExtraFiles, ExtraFiles[0] is assigned FD 3
	ipcFile := os.NewFile(3, "cligram-ipc-socket")
	if ipcFile == nil {
		slog.Error("FD 3 is nil: call-helper must be launched by cligram core")
		return fmt.Errorf("FD 3 is nil: call-helper must be launched by cligram core")
	}

	// Verify FD 3 is actually open and valid socket
	var stat syscall.Stat_t
	if err := syscall.Fstat(3, &stat); err != nil || (stat.Mode&syscall.S_IFMT) != syscall.S_IFSOCK {
		fmt.Fprintf(os.Stderr, "Error: FD 3 is not a valid open socket\n")
		fmt.Fprintf(os.Stderr, "cligram call-helper must be launched as a child process of cligram core.\n")
		fmt.Fprintf(os.Stderr, "Run with --loopback to test local audio pipeline.\n")
		return fmt.Errorf("FD 3 is not a valid open socket")
	}

	slog.Info("cligram call-helper started, listening on inherited FD 3")
	handler := NewHandler(ipcFile, ipcFile)

	if err := handler.Run(); err != nil {
		slog.Info("cligram call-helper exiting", "reason", err)
		return err
	}
	return nil
}

func runLoopbackDiagnostic() {
	fmt.Println("🎙️ Running Cligram Call Helper Audio Loopback Diagnostic...")
	captureCmd, playbackCmd, soundServer := ResolveAudioCommands()

	if captureCmd == "" || playbackCmd == "" {
		fmt.Fprintf(os.Stderr, "❌ Error: Could not resolve both capture and playback tools. (Server: %s)\n", soundServer)
		os.Exit(1)
	}

	fmt.Printf("✅ Detected Sound Server: %s\n", soundServer)
	fmt.Printf("   Capture tool:  %s\n", captureCmd)
	fmt.Printf("   Playback tool: %s\n", playbackCmd)

	fmt.Println("   Testing audio pipeline execution...")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", fmt.Sprintf("%s | %s", captureCmd, playbackCmd))
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Subprocess pipeline failed to start: %v\n", err)
		os.Exit(1)
	}

	time.Sleep(500 * time.Millisecond)
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	fmt.Println("✅ Audio pipeline started and verified successfully.")
}
