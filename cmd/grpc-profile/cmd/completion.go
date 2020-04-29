package cmd

import (
	"errors"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(completionCmd)
}

var completionCmd = &cobra.Command{
	Use:   "completion",
	Short: "Generates bash completion scripts",
	Long: `To load completion run

. <(` + applName + ` completion bash)

To configure your bash shell to load completions for each session add to your bashrc

# ~/.bashrc or ~/.profile
. <(` + applName + ` completion bash)
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			return invalidArgumentsError
		}
		var shell string
		if len(args) == 1 {
			shell = args[0]
		} else {
			if os.Getenv("ZSH_VERSION") != "" {
				shell = "zsh"
			} else if os.Getenv("BASH_VERSION") != "" {
				shell = "bash"
			} else if os.Getenv("FISH_VERSION") != "" {
				shell = "fish"
			} else if os.Getenv("PSVersionTable.PSVersion") != "" {
				shell = "powershell"
			} else {
				shell = "unknown"
			}
		}

		switch shell {
		case "bash":
			return rootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			return rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			return rootCmd.GenFishCompletion(os.Stdout, true)
		case "powershell":
			return rootCmd.GenPowerShellCompletion(os.Stdout)
		default:
			return errors.New("completion for shell " + shell + " is not supported yet")
		}
	},
	ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
}
