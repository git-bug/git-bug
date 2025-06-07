package execenv

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"

	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/repository"
	"github.com/git-bug/git-bug/util/interrupt"
)

// LoadRepo is a pre-run function that load the repository for use in a command
func LoadRepo(env *Env) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		repoPath, err := getRepoPath(env)
		if err != nil {
			return err
		}

		// Note: we are not loading clocks here because we assume that LoadRepo is only used
		//  when we don't manipulate entities, or as a child call of LoadBackend which will
		//  read all clocks anyway.
		env.Repo, err = repository.OpenGoGitRepo(repoPath, gitBugNamespace, nil)
		if err == repository.ErrNotARepo {
			return fmt.Errorf("%s must be run from within a git Repo", RootCommandName)
		}
		if err != nil {
			return err
		}

		return nil
	}
}

// LoadRepoEnsureUser is the same as LoadRepo, but also ensure that the user has configured
// an identity. Use this pre-run function when an error after using the configured user won't
// do.
func LoadRepoEnsureUser(env *Env) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		err := LoadRepo(env)(cmd, args)
		if err != nil {
			return err
		}

		_, err = identity.GetUserIdentity(env.Repo)
		if err != nil {
			return err
		}

		return nil
	}
}

// LoadBackend is a pre-run function that load the repository and the Backend for use in a command
// When using this function you also need to use CloseBackend as a post-run
func LoadBackend(env *Env) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		err := LoadRepo(env)(cmd, args)
		if err != nil {
			return err
		}

		var events chan cache.BuildEvent
		env.Backend, events = cache.NewRepoCache(env.Repo)

		err = CacheBuildProgressBar(env, events)
		if err != nil {
			return err
		}

		cleaner := func(env *Env) interrupt.CleanerFunc {
			return func() error {
				if env.Backend != nil {
					err := env.Backend.Close()
					env.Backend = nil
					return err
				}
				return nil
			}
		}

		// Cleanup properly on interrupt
		interrupt.RegisterCleaner(cleaner(env))
		return nil
	}
}

// LoadBackendEnsureUser is the same as LoadBackend, but also ensure that the user has configured
// an identity. Use this pre-run function when an error after using the configured user won't
// do.
func LoadBackendEnsureUser(env *Env) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		err := LoadBackend(env)(cmd, args)
		if err != nil {
			return err
		}

		_, err = identity.GetUserIdentity(env.Repo)
		if err != nil {
			return err
		}

		return nil
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

func CacheBuildProgressBar(env *Env, events chan cache.BuildEvent) error {
	var progress *mpb.Progress
	var bars = make(map[string]*mpb.Bar)

	for event := range events {
		if event.Err != nil {
			return event.Err
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
		}
	}

	if progress != nil {
		progress.Wait()
	}

	return nil
}

func getRepoPath(env *Env) (string, error) {
	if len(env.RepoPath) > 0 {
		return filepath.Join(env.RepoPath...), nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("unable to get the current working directory: %q", err)
	}
	return cwd, nil
}
