package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(gcCmd)
}

var (
	gcCmd = &cobra.Command{
		Use:     "gc",
		Short:   "Run forced GC on remote server",
		Long:    `Run forced GC on remote server where the agent is running`,
		PreRunE: connect,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return errInvalidArguments
			}
			return client.GC(cmd.Context())
		},
	}
)
