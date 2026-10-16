package bridgecmd

import (
	"context"
	"fmt"
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
	verbose bool
}

func newBridgePushCommand(env *execenv.Env) *cobra.Command {
	options := bridgePushOptions{}

	cmd := &cobra.Command{
		Use:     "push [NAME]",
		Short:   "Push updates to remote bug tracker",
		PreRunE: execenv.LoadBackend(env, execenv.EnsureUser()),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBridgePush(env, options, args)
		}),
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completion.Bridge(env),
	}

	flags := cmd.Flags()
	flags.SortFlags = false
	flags.BoolVarP(&options.verbose, "verbose", "v", false,
		"Explain why issues were not pushed")

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

	events, err := b.ExportAll(ctx, time.Time{})
	if err != nil {
		return err
	}

	reportErr := reportExportResults(env.Out, b.Name, events, opts.verbose)

	// send done signal
	close(done)

	return reportErr
}

// reportExportResults prints the outcome of a push and returns an error if
// any issue failed to export. With verbose, it also summarizes why issues
// were skipped.
func reportExportResults(out execenv.Out, name string, events <-chan core.ExportResult, verbose bool) error {
	exportedIssues := 0
	exportErrors := 0
	// Skipped issues are summarized by reason instead of listed one by one.
	skipped := map[string]int{}
	var skipOrder []string
	for result := range events {
		if result.Event == core.ExportEventNothing && result.Reason != core.ReasonNothingExported {
			if skipped[result.Reason] == 0 {
				skipOrder = append(skipOrder, result.Reason)
			}
			skipped[result.Reason]++
		}
		if result.Event != core.ExportEventNothing {
			out.Println(result.String())
		}

		switch result.Event {
		case core.ExportEventBug:
			exportedIssues++
		case core.ExportEventError:
			exportErrors++
		}
	}

	if verbose {
		for _, reason := range skipOrder {
			out.Printf("skipped %d issues: %s\n", skipped[reason], reason)
		}
	}
	out.Printf("exported %d issues with %s bridge\n", exportedIssues, name)

	if exportErrors > 0 {
		return fmt.Errorf("%d export error(s) with %s bridge", exportErrors, name)
	}
	return nil
}
