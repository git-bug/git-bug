package commands

import (
	"os"

	"github.com/spf13/cobra"

	bridgecmd "github.com/git-bug/git-bug/commands/bridge"
	bugcmd "github.com/git-bug/git-bug/commands/bug"
	"github.com/git-bug/git-bug/commands/execenv"
	usercmd "github.com/git-bug/git-bug/commands/user"
)

func NewRootCommand(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   execenv.RootCommandName,
		Short: "A bug tracker embedded in Git",
		Long: `git-bug is a bug tracker embedded in git.

git-bug use git objects to store the bug tracking separated from the files
history. As bugs are regular git objects, they can be pushed and pulled from/to
the same git remote you are already using to collaborate with other people.

By default, git-bug manages bugs and users in the Git repository containing
the current working directory.  This behavior can be changed using the --git-dir
flag described below, or by setting the GIT-DIR environment variable, which
takes precedence over the CLI flag.
`,

		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			root := cmd.Root()
			root.Version = version
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

	env := execenv.NewEnv()
	cmd.PersistentFlags().StringVar(&env.GitDir, "git-dir", "", `run as if git-bug was started in <path>`)

	addCmdWithGroup(bugcmd.NewBugCommand(env), entityGroup)
	addCmdWithGroup(usercmd.NewUserCommand(env), entityGroup)
	addCmdWithGroup(newLabelCommand(env), entityGroup)

	addCmdWithGroup(newTermUICommand(env), uiGroup)
	addCmdWithGroup(newWebUICommand(env), uiGroup)

	addCmdWithGroup(newPullCommand(env), remoteGroup)
	addCmdWithGroup(newPushCommand(env), remoteGroup)
	addCmdWithGroup(bridgecmd.NewBridgeCommand(env), remoteGroup)

	cmd.AddCommand(newVersionCommand(env))
	cmd.AddCommand(newWipeCommand(env))

	return cmd
}
