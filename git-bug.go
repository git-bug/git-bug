package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/commands"
)

const usageMessageTemplate = `Usage: %s <command>

Where <command> is one of:
  %s

For individual command usage, run:
  %s help <command>
`

func usage() {
	command := os.Args[0]
	var subcommands []string
	for subcommand := range commands.CommandMap {
		subcommands = append(subcommands, subcommand)
	}
	sort.Strings(subcommands)
	fmt.Printf(usageMessageTemplate, command, strings.Join(subcommands, "\n  "), command)
}

func help() {
	if len(os.Args) < 3 {
		usage()
		return
	}
	subcommand, ok := commands.CommandMap[os.Args[2]]
	if !ok {
		fmt.Printf("Unknown command %q\n", os.Args[2])
		usage()
		return
	}
	subcommand.Usage(os.Args[0])
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "help" {
		help()
		return
	}
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Unable to get the current working directory: %q\n", err)
		return
	}
	repo, err := repository.NewGitRepo(cwd)
	if err != nil {
		fmt.Printf("%s must be run from within a git repo.\n", os.Args[0])
		return
	}
	if len(os.Args) < 2 {
		// default behavior
		fmt.Println("Not implemented")
		return
	}
	subcommand, ok := commands.CommandMap[os.Args[1]]
	if !ok {
		fmt.Printf("Unknown command: %q\n", os.Args[1])
		usage()
		return
	}
	if err := subcommand.Run(repo, os.Args[2:]); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
