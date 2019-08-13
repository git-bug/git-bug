package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/util/interrupt"
)

func runLsID(cmd *cobra.Command, args []string) error {

	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	var prefix = ""
	if len(args) != 0 {
		prefix = args[0]
	}

	for _, id := range backend.AllBugsIds() {
		if prefix == "" || id.HasPrefix(prefix) {
			fmt.Println(id)
		}
	}

	return nil
}

var listBugIDCmd = &cobra.Command{
	Use:     "ls-id [<prefix>]",
	Short:   "List bug identifiers.",
	PreRunE: loadRepo,
	RunE:    runLsID,
}

func init() {
	RootCmd.AddCommand(listBugIDCmd)
}
