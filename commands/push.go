package commands

import (
	"errors"
	"fmt"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/util/interrupt"
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

	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	stdout, err := backend.Push(remote)
	if err != nil {
		return err
	}

	fmt.Println(stdout)

	return nil
}

// showCmd defines the "push" subcommand.
var pushCmd = &cobra.Command{
	Use:     "push [<remote>]",
	Short:   "Push bugs update to a git remote",
	PreRunE: loadRepo,
	RunE:    runPush,
}

func init() {
	RootCmd.AddCommand(pushCmd)
}
