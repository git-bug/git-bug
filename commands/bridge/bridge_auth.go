package bridgecmd

import (
	"sort"
	"strings"

	text "github.com/MichaelMure/go-term-text"
	"github.com/spf13/cobra"

	"github.com/git-bug/git-bug/bridge/core/auth"
	"github.com/git-bug/git-bug/commands/execenv"
	"github.com/git-bug/git-bug/util/colors"
)

func newBridgeAuthCommand(env *execenv.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "auth",
		Short:   "List all known bridge authentication credentials",
		PreRunE: execenv.LoadBackend(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBridgeAuth(env)
		}),
		Args: cobra.NoArgs,
	}

	cmd.AddCommand(newBridgeAuthAddTokenCommand(env))
	cmd.AddCommand(newBridgeAuthRm(env))
	cmd.AddCommand(newBridgeAuthShow(env))

	return cmd
}

func runBridgeAuth(env *execenv.Env) error {
	creds, err := auth.List(env.Backend)
	if err != nil {
		return err
	}

	for _, cred := range creds {
		targetFmt := text.LeftPadMaxLine(cred.Target(), 10, 0)

		var value string
		switch cred := cred.(type) {
		case *auth.Token:
			value = cred.Value
		}

		meta := make([]string, 0, len(cred.Metadata()))
		for k, v := range cred.Metadata() {
			meta = append(meta, k+":"+v)
		}
		sort.Strings(meta)
		metaFmt := strings.Join(meta, ",")

		env.Out.Printf("%s %s %s %s %s\n",
			colors.Cyan(cred.ID().Human()),
			colors.Yellow(targetFmt),
			colors.Magenta(cred.Kind()),
			value,
			metaFmt,
		)
	}

	return nil
}
