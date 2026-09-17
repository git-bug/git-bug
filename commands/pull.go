package commands

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/git-bug/git-bug/commands/completion"
	"github.com/git-bug/git-bug/commands/execenv"
	"github.com/git-bug/git-bug/entities/bug"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/repository"
)

func newPullCommand(env *execenv.Env) *cobra.Command {
	var verbose bool

	cmd := &cobra.Command{
		Use:     "pull [REMOTE]",
		Short:   "Pull updates from a git remote",
		PreRunE: execenv.LoadBackend(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runPull(env, args, verbose)
		}),
		ValidArgsFunction: completion.GitRemote(env),
	}

	flags := cmd.Flags()
	flags.BoolVarP(&verbose, "verbose", "v", false, "log each operation to stderr")

	return cmd
}

func runPull(env *execenv.Env, args []string, verbose bool) error {
	var remote string
	switch {
	case len(args) > 1:
		return errors.New("Only pulling from one remote at a time is supported")
	case len(args) == 1:
		remote = args[0]
	default:
		v, err := repository.GetDefaultString("git-bug.remote", env.Repo.AnyConfig(), "origin")
		if err != nil {
			return err
		}
		remote = v
	}

	if verbose {
		env.Err.Println("Fetching remote ...")
	} else {
		env.Out.Println("Fetching remote ...")
	}

	stdout, err := env.Backend.Fetch(remote)
	if err != nil {
		return err
	}

	if verbose {
		env.Err.Println(stdout)
		env.Err.Println("Merging data ...")
	} else {
		env.Out.Println(stdout)
		env.Out.Println("Merging data ...")
	}

	// Track statistics
	newBugs := 0
	updatedBugs := 0
	newIdentities := 0
	updatedIdentities := 0

	for result := range env.Backend.MergeAll(remote) {
		if result.Err != nil {
			if verbose {
				env.Err.Printf("Error: %v\n", result.Err)
			} else {
				env.Err.Println(result.Err)
			}
			continue
		}

		if result.Status != entity.MergeStatusNothing {
			if verbose {
				env.Err.Printf("%s: %s\n", result.Id.Human(), result)
			} else {
				env.Out.Printf("%s: %s\n", result.Id.Human(), result)
			}

			// Count entity changes by checking the entity type
			if result.Entity != nil {
				switch result.Entity.(type) {
				case *bug.Bug:
					if result.Status == entity.MergeStatusNew {
						newBugs++
					} else if result.Status == entity.MergeStatusUpdated {
						updatedBugs++
					}
				case *identity.Identity:
					if result.Status == entity.MergeStatusNew {
						newIdentities++
					} else if result.Status == entity.MergeStatusUpdated {
						updatedIdentities++
					}
				}
			}
		}
	}

	// Print summary
	if !verbose {
		if newBugs > 0 || updatedBugs > 0 || newIdentities > 0 || updatedIdentities > 0 {
			env.Out.Printf("Summary: %d new bugs, %d updated bugs, %d new identities, %d updated identities\n",
				newBugs, updatedBugs, newIdentities, updatedIdentities)
		} else {
			env.Out.Println("No new changes")
		}
	}

	return nil
}
