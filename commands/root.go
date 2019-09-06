// Package commands contains the CLI commands
package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/repository"
)

const rootCommandName = "git-bug"

// package scoped var to hold the repo after the PreRun execution
var repo repository.ClockedRepo

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   rootCommandName,
	Short: "A bug tracker embedded in Git.",
	Long: `git-bug is a bug tracker embedded in git.

git-bug use git objects to store the bug tracking separated from the files
history. As bugs are regular git objects, they can be pushed and pulled from/to
the same git remote your are already using to collaborate with other peoples.

`,

	// For the root command, force the execution of the PreRun
	// even if we just display the help. This is to make sure that we check
	// the repository and give the user early feedback.
	Run: func(cmd *cobra.Command, args []string) {
		if err := cmd.Help(); err != nil {
			os.Exit(1)
		}
	},

	SilenceUsage:      true,
	DisableAutoGenTag: true,

	// Custom bash code to connect the git completion for "git bug" to the
	// git-bug completion for "git-bug"
	BashCompletionFunction: `
_git_bug() {
    __start_git-bug "$@"
}
`,
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// loadRepo is a pre-run function that load the repository for use in a command
func loadRepo(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("unable to get the current working directory: %q", err)
	}

	repo, err = repository.NewGitRepo(cwd, bug.Witnesser)
	if err == repository.ErrNotARepo {
		return fmt.Errorf("%s must be run from within a git repo", rootCommandName)
	}

	if err != nil {
		return err
	}

	return nil
}

// loadRepoEnsureUser is the same as loadRepo, but also ensure that the user has configured
// an identity. Use this pre-run function when an error after using the configured user won't
// do.
func loadRepoEnsureUser(cmd *cobra.Command, args []string) error {
	err := loadRepo(cmd, args)
	if err != nil {
		return err
	}

	set, err := identity.IsUserIdentitySet(repo)
	if err != nil {
		return err
	}

	if !set {
		// Print the error directly to not confuse a user
		_, _ = fmt.Fprintln(os.Stderr, identity.ErrNoIdentitySet.Error())
		os.Exit(-1)
	}

	return nil
}
