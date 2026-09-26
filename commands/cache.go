package commands

import (
	"github.com/spf13/cobra"

	"github.com/git-bug/git-bug/commands/execenv"
)

func newCacheCommand(env *execenv.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Manage the local cache",
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(newCacheRebuildCommand(env))

	return cmd
}

func newCacheRebuildCommand(env *execenv.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rebuild",
		Short: "Rebuild the local cache from the git data",
		Long: `Rebuild the local cache from the git data.

The cache is not aware of changes made to git-bug's references by other tools, for example when fetching or pushing them with git directly. Use this command to make git-bug reflect them.`,
		PreRunE: execenv.LoadBackendRebuild(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return nil
		}),
		Args: cobra.NoArgs,
	}

	return cmd
}
