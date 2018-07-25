package commands

import (
	"errors"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/spf13/cobra"
	"os"
)

func runPull(cmd *cobra.Command, args []string) error {
	if len(args) > 1 {
		return errors.New("Only pulling from one remote at a time is supported")
	}

	remote := "origin"
	if len(args) == 1 {
		remote = args[0]
	}

	return bug.Pull(repo, os.Stdout, remote)
}

// showCmd defines the "push" subcommand.
var pullCmd = &cobra.Command{
	Use:   "pull [<remote>]",
	Short: "Pull bugs update from a git remote",
	RunE:  runPull,
}

func init() {
	RootCmd.AddCommand(pullCmd)
}
