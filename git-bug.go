//go:generate go run pack_webui.go

package main

import (
	"fmt"
	"github.com/MichaelMure/git-bug/commands"
	"github.com/MichaelMure/git-bug/repository"
	"os"
	"sort"
	"strings"
)

const rootCommandName = "git bug"
const usageMessageTemplate = `Usage: %s <command>

Where <command> is one of:
  %s

For individual command usage, run:
  %s help <command>
`

func rootUsage() {
	var subcommands []string
	for subcommand := range commands.CommandMap {
		subcommands = append(subcommands, subcommand)
	}
	sort.Strings(subcommands)
	fmt.Printf(usageMessageTemplate, rootCommandName, strings.Join(subcommands, "\n  "), rootCommandName)
}

func help(command string) {
	subcommand, ok := commands.CommandMap[command]
	if !ok {
		fmt.Printf("Unknown command %q\n", command)
		rootUsage()
		return
	}
	subcommand.PrintUsage(rootCommandName, command)
}

func main() {
	args := os.Args

	// git bug
	if len(args) == 1 {
		fmt.Println("Will list bugs, not implemented yet")
		//TODO: list bugs
		return
	}

	if args[1] == "help" {
		if len(args) == 2 {
			// git bug help
			rootUsage()
		} else {
			// git bug help <command>
			help(args[2])
		}
		return
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Unable to get the current working directory: %q\n", err)
		return
	}
	repo, err := repository.NewGitRepo(cwd)
	if err != nil {
		fmt.Printf("%s must be run from within a git repo.\n", rootCommandName)
		return
	}

	subcommand, ok := commands.CommandMap[args[1]]
	if !ok {
		fmt.Printf("Unknown command: %q\n", args[1])
		rootUsage()
		return
	}
	if err := subcommand.Run(repo, args[2:]); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
