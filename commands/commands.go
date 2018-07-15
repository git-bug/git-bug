// Package commands contains the assorted sub commands supported by the git-bug tool.
package commands

import (
	"github.com/MichaelMure/git-bug/repository"
)

const messageFilename = "BUG_MESSAGE_EDITMSG"

// Command represents the definition of a single command.
type Command struct {
	Usage     func(string)
	RunMethod func(repository.Repo, []string) error
}

// Run executes a command, given its arguments.
//
// The args parameter is all of the command line args that followed the
// subcommand.
func (cmd *Command) Run(repo repository.Repo, args []string) error {
	return cmd.RunMethod(repo, args)
}

// CommandMap defines all of the available (sub)commands.
var CommandMap = map[string]*Command{
	"comment": commentCmd,
	"ls":      lsCmd,
	"new":     newCmd,
	"pull":    pullCmd,
	"push":    pushCmd,
}
