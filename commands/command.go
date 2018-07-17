// Package commands contains the assorted sub commands supported by the git-bug tool.
package commands

import (
	"flag"
	"fmt"
	"github.com/MichaelMure/git-bug/repository"
)

const messageFilename = "BUG_MESSAGE_EDITMSG"

// Command represents the definition of a single command.
type Command struct {
	// Short description of the command
	Description string
	// Command line usage
	Usage string
	// Flag set of the command
	flagSet *flag.FlagSet
	// Execute the command
	RunMethod func(repository.Repo, []string) error
}

// Run executes a command, given its arguments.
//
// The args parameter is all of the command line args that followed the
// subcommand.
func (cmd *Command) Run(repo repository.Repo, args []string) error {
	return cmd.RunMethod(repo, args)
}

func (cmd *Command) PrintUsage(rootCommand string, cmdName string) {
	fmt.Printf("Usage: %s %s %s\n", rootCommand, cmdName, cmd.Usage)

	if cmd.flagSet != nil {
		fmt.Printf("\nOptions:\n")
		cmd.flagSet.PrintDefaults()
	}
}

// CommandMap defines all of the available (sub)commands.
var CommandMap map[string]*Command

// We use init() to avoid a cycle in the data initialization because of the "commands" command
func init() {
	CommandMap = map[string]*Command{
		"close":    closeCmd,
		"commands": commandsCmd,
		"comment":  commentCmd,
		"ls":       lsCmd,
		"new":      newCmd,
		"open":     openCmd,
		"pull":     pullCmd,
		"push":     pushCmd,
		"webui":    webUICmd,
	}
}
