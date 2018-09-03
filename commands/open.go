package commands

import (
	"errors"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/spf13/cobra"
)

func runOpenBug(cmd *cobra.Command, args []string) error {
	if len(args) > 1 {
		return errors.New("Only opening one bug at a time is supported")
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

	err = b.Open()
	if err != nil {
		return err
	}

	return b.Commit()
}

var openCmd = &cobra.Command{
	Use:   "open <id>",
	Short: "Mark the bug as open",
	RunE:  runOpenBug,
}

func init() {
	RootCmd.AddCommand(openCmd)
}
