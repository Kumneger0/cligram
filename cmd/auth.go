package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/kumneger0/cligram/internal/runner"
	"github.com/spf13/cobra"
)

func login() *cobra.Command {
	return &cobra.Command{
		Use:          "login",
		Short:        "cligram login",
		Args:          cobra.NoArgs,
		SilenceUsage: true,
		Run: func(cmd *cobra.Command, args []string) {
			jsExcutable, err := runner.GetJSExcutable()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to get JS executable: %v\n", err)
				return
			}
			jsExcute := exec.Command(*jsExcutable, "login")
			jsExcute.Stdin = os.Stdin
			jsExcute.Stdout = os.Stdout
			jsExcute.Stderr = os.Stderr
			jsExcute.Run()
		},
	}
}

func logout() *cobra.Command {
	return &cobra.Command{
		Use:          "logout",
		Short:        "cligram logout",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		Run: func(cmd *cobra.Command, args []string) {
			jsExcutable, err := runner.GetJSExcutable()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to get JS executable: %v\n", err)
				return
			}
			jsExcute := exec.Command(*jsExcutable, "logout")
			jsExcute.Stdin = os.Stdin
			jsExcute.Stdout = os.Stdout
			jsExcute.Stderr = os.Stderr
			jsExcute.Run()
		},
	}
}
