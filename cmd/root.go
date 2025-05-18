package cmd

import (
	"fmt"

	list "github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	ui "github.com/kumneger0/cligram/internal/ui"
	"github.com/spf13/cobra"
)

func newRootCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cligram",
		Short: "cligram a cli based telegram client",
		RunE: func(cmd *cobra.Command, args []string) error {

			userList := list.New(ui.GetFakeData(), ui.CustomDelegate{}, 10, 30)
			m := ui.Model{
				Users: userList,
			}
			p := tea.NewProgram(m, tea.WithAltScreen())

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
