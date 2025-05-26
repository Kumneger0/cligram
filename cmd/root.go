package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	list "github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	viewport "github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kumneger0/cligram/internal/rpc"
	ui "github.com/kumneger0/cligram/internal/ui"
	"github.com/spf13/cobra"
)

func getJsFilePath() string {
	cwd, _ := os.Getwd()
	//TODO: don't forget update this 
	// the file may be diffrent after build
	jsFilePath := filepath.Join(cwd, "js", "src", "index.ts")

	return jsFilePath
}

func startSeparateJsProces(wg *sync.WaitGroup) {
	jsFilePath := getJsFilePath()

	jsExcute := exec.Command("bun", jsFilePath)

	stdin, err := jsExcute.StdinPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create stdin pipe: %v\n", err)
		wg.Done()
		return
	}

	stdout, err := jsExcute.StdoutPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create stdout pipe: %v\n", err)
		stdin.Close()
		wg.Done()
		return
	}

	logDir := filepath.Join(os.TempDir(), "tg-cli-logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create log directory: %v\n", err)
	}

	logFile, err := os.Create(filepath.Join(logDir, "js-process.log"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create log file, stderr will be discarded: %v\n", err)
		jsExcute.Stderr = nil
	} else {
		jsExcute.Stderr = logFile
		defer func() {
			logFile.Close()
		}()
	}

	if err := jsExcute.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start JavaScript process: %v\n", err)
		stdin.Close()
		if logFile != nil {
			logFile.Close()
		}
		wg.Done()
		return
	}

	rpc.JsProcess = jsExcute.Process

	rpc.RpcClient = &rpc.JsonRpcClient{
		Stdin:  stdin,
		Stdout: stdout,
		Cmd:    jsExcute,
		NextID: 1,
	}

	go func() {
		if err := jsExcute.Wait(); err != nil {
			if logFile != nil {
				fmt.Fprintf(logFile, "JavaScript process exited with error: %v\n", err)
			}
		}
	}()

	wg.Done()
}

func newRootCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cligram",
		Short: "cligram a cli based telegram client",
		RunE: func(cmd *cobra.Command, args []string) error {

			var wg sync.WaitGroup
			wg.Add(1)
			go startSeparateJsProces(&wg)

			users := []list.Item{}

			userList := list.New(users, ui.CustomDelegate{}, 10, 20)
			channels := []list.Item{}
			channelList := (list.New(channels, ui.CustomDelegate{}, 10, 20))
			groups := []list.Item{}
			groupList := (list.New(groups, ui.CustomDelegate{}, 10, 20))

			userList.SetShowHelp(false)
			channelList.SetShowHelp(false)
			groupList.SetShowHelp(false)

			input := textinput.New()
			input.Placeholder = "Type a message..."
			input.Prompt = "> "
			input.CharLimit = 256

			m := ui.Model{
				Input:           input,
				Users:           userList,
				Groups:          groupList,
				Channels:        channelList,
				Mode:            "users",
				FocusedOn:       "sideBar",
				Vp:              viewport.New(0, 0),
			}
			p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())

			_, err := p.Run()
			if err != nil {
				return fmt.Errorf("failed to start TUI: %w", err)
			}

			return nil
		},
	}

	cmd.AddCommand(newVersionCmd(version))
	cmd.AddCommand(login())
	cmd.AddCommand(logout())

	return cmd
}

func Execute(version string) error {
	if err := newRootCmd(version).Execute(); err != nil {
		return fmt.Errorf("error executing root command: %w", err)
	}

	return nil
}
