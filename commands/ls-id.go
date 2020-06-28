package commands

import (
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/util/interrupt"
)

func newLsIdCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:     "ls-id [<prefix>]",
		Short:   "List bug identifiers.",
		PreRunE: loadRepo(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLsId(env, args)
		},
	}

	return cmd
}

func runLsId(env *Env, args []string) error {
	backend, err := cache.NewRepoCache(env.repo)
	if err != nil {
		return err
	}
	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	var prefix = ""
	if len(args) != 0 {
		prefix = args[0]
	}

	for _, id := range backend.AllBugsIds() {
		if prefix == "" || id.HasPrefix(prefix) {
			env.out.Println(id)
		}
	}

	return nil
}
