package commands

import (
	"fmt"
	"strings"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/spf13/cobra"
)

func runLsID(cmd *cobra.Command, args []string) error {

	if len(args) < 1 {
		_, err := ListAllID()

		if err != nil {
			return err
		}

		return nil
	}
	answer, err := ListID(args[0])

	if err != nil {
		return err
	}

	if answer == "" {
		fmt.Printf("No matching bug Id with prefix %s\n", args[0])
	} else {
		fmt.Println(answer)
	}

	return nil
}

//ListID lists the local bug id after taking the prefix as input
func ListID(prefix string) (string, error) {

	IDlist, err := bug.ListLocalIds(repo)

	if err != nil {
		return "", err
	}

	for _, id := range IDlist {
		if strings.HasPrefix(id, prefix) {
			return id, nil
		}
	}

	return "", nil

}

//ListAllID lists all the local bug id
func ListAllID() (string, error) {

	IDlist, err := bug.ListLocalIds(repo)
	if err != nil {
		return "", err
	}

	for _, id := range IDlist {
		fmt.Println(id)
	}

	return "", nil
}

var listBugIDCmd = &cobra.Command{
	Use:     "ls-id [<prefix>]",
	Short:   "List Bug Id",
	PreRunE: loadRepo,
	RunE:    runLsID,
}

func init() {
	RootCmd.AddCommand(listBugIDCmd)
}
