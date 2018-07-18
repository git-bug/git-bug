package commands

import (
	"errors"
	"fmt"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util"
	"strings"
)

func runShowBug(repo repository.Repo, args []string) error {
	if len(args) > 1 {
		return errors.New("Only showing one bug at a time is supported")
	}

	if len(args) == 0 {
		return errors.New("You must provide a bug id")
	}

	prefix := args[0]

	b, err := bug.FindBug(repo, prefix)
	if err != nil {
		return err
	}

	snapshot := b.Compile()

	if len(snapshot.Comments) == 0 {
		return errors.New("Invalid bug: no comment")
	}

	firstComment := snapshot.Comments[0]

	// Header
	fmt.Printf("[%s] %s %s\n\n",
		util.Yellow(snapshot.Status),
		util.Cyan(snapshot.HumanId()),
		snapshot.Title,
	)

	fmt.Printf("%s opened this issue %s\n\n",
		util.Magenta(firstComment.Author.Name),
		firstComment.FormatTime(),
	)

	var labels = make([]string, len(snapshot.Labels))
	for i := range snapshot.Labels {
		labels[i] = string(snapshot.Labels[i])
	}

	fmt.Printf("labels: %s\n\n",
		strings.Join(labels, ", "),
	)

	// Comments
	indent := "  "

	for i, comment := range snapshot.Comments {
		fmt.Printf("%s#%d %s <%s>\n\n",
			indent,
			i,
			comment.Author.Name,
			comment.Author.Email,
		)

		fmt.Printf("%s%s\n\n\n",
			indent,
			comment.Message,
		)
	}

	return nil
}

var showCmd = &Command{
	Description: "Display the details of a bug",
	Usage:       "<id>",
	RunMethod:   runShowBug,
}
