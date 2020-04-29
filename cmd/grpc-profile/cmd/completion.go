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
		var shell string
		writer := os.Stdout
		if len(args) == 2 {
			shell = args[0]
			var err error
			writer, err = os.Create(args[1])
			if err != nil {
				return err
			}
		} else if len(args) == 1 {
			shell = args[0]
		} else if len(args) == 0 {
			shell = getShell()
		} else {
			return errInvalidArguments
		}
		defer writer.Close()

		switch shell {
		case "bash":
			return rootCmd.GenBashCompletion(writer)
		case "zsh":
			return rootCmd.GenZshCompletion(writer)
		case "fish":
			return rootCmd.GenFishCompletion(writer, true)
		case "powershell":
			return rootCmd.GenPowerShellCompletion(writer)
		default:
			return errors.New("completion for shell " + shell + " is not supported yet")
		}
	},
	ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
}

func getShell() (shell string) {
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
	return
}
