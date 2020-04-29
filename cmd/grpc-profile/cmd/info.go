package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(infoCmd)
}

var (
	infoCmd = &cobra.Command{
		Use:     "info",
		Short:   "Get information about the server",
		Long:    `Get information about the server where the agent is running`,
		PreRunE: connect,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return errInvalidArguments
			}
			info, err := client.GetInfo(cmd.Context())
			if err != nil {
				return err
			}

			out, err := json.MarshalIndent(info, "", "  ")
			if err != nil {
				return err
			}

			fmt.Println("Information:")
			fmt.Println(string(out))
			return nil
		},
	}
)
