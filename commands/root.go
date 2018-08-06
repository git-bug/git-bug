package commands

import (
	"fmt"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/spf13/cobra"
	"os"
)

// Will display "git bug"
// \u00A0 is a non-breaking space
// It's used to avoid cobra to split the Use string at the first space to get the root command name
//const rootCommandName = "git\u00A0bug"
const rootCommandName = "git-bug"

// package scoped var to hold the repo after the PreRun execution
var repo repository.Repo

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   rootCommandName,
	Short: "A bugtracker embedded in Git",
	Long: `git-bug is a bugtracker embedded in git.

It use the same internal storage so it doesn't pollute your project. As you would do with commits and branches, you can push your bugs to the same git remote your are already using to collaborate with other peoples.`,

	// Force the execution of the PreRun while still displaying the help
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},

	// Load the repo before any command execution
	// Note, this concern only commands that actually have a Run function
	PersistentPreRunE: loadRepo,

	DisableAutoGenTag: true,
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func loadRepo(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("Unable to get the current working directory: %q\n", err)
	}

	repo, err = repository.NewGitRepo(cwd, bug.Witnesser)
	if err == repository.ErrNotARepo {
		return fmt.Errorf("%s must be run from within a git repo.\n", rootCommandName)
	}

	if err != nil {
		return err
	}

	return nil
}
