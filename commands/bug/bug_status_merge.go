package bugcmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/git-bug/git-bug/commands/execenv"
	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/util/text"
)

type bugStatusMergeOptions struct {
	mergeCommit string
}

func newBugStatusMergeCommand(env *execenv.Env) *cobra.Command {
	opts := bugStatusMergeOptions{}

	cmd := &cobra.Command{
		Use:     "merge [BUG_ID]",
		Short:   "Mark a pull-request as merged",
		Long:    "Mark a pull-request as merged and record the merge commit hash. Only valid for bugs whose kind is pr.",
		PreRunE: execenv.LoadBackendEnsureUser(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBugStatusMerge(env, opts, args)
		}),
		ValidArgsFunction: BugCompletion(env),
	}

	cmd.Flags().StringVarP(&opts.mergeCommit, "commit", "c", "",
		"The merge commit hash. Required.")

	return cmd
}

func runBugStatusMerge(env *execenv.Env, opts bugStatusMergeOptions, args []string) error {
	mergeCommit := text.CleanupOneLine(opts.mergeCommit)
	if mergeCommit == "" {
		return fmt.Errorf("merge requires --commit <hash>")
	}

	b, _, err := ResolveSelected(env.Backend, args)
	if err != nil {
		return err
	}

	if b.Snapshot().Kind != common.PRKind {
		return fmt.Errorf("%s is not a pull-request (kind=%s)", b.Id().Human(), b.Snapshot().Kind)
	}

	if _, err := b.Merge(mergeCommit); err != nil {
		return err
	}
	return b.Commit()
}
