package boardcmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	buginput "github.com/git-bug/git-bug/commands/bug/input"
	"github.com/git-bug/git-bug/commands/execenv"
	_select "github.com/git-bug/git-bug/commands/select"
	"github.com/git-bug/git-bug/entity"
)

type boardAddDraftOptions struct {
	title          string
	messageFile    string
	message        string
	column         string
	nonInteractive bool
}

func newBoardAddDraftCommand() *cobra.Command {
	env := execenv.NewEnv()
	options := boardAddDraftOptions{}

	cmd := &cobra.Command{
		Use:     "add-draft [BOARD_ID]",
		Short:   "Add a draft item to a board",
		PreRunE: execenv.LoadBackend(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBoardAddDraft(env, options, args)
		}),
		ValidArgsFunction: BoardCompletion(env),
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	flags.StringVarP(&options.title, "title", "t", "",
		"Provide the title to describe the draft item")
	flags.StringVarP(&options.message, "message", "m", "",
		"Provide the message of the draft item")
	flags.StringVarP(&options.messageFile, "file", "F", "",
		"Take the message from the given file. Use - to read the message from the standard input")
	flags.StringVarP(&options.column, "column", "c", "1",
		"The column to add to. Either a column Id or prefix, or the column number starting from 1.")
	_ = cmd.RegisterFlagCompletionFunc("column", ColumnCompletion(env))
	flags.BoolVar(&options.nonInteractive, "non-interactive", false, "Do not ask for user input")

	return cmd
}

func runBoardAddDraft(env *execenv.Env, opts boardAddDraftOptions, args []string) error {
	b, args, err := ResolveSelected(env.Backend, args)

	var columnId entity.CombinedId

	switch {
	case err == nil:
		// try to parse as column number
		index, err := strconv.Atoi(opts.column)
		if err == nil {
			if index-1 >= 0 && index-1 < len(b.Snapshot().Columns) {
				columnId = b.Snapshot().Columns[index-1].CombinedId
			} else {
				return fmt.Errorf("invalid column")
			}
		}
		fallthrough // could be an Id
	case _select.IsErrNoValidId(err):
		b, columnId, err = env.Backend.Boards().ResolveColumn(opts.column)
		if err != nil {
			return err
		}
	default:
		// actual error
		return err
	}

	if opts.messageFile != "" && opts.message == "" {
		// Note: reuse the bug inputs
		opts.title, opts.message, err = buginput.BugCreateFileInput(opts.messageFile)
		if err != nil {
			return err
		}
	}

	if !opts.nonInteractive && opts.messageFile == "" && (opts.message == "" || opts.title == "") {
		opts.title, opts.message, err = buginput.BugCreateEditorInput(env.Backend, opts.title, opts.message)
		if err == buginput.ErrEmptyTitle {
			env.Out.Println("Empty title, aborting.")
			return nil
		}
		if err != nil {
			return err
		}
	}

	id, _, err := b.AddItemDraft(columnId, opts.title, opts.message, nil)
	if err != nil {
		return err
	}

	env.Out.Printf("%s created\n", id.Human())

	return b.Commit()
}
