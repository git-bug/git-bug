package commands

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/araddon/dateparse"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/bridge"
	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/util/interrupt"
)

var (
	bridgePullImportSince string
	bridgePullNoResume    bool
)

func runBridgePull(cmd *cobra.Command, args []string) error {
	if bridgePullNoResume && bridgePullImportSince != "" {
		return fmt.Errorf("only one of --no-resume and --since flags should be used")
	}

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

	var events <-chan core.ImportResult
	switch {
	case bridgePullNoResume:
		events, err = b.ImportAllSince(ctx, time.Time{})
	case bridgePullImportSince != "":
		since, err2 := parseSince(bridgePullImportSince)
		if err2 != nil {
			return errors.Wrap(err2, "import time parsing")
		}
		events, err = b.ImportAllSince(ctx, since)
	default:
		events, err = b.ImportAll(ctx)
	}

	if err != nil {
		return err
	}

	importedIssues := 0
	importedIdentities := 0
	for result := range events {
		if result.Event != core.ImportEventNothing {
			fmt.Println(result.String())
		}

		switch result.Event {
		case core.ImportEventBug:
			importedIssues++
		case core.ImportEventIdentity:
			importedIdentities++
		}
	}

	fmt.Printf("imported %d issues and %d identities with %s bridge\n", importedIssues, importedIdentities, b.Name)

	// send done signal
	close(done)

	return nil
}

func parseSince(since string) (time.Time, error) {
	duration, err := time.ParseDuration(since)
	if err == nil {
		return time.Now().Add(-duration), nil
	}

	return dateparse.ParseLocal(since)
}

var bridgePullCmd = &cobra.Command{
	Use:     "pull [<name>]",
	Short:   "Pull updates.",
	PreRunE: loadRepoEnsureUser,
	RunE:    runBridgePull,
	Args:    cobra.MaximumNArgs(1),
}

func init() {
	bridgeCmd.AddCommand(bridgePullCmd)
	bridgePullCmd.Flags().BoolVarP(&bridgePullNoResume, "no-resume", "n", false, "force importing all bugs")
	bridgePullCmd.Flags().StringVarP(&bridgePullImportSince, "since", "s", "", "import only bugs updated after the given date (ex: \"200h\" or \"june 2 2019\")")
}
