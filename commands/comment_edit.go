package commands

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	_select "github.com/MichaelMure/git-bug/commands/select"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/input"
)

type commentEditOptions struct {
	messageFile string
	message     string
}

func newCommentEditCommand() *cobra.Command {
	env := newEnv()
	options := commentEditOptions{}

	cmd := &cobra.Command{
		Use:      "edit [<id>]",
		Short:    "Edit an existing comment on a bug.",
		PreRunE:  loadBackendEnsureUser(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommentEdit(env, options, args)
		},
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	flags.StringVarP(&options.messageFile, "file", "F", "",
		"Take the message from the given file. Use - to read the message from the standard input")

	flags.StringVarP(&options.message, "message", "m", "",
		"Provide the new message from the command line")

	return cmd
}

func ResolveComment(repo *cache.RepoCache, args []string) (*cache.BugCache, entity.Id, error) {
	fullId := args[0]
	bugId, _ := bug.UnpackCommentId(args[0])
	args[0] = bugId
	b, args, err := _select.ResolveBug(repo, args)
	if err != nil {
		return nil, entity.UnsetId, err
	}

	matching := make([]entity.Id, 0, 5)

	for _, comment := range b.Snapshot().Comments {
		if comment.Id().HasPrefix(fullId) {
			matching = append(matching, comment.Id())
		}
	}

	if len(matching) > 1 {
		return nil, entity.UnsetId, entity.NewErrMultipleMatch("comment", matching)
	} else if len(matching) == 0 {
		return nil, entity.UnsetId, errors.New("comment doesn't exist")
	}

	return b, matching[0], nil
}

func runCommentEdit(env *Env, opts commentEditOptions, args []string) error {
	b, c, err := ResolveComment(env.backend, args)

	if err != nil {
		return err
	}

	if opts.messageFile != "" && opts.message == "" {
		opts.message, err = input.BugCommentFileInput(opts.messageFile)
		if err != nil {
			return err
		}
	}

	if opts.messageFile == "" && opts.message == "" {
		opts.message, err = input.BugCommentEditorInput(env.backend, "")
		if err == input.ErrEmptyMessage {
			env.err.Println("Empty message, aborting.")
			return nil
		}
		if err != nil {
			return err
		}
	}

	_, err = b.EditComment(c, opts.message)
	if err != nil {
		return err
	}

	return b.Commit()
}
