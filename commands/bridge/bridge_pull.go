package bridgecmd

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/araddon/dateparse"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/git-bug/git-bug/bridge"
	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/commands/completion"
	"github.com/git-bug/git-bug/commands/execenv"
	"github.com/git-bug/git-bug/util/interrupt"
)

type bridgePullOptions struct {
	importSince string
	noResume    bool
}

func newBridgePullCommand(env *execenv.Env) *cobra.Command {
	options := bridgePullOptions{}

	cmd := &cobra.Command{
		Use:     "pull [NAME]",
		Short:   "Pull updates from a remote bug tracker",
		PreRunE: execenv.LoadBackend(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBridgePull(env, options, args)
		}),
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completion.Bridge(env),
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	flags.BoolVarP(&options.noResume, "no-resume", "n", false, "force importing all bugs")
	flags.StringVarP(&options.importSince, "since", "s", "", "import only bugs updated after the given date (ex: \"200h\" or \"june 2 2019\")")

	return cmd
}

func runBridgePull(env *execenv.Env, opts bridgePullOptions, args []string) error {
	if opts.noResume && opts.importSince != "" {
		return fmt.Errorf("only one of --no-resume and --since flags should be used")
	}

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

	// buffered channel to avoid send block at the end
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

	var events <-chan core.ImportResult
	switch {
	case opts.noResume:
		events, err = b.ImportAllSince(ctx, time.Time{})
	case opts.importSince != "":
		since, err2 := parseSince(opts.importSince)
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

	reportErr := reportImportResults(env.Out, b.Name, events)

	// send done signal
	close(done)

	return reportErr
}

// reportImportResults prints the outcome of a pull and returns an error if
// any import failed. Cancellation does not count as a failure.
func reportImportResults(out execenv.Out, name string, events <-chan core.ImportResult) error {
	importedIssues := 0
	importedIdentities := 0
	importErrors := 0
	for result := range events {
		switch result.Event {
		case core.ImportEventNothing:
			// filtered

		case core.ImportEventBug:
			importedIssues++
			out.Println(result.String())

		case core.ImportEventIdentity:
			importedIdentities++
			out.Println(result.String())

		case core.ImportEventError:
			if result.Err != context.Canceled {
				importErrors++
				out.Println(result.String())
			}

		default:
			out.Println(result.String())
		}
	}

	out.Printf("imported %d issues and %d identities with %s bridge\n", importedIssues, importedIdentities, name)

	if importErrors > 0 {
		return fmt.Errorf("%d import error(s) with %s bridge", importErrors, name)
	}
	return nil
}

func parseSince(since string) (time.Time, error) {
	duration, err := time.ParseDuration(since)
	if err == nil {
		return time.Now().Add(-duration), nil
	}

	return dateparse.ParseLocal(since)
}
