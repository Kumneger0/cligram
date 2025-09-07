package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/term"
	"github.com/kumneger0/cligram/internal/telegram"
	"github.com/kumneger0/cligram/internal/telegram/types"
	"github.com/kumneger0/cligram/internal/ui"
	overlay "github.com/rmhubbert/bubbletea-overlay"
	"github.com/spf13/cobra"
)

var (
	Program *tea.Program
)

func newRootCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cligram",
		Short: "cligram a cli based telegram client",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithCancel(context.Background())
			updateChannel := make(chan types.Notification, 128)

			cligram, err := telegram.NewClient(ctx, updateChannel)
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

				userChatsResult, err := telegram.Cligram.GetChatManager().GetUserChats(ctx, false, 0, 0)
				modalContent := ""
				isModalVisible := false
				userChats := userChatsResult.Data

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
				for _, du := range result {
					users = append(users, types.UserInfo{
						UnreadCount: du.UnreadCount,
						FirstName:   du.FirstName,
						IsBot:       du.IsBot,
						PeerID:      du.PeerID,
						AccessHash:  du.AccessHash,
						LastSeen:    du.LastSeen,
						IsOnline:    du.IsOnline,
					})
				}
				model := &ui.Model{}
				userList := list.New(users, ui.CustomDelegate{Model: model}, 10, 20)
				userList.SetShowPagination(false)
				channels := []list.Item{}
				channelList := (list.New(channels, ui.CustomDelegate{Model: model}, 10, 20))
				channelList.SetShowPagination(false)
				groups := []list.Item{}
				groupList := (list.New(groups, ui.CustomDelegate{Model: model}, 10, 20))
				groupList.SetShowPagination(false)

				userList.SetShowHelp(false)
				channelList.SetShowHelp(false)
				groupList.SetShowHelp(false)

				input := textinput.New()
				input.Placeholder = "Type a message..."
				input.Prompt = "> "
				input.CharLimit = 256

				width, height, _ := term.GetSize(os.Stdout.Fd())

				chatList := list.New([]list.Item{}, ui.MessagesDelegate{Model: model}, 10, 20)
				chatList.SetShowPagination(false)
				chatList.SetShowHelp(false)
				chatList.SetShowFilter(false)
				chatList.SetShowTitle(false)
				chatList.SetShowStatusBar(false)

				fp := filepicker.New()
				fp.AllowedTypes = []string{}
				fp.DirAllowed = false
				fp.CurrentDirectory, _ = os.UserHomeDir()

				model.AreWeSwitchingModes = false
				model.Filepicker = fp
				model.Input = input
				model.Users = userList
				model.Groups = groupList
				model.ModalContent = modalContent
				model.Height = height - 4
				model.Width = width - 4
				model.Channels = channelList
				model.IsModalVisible = isModalVisible
				model.Mode = ui.ModeUsers
				model.FocusedOn = ui.SideBar
				model.ChatUI = chatList
				model.SelectedFile = ""
				model.OffsetDate = userChatsResult.OffsetDate
				model.OffsetID = userChatsResult.OffsetID

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
	}

	cmd.AddCommand(newVersionCmd(version))
	cmd.AddCommand(upgradeCligram(version))
	cmd.AddCommand(cligramLog())
	cmd.AddCommand(ManCmd(cmd))
	return cmd
}

func Execute(version string) error {
	if err := newRootCmd(version).Execute(); err != nil {
		return fmt.Errorf("error executing root command: %w", err)
	}
	return nil
}
