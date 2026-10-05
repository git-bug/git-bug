package bridgecmd

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/git-bug/git-bug/bridge"
	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/commands/completion"
	"github.com/git-bug/git-bug/commands/execenv"
	"github.com/git-bug/git-bug/util/interrupt"
)

type bridgePushOptions struct {
	foreign bool
}

func newBridgePushCommand(env *execenv.Env) *cobra.Command {
	options := bridgePushOptions{}

	cmd := &cobra.Command{
		Use:   "push [NAME]",
		Short: "Push updates to remote bug tracker",
		Long: `Push updates to remote bug tracker.

By default, bugs imported from another bug tracker are not pushed. With
--foreign they are mirrored too, which makes the same bug available in
several trackers. The first foreign push needs a remote tracker without any
issue or pull request; later ones need a tracker holding only mirrored
issues. Mirrored content appears as written by the token owner, with a note
naming the original author.`,
		PreRunE: execenv.LoadBackendEnsureUser(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBridgePush(env, options, args)
		}),
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completion.Bridge(env),
	}

	flags := cmd.Flags()
	flags.SortFlags = false
	flags.BoolVar(&options.foreign, "foreign", false,
		"Also push bugs imported from other bug trackers (needs an empty or mirror-only remote tracker)")

	return cmd
}

func runBridgePush(env *execenv.Env, opts bridgePushOptions, args []string) error {
	var b *core.Bridge
	var err error

	if len(args) == 0 {
		b, err = bridge.DefaultBridge(env.Backend)
	} else {
		b, err = bridge.LoadBridge(env.Backend, args[0])
	}

	if err != nil {
		return err
	}

	parentCtx := context.Background()
	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()

	done := make(chan struct{}, 1)

	var mu sync.Mutex
	interruptCount := 0
	interrupt.RegisterCleaner(func() error {
		mu.Lock()
		if interruptCount > 0 {
			env.Err.Println("Received another interrupt before graceful stop, terminating...")
			os.Exit(0)
		}

		interruptCount++
		mu.Unlock()

		env.Err.Println("Received interrupt signal, stopping the import...\n(Hit ctrl-c again to kill the process.)")

		// send signal to stop the importer
		cancel()

		// block until importer gracefully shutdown
		<-done
		return nil
	})

	events, err := b.ExportAllWithOptions(ctx, time.Time{}, core.ExportOptions{Foreign: opts.foreign})
	if err != nil {
		return err
	}

	exportedIssues := 0
	for result := range events {
		if result.Event != core.ExportEventNothing {
			env.Out.Println(result.String())
		}

		switch result.Event {
		case core.ExportEventBug:
			exportedIssues++
		}
	}

	env.Out.Printf("exported %d issues with %s bridge\n", exportedIssues, b.Name)

	// send done signal
	close(done)
	return nil
}
