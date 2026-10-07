package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func main() {
	loopbackFlag := flag.Bool("loopback", false, "Run audio capture and playback loopback diagnostic")
	flag.Parse()

	if *loopbackFlag {
		runLoopbackDiagnostic()
		return
	}

	// In child process spawned via exec.Cmd.ExtraFiles, ExtraFiles[0] is assigned FD 3
	ipcFile := os.NewFile(3, "cligram-ipc-socket")
	if ipcFile == nil {
		slog.Error("FD 3 is nil: cligram-voip must be launched by cligram core")
		os.Exit(1)
	}

	// Verify FD 3 is actually open and valid
	var stat syscall.Stat_t
	if err := syscall.Fstat(3, &stat); err != nil {
		fmt.Fprintf(os.Stderr, "Error: FD 3 is not a valid open socket: %v\n", err)
		fmt.Fprintf(os.Stderr, "cligram-voip must be launched as a child process of cligram core.\n")
		fmt.Fprintf(os.Stderr, "Run with --loopback to test local audio pipeline.\n")
		os.Exit(1)
	}

	slog.Info("cligram-voip started, listening on inherited FD 3")
	handler := NewSidecarHandler(ipcFile, ipcFile)

	if err := handler.Run(); err != nil {
		slog.Info("cligram-voip exiting", "reason", err)
	}
}

func runLoopbackDiagnostic() {
	fmt.Println("🎙️ Running Cligram VoIP Audio Loopback Diagnostic...")
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
