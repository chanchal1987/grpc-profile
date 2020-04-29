package cmd

import (
	"os"
	"time"

	profile "github.com/chanchal1987/grpc-profile"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(profileCmd)
}

var (
	profileCmd = &cobra.Command{
		Use:     "profile <profile-type> [duration] <file-name>",
		Short:   "Run profile on remote server",
		Long:    `Run profile on remote server where the agent is running`,
		PreRunE: connect,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return []string{
					"heap",
					"mutex",
					"block",
					"threadcreate",
					"goroutine",
					"cpu",
					"trace",
				}, cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 2 {
				file, err := os.Create(args[1])
				if err != nil {
					return err
				}
				defer file.Close()
				var prof profile.LookupType
				switch args[0] {
				case "heap", "memory":
					prof = profile.HeapType
				case "mutex":
					prof = profile.MutexType
				case "block":
					prof = profile.BlockType
				case "threadcreate", "thread-create":
					prof = profile.ThreadCreateType
				case "goroutine", "go-routine":
					prof = profile.GoRoutineType
				default:
					return invalidArgumentsError
				}
				return client.LookupProfile(cmd.Context(), prof, file)
			} else if len(args) == 3 {
				dur, err := time.ParseDuration(args[1])
				if err != nil {
					return err
				}
				file, err := os.Create(args[2])
				if err != nil {
					return err
				}
				defer file.Close()
				var prof profile.NonLookupType
				switch args[0] {
				case "cpu":
					prof = profile.CPUType
				case "trace":
					prof = profile.TraceType
				default:
					return invalidArgumentsError
				}
				return client.NonLookupProfile(cmd.Context(), prof, dur, file)
			}
			return invalidArgumentsError
		},
	}
)
