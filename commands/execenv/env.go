package execenv

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/entities/identity"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/interrupt"
)

const RootCommandName = "git-bug"

const gitBugNamespace = "git-bug"

// Env is the environment of a command
type Env struct {
	Repo    repository.ClockedRepo
	Backend *cache.RepoCache
	Out     Out
	Err     Out
}

func NewEnv() *Env {
	return &Env{
		Repo: nil,
		Out:  out{Writer: os.Stdout},
		Err:  out{Writer: os.Stderr},
	}
}

type Out interface {
	io.Writer
	Printf(format string, a ...interface{})
	Print(a ...interface{})
	Println(a ...interface{})

	// String returns what have been written in the output before, as a string.
	// This only works in test scenario.
	String() string
	// Bytes returns what have been written in the output before, as []byte.
	// This only works in test scenario.
	Bytes() []byte
	// Reset clear what has been recorded as written in the output before.
	// This only works in test scenario.
	Reset()
}

type out struct {
	io.Writer
}

func (o out) Printf(format string, a ...interface{}) {
	_, _ = fmt.Fprintf(o, format, a...)
}

func (o out) Print(a ...interface{}) {
	_, _ = fmt.Fprint(o, a...)
}

func (o out) Println(a ...interface{}) {
	_, _ = fmt.Fprintln(o, a...)
}

func (o out) String() string {
	panic("only work with a test env")
}

func (o out) Bytes() []byte {
	panic("only work with a test env")
}

func (o out) Reset() {
	panic("only work with a test env")
}

// LoadRepo is a pre-run function that load the repository for use in a command
func LoadRepo(env *Env) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("unable to get the current working directory: %q", err)
		}

		// Note: we are not loading clocks here because we assume that LoadRepo is only used
		//  when we don't manipulate entities, or as a child call of LoadBackend which will
		//  read all clocks anyway.
		env.Repo, err = repository.OpenGoGitRepo(cwd, gitBugNamespace, nil)
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

		for event := range events {
			if event.Err != nil {
				return event.Err
			}
			switch event.Event {
			case cache.BuildEventCacheIsBuilt:
				env.Err.Println("Building cache... ")
			case cache.BuildEventStarted:
				env.Err.Printf("[%s] started\n", event.Typename)
			case cache.BuildEventFinished:
				env.Err.Printf("[%s] done\n", event.Typename)
			}
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
