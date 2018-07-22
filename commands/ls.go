package commands

import (
	"fmt"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/util"
	"github.com/spf13/cobra"
)

func runLsBug(cmd *cobra.Command, args []string) error {
	bugs := bug.ReadAllLocalBugs(repo)

	for b := range bugs {
		if b.Err != nil {
			return b.Err
		}

		snapshot := b.Bug.Compile()

		var author bug.Person

		if len(snapshot.Comments) > 0 {
			create := snapshot.Comments[0]
			author = create.Author
		}

		// truncate + pad if needed
		titleFmt := fmt.Sprintf("%-50.50s", snapshot.Title)
		authorFmt := fmt.Sprintf("%-15.15s", author.Name)

		fmt.Printf("%s %s\t%s\t%s\t%s\n",
			util.Cyan(b.Bug.HumanId()),
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
	RootCmd.AddCommand(lsCmd)
}
