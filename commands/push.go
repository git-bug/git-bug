package commands

import (
	"errors"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/spf13/cobra"
)

func runPush(cmd *cobra.Command, args []string) error {
	if len(args) > 1 {
		return errors.New("Only pushing to one remote at a time is supported")
	}

	remote := "origin"
	if len(args) == 1 {
		remote = args[0]
	}

	if err := repo.PushRefs(remote, bug.BugsRefPattern+"*"); err != nil {
		return err
	}
	return nil
}

// showCmd defines the "push" subcommand.
var pushCmd = &cobra.Command{
	Use:   "push [<remote>]",
	Short: "Push bugs update to a git remote",
	RunE:  runPush,
}

func init() {
	rootCmd.AddCommand(pushCmd)
}
