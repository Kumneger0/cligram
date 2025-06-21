package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	list "github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kumneger0/cligram/internal/rpc"
	"github.com/kumneger0/cligram/internal/runner"
	ui "github.com/kumneger0/cligram/internal/ui"
	overlay "github.com/rmhubbert/bubbletea-overlay"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	Program *tea.Program
)

func getCligramLogFilePath() string {
	return filepath.Join(os.TempDir(), "cligram.log")
}

func startSeparateJsProces(wg *sync.WaitGroup) {
	jsExcutable, err := runner.GetJSExcutable()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get JS executable: %v\n", err)
		wg.Done()
		return
	}

	jsExcute := exec.Command(*jsExcutable)

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

	logFile, err := os.Create(getCligramLogFilePath())
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
			wg.Wait()

			notificationChannel := make(chan rpc.Notification)

			go rpc.ProcessIncomingNotifications(notificationChannel)

			msg := rpc.RpcClient.GetUserChats()

			modalContent := ""
			isModalVisible := false

			var result []rpc.UserInfo = []rpc.UserInfo{}

			if msg.Err != nil {
				modalContent = msg.Err.Error()
				isModalVisible = true
			} else {
				if msg.Response.Result != nil {
					result = msg.Response.Result
				}
			}

			var users []list.Item = []list.Item{}
			for _, du := range result {
				users = append(users, rpc.UserInfo{
					UnreadCount: du.UnreadCount,
					FirstName:   du.FirstName,
					IsBot:       du.IsBot,
					PeerID:      du.PeerID,
					AccessHash:  du.PeerID,
					LastSeen:    du.LastSeen,
					IsOnline:    du.IsOnline,
				})
			}

			userList := list.New(users, ui.CustomDelegate{}, 10, 20)
			userList.SetShowPagination(false)
			channels := []list.Item{}
			channelList := (list.New(channels, ui.CustomDelegate{}, 10, 20))
			channelList.SetShowPagination(false)
			groups := []list.Item{}
			groupList := (list.New(groups, ui.CustomDelegate{}, 10, 20))
			groupList.SetShowPagination(false)

			userList.SetShowHelp(false)
			channelList.SetShowHelp(false)
			groupList.SetShowHelp(false)

			input := textinput.New()
			input.Placeholder = "Type a message..."
			input.Prompt = "> "
			input.CharLimit = 256

			fd := int(os.Stdout.Fd())
			width, height, _ := term.GetSize(fd)

			chatList := list.New([]list.Item{}, ui.MessagesDelegate{}, 10, 20)
			chatList.SetShowPagination(false)
			chatList.SetShowHelp(false)
			chatList.SetShowFilter(false)
			chatList.SetShowTitle(false)
			chatList.SetShowStatusBar(false)

			fp := filepicker.New()
			fp.AllowedTypes = []string{}
			fp.DirAllowed = false
			fp.CurrentDirectory, _ = os.UserHomeDir()

			m := ui.Model{
				Filepicker:     fp,
				Input:          input,
				Users:          userList,
				Groups:         groupList,
				ModalContent:   modalContent,
				Height:         height - 4,
				Width:          width - 4,
				Channels:       channelList,
				IsModalVisible: isModalVisible,
				Mode:           "users",
				FocusedOn:      "sideBar",
				ChatUI:         chatList,
				SelectedFile:   "",
			}

			backgorund := m
			forground := &ui.Foreground{}

			manager := ui.Manager{
				Foreground: forground,
				Background: backgorund,
				State:      ui.MainView,
				Overlay: overlay.New(
					forground,
					backgorund,
					overlay.Center,
					overlay.Top,
					0,
					0,
				),
			}

			Program = tea.NewProgram(manager, tea.WithAltScreen(), tea.WithMouseCellMotion())

			go func() {
				for {
					time.Sleep(1 * time.Second)
					select {
					case msg := <-notificationChannel:
						if msg.NewMessageMsg != (rpc.NewMessageMsg{}) {
							Program.Send(msg.NewMessageMsg)
						}
						if msg.UserOnlineOfflineMsg != (rpc.UserOnlineOffline{}) {
							Program.Send(msg.UserOnlineOfflineMsg)
						}

					case <-time.After(1 * time.Second):
						// fmt.Println("sent tick")
					}
				}
			}()

			_, err := Program.Run()
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
