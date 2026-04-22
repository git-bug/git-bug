package bugcmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/git-bug/git-bug/commands/execenv"
	"github.com/git-bug/git-bug/entities/common"
)

func newBugStatusReadyCommand(env *execenv.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "ready [BUG_ID]",
		Short:   "Mark a draft pull-request as ready for review",
		Long:    "Transition a pull-request from draft to open. No-op for non-draft PRs and rejected for issues.",
		PreRunE: execenv.LoadBackendEnsureUser(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBugStatusReady(env, args)
		}),
		ValidArgsFunction: BugCompletion(env),
	}

	return cmd
}

func runBugStatusReady(env *execenv.Env, args []string) error {
	b, _, err := ResolveSelected(env.Backend, args)
	if err != nil {
		return err
	}

	snap := b.Snapshot()
	if snap.Kind != common.PRKind {
		return fmt.Errorf("%s is not a pull-request (kind=%s)", b.Id().Human(), snap.Kind)
	}
	if snap.Status != common.DraftStatus {
		return fmt.Errorf("%s is not a draft (status=%s)", b.Id().Human(), snap.Status)
	}

	if _, err := b.MarkReady(); err != nil {
		return err
	}
	return b.Commit()
}
