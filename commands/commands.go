package commands

import (
	"flag"
	"fmt"
	"github.com/MichaelMure/git-bug/repository"
	"sort"
)

var commandsFlagSet = flag.NewFlagSet("commands", flag.ExitOnError)

var (
	commandsDesc = commandsFlagSet.Bool("pretty", false, "Output the command description as well as Markdown compatible comment")
)

func runCommands(repo repository.Repo, args []string) error {
	commandsFlagSet.Parse(args)
	args = commandsFlagSet.Args()

	first := true

	keys := make([]string, 0, len(CommandMap))

	for key := range CommandMap {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	for _, key := range keys {
		if !first {
			fmt.Println()
		}

		first = false

		cmd := CommandMap[key]

		if *commandsDesc {
			fmt.Printf("# %s\n", cmd.Description)
		}

		// TODO: the root name command ("git bug") should be passed from git-bug.go but well ...
		fmt.Printf("%s %s %s\n", "git bug", key, cmd.Usage)
	}

	return nil
}

var commandsCmd = &Command{
	Description: "Display available commands",
	Usage:       "[<option>...]",
	flagSet:     commandsFlagSet,
	RunMethod:   runCommands,
}
