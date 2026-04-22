package bugcmd

import (
	text "github.com/MichaelMure/go-term-text"
	"github.com/spf13/cobra"

	"github.com/git-bug/git-bug/commands/execenv"
	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/util/colors"
)

func newBugReviewCommand(env *execenv.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "review [BUG_ID]",
		Short:   "List reviews on a pull-request",
		Long:    "List each review on the given pull-request, with its verdict, body, and line-anchored review comments.",
		PreRunE: execenv.LoadBackend(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBugReview(env, args)
		}),
		ValidArgsFunction: BugCompletion(env),
	}

	cmd.AddCommand(newBugReviewAddCommand(env))
	cmd.AddCommand(newBugReviewCommentCommand(env))

	return cmd
}

func runBugReview(env *execenv.Env, args []string) error {
	b, _, err := ResolveSelected(env.Backend, args)
	if err != nil {
		return err
	}
	snap := b.Snapshot()
	if snap.Kind != common.PRKind {
		env.Err.Printf("%s is not a pull-request (kind=%s); nothing to review.\n", b.Id().Human(), snap.Kind)
		return nil
	}
	if len(snap.Reviews) == 0 {
		env.Out.Println("No reviews yet.")
		return nil
	}

	for i, r := range snap.Reviews {
		if i != 0 {
			env.Out.Println()
		}
		env.Out.Printf("Review: %s\n", colors.Cyan(r.CombinedId().Human()))
		env.Out.Printf("Author: %s\n", colors.Magenta(r.Author.DisplayName()))
		env.Out.Printf("State:  %s\n", colors.Yellow(r.State.String()))
		env.Out.Printf("Commit: %s\n", r.CommitHash)
		if r.Body != "" {
			env.Out.Println()
			env.Out.Println(text.LeftPadLines(r.Body, 4))
		}
		for _, c := range r.Comments {
			env.Out.Println()
			env.Out.Printf("  Comment: %s\n", colors.Cyan(c.CombinedId().Human()))
			env.Out.Printf("  Author:  %s\n", colors.Magenta(c.Author.DisplayName()))
			anchor := c.Path + ":"
			if c.EndLine != 0 && c.EndLine != c.StartLine {
				anchor += colors.Green("")
			}
			env.Out.Printf("  Anchor:  %s:%d", c.Path, c.StartLine)
			if c.EndLine != 0 && c.EndLine != c.StartLine {
				env.Out.Printf("-%d", c.EndLine)
			}
			env.Out.Printf(" @%s", c.CommitHash)
			if c.ReplyTo != "" {
				env.Out.Printf(" (reply to %s)", c.ReplyTo.Human())
			}
			env.Out.Println()
			if c.Body != "" {
				env.Out.Println(text.LeftPadLines(c.Body, 6))
			}
		}
	}

	return nil
}
