package commands

import (
	"fmt"
	"github.com/spf13/cobra"
)

var commandsDesc bool

func runCommands(cmd *cobra.Command, args []string) error {
	first := true

	allCmds := cmd.Root().Commands()

	for _, cmd := range allCmds {
		if !first {
			fmt.Println()
		}

		first = false

		if commandsDesc {
			fmt.Printf("# %s\n", cmd.Short)
		}

		fmt.Printf("%s %s",
			rootCommandName,
			cmd.Use,
		)

		if commandsDesc {
			fmt.Println()
		}
	}

	if !commandsDesc {
		fmt.Println()
	}

	return nil
}

var commandsCmd = &cobra.Command{
	Use:   "commands [<option>...]",
	Short: "Display available commands",
	RunE:  runCommands,
}

func init() {
	rootCmd.AddCommand(commandsCmd)

	commandsCmd.Flags().BoolVarP(&commandsDesc, "pretty", "p", false,
		"Output the command description as well as Markdown compatible comment",
	)
}
