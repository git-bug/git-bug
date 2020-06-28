package commands

import (
	"sort"
	"strings"

	"github.com/spf13/cobra"

	text "github.com/MichaelMure/go-term-text"

	"github.com/MichaelMure/git-bug/bridge/core/auth"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/util/colors"
	"github.com/MichaelMure/git-bug/util/interrupt"
)

func newBridgeAuthCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:     "auth",
		Short:   "List all known bridge authentication credentials.",
		PreRunE: loadRepo(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBridgeAuth(env)
		},
		Args: cobra.NoArgs,
	}

	cmd.AddCommand(newBridgeAuthAddTokenCommand())
	cmd.AddCommand(newBridgeAuthRm())
	cmd.AddCommand(newBridgeAuthShow())

	return cmd
}

func runBridgeAuth(env *Env) error {
	backend, err := cache.NewRepoCache(env.repo)
	if err != nil {
		return err
	}
	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	creds, err := auth.List(backend)
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

		env.out.Printf("%s %s %s %s %s\n",
			colors.Cyan(cred.ID().Human()),
			colors.Yellow(targetFmt),
			colors.Magenta(cred.Kind()),
			value,
			metaFmt,
		)
	}

	return nil
}
