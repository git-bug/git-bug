package commands

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// These variables are initialized externally during the build. See the Makefile.
var GitCommit string
var GitLastTag string
var GitExactTag string

var (
	versionNumber bool
	versionCommit bool
	versionAll    bool
)

func runVersionCmd(cmd *cobra.Command, args []string) {
	if versionAll {
		fmt.Printf("%s version: %s\n", rootCommandName, RootCmd.Version)
		fmt.Printf("System version: %s/%s\n", runtime.GOARCH, runtime.GOOS)
		fmt.Printf("Golang version: %s\n", runtime.Version())
		return
	}

	if versionNumber {
		fmt.Println(RootCmd.Version)
		return
	}

	if versionCommit {
		fmt.Println(GitCommit)
		return
	}

	fmt.Printf("%s version: %s\n", rootCommandName, RootCmd.Version)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show git-bug version information.",
	Run:   runVersionCmd,
}

func init() {
	if GitExactTag == "undefined" {
		GitExactTag = ""
	}

	RootCmd.Version = GitLastTag

	if GitExactTag == "" {
		RootCmd.Version = fmt.Sprintf("%s-dev-%.10s", RootCmd.Version, GitCommit)
	}

	RootCmd.AddCommand(versionCmd)
	versionCmd.Flags().SortFlags = false

	versionCmd.Flags().BoolVarP(&versionNumber, "number", "n", false,
		"Only show the version number",
	)
	versionCmd.Flags().BoolVarP(&versionCommit, "commit", "c", false,
		"Only show the commit hash",
	)
	versionCmd.Flags().BoolVarP(&versionAll, "all", "a", false,
		"Show all version information",
	)
}
