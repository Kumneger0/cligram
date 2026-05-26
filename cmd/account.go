package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kumneger0/cligram/internal/telegram"
	"github.com/kumneger0/cligram/internal/telegram/types"
	"github.com/spf13/cobra"
)

type AccountsOnDeviceInfo struct {
	accountName string
	path        string
}

type accountModel struct {
	accounts []AccountsOnDeviceInfo
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
			m.selected = m.accounts[m.cursor].path
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
			nameStyle.Render(prefix+acc.accountName),
			pathLabelStyle.Render("    Path:"),
			pathValueStyle.Render(acc.path),
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
			onlyPaths := getAccountDirsOnThisDevice()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			updateChannel := make(chan types.Notification, 128)

			var accountsOnThisDevice []AccountsOnDeviceInfo
			var mu sync.Mutex
			var wg sync.WaitGroup
			for _, p := range onlyPaths {
				wg.Add(1)
				go func(path string) {
					defer wg.Done()

					clientCtx, clientCancel := context.WithCancel(ctx)
					defer clientCancel()

					cligram, err := telegram.NewClient(clientCtx, updateChannel, telegramAPIID, telegramAPIHash, path)
					if err != nil {
						slog.Error("failed to create client", "path", path, "error", err)
						return
					}

					err = cligram.Client.Run(clientCtx, func(ctx context.Context) error {
						self, err := cligram.Client.Self(ctx)
						if err != nil {
							slog.Error(err.Error())
							return err
						}

						mu.Lock()
						accountsOnThisDevice = append(accountsOnThisDevice, AccountsOnDeviceInfo{
							accountName: self.FirstName,
							path:        path,
						})
						mu.Unlock()

						clientCancel()
						return nil
					})

					if err != nil {
						slog.Debug("client run finished", "path", path, "error", err)
					}
				}(p)
			}
			wg.Wait()
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
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		slog.Error(err.Error())
		return "account1"
	}
	sessionDir := filepath.Join(userHomeDir, ".cligram")

	if _, err := os.Stat(sessionDir); os.IsNotExist(err) {
		return "account1"
	}

	dirEntry, err := os.ReadDir(sessionDir)
	if err != nil {
		slog.Error(err.Error())
		return "account1"
	}

	re := regexp.MustCompile(`^account(\d+)$`)
	maxID := 0

	for _, dir := range dirEntry {
		if dir.IsDir() {
			matches := re.FindStringSubmatch(dir.Name())
			if len(matches) == 2 {
				id, err := strconv.Atoi(matches[1])
				if err == nil && id > maxID {
					maxID = id
				}
			}
		}
	}
	return fmt.Sprintf("account%d", maxID+1)
}

func getAccountDirsOnThisDevice() []string {
	var onlyPaths []string

	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		slog.Error(err.Error())
		return onlyPaths
	}
	sessionDir := filepath.Join(userHomeDir, ".cligram")
	dirEntry, err := os.ReadDir(sessionDir)
	if err != nil {
		slog.Error(err.Error())
		return onlyPaths
	}

	type dirInfo struct {
		name    string
		modTime int64
	}
	var dirs []dirInfo

	for _, dir := range dirEntry {
		if dir.IsDir() {
			info, err := dir.Info()
			if err != nil {
				slog.Error(err.Error())
				continue
			}
			dirs = append(dirs, dirInfo{
				name:    dir.Name(),
				modTime: info.ModTime().Unix(),
			})
		}
	}

	slices.SortFunc(dirs, func(a, b dirInfo) int {
		if a.modTime < b.modTime {
			return -1
		}
		if a.modTime > b.modTime {
			return 1
		}
		return 0
	})

	for _, d := range dirs {
		onlyPaths = append(onlyPaths, d.name)
	}

	if len(onlyPaths) == 0 {
		dir := generateDir()
		return []string{dir}
	}
	return onlyPaths
}
