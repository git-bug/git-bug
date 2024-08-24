package bridgecmd

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/git-bug/git-bug/bridge/core/auth"
	"github.com/git-bug/git-bug/commands/completion"
	"github.com/git-bug/git-bug/commands/execenv"
)

func newBridgeAuthShow(env *execenv.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "show",
		Short:   "Display an authentication credential",
		PreRunE: execenv.LoadBackend(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBridgeAuthShow(env, args)
		}),
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completion.BridgeAuth(env),
	}

	return cmd
}

func runBridgeAuthShow(env *execenv.Env, args []string) error {
	cred, err := auth.LoadWithPrefix(env.Repo, args[0])
	if err != nil {
		return err
	}

	env.Out.Printf("Id: %s\n", cred.ID())
	env.Out.Printf("Target: %s\n", cred.Target())
	env.Out.Printf("Kind: %s\n", cred.Kind())
	env.Out.Printf("Creation: %s\n", cred.CreateTime().Format(time.RFC822))

	switch cred := cred.(type) {
	case *auth.Token:
		env.Out.Printf("Value: %s\n", cred.Value)
	}

	env.Out.Println("Metadata:")

	meta := make([]string, 0, len(cred.Metadata()))
	for key, value := range cred.Metadata() {
		meta = append(meta, fmt.Sprintf("    %s --> %s\n", key, value))
	}
	sort.Strings(meta)

	env.Out.Print(strings.Join(meta, ""))

	return nil
}
