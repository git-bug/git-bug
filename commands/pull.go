package commands

import (
	"errors"
	"fmt"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/spf13/cobra"
)

func runPull(cmd *cobra.Command, args []string) error {
	if len(args) > 1 {
		return errors.New("Only pulling from one remote at a time is supported")
	}

	remote := "origin"
	if len(args) == 1 {
		remote = args[0]
	}

	fmt.Printf("Fetching remote ...\n\n")

	if err := bug.Fetch(repo, remote); err != nil {
		return err
	}

	fmt.Printf("\nMerging data ...\n\n")

	for merge := range bug.MergeAll(repo, remote) {
		if merge.Err != nil {
			return merge.Err
		}

		if merge.Status != bug.MsgNothing {
			fmt.Printf("%s: %s\n", merge.HumanId, merge.Status)
		}
	}

	return nil
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
