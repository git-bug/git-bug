// Package commands contains the CLI commands
package commands

import (
	"fmt"
	"os"
	"os/exec"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/commands/bridge"
	"github.com/MichaelMure/git-bug/commands/bug"
	"github.com/MichaelMure/git-bug/commands/execenv"
	"github.com/MichaelMure/git-bug/commands/user"
)

// These variables are initialized externally during the build. See the Makefile.
var GitCommit string
var GitLastTag string
var GitExactTag string

func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   execenv.RootCommandName,
		Short: "A bug tracker embedded in Git",
		Long: `git-bug is a bug tracker embedded in git.

git-bug use git objects to store the bug tracking separated from the files
history. As bugs are regular git objects, they can be pushed and pulled from/to
the same git remote you are already using to collaborate with other people.

`,

		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			GitLastTag, GitExactTag, GitCommit = getTagsAndCommit()

			root := cmd.Root()

			if GitExactTag == "undefined" {
				GitExactTag = ""
			}
			root.Version = GitLastTag
			if GitExactTag == "" {
				root.Version = fmt.Sprintf("%s-dev-%.10s", root.Version, GitCommit)
			}
		},

		// For the root command, force the execution of the PreRun
		// even if we just display the help. This is to make sure that we check
		// the repository and give the user early feedback.
		Run: func(cmd *cobra.Command, args []string) {
			if err := cmd.Help(); err != nil {
				os.Exit(1)
			}
		},

		SilenceUsage:      true,
		DisableAutoGenTag: true,
	}

	const entityGroup = "entity"
	const uiGroup = "ui"
	const remoteGroup = "remote"

	cmd.AddGroup(&cobra.Group{ID: entityGroup, Title: "Entities"})
	cmd.AddGroup(&cobra.Group{ID: uiGroup, Title: "Interactive interfaces"})
	cmd.AddGroup(&cobra.Group{ID: remoteGroup, Title: "Interaction with the outside world"})

	addCmdWithGroup := func(child *cobra.Command, groupID string) {
		cmd.AddCommand(child)
		child.GroupID = groupID
	}

	addCmdWithGroup(bugcmd.NewBugCommand(), entityGroup)
	addCmdWithGroup(usercmd.NewUserCommand(), entityGroup)
	addCmdWithGroup(newLabelCommand(), entityGroup)

	addCmdWithGroup(newTermUICommand(), uiGroup)
	addCmdWithGroup(newWebUICommand(), uiGroup)

	addCmdWithGroup(newPullCommand(), remoteGroup)
	addCmdWithGroup(newPushCommand(), remoteGroup)
	addCmdWithGroup(bridgecmd.NewBridgeCommand(), remoteGroup)

	cmd.AddCommand(newCommandsCommand())
	cmd.AddCommand(newVersionCommand())

	return cmd
}

func Execute() {
	if err := NewRootCommand().Execute(); err != nil {
		os.Exit(1)
	}
}

func getTagsAndCommit() (tag, exacttag, commit string) {
	var t, e, c string

	info, ok := debug.ReadBuildInfo()

	if !ok {
		fmt.Println("could not get commit")
	}

	for _, kv := range info.Settings {
		switch kv.Key {
		case "vcs.revision":
			c = kv.Value
		}
	}

	cmd := exec.Command("git", "describe", "--tags", "--abbrev=0")
	stdout, err := cmd.Output()

	if err != nil {
		fmt.Printf("could not get last tag: %v\n", err.Error())
	}

	t = strings.TrimSuffix(string(stdout), "\n")

	cmd = exec.Command("git", "name-rev", "--name-only", "--tags", "HEAD")
	stdout, err = cmd.Output()

	if err != nil {
		fmt.Printf("could not get exact tag: %v\n", err.Error())
	}

	e = strings.TrimSuffix(string(stdout), "\n")

	return t, e, c
}
