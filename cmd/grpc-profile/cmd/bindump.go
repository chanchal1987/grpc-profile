package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(binDumpCmd)
}

var (
	binDumpCmd = &cobra.Command{
		Use:     "bin-dump <file-name>",
		Short:   "Get a dumo of the binary file where the agent is running",
		Long:    `Get a dumo of the binary file where the agent is running`,
		PreRunE: connect,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return invalidArgumentsError
			}
			file, err := os.Create(args[0])
			if err != nil {
				return err
			}
			defer file.Close()
			return client.BinaryDump(cmd.Context(), file)
		},
	}
)
