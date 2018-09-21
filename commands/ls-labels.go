package commands

import (
	"fmt"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/spf13/cobra"
)

func runLsLabel(cmd *cobra.Command, args []string) error {
	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()

	labels := backend.ValidLabels()

	for _, l := range labels {
		fmt.Println(l)
	}

	return nil
}

var lsLabelCmd = &cobra.Command{
	Use:   "ls-label",
	Short: "List valid labels",
	Long: `List valid labels.

Note: in the future, a proper label policy could be implemented where valid labels are defined in a configuration file. Until that, the default behavior is to return the list of labels already used.`,
	RunE: runLsLabel,
}

func init() {
	RootCmd.AddCommand(lsLabelCmd)
}
