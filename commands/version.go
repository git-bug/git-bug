package commands

import (
	"github.com/spf13/cobra"

	"github.com/git-bug/git-bug/commands/execenv"
)

func newVersionCommand(env *execenv.Env) *cobra.Command {
	return &cobra.Command{
		Use:     "version",
		Short:   "Print version information",
		Example: "git bug version",
		Long: `
Print version information.

Format:
  git-bug <version> [commit[/dirty]] <compiler version> <platform> <arch>

Format Description:
  <version> may be one of:
  	- A semantic version string, prefixed with a "v", e.g. v1.2.3
  	- "undefined" (if not provided, or built with an invalid version string)

  [commit], if present, is the commit hash that was checked out during the
  build. This may be suffixed with '/dirty' if there were local file
  modifications. This is indicative of your build being patched, or modified in
  some way from the commit.

  <compiler version> is the version of the go compiler used for the build.

  <platform> is the target platform (GOOS).

  <arch> is the target architecture (GOARCH).
`,
		Run: func(cmd *cobra.Command, args []string) {
			env.Out.Printf("%s %s", execenv.RootCommandName, cmd.Root().Version)
		},
	}
}
