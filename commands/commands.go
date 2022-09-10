package commands

import (
	"sort"

	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/commands/execenv"
)

type commandOptions struct {
	desc bool
}

func newCommandsCommand() *cobra.Command {
	env := execenv.NewEnv()
	options := commandOptions{}

	cmd := &cobra.Command{
		Use:   "commands",
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

func runCommands(env *execenv.Env, opts commandOptions) error {
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
			env.Out.Println()
		}

		first = false

		if opts.desc {
			env.Out.Printf("# %s\n", cmd.Short)
		}

		env.Out.Print(cmd.UseLine())

		if opts.desc {
			env.Out.Println()
		}
	}

	if !opts.desc {
		env.Out.Println()
	}

	return nil
}

type commandSorterByName []*cobra.Command

func (c commandSorterByName) Len() int           { return len(c) }
func (c commandSorterByName) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }
func (c commandSorterByName) Less(i, j int) bool { return c[i].CommandPath() < c[j].CommandPath() }
