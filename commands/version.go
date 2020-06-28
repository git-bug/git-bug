package commands

import (
	"runtime"

	"github.com/spf13/cobra"
)

type versionOptions struct {
	number bool
	commit bool
	all    bool
}

func newVersionCommand() *cobra.Command {
	env := newEnv()
	options := versionOptions{}

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show git-bug version information.",
		Run: func(cmd *cobra.Command, args []string) {
			runVersion(env, options, cmd.Root())
		},
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	flags.BoolVarP(&options.number, "number", "n", false,
		"Only show the version number",
	)
	flags.BoolVarP(&options.commit, "commit", "c", false,
		"Only show the commit hash",
	)
	flags.BoolVarP(&options.all, "all", "a", false,
		"Show all version information",
	)

	return cmd
}

func runVersion(env *Env, opts versionOptions, root *cobra.Command) {
	if opts.all {
		env.out.Printf("%s version: %s\n", rootCommandName, root.Version)
		env.out.Printf("System version: %s/%s\n", runtime.GOARCH, runtime.GOOS)
		env.out.Printf("Golang version: %s\n", runtime.Version())
		return
	}

	if opts.number {
		env.out.Println(root.Version)
		return
	}

	if opts.commit {
		env.out.Println(GitCommit)
		return
	}

	env.out.Printf("%s version: %s\n", rootCommandName, root.Version)
}
