package cmd

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

func login() *cobra.Command {
	cwd, _ := os.Getwd()
	jsFilePath := filepath.Join(cwd, "js", "src", "index.ts")

	return &cobra.Command{
		Use:          "login",
		Short:        "cligram login",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		Run: func(cmd *cobra.Command, args []string) {
			jsExcute := exec.Command("bun", jsFilePath, "login")
			jsExcute.Stdin = os.Stdin
			jsExcute.Stdout = os.Stdout
			jsExcute.Stderr = os.Stderr
			jsExcute.Run()
		},
	}	
}


func logout() *cobra.Command {
	cwd, _ := os.Getwd()
	jsFilePath := filepath.Join(cwd, "js", "src", "index.ts")

	return &cobra.Command{
		Use:          "logout",
		Short:        "cligram logout",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		Run: func(cmd *cobra.Command, args []string) {
			jsExcute := exec.Command("bun", jsFilePath, "logout")
			jsExcute.Stdin = os.Stdin
			jsExcute.Stdout = os.Stdout
			jsExcute.Stderr = os.Stderr
			jsExcute.Run()
		},
	}
}