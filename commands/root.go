// Package commands contains the CLI commands
package commands

import (
	"errors"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/spf13/cobra"

	bridgecmd "github.com/MichaelMure/git-bug/commands/bridge"
	bugcmd "github.com/MichaelMure/git-bug/commands/bug"
	"github.com/MichaelMure/git-bug/commands/execenv"
	usercmd "github.com/MichaelMure/git-bug/commands/user"
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
			root := cmd.Root()

			if GitExactTag == "undefined" {
				GitExactTag = ""
			}

			// release version
			root.Version = GitLastTag

			// dev version
			if GitExactTag == "" {
				if root.Version != "" {
					// if we have a tag, append the commit hash
					root.Version = fmt.Sprintf("%s-dev-%.10s", root.Version, GitCommit)
				} else {
					// if we don't have a tag, try to read
					// commit and dirty state from the build info
					if commit, dirty, err := getCommitAndDirty(); err == nil {
						root.Version = fmt.Sprintf("dev-%.10s", commit)

						if dirty != "" {
							root.Version = fmt.Sprintf("%s-dirty", root.Version)
						}
					} else {
						root.Version = "dev-unknown"
					}
				}
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

func getCommitAndDirty() (commit, dirty string, err error) {
	var d, c string

	info, ok := debug.ReadBuildInfo()

	if !ok {
		return d, c, errors.New("unable to read build info")
	}

	// get the commit and modified status
	// (that is the flag for repository dirty or not)
	for _, kv := range info.Settings {
		switch kv.Key {
		case "vcs.revision":
			c = kv.Value
		case "vcs.modified":
			if kv.Value == "true" {
				d = "dirty"
			}
		}
	}

	return c, d, nil
}
