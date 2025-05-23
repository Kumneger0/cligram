package cmd

import (
	"fmt"

	list "github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	viewport "github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	ui "github.com/kumneger0/cligram/internal/ui"
	"github.com/spf13/cobra"
)

func newRootCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cligram",
		Short: "cligram a cli based telegram client",
		RunE: func(cmd *cobra.Command, args []string) error {
			users := ui.GetFakeData()

			userList := list.New(users, ui.CustomDelegate{}, 10, 20)
            channels := ui.GetFakeChannels()
            channelList := (list.New(channels, ui.CustomDelegate{}, 10, 20))
            groups := ui.GetFakeChannels()
            groupList := (list.New(groups, ui.CustomDelegate{}, 10, 20))



			userList.SetShowHelp(false)
            channelList.SetShowHelp(false)
            groupList.SetShowHelp(false)
			initiallySelectedUser := users[0].(ui.UserInfo)
			initiallySelectedChannel := channels[0].(ui.ChannelAndGroupInfo)

			input := textinput.New()
			input.Placeholder = "Type a message..."
			input.Prompt = "> "
			input.CharLimit = 256

			m := ui.Model{
				Input: input,
				Users:        userList,
				SelectedUser: initiallySelectedUser,
				SelectedChannel: initiallySelectedChannel,
				Groups: groupList,
				SelectedGroup: groups[0].(ui.ChannelAndGroupInfo),
				Channels: channelList,
			    Mode: "users",
				Conversations: ui.FakeConversations(),
				FocusedOn: "sideBar",
		        Vp: viewport.New(0, 0),
			}
			p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())

			_, err := p.Run()
			if err != nil {
				return fmt.Errorf("failed to start TUI: %w", err)
			}

			fmt.Println("Exited TUI normally.")
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
