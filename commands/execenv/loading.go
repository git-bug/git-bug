package execenv

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"

	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/repository"
)

// LoadRepo is a pre-run function that load the repository for use in a command
func LoadRepo(env *Env) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("unable to get the current working directory: %q", err)
		}

		env.Repo, err = repository.OpenGoGitRepo(cwd, gitBugNamespace)
		if errors.Is(err, repository.ErrNotARepo) {
			return fmt.Errorf("%s must be run from within a git Repo", RootCommandName)
		}
		if err != nil {
			return err
		}

		return nil
	}
}

// LoadOption configures LoadBackend.
type LoadOption func(*loadOptions)

type loadOptions struct {
	ensureUser    bool
	followChanges bool
	noProgressBar bool
}

// EnsureUser has LoadBackend also ensure that the user has configured an
// identity. Use it when an error after using the configured user won't do.
func EnsureUser() LoadOption {
	return func(o *loadOptions) { o.ensureUser = true }
}

// FollowChanges has the Backend follow the changes made to the repository
// outside of it, as configured, see changeSource. Use it for a command that
// keeps running, such as an interactive UI.
func FollowChanges() LoadOption {
	return func(o *loadOptions) { o.followChanges = true }
}

// NoProgressBar has LoadBackend build the cache silently, rather than showing
// its progress. Use it where nothing should be printed, such as shell
// completion.
func NoProgressBar() LoadOption {
	return func(o *loadOptions) { o.noProgressBar = true }
}

// LoadBackend is a pre-run function that load the repository and the Backend for use in a command
// When using this function you also need to use CloseBackend as a post-run
func LoadBackend(env *Env, opts ...LoadOption) func(*cobra.Command, []string) error {
	var o loadOptions
	for _, opt := range opts {
		opt(&o)
	}

	return func(cmd *cobra.Command, args []string) error {
		err := LoadRepo(env)(cmd, args)
		if err != nil {
			return err
		}

		var source repository.ChangeSource
		if o.followChanges {
			source, err = changeSource(env)
			if err != nil {
				return err
			}
		}

		var events chan cache.BuildEvent
		env.Backend, events = cache.NewRepoCache(env.Repo, source)

		// wait for the build, showing its progress unless asked not to
		var progress *mpb.Progress
		bars := make(map[string]*mpb.Bar)
	build:
		for {
			select {
			case <-env.Ctx.Done():
				// TODO: stop the build itself, rather than only waiting for it
				if progress != nil {
					progress.Shutdown()
				}
				return env.Ctx.Err()
			case event, ok := <-events:
				if !ok {
					break build
				}
				if event.Err != nil {
					return event.Err
				}
				if o.noProgressBar {
					continue
				}

				if progress == nil {
					progress = mpb.New(mpb.WithOutput(env.Err.Raw()))
				}

				switch event.Event {
				case cache.BuildEventCacheIsBuilt:
					env.Err.Println("Building cache... ")
				case cache.BuildEventStarted:
					bars[event.Typename] = progress.AddBar(-1,
						mpb.BarRemoveOnComplete(),
						mpb.PrependDecorators(
							decor.Name(event.Typename, decor.WCSyncSpace),
							decor.CountersNoUnit("%d / %d", decor.WCSyncSpace),
						),
						mpb.AppendDecorators(decor.Percentage(decor.WCSyncSpace)),
					)
				case cache.BuildEventProgress:
					bars[event.Typename].SetCurrent(event.Progress)
					bars[event.Typename].SetTotal(event.Total, event.Progress == event.Total)
				case cache.BuildEventFinished:
					if bar := bars[event.Typename]; !bar.Completed() {
						bar.SetTotal(0, true)
					}
				case cache.BuildEventWarning:
					env.Err.Printf("Warning: %v\n", event.Warning)
				}
			}
		}
		if progress != nil {
			progress.Wait()
		}

		if o.ensureUser {
			return ensureUser(env.Repo)
		}
		return nil
	}
}

// changesNotifierConfigKey selects how a long-running command learns of the
// changes made to the repository outside of it, see changeSource.
const changesNotifierConfigKey = "git-bug.changes.notifier"

// changesPollPeriod is how often the refs are checked when polling.
const changesPollPeriod = 5 * time.Second

// changesBackstopPeriod is how often every ref is compared with the cache,
// with any value but none: it bounds how long a change the notifier missed
// stays unnoticed.
const changesBackstopPeriod = time.Minute

// changeSource returns the source a long-running command follows the changes
// made outside with, as configured in git-bug.changes.notifier:
//   - auto (the default), watch or poll: that notifier, backed by a periodic
//     comparison of every ref, the one source that can't miss a change,
//   - periodic: that periodic comparison alone,
//   - none: no source at all, nil, for a repository only written through this
//     process.
func changeSource(env *Env) (repository.ChangeSource, error) {
	notifier, err := repository.GetDefaultString(changesNotifierConfigKey, env.Repo.AnyConfig(), "auto")
	if err != nil {
		return nil, err
	}

	backstop := repository.Periodic(changesBackstopPeriod)

	switch notifier {
	case "none":
		return nil, nil
	case "periodic":
		return backstop, nil
	case "auto", "watch", "poll":
	default:
		return nil, fmt.Errorf("invalid %s %q: expected auto, watch, poll, periodic or none", changesNotifierConfigKey, notifier)
	}

	// the notifiers need the git directory
	repo, ok := env.Repo.(*repository.GoGitRepo)
	if !ok {
		return backstop, nil
	}

	switch notifier {
	case "auto":
		return repository.Merge(repository.NewAutoSource(repo, changesPollPeriod), backstop), nil
	case "watch":
		return repository.Merge(repository.NewWatchSource(repo), backstop), nil
	default: // poll
		return repository.Merge(repository.NewPollSource(repo, changesPollPeriod), backstop), nil
	}
}

// CloseBackend is a wrapper for a RunE function that will close the Backend properly
// if it has been opened.
// This wrapper style is necessary because a Cobra PostE function does not run if RunE return an error.
func CloseBackend(env *Env, runE func(cmd *cobra.Command, args []string) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		errRun := runE(cmd, args)

		if env.Backend == nil {
			return nil
		}
		err := env.Backend.Close()
		env.Backend = nil

		// prioritize the RunE error
		if errRun != nil {
			return errRun
		}
		return err
	}
}

// ensureUser checks that a valid user identity is configured for the repo.
func ensureUser(repo repository.Repo) error {
	_, err := identity.GetUserIdentity(repo)
	if entity.IsErrNotFound(err) {
		// GetUserIdentity already removed the dangling configuration
		return fmt.Errorf("the configured user identity doesn't exist in this repository and has been unset; " +
			"select an existing one with \"git bug user adopt\" or create one with \"git bug user new\"")
	}
	return err
}
