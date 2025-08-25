package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/charmbracelet/bubbles/filepicker"
	list "github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kumneger0/cligram/internal/telegram"
	ui "github.com/kumneger0/cligram/internal/ui"
	overlay "github.com/rmhubbert/bubbletea-overlay"
	"github.com/spf13/cobra"
	"golang.org/x/term"
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
			updateChannel := make(chan telegram.Notification)
			telegram.Cligram = telegram.GetTelegramClient(ctx, updateChannel)
			err := telegram.Cligram.Run(ctx, func(ctx context.Context) error {
				err := telegram.Cligram.Auth(ctx)
				if err != nil {
					slog.Error(err.Error())
					return nil
				}
				msg, err := telegram.Cligram.GetUserChats(false)
				if err != nil {
					slog.Error(err.Error())
					return nil
				}

				modalContent := ""
				isModalVisible := false

				var result []telegram.UserInfo = []telegram.UserInfo{}

				if msg.Err != nil {
					modalContent = msg.Err.Error()
					isModalVisible = true
				} else {
					if msg.Response != nil {
						result = *msg.Response
					}
				}

				var users []list.Item = []list.Item{}
				for _, du := range result {
					users = append(users, telegram.UserInfo{
						UnreadCount: du.UnreadCount,
						FirstName:   du.FirstName,
						IsBot:       du.IsBot,
						PeerID:      du.PeerID,
						AccessHash:  du.PeerID,
						LastSeen:    du.LastSeen,
						IsOnline:    du.IsOnline,
					})
				}
				model := ui.Model{}
				userList := list.New(users, ui.CustomDelegate{Model: &model}, 10, 20)
				userList.SetShowPagination(false)
				channels := []list.Item{}
				channelList := (list.New(channels, ui.CustomDelegate{Model: &model}, 10, 20))
				channelList.SetShowPagination(false)
				groups := []list.Item{}
				groupList := (list.New(groups, ui.CustomDelegate{Model: &model}, 10, 20))
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

				chatList := list.New([]list.Item{}, ui.MessagesDelegate{Model: &model}, 10, 20)
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

				background := model
				forground := &ui.Foreground{}

				ui.TUIManager = ui.Manager{
					Foreground: forground,
					Background: background,
					State:      ui.MainView,
					Overlay: overlay.New(
						forground,
						background,
						overlay.Center,
						overlay.Top,
						0,
						0,
					),
				}

				Program = tea.NewProgram(ui.TUIManager, tea.WithAltScreen())

				go func() {
					for msg := range updateChannel {
						if msg.NewMessageMsg != (telegram.NewMessageMsg{}) {
							Program.Send(msg.NewMessageMsg)
						}
					}
				}()
				_, err = Program.Run()

				cancel()
				if err != nil {
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
