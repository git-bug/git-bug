package commands

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/bridge"
	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/util/interrupt"
)

func newBridgePushCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:     "push [<name>]",
		Short:   "Push updates.",
		PreRunE: loadRepoEnsureUser(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBridgePush(env, args)
		},
		Args: cobra.MaximumNArgs(1),
	}

	return cmd
}

func runBridgePush(env *Env, args []string) error {
	backend, err := cache.NewRepoCache(env.repo)
	if err != nil {
		return err
	}
	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	var b *core.Bridge

	if len(args) == 0 {
		b, err = bridge.DefaultBridge(backend)
	} else {
		b, err = bridge.LoadBridge(backend, args[0])
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
			env.err.Println("Received another interrupt before graceful stop, terminating...")
			os.Exit(0)
		}

		interruptCount++
		mu.Unlock()

		env.err.Println("Received interrupt signal, stopping the import...\n(Hit ctrl-c again to kill the process.)")

		// send signal to stop the importer
		cancel()

		// block until importer gracefully shutdown
		<-done
		return nil
	})

	events, err := b.ExportAll(ctx, time.Time{})
	if err != nil {
		return err
	}

	exportedIssues := 0
	for result := range events {
		if result.Event != core.ExportEventNothing {
			env.out.Println(result.String())
		}

		switch result.Event {
		case core.ExportEventBug:
			exportedIssues++
		}
	}

	env.out.Printf("exported %d issues with %s bridge\n", exportedIssues, b.Name)

	// send done signal
	close(done)
	return nil
}
