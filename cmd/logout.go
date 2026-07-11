package cmd

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

func Logout(telegramAPIID, telegramAPIHash string) *cobra.Command {
	var account string
	c := &cobra.Command{
		Use:          "logout",
		Short:        "Log out from cligram",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			accountsOnThisDevice := getAccountDirsOnThisDevice(telegramAPIID, telegramAPIHash)
			if len(accountsOnThisDevice) == 0 {
				fmt.Println("No active accounts found.")
				return nil
			}

			var accountToLogout string
			if account != "" || len(accountsOnThisDevice) == 1 {
				if account != "" {
					accountToLogout = account
				} else {
					accountToLogout = accountsOnThisDevice[0].Path
				}
			} else {
				m := &accountModel{accounts: accountsOnThisDevice}
				p := tea.NewProgram(m)
				if _, err := p.Run(); err != nil {
					return err
				}
				if m.selected == "" {
					return nil
				}
				accountToLogout = m.selected
			}

			userHomeDir, err := os.UserHomeDir()
			if err != nil {
				slog.Error(err.Error())
				errorLink := fmt.Sprintf("https://github.com/kumneger0/cligram/issues/new?title=%s&body=%s",
					url.QueryEscape("Logout Error"),
					url.QueryEscape("Error detail:\n\n"+err.Error()))
				fmt.Fprintf(cmd.OutOrStderr(), "We failed to log you out. Please report this error by clicking the following link:\n%s\n", errorLink)
				return err
			}
			sessionDir, err := filepath.Abs(filepath.Join(userHomeDir, ".cligram"))
			if err != nil {
				return err
			}
			cligramWorkingDIR, err := filepath.Abs(filepath.Join(sessionDir, accountToLogout))
			if err != nil {
				return err
			}

			if !strings.HasPrefix(cligramWorkingDIR, sessionDir+string(os.PathSeparator)) {
				return fmt.Errorf("invalid account name: %s", accountToLogout)
			}

			err = os.RemoveAll(cligramWorkingDIR)
			if err != nil {
				slog.Error(err.Error())
				errorLink := fmt.Sprintf("https://github.com/kumneger0/cligram/issues/new?title=%s&body=%s",
					url.QueryEscape("Logout Error "),
					url.QueryEscape("Error detail:\n\n"+err.Error()))
				fmt.Fprintf(cmd.OutOrStderr(), "We failed to log you out. Please report this error by clicking the following link:\n%s\n", errorLink)
				return err
			}
			fmt.Println(headerStyle.Render(fmt.Sprintf("\n✅ Successfully logged out from account: %s", accountToLogout)))
			return nil
		},
	}
	c.Flags().StringVarP(&account, "account", "a", "", "account to log out from")
	return c
}
