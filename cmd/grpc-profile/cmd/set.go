package cmd

import (
	"errors"
	"fmt"
	"strconv"

	profile "github.com/chanchal1987/grpc-profile"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(setCmd)
}

var (
	setList = map[string]profile.Variable{
		"MemProfRate":          profile.MemProfRate,
		"CPUProfRate":          profile.CPUProfRate,
		"MutexProfileFraction": profile.MutexProfileFraction,
		"BlockProfileRate":     profile.BlockProfileRate,
	}

	setCmd = &cobra.Command{
		Use:     "set <variable> <value>",
		Short:   "Set veriable in agent",
		Long:    `Set a variable in the agent where this server is connected`,
		PreRunE: connect,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				argList := make([]string, len(setList))
				i := 0
				for k := range setList {
					argList[i] = k
					i++
				}

				return argList, cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				return invalidArgumentsError
			}
			val, ok := setList[args[0]]
			if !ok {
				return errors.New("unknown variable")
			}
			rt, err := strconv.Atoi(args[1])
			if err != nil {
				return err
			}
			pRt, err := client.Set(cmd.Context(), val, rt)
			if err != nil {
				return err
			}
			fmt.Println("Changed valus of", args[0], "from", pRt, "to", rt)
			return nil
		},
	}
)
