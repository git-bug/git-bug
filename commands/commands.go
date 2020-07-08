package commands

import (
	"sort"

	"github.com/spf13/cobra"
)

type commandOptions struct {
	desc bool
}

func newCommandsCommand() *cobra.Command {
	env := newEnv()
	options := commandOptions{}

	cmd := &cobra.Command{
		Use:   "commands [<option>...]",
		Short: "Display available commands.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommands(env, options)
		},
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	flags.BoolVarP(&options.desc, "pretty", "p", false,
		"Output the command description as well as Markdown compatible comment",
	)

	return cmd
}

func runCommands(env *Env, opts commandOptions) error {
	first := true

	var allCmds []*cobra.Command
	queue := []*cobra.Command{NewRootCommand()}

	for len(queue) > 0 {
		cmd := queue[0]
		queue = queue[1:]
		allCmds = append(allCmds, cmd)
		queue = append(queue, cmd.Commands()...)
	}

	sort.Sort(commandSorterByName(allCmds))

	for _, cmd := range allCmds {
		if !first {
			env.out.Println()
		}

		first = false

		if opts.desc {
			env.out.Printf("# %s\n", cmd.Short)
		}

		env.out.Print(cmd.UseLine())

		if opts.desc {
			env.out.Println()
		}
	}

	if !opts.desc {
		env.out.Println()
	}

	return nil
}

type commandSorterByName []*cobra.Command

func (c commandSorterByName) Len() int           { return len(c) }
func (c commandSorterByName) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }
func (c commandSorterByName) Less(i, j int) bool { return c[i].CommandPath() < c[j].CommandPath() }
