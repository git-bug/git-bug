package boardcmd

import (
	"github.com/spf13/cobra"

	bugcmd "github.com/git-bug/git-bug/commands/bug"
	"github.com/git-bug/git-bug/commands/execenv"
)

type boardAddBugOptions struct {
	column string
}

func newBoardAddBugCommand() *cobra.Command {
	env := execenv.NewEnv()
	options := boardAddBugOptions{}

	cmd := &cobra.Command{
		Use:     "add-bug [BOARD_ID] [BUG_ID]",
		Short:   "Add a bug to a board",
		PreRunE: execenv.LoadBackend(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBoardAddBug(env, options, args)
		}),
		ValidArgsFunction: BoardAndBugCompletion(env),
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	flags.StringVarP(&options.column, "column", "c", "1",
		"The column to add to. Either a column Id or prefix, or the column number starting from 1.")
	_ = cmd.RegisterFlagCompletionFunc("column", ColumnCompletion(env))

	return cmd
}

func runBoardAddBug(env *execenv.Env, opts boardAddBugOptions, args []string) error {
	b, columnId, err := resolveColumnId(env, opts.column, args)
	if err != nil {
		return err
	}

	bug, _, err := bugcmd.ResolveSelected(env.Backend, args)
	if err != nil {
		return err
	}

	id, _, err := b.AddItemEntity(columnId, bug)
	if err != nil {
		return err
	}

	env.Out.Printf("%s created\n", id.Human())

	return b.Commit()
}
