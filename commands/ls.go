package commands

import (
	"fmt"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/util"
	"github.com/spf13/cobra"
)

func runLsBug(cmd *cobra.Command, args []string) error {
	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()

	var query *cache.Query
	if len(args) >= 1 {
		query, err = cache.ParseQuery(args[0])

		if err != nil {
			return err
		}
	}

	allIds := backend.QueryBugs(query)

	for _, id := range allIds {
		b, err := backend.ResolveBug(id)
		if err != nil {
			return err
		}

		snapshot := b.Snapshot()

		var author bug.Person

		if len(snapshot.Comments) > 0 {
			create := snapshot.Comments[0]
			author = create.Author
		}

		// truncate + pad if needed
		titleFmt := fmt.Sprintf("%-50.50s", snapshot.Title)
		authorFmt := fmt.Sprintf("%-15.15s", author.Name)

		fmt.Printf("%s %s\t%s\t%s\t%s\n",
			util.Cyan(b.HumanId()),
			util.Yellow(snapshot.Status),
			titleFmt,
			util.Magenta(authorFmt),
			snapshot.Summary(),
		)
	}

	return nil
}

var lsCmd = &cobra.Command{
	Use:   "ls <query>",
	Short: "Display a summary of all bugs",
	RunE:  runLsBug,
}

func init() {
	RootCmd.AddCommand(lsCmd)
}
