package commands

import (
	"flag"
	"fmt"
	"github.com/MichaelMure/git-bug/repository"
)

var commandsFlagSet = flag.NewFlagSet("commands", flag.ExitOnError)

var (
	commandsDesc = commandsFlagSet.Bool("pretty", false, "Output the command description as well as Markdown compatible comment")
)

func runCommands(repo repository.Repo, args []string) error {
	commandsFlagSet.Parse(args)
	args = commandsFlagSet.Args()

	first := true

	for name, cmd := range CommandMap {
		if !first {
			fmt.Println()
		}

		first = false

		if *commandsDesc {
			fmt.Printf("# %s\n", cmd.Description)
		}

		// TODO: the root name command ("git bug") should be passed from git-bug.go but well ...
		fmt.Printf("%s %s %s\n", "git bug", name, cmd.Usage)
	}

	return nil
}

var commandsCmd = &Command{
	Description: "Display available commands",
	Usage:       "[<option>...]",
	flagSet:     commandsFlagSet,
	RunMethod:   runCommands,
}
