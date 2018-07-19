package commands

import (
	"fmt"
	b "github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/util"
	"github.com/spf13/cobra"
)

func runLsBug(cmd *cobra.Command, args []string) error {
	ids, err := repo.ListRefs(b.BugsRefPattern)

	if err != nil {
		return err
	}

	for _, ref := range ids {
		bug, err := b.ReadBug(repo, b.BugsRefPattern+ref)

		if err != nil {
			return err
		}

		snapshot := bug.Compile()

		var author b.Person

		if len(snapshot.Comments) > 0 {
			create := snapshot.Comments[0]
			author = create.Author
		}

		// truncate + pad if needed
		titleFmt := fmt.Sprintf("%-50.50s", snapshot.Title)
		authorFmt := fmt.Sprintf("%-15.15s", author.Name)

		fmt.Printf("%s %s\t%s\t%s\t%s\n",
			util.Cyan(bug.HumanId()),
			util.Yellow(snapshot.Status),
			titleFmt,
			util.Magenta(authorFmt),
			snapshot.Summary(),
		)
	}

	return nil
}

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "Display a summary of all bugs",
	RunE:  runLsBug,
}

func init() {
	rootCmd.AddCommand(lsCmd)
}
