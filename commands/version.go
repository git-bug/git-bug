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
		Run: func(cmd *cobra.Command, _ []string) {
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
		env.err.Printf("%s version: ", rootCommandName)
		env.out.Printf("%s\n", root.Version)
		env.err.Print("System version: ")
		env.out.Printf("%s/%s\n", runtime.GOARCH, runtime.GOOS)
		env.err.Print("Golang version: ")
		env.out.Printf("%s\n", runtime.Version())
		return
	}

	if opts.number {
		env.out.Print(root.Version)
		env.err.Println()
		return
	}

	if opts.commit {
		env.out.Print(GitCommit)
		env.err.Println()
		return
	}

	printVersion(env, root)
}

func printVersion(env *Env, root *cobra.Command) {
	env.err.Printf("%s version: ", rootCommandName)
	env.out.Printf("%s", root.Version)
	env.err.Println()
}
