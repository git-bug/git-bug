package commands

import (
	"fmt"
	b "github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/repository"
)

func RunListBug(repo repository.Repo, args []string) error {
	refs, err := repo.ListRefs(b.BugsRefPattern)

	if err != nil {
		return err
	}

	for _, ref := range refs {
		bug, err := b.ReadBug(repo, ref)

		if err != nil {
			return err
		}

		snapshot := bug.Compile()

		fmt.Printf("%s %s\n", bug.HumanId(), snapshot.Title)
	}

	return nil
}

var listCmd = &Command{
	Usage: func(arg0 string) {
		fmt.Printf("Usage: %s\n", arg0)
	},
	RunMethod: RunListBug,
}
