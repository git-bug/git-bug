package bugcmd

import (
	"fmt"

	"github.com/spf13/cobra"

	buginput "github.com/git-bug/git-bug/commands/bug/input"
	"github.com/git-bug/git-bug/commands/execenv"
	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/util/text"
)

type bugReviewCommentOptions struct {
	path           string
	line           int
	endLine        int
	replyTo        string
	commitHash     string
	message        string
	messageFile    string
	nonInteractive bool
}

func newBugReviewCommentCommand(env *execenv.Env) *cobra.Command {
	opts := bugReviewCommentOptions{}

	cmd := &cobra.Command{
		Use:   "comment REVIEW_ID",
		Short: "Add a line-anchored comment to an existing review",
		Long: `Attach a line-anchored comment to an existing review thread on a
pull-request. REVIEW_ID is the review's combined id (or a unique prefix).

For new threads, supply --path and --line (optionally --end-line). For
replies to an existing review comment, supply --reply-to <COMMENT_ID>; --path,
--line and --commit will default to those of the comment being replied to.

--commit defaults to the PR's current head commit.`,
		PreRunE: execenv.LoadBackendEnsureUser(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBugReviewComment(env, opts, args)
		}),
	}

	flags := cmd.Flags()
	flags.SortFlags = false
	flags.StringVar(&opts.path, "path", "", "File path the comment anchors to")
	flags.IntVar(&opts.line, "line", 0, "Start line (1-based) in the file")
	flags.IntVar(&opts.endLine, "end-line", 0, "End line (inclusive). Defaults to --line for a single-line anchor.")
	flags.StringVar(&opts.replyTo, "reply-to", "", "Combined id of a review comment to reply to")
	flags.StringVar(&opts.commitHash, "commit", "", "Commit hash the comment is anchored to (defaults to PR head)")
	flags.StringVarP(&opts.message, "message", "m", "", "Comment body")
	flags.StringVarP(&opts.messageFile, "file", "F", "", "Take the body from the given file (- for stdin)")
	flags.BoolVar(&opts.nonInteractive, "non-interactive", false, "Do not ask for user input")

	return cmd
}

func runBugReviewComment(env *execenv.Env, opts bugReviewCommentOptions, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("review id argument required")
	}
	reviewPrefix := args[0]

	b, reviewCombined, err := env.Backend.Bugs().ResolveComment(reviewPrefix)
	if err != nil {
		// ResolveComment searches Comments; review ids live in snap.Reviews,
		// so fall back to a manual prefix scan of reviews.
		b, reviewCombined, err = resolveReviewByPrefix(env, reviewPrefix)
		if err != nil {
			return err
		}
	}

	snap := b.Snapshot()
	if snap.Kind != common.PRKind {
		return fmt.Errorf("%s is not a pull-request", b.Id().Human())
	}

	// If replying, inherit anchor from parent unless explicitly overridden.
	var replyTo entity.CombinedId
	path := opts.path
	startLine := opts.line
	endLine := opts.endLine
	commitHash := opts.commitHash

	if opts.replyTo != "" {
		parent, found := findReviewCommentByPrefix(snap, opts.replyTo)
		if !found {
			return fmt.Errorf("no review comment matching prefix %q", opts.replyTo)
		}
		replyTo = parent.CombinedId()
		if path == "" {
			path = parent.Path
		}
		if startLine == 0 {
			startLine = parent.StartLine
		}
		if endLine == 0 {
			endLine = parent.EndLine
		}
		if commitHash == "" {
			commitHash = parent.CommitHash
		}
	}

	if path == "" {
		return fmt.Errorf("--path is required (or --reply-to to inherit)")
	}
	if startLine <= 0 {
		return fmt.Errorf("--line must be a positive integer")
	}
	if commitHash == "" {
		commitHash = snap.HeadCommit
	}
	if commitHash == "" {
		return fmt.Errorf("no commit to anchor to; supply --commit or set a PR head")
	}

	if opts.messageFile != "" && opts.message == "" {
		opts.message, err = buginput.BugCommentFileInput(opts.messageFile)
		if err != nil {
			return err
		}
	}
	if opts.message == "" && opts.messageFile == "" && !opts.nonInteractive {
		opts.message, err = buginput.BugCommentEditorInput(env.Backend, "")
		if err == buginput.ErrEmptyMessage {
			env.Err.Println("Empty body, aborting.")
			return nil
		}
		if err != nil {
			return err
		}
	}
	if opts.message == "" {
		return fmt.Errorf("review comment body is required")
	}

	_, _, err = b.AddReviewComment(
		reviewCombined,
		text.Cleanup(opts.message),
		commitHash,
		path,
		startLine,
		endLine,
		replyTo,
	)
	if err != nil {
		return err
	}
	return b.Commit()
}
