package commands

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/bridge"
	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/util/interrupt"
)

func runBridgePull(cmd *cobra.Command, args []string) error {
	backend, err := cache.NewRepoCache(repo)
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

	// buffered channel to avoid send block at the end
	done := make(chan struct{}, 1)

	var mu sync.Mutex
	interruptCount := 0
	interrupt.RegisterCleaner(func() error {
		mu.Lock()
		if interruptCount > 0 {
			fmt.Println("Received another interrupt before graceful stop, terminating...")
			os.Exit(0)
		}

		interruptCount++
		mu.Unlock()

		fmt.Println("Received interrupt signal, stopping the import...\n(Hit ctrl-c again to kill the process.)")

		// send signal to stop the importer
		cancel()

		// block until importer gracefully shutdown
		<-done
		return nil
	})

	// TODO: by default import only new events
	events, err := b.ImportAll(ctx, time.Time{})
	if err != nil {
		return err
	}

	importedIssues := 0
	importedIdentities := 0
	for result := range events {
		fmt.Println(result.String())

		switch result.Event {
		case core.ImportEventBug:
			importedIssues++
		case core.ImportEventIdentity:
			importedIdentities++
		}
	}

	// send done signal
	close(done)

	fmt.Printf("Successfully imported %d issues and %d identities with %s bridge\n", importedIssues, importedIdentities, b.Name)

	return nil
}

var bridgePullCmd = &cobra.Command{
	Use:     "pull [<name>]",
	Short:   "Pull updates.",
	PreRunE: loadRepo,
	RunE:    runBridgePull,
	Args:    cobra.MaximumNArgs(1),
}

func init() {
	bridgeCmd.AddCommand(bridgePullCmd)
}
