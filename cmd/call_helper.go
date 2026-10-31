package cmd

import (
	"github.com/kumneger0/cligram/cmd/callhelper"
	"github.com/spf13/cobra"
)

func newCallHelperCmd() *cobra.Command {
	var loopback bool
	cmd := &cobra.Command{
		Use:          "call-helper",
		Short:        "Start voice call media and signaling worker directly",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return callhelper.Run(loopback)
		},
	}
	cmd.Flags().BoolVar(&loopback, "loopback", false, "Run audio capture and playback loopback diagnostic")
	return cmd
}
