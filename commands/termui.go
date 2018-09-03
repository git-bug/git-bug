package commands

import (
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/termui"
	"github.com/spf13/cobra"
)

func runTermUI(cmd *cobra.Command, args []string) error {
	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()

	return termui.Run(backend)
}

var termUICmd = &cobra.Command{
	Use:   "termui",
	Short: "Launch the terminal UI",
	RunE:  runTermUI,
}

func init() {
	RootCmd.AddCommand(termUICmd)
}
