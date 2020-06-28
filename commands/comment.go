package commands

import (
	text "github.com/MichaelMure/go-term-text"
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/commands/select"
	"github.com/MichaelMure/git-bug/util/colors"
	"github.com/MichaelMure/git-bug/util/interrupt"
)

func newCommentCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:     "comment [<id>]",
		Short:   "Display or add comments to a bug.",
		PreRunE: loadRepo(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runComment(env, args)
		},
	}

	cmd.AddCommand(newCommentAddCommand())

	return cmd
}

func runComment(env *Env, args []string) error {
	backend, err := cache.NewRepoCache(env.repo)
	if err != nil {
		return err
	}
	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	b, args, err := _select.ResolveBug(backend, args)
	if err != nil {
		return err
	}

	snap := b.Snapshot()

	for i, comment := range snap.Comments {
		if i != 0 {
			env.out.Println()
		}

		env.out.Printf("Author: %s\n", colors.Magenta(comment.Author.DisplayName()))
		env.out.Printf("Id: %s\n", colors.Cyan(comment.Id().Human()))
		env.out.Printf("Date: %s\n\n", comment.FormatTime())
		env.out.Println(text.LeftPadLines(comment.Message, 4))
	}

	return nil
}
