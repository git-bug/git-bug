package commands

import (
	"errors"
	"fmt"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/util/git"
	"github.com/spf13/cobra"
)

var (
	defaultRemote bool
)

func runPush(cmd *cobra.Command, args []string) error {
	if len(args) > 1 {
		return errors.New("Only pushing to one remote at a time is supported")
	}

	if len(args) == 0 && defaultRemote {
		str, err := git.GetConfig("gitbug.defaultremote")
		if _, ok := err.(*git.ErrNotFound); ok {
			return errors.New("No default remote set")
		} else {
			return err
		}

		fmt.Println("Default remote: ", str)
		return nil
	}

	remote := ""
	if len(args) == 1 {
		remote = args[0]
	} else if str, err := git.GetConfig("gitbug.defaultremote"); err == nil {
		remote = str
	} else {
		return errors.New("No remote provided, and no defaults specified.")
	}

	if defaultRemote {
		if err := git.SetConfig("gitbug.defaultremote", remote); err != nil {
			return err
		}
	}

	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()

	stdout, err := backend.Push(remote)
	if err != nil {
		return err
	}

	fmt.Println(stdout)

	return nil
}

// showCmd defines the "push" subcommand.
var pushCmd = &cobra.Command{
	Use:   "push [<remote>]",
	Short: "Push bugs update to a git remote",
	RunE:  runPush,
}

func init() {
	RootCmd.AddCommand(pushCmd)

	pushCmd.Flags().SortFlags = false

	pushCmd.Flags().BoolVarP(&defaultRemote, "default", "d", false,
		"If a remote is provided, set the default remote for future pushes. Otherwise list the default remote if one exists.")
}
