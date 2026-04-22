package bugcmd

import (
	"fmt"

	"github.com/spf13/cobra"

	buginput "github.com/git-bug/git-bug/commands/bug/input"
	"github.com/git-bug/git-bug/commands/execenv"
	"github.com/git-bug/git-bug/entities/bug"
	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/util/text"
)

type bugReviewAddOptions struct {
	approve         bool
	requestChanges  bool
	comment         bool
	message         string
	messageFile     string
	commitHash      string
	nonInteractive  bool
}

func newBugReviewAddCommand(env *execenv.Env) *cobra.Command {
	opts := bugReviewAddOptions{}

	cmd := &cobra.Command{
		Use:   "add [BUG_ID]",
		Short: "Record a review on a pull-request",
		Long: `Record a review verdict (approve, request-changes, or comment) on a
pull-request. Exactly one of --approve / --request-changes / --comment must
be supplied. By default the review is anchored to the PR's current head
commit; override with --commit.`,
		PreRunE: execenv.LoadBackendEnsureUser(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBugReviewAdd(env, opts, args)
		}),
		ValidArgsFunction: BugCompletion(env),
	}

	flags := cmd.Flags()
	flags.SortFlags = false
	flags.BoolVar(&opts.approve, "approve", false, "Approve the pull-request")
	flags.BoolVar(&opts.requestChanges, "request-changes", false, "Request changes on the pull-request")
	flags.BoolVar(&opts.comment, "comment", false, "Leave a comment-only review (no verdict)")
	flags.StringVarP(&opts.message, "message", "m", "", "Review body")
	flags.StringVarP(&opts.messageFile, "file", "F", "", "Take the review body from the given file (- for stdin)")
	flags.StringVar(&opts.commitHash, "commit", "", "Commit hash the review is anchored to (defaults to the PR's head commit)")
	flags.BoolVar(&opts.nonInteractive, "non-interactive", false, "Do not ask for user input")

	return cmd
}

func runBugReviewAdd(env *execenv.Env, opts bugReviewAddOptions, args []string) error {
	b, _, err := ResolveSelected(env.Backend, args)
	if err != nil {
		return err
	}
	snap := b.Snapshot()
	if snap.Kind != common.PRKind {
		return fmt.Errorf("%s is not a pull-request (kind=%s)", b.Id().Human(), snap.Kind)
	}

	verdicts := 0
	var state bug.ReviewState
	switch {
	case opts.approve:
		verdicts++
		state = bug.ReviewApproved
	}
	if opts.requestChanges {
		verdicts++
		state = bug.ReviewChangesRequested
	}
	if opts.comment {
		verdicts++
		state = bug.ReviewCommented
	}
	if verdicts != 1 {
		return fmt.Errorf("exactly one of --approve / --request-changes / --comment is required")
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
			// An empty body is acceptable for approvals/request-changes — GitHub
			// treats them as a verdict without body text. Only warn, don't abort.
			opts.message = ""
		} else if err != nil {
			return err
		}
	}

	commitHash := opts.commitHash
	if commitHash == "" {
		commitHash = snap.HeadCommit
	}
	if commitHash == "" {
		return fmt.Errorf("--commit not supplied and PR has no recorded head commit")
	}

	_, _, err = b.AddReview(state, text.Cleanup(opts.message), commitHash)
	if err != nil {
		return err
	}
	return b.Commit()
}
