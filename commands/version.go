package commands

import (
	"log/slog"
	"strings"

	"github.com/spf13/cobra"

	"github.com/git-bug/git-bug/commands/execenv"
)

// TODO: 0.12.0: remove deprecated build vars
var (
	GitCommit   string
	GitLastTag  string
	GitExactTag string
)

func newVersionCommand(env *execenv.Env) *cobra.Command {
	var raw bool

	cmd := &cobra.Command{
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

With --raw, print <version> alone, on a line of its own, with no "git-bug"
prefix and no build metadata. That is the form to compare against another
version or to embed in a script.
`,
		Run: func(cmd *cobra.Command, args []string) {
			defer warnDeprecated()
			if raw {
				// <version> is always the first space-separated field of the
				// full string, so the raw form stays a prefix of the default
				// output rather than a second source of truth.
				env.Out.Printf("%s\n", rawVersion(cmd.Root().Version))
				return
			}
			env.Out.Printf("%s %s", execenv.RootCommandName, cmd.Root().Version)
		},
	}

	cmd.Flags().BoolVar(&raw, "raw", false, "Print only the version string, with no prefix or build metadata")

	return cmd
}

// rawVersion returns <version> on its own, extracted from the same string
// `git-bug version` prints, so the two can never disagree.
func rawVersion(full string) string {
	version, _, _ := strings.Cut(full, " ")
	return version
}

// warnDeprecated warns about deprecated build variables
// TODO: 0.12.0: remove support for old build tags
func warnDeprecated() {
	msg := "please contact your package maintainer"
	reason := "deprecated build variable"
	if GitLastTag != "" {
		slog.Warn(msg, "reason", reason, "name", "GitLastTag", "value", GitLastTag)
	}
	if GitExactTag != "" {
		slog.Warn(msg, "reason", reason, "name", "GitExactTag", "value", GitExactTag)
	}
	if GitCommit != "" {
		slog.Warn(msg, "reason", reason, "name", "GitCommit", "value", GitCommit)
	}
}
