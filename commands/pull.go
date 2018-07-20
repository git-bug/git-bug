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

	if err := repo.FetchRefs(remote, bug.BugsRefPattern+"*", bug.BugsRemoteRefPattern+"*"); err != nil {
		return err
	}

	fmt.Printf("\nMerging data ...\n\n")

	remoteRefSpec := fmt.Sprintf(bug.BugsRemoteRefPattern, remote)
	remoteRefs, err := repo.ListRefs(remoteRefSpec)

	if err != nil {
		return err
	}

	for _, ref := range remoteRefs {
		remoteRef := fmt.Sprintf(bug.BugsRemoteRefPattern, remote) + ref
		remoteBug, err := bug.ReadBug(repo, remoteRef)

		if err != nil {
			return err
		}

		// Check for error in remote data
		if !remoteBug.IsValid() {
			fmt.Printf("%s: %s\n", remoteBug.HumanId(), "invalid remote data")
			continue
		}

		localRef := bug.BugsRefPattern + remoteBug.Id()
		localExist, err := repo.RefExist(localRef)

		// the bug is not local yet, simply create the reference
		if !localExist {
			err := repo.CopyRef(remoteRef, localRef)

			if err != nil {
				return err
			}

			fmt.Printf("%s: %s\n", remoteBug.HumanId(), "new")
			continue
		}

		localBug, err := bug.ReadBug(repo, localRef)

		if err != nil {
			return err
		}

		updated, err := localBug.Merge(repo, remoteBug)

		if err != nil {
			return err
		}

		if updated {
			fmt.Printf("%s: %s\n", remoteBug.HumanId(), "updated")
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
