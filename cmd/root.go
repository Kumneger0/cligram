package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime/pprof"
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/list" // Legacy groups have no access hash; supergroups (migrated) do.
	"github.com/gotd/td/tg"
	"go.dalton.dog/bubbleup"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/term"
	"github.com/kumneger0/cligram/internal/telegram"
	"github.com/kumneger0/cligram/internal/telegram/client"
	"github.com/kumneger0/cligram/internal/telegram/types"
	"github.com/kumneger0/cligram/internal/ui"
	overlay "github.com/rmhubbert/bubbletea-overlay"
	"github.com/spf13/cobra"
)

var (
	Program        *tea.Program
	memFile        string
	cpuFile        string
	cpuProfileFile *os.File
)

func newRootCmd(version string, telegramAPIID, telegramAPIHash string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cligram",
		Short: "cligram a cli based telegram client",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithCancel(context.Background())
			updateChannel := make(chan types.Notification, 128)

			cligram, err := telegram.NewClient(ctx, updateChannel, telegramAPIID, telegramAPIHash)
			if err != nil {
				slog.Error(err.Error())
				cancel()
				return fmt.Errorf("failed to initialize telegram client: %w", err)
			}
			telegram.Cligram = cligram
			err = telegram.Cligram.Run(ctx, func(ctx context.Context) error {
				if err := cligram.Auth(ctx); err != nil {
					cancel()
					slog.Error(err.Error())
					return fmt.Errorf("authentication failed: %w", err)
				}

				userChatsResult, err := telegram.Cligram.GetAllChats(ctx, 0, 0)
				modalContent := ""
				isModalVisible := false
				userChats := userChatsResult.PrivateChats
				groupsChat := userChatsResult.Groups
				channelsChat := userChatsResult.Channels

				var result []types.UserInfo = []types.UserInfo{}

				if err != nil {
					modalContent = err.Error()
					isModalVisible = true
				} else {
					if userChats != nil {
						result = userChats
					}
				}

				var users []list.Item = []list.Item{}
				var bots []list.Item = []list.Item{}
				for _, du := range result {
					if du.IsBot {
						bots = append(bots, du)
						continue
					}
					users = append(users, du)
				}

				channelsList := []list.Item{}
				for _, channel := range channelsChat {
					channelsList = append(channelsList, channel)
				}

				groupsList := []list.Item{}
				for _, group := range groupsChat {
					groupsList = append(groupsList, group)
				}

				model := &ui.Model{}
				model.Alert = *bubbleup.NewAlertModel(80, true, 10*time.Second)
				model.CustomEmojis = make(map[int64]*tg.Document)
				userList := list.New(users, ui.CustomDelegate{Model: model}, 10, 20)
				userList.SetShowPagination(false)
				channels := list.New(channelsList, ui.CustomDelegate{Model: model}, 10, 20)
				groups := (list.New(groupsList, ui.CustomDelegate{Model: model}, 10, 20))
				botsList := list.New(bots, ui.CustomDelegate{Model: model}, 10, 20)
				channels.SetShowPagination(false)
				groups.SetShowPagination(false)

				userList.SetShowHelp(false)
				channels.SetShowHelp(false)
				groups.SetShowHelp(false)
				botsList.SetShowHelp(false)

				userList.SetShowTitle(false)
				channels.SetShowTitle(false)
				groups.SetShowTitle(false)
				botsList.SetShowTitle(false)
				input := textinput.New()
				input.Placeholder = "Type a message..."
				input.Prompt = "> "
				input.CharLimit = 256

				width, height, _ := term.GetSize(os.Stdout.Fd())

				chatList := list.New([]list.Item{}, ui.MessagesDelegate{Model: model}, 5, 2)
				chatList.SetShowPagination(false)
				chatList.SetShowHelp(false)
				chatList.SetShowFilter(false)
				chatList.SetShowTitle(false)
				chatList.SetShowStatusBar(false)

				fp := filepicker.New()
				fp.AllowedTypes = []string{}
				fp.DirAllowed = false
				fp.CurrentDirectory, _ = os.UserHomeDir()

				model.Filepicker = fp
				model.Input = input
				model.Users = userList
				model.Groups = groups
				model.ModalContent = modalContent
				model.Height = height - 4
				model.Width = width - 4
				model.Channels = channels
				model.IsModalVisible = isModalVisible
				model.Mode = ui.ModeUsers
				model.FocusedOn = ui.SideBar
				model.ChatUI = chatList
				model.SelectedFile = ""
				model.OffsetDate = userChatsResult.OffsetDate
				model.OffsetID = userChatsResult.OffsetID
				model.OnPagination = false
				model.Bots = botsList

				model.Stories = []types.Stories{}

				background := model
				foreground := &ui.Foreground{}

				manager := ui.Manager{
					Foreground: foreground,
					Background: background,
					State:      ui.MainView,
					Overlay: overlay.New(
						foreground,
						background,
						overlay.Center,
						overlay.Top,
						0,
						0,
					),
				}

				Program = tea.NewProgram(manager, tea.WithAltScreen())
				go func() {
					for {
						select {
						case <-ctx.Done():
							return
						case msg, ok := <-updateChannel:
							if !ok {
								return
							}
							if msg.NewMessage != nil {
								Program.Send(*msg.NewMessage)
							}
							if msg.UserStatus != nil {
								Program.Send(*msg.UserStatus)
							}
							if msg.Error != nil {
								Program.Send(*msg.Error)
							}
							if msg.UserTyping != nil {
								Program.Send(*msg.UserTyping)
							}
							if msg.SearchResult != nil {
								Program.Send(*msg.SearchResult)
							}
						}
					}
				}()
				_, err = Program.Run()

				cancel()
				if err != nil {
					cancel()
					return fmt.Errorf("failed to start TUI: %w", err)
				}

				return nil
			})
			if err != nil {
				fmt.Println(err)
				slog.Error(err.Error())
			}
			return nil
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			if cpuFile != "" {
				pprof.StopCPUProfile()
			}
			if cpuProfileFile != nil {
				cpuProfileFile.Close()
			}
			if memFile != "" {
				f, err := os.Create(memFile)
				if err != nil {
					slog.Error(err.Error())
				}
				defer f.Close()
				if err := pprof.WriteHeapProfile(f); err != nil {
					slog.Error(err.Error())
				}
			}
		},
	}

	cmd.AddCommand(newVersionCmd(version))
	cmd.AddCommand(upgradeCligram(version))
	cmd.AddCommand(cligramLog())
	cmd.AddCommand(ManCmd(cmd))
	cmd.AddCommand(client.Logout())
	return cmd
}

func Execute(version string, telegramAPIID, telegramAPIHash string) error {
	cmd := newRootCmd(version, telegramAPIID, telegramAPIHash)

	cmd.PersistentFlags().StringVar(&cpuFile, "cpuprofile", "", "write cpu profile to `file`")
	cmd.PersistentFlags().StringVar(&memFile, "memprofile", "", "write memory profile to `file`")

	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if cpuFile != "" {
			f, err := os.Create(cpuFile)
			if err != nil {
				return fmt.Errorf("could not create CPU profile: %w", err)
			}
			if err := pprof.StartCPUProfile(f); err != nil {
				f.Close()
				return fmt.Errorf("could not start CPU profile: %w", err)
			}
			cpuProfileFile = f
		}
		return nil
	}

	if err := cmd.Execute(); err != nil {
		return fmt.Errorf("error executing root command: %w", err)
	}
	return nil
}
