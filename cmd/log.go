package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func cligramLog() *cobra.Command {
	return &cobra.Command{
		Use:          "log",
		Short:        "cligram log",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		Run: func(cmd *cobra.Command, args []string) {
			logs, err := os.ReadFile("/tmp/cligram.log")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading log file: %v", err)
				os.Exit(1)
			}
			fmt.Print(string(logs))
		},
	}
}
