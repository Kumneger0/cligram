package cmd

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kumneger0/cligram/internal/telegram"
	"github.com/kumneger0/cligram/internal/telegram/types"
	"github.com/spf13/cobra"
)

type accountModel struct {
	accounts []types.AccountsOnDeviceInfo
	cursor   int
	selected string
}

func (m accountModel) Init() tea.Cmd {
	return nil
}

func (m *accountModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.accounts)-1 {
				m.cursor++
			}
		case "enter":
			m.selected = m.accounts[m.cursor].Path
			return m, tea.Quit
		}
	}
	return m, nil
}

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#818CF8")).
			Padding(0, 1).
			MarginBottom(1)

	accountCardStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#1E293B")).
				Padding(0, 1).
				MarginBottom(1).
				Width(50)

	selectedCardStyle = accountCardStyle.
				BorderForeground(lipgloss.Color("#818CF8")).
				Background(lipgloss.Color("#1E293B"))

	nameStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#E2E8F0"))

	pathLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#94A3B8")).
			Italic(true)

	pathValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#818CF8"))
)

func (m accountModel) View() string {
	var s strings.Builder

	s.WriteString(headerStyle.Render("👤 Select a Telegram Account"))
	s.WriteString("\n\n")

	for i, acc := range m.accounts {
		style := accountCardStyle
		prefix := "  "
		if i == m.cursor {
			style = selectedCardStyle
			prefix = "> "
		}

		content := fmt.Sprintf("%s\n%s %s",
			nameStyle.Render(prefix+acc.AccountName),
			pathLabelStyle.Render("    Path:"),
			pathValueStyle.Render(acc.Path),
		)
		s.WriteString(style.Render(content))
		s.WriteString("\n")
	}

	s.WriteString("\n" + pathLabelStyle.Render("↑/↓: navigate • enter: select • q: cancel"))

	return s.String()
}

func Account(telegramAPIID, telegramAPIHash string) *cobra.Command {
	var add bool
	c := &cobra.Command{
		Use:                   "account",
		Aliases:               []string{"accounts"},
		Short:                 "Manage Telegram accounts",
		SilenceUsage:          true,
		DisableFlagsInUseLine: true,
		Hidden:                false,
		Args:                  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if add {
				newAcc := generateDir()
				root := cmd.Root()
				if err := root.Flags().Set("account", newAcc); err != nil {
					return err
				}
				if root.RunE != nil {
					return root.RunE(root, args)
				}
				return nil
			}
			accountsOnThisDevice := getAccountDirsOnThisDevice(telegramAPIID, telegramAPIHash)

			if len(accountsOnThisDevice) == 0 {
				fmt.Println("No active accounts found.")
				return nil
			}

			m := &accountModel{accounts: accountsOnThisDevice}
			p := tea.NewProgram(m)
			if _, err := p.Run(); err != nil {
				return err
			}

			if m.selected != "" {
				fmt.Println(headerStyle.Render("\n🔄 Switching account..."))
				root := cmd.Root()
				if err := root.Flags().Set("account", m.selected); err != nil {
					return err
				}
				if root.RunE != nil {
					return root.RunE(root, args)
				}
			}
			return nil
		},
	}

	c.Flags().BoolVar(&add, "add", false, "Add a new account")
	return c
}

func generateDir() string {
	return telegram.GenerateDir()
}

type DirInfo = telegram.DirInfo

func getCligramDir() *string {
	return telegram.GetCligramDir()
}

func getAccountPaths() []DirInfo {
	return telegram.GetAccountPaths()
}

func getAccountDirsOnThisDevice(telegramAPIID, telegramAPIHash string) []types.AccountsOnDeviceInfo {
	return telegram.GetAccountDirsOnThisDevice(telegramAPIID, telegramAPIHash)
}
