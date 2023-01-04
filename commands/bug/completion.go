package bugcmd

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/commands/completion"
	"github.com/git-bug/git-bug/commands/execenv"
	_select "github.com/git-bug/git-bug/commands/select"
	"github.com/git-bug/git-bug/entities/common"
)

// BugCompletion complete a bug id
func BugCompletion(env *execenv.Env) completion.ValidArgsFunction {
	return func(cmd *cobra.Command, args []string, toComplete string) (completions []string, directives cobra.ShellCompDirective) {
		if err := execenv.LoadBackend(env)(cmd, args); err != nil {
			return completion.HandleError(err)
		}
		defer func() {
			_ = env.Backend.Close()
		}()

		return BugWithBackend(env.Backend, toComplete)
	}
}

func BugWithBackend(backend *cache.RepoCache, toComplete string) (completions []string, directives cobra.ShellCompDirective) {
	for _, id := range backend.Bugs().AllIds() {
		if strings.Contains(id.String(), strings.TrimSpace(toComplete)) {
			excerpt, err := backend.Bugs().ResolveExcerpt(id)
			if err != nil {
				return completion.HandleError(err)
			}
			completions = append(completions, id.Human()+"\t"+excerpt.Title)
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

// BugAndLabelsCompletion complete either a bug ID or a label if we know about the bug
func BugAndLabelsCompletion(env *execenv.Env, addOrRemove bool) completion.ValidArgsFunction {
	return func(cmd *cobra.Command, args []string, toComplete string) (completions []string, directives cobra.ShellCompDirective) {
		if err := execenv.LoadBackend(env)(cmd, args); err != nil {
			return completion.HandleError(err)
		}
		defer func() {
			_ = env.Backend.Close()
		}()

		b, cleanArgs, err := ResolveSelected(env.Backend, args)
		if _select.IsErrNoValidId(err) {
			// we need a bug first to complete labels
			return BugWithBackend(env.Backend, toComplete)
		}
		if err != nil {
			return completion.HandleError(err)
		}

		snap := b.Snapshot()

		seenLabels := map[common.Label]bool{}
		for _, label := range cleanArgs {
			seenLabels[common.Label(label)] = addOrRemove
		}

		var labels []common.Label
		if addOrRemove {
			for _, label := range snap.Labels {
				seenLabels[label] = true
			}

			allLabels := env.Backend.Bugs().ValidLabels()
			labels = make([]common.Label, 0, len(allLabels))
			for _, label := range allLabels {
				if !seenLabels[label] {
					labels = append(labels, label)
				}
			}
		} else {
			labels = make([]common.Label, 0, len(snap.Labels))
			for _, label := range snap.Labels {
				if seenLabels[label] {
					labels = append(labels, label)
				}
			}
		}

		completions = make([]string, len(labels))
		for i, label := range labels {
			completions[i] = string(label) + "\t" + "Label"
		}

		return completions, cobra.ShellCompDirectiveNoFileComp
	}
}
