package commands

import (
	"fmt"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func runStatus(cmd *cobra.Command, args []string) error {
	var err error

	if len(args) > 1 {
		return errors.New("Only one bug id is supported")
	}

	if len(args) == 0 {
		return errors.New("You must provide a bug id")
	}

	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()

	prefix := args[0]

	b, err := backend.ResolveBugPrefix(prefix)
	if err != nil {
		return err
	}

	snap := b.Snapshot()

	fmt.Println(snap.Status)

	return nil
}

var statusCmd = &cobra.Command{
	Use:   "status <id>",
	Short: "Show the bug status",
	RunE:  runStatus,
}

func init() {
	RootCmd.AddCommand(statusCmd)
}
