package cmd

import (
	profile "github.com/chanchal1987/grpc-profile"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(stopCmd)
}

var (
	stopCmd = &cobra.Command{
		Use:       "stop <cpu|trace>",
		Short:     "Stop running profile on remote server",
		Long:      `Stop running profile on remote server where the agent is running`,
		PreRunE:   connect,
		ValidArgs: []string{"cpu", "trace"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return invalidArgumentsError
			}
			var prof profile.NonLookupType
			switch args[0] {
			case "cpu":
				prof = profile.CPUType
			case "trace":
				prof = profile.TraceType
			default:
				return invalidArgumentsError
			}
			return client.StopNonLookupProfile(cmd.Context(), prof)
		},
	}
)
