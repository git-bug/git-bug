package commands

import (
	"fmt"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/input"
	"github.com/MichaelMure/git-bug/util/interrupt"
	"github.com/spf13/cobra"
)

var (
	addTitle       string
	addMessage     string
	addMessageFile string
)

func runAddBug(cmd *cobra.Command, args []string) error {
	var err error

	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	if addMessageFile != "" && addMessage == "" {
		addTitle, addMessage, err = input.BugCreateFileInput(addMessageFile)
		if err != nil {
			return err
		}
	}

	if addMessageFile == "" && (addMessage == "" || addTitle == "") {
		addTitle, addMessage, err = input.BugCreateEditorInput(backend, addTitle, addMessage)

		if err == input.ErrEmptyTitle {
			fmt.Println("Empty title, aborting.")
			return nil
		}
		if err != nil {
			return err
		}
	}

	b, err := backend.NewBug(addTitle, addMessage)
	if err != nil {
		return err
	}

	fmt.Printf("%s created\n", b.HumanId())

	return nil
}

var addCmd = &cobra.Command{
	Use:     "add",
	Short:   "Create a new bug.",
	PreRunE: loadRepo,
	RunE:    runAddBug,
}

func init() {
	RootCmd.AddCommand(addCmd)

	addCmd.Flags().SortFlags = false

	addCmd.Flags().StringVarP(&addTitle, "title", "t", "",
		"Provide a title to describe the issue",
	)
	addCmd.Flags().StringVarP(&addMessage, "message", "m", "",
		"Provide a message to describe the issue",
	)
	addCmd.Flags().StringVarP(&addMessageFile, "file", "F", "",
		"Take the message from the given file. Use - to read the message from the standard input",
	)
}
