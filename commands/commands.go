package commands

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
)

var (
	commandsDesc bool
)

type commandSorterByName []*cobra.Command

func (c commandSorterByName) Len() int           { return len(c) }
func (c commandSorterByName) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }
func (c commandSorterByName) Less(i, j int) bool { return c[i].CommandPath() < c[j].CommandPath() }

func runCommands(cmd *cobra.Command, args []string) error {
	first := true

	var allCmds []*cobra.Command
	queue := []*cobra.Command{RootCmd}

	for len(queue) > 0 {
		cmd := queue[0]
		queue = queue[1:]
		allCmds = append(allCmds, cmd)
		for _, c := range cmd.Commands() {
			queue = append(queue, c)
		}
	}

	sort.Sort(commandSorterByName(allCmds))

	for _, cmd := range allCmds {
		if !first {
			fmt.Println()
		}

		first = false

		if commandsDesc {
			fmt.Printf("# %s\n", cmd.Short)
		}

		fmt.Print(cmd.UseLine())

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
	RootCmd.AddCommand(commandsCmd)

	commandsCmd.Flags().SortFlags = false

	commandsCmd.Flags().BoolVarP(&commandsDesc, "pretty", "p", false,
		"Output the command description as well as Markdown compatible comment",
	)
}
