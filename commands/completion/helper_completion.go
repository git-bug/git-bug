package completion

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/bridge"
	"github.com/MichaelMure/git-bug/bridge/core/auth"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/commands/bug/select"
	"github.com/MichaelMure/git-bug/commands/execenv"
	"github.com/MichaelMure/git-bug/entities/bug"
)

type ValidArgsFunction func(cmd *cobra.Command, args []string, toComplete string) (completions []string, directives cobra.ShellCompDirective)

func handleError(err error) (completions []string, directives cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveError
}

func Bridge(env *execenv.Env) ValidArgsFunction {
	return func(cmd *cobra.Command, args []string, toComplete string) (completions []string, directives cobra.ShellCompDirective) {
		if err := execenv.LoadBackend(env)(cmd, args); err != nil {
			return handleError(err)
		}
		defer func() {
			_ = env.Backend.Close()
		}()

		bridges, err := bridge.ConfiguredBridges(env.Backend)
		if err != nil {
			return handleError(err)
		}

		completions = make([]string, len(bridges))
		for i, bridge := range bridges {
			completions[i] = bridge + "\t" + "Bridge"
		}

		return completions, cobra.ShellCompDirectiveNoFileComp
	}
}

func BridgeAuth(env *execenv.Env) ValidArgsFunction {
	return func(cmd *cobra.Command, args []string, toComplete string) (completions []string, directives cobra.ShellCompDirective) {
		if err := execenv.LoadBackend(env)(cmd, args); err != nil {
			return handleError(err)
		}
		defer func() {
			_ = env.Backend.Close()
		}()

		creds, err := auth.List(env.Backend)
		if err != nil {
			return handleError(err)
		}

		completions = make([]string, len(creds))
		for i, cred := range creds {
			meta := make([]string, 0, len(cred.Metadata()))
			for k, v := range cred.Metadata() {
				meta = append(meta, k+":"+v)
			}
			sort.Strings(meta)
			metaFmt := strings.Join(meta, ",")

			completions[i] = cred.ID().Human() + "\t" + cred.Target() + " " + string(cred.Kind()) + " " + metaFmt
		}

		return completions, cobra.ShellCompDirectiveNoFileComp
	}
}

func Bug(env *execenv.Env) ValidArgsFunction {
	return func(cmd *cobra.Command, args []string, toComplete string) (completions []string, directives cobra.ShellCompDirective) {
		if err := execenv.LoadBackend(env)(cmd, args); err != nil {
			return handleError(err)
		}
		defer func() {
			_ = env.Backend.Close()
		}()

		return bugWithBackend(env.Backend, toComplete)
	}
}

func bugWithBackend(backend *cache.RepoCache, toComplete string) (completions []string, directives cobra.ShellCompDirective) {
	allIds := backend.AllBugsIds()
	bugExcerpt := make([]*cache.BugExcerpt, len(allIds))
	for i, id := range allIds {
		var err error
		bugExcerpt[i], err = backend.ResolveBugExcerpt(id)
		if err != nil {
			return handleError(err)
		}
	}

	for i, id := range allIds {
		if strings.Contains(id.String(), strings.TrimSpace(toComplete)) {
			completions = append(completions, id.Human()+"\t"+bugExcerpt[i].Title)
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

func BugAndLabels(env *execenv.Env, addOrRemove bool) ValidArgsFunction {
	return func(cmd *cobra.Command, args []string, toComplete string) (completions []string, directives cobra.ShellCompDirective) {
		if err := execenv.LoadBackend(env)(cmd, args); err != nil {
			return handleError(err)
		}
		defer func() {
			_ = env.Backend.Close()
		}()

		b, args, err := _select.ResolveBug(env.Backend, args)
		if err == _select.ErrNoValidId {
			// we need a bug first to complete labels
			return bugWithBackend(env.Backend, toComplete)
		}
		if err != nil {
			return handleError(err)
		}

		snap := b.Snapshot()

		seenLabels := map[bug.Label]bool{}
		for _, label := range args {
			seenLabels[bug.Label(label)] = addOrRemove
		}

		var labels []bug.Label
		if addOrRemove {
			for _, label := range snap.Labels {
				seenLabels[label] = true
			}

			allLabels := env.Backend.ValidLabels()
			labels = make([]bug.Label, 0, len(allLabels))
			for _, label := range allLabels {
				if !seenLabels[label] {
					labels = append(labels, label)
				}
			}
		} else {
			labels = make([]bug.Label, 0, len(snap.Labels))
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

func From(choices []string) ValidArgsFunction {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return choices, cobra.ShellCompDirectiveNoFileComp
	}
}

func GitRemote(env *execenv.Env) ValidArgsFunction {
	return func(cmd *cobra.Command, args []string, toComplete string) (completions []string, directives cobra.ShellCompDirective) {
		if err := execenv.LoadBackend(env)(cmd, args); err != nil {
			return handleError(err)
		}
		defer func() {
			_ = env.Backend.Close()
		}()

		remoteMap, err := env.Backend.GetRemotes()
		if err != nil {
			return handleError(err)
		}
		completions = make([]string, 0, len(remoteMap))
		for remote, url := range remoteMap {
			completions = append(completions, remote+"\t"+"Remote: "+url)
		}
		sort.Strings(completions)
		return completions, cobra.ShellCompDirectiveNoFileComp
	}
}

func Label(env *execenv.Env) ValidArgsFunction {
	return func(cmd *cobra.Command, args []string, toComplete string) (completions []string, directives cobra.ShellCompDirective) {
		if err := execenv.LoadBackend(env)(cmd, args); err != nil {
			return handleError(err)
		}
		defer func() {
			_ = env.Backend.Close()
		}()

		labels := env.Backend.ValidLabels()
		completions = make([]string, len(labels))
		for i, label := range labels {
			if strings.Contains(label.String(), " ") {
				completions[i] = fmt.Sprintf("\"%s\"\tLabel", label.String())
			} else {
				completions[i] = fmt.Sprintf("%s\tLabel", label.String())
			}
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	}
}

func Ls(env *execenv.Env) ValidArgsFunction {
	return func(cmd *cobra.Command, args []string, toComplete string) (completions []string, directives cobra.ShellCompDirective) {
		if strings.HasPrefix(toComplete, "status:") {
			completions = append(completions, "status:open\tOpen bugs")
			completions = append(completions, "status:closed\tClosed bugs")
			return completions, cobra.ShellCompDirectiveDefault
		}

		byPerson := []string{"author:", "participant:", "actor:"}
		byLabel := []string{"label:", "no:"}
		needBackend := false
		for _, key := range append(byPerson, byLabel...) {
			if strings.HasPrefix(toComplete, key) {
				needBackend = true
			}
		}

		if needBackend {
			if err := execenv.LoadBackend(env)(cmd, args); err != nil {
				return handleError(err)
			}
			defer func() {
				_ = env.Backend.Close()
			}()
		}

		for _, key := range byPerson {
			if !strings.HasPrefix(toComplete, key) {
				continue
			}
			ids := env.Backend.AllIdentityIds()
			completions = make([]string, len(ids))
			for i, id := range ids {
				user, err := env.Backend.ResolveIdentityExcerpt(id)
				if err != nil {
					return handleError(err)
				}
				var handle string
				if user.Login != "" {
					handle = user.Login
				} else {
					// "author:John Doe" does not work yet, so use the first name.
					handle = strings.Split(user.Name, " ")[0]
				}
				completions[i] = key + handle + "\t" + user.DisplayName()
			}
			return completions, cobra.ShellCompDirectiveNoFileComp
		}

		for _, key := range byLabel {
			if !strings.HasPrefix(toComplete, key) {
				continue
			}
			labels := env.Backend.ValidLabels()
			completions = make([]string, len(labels))
			for i, label := range labels {
				if strings.Contains(label.String(), " ") {
					completions[i] = key + "\"" + string(label) + "\""
				} else {
					completions[i] = key + string(label)
				}
			}
			return completions, cobra.ShellCompDirectiveNoFileComp
		}

		completions = []string{
			"actor:\tFilter by actor",
			"author:\tFilter by author",
			"label:\tFilter by label",
			"no:\tExclude bugs by label",
			"participant:\tFilter by participant",
			"status:\tFilter by open/close status",
			"title:\tFilter by title",
		}
		return completions, cobra.ShellCompDirectiveNoSpace
	}
}

func User(env *execenv.Env) ValidArgsFunction {
	return func(cmd *cobra.Command, args []string, toComplete string) (completions []string, directives cobra.ShellCompDirective) {
		if err := execenv.LoadBackend(env)(cmd, args); err != nil {
			return handleError(err)
		}
		defer func() {
			_ = env.Backend.Close()
		}()

		ids := env.Backend.AllIdentityIds()
		completions = make([]string, len(ids))
		for i, id := range ids {
			user, err := env.Backend.ResolveIdentityExcerpt(id)
			if err != nil {
				return handleError(err)
			}
			completions[i] = user.Id.Human() + "\t" + user.DisplayName()
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	}
}

func UserForQuery(env *execenv.Env) ValidArgsFunction {
	return func(cmd *cobra.Command, args []string, toComplete string) (completions []string, directives cobra.ShellCompDirective) {
		if err := execenv.LoadBackend(env)(cmd, args); err != nil {
			return handleError(err)
		}
		defer func() {
			_ = env.Backend.Close()
		}()

		ids := env.Backend.AllIdentityIds()
		completions = make([]string, len(ids))
		for i, id := range ids {
			user, err := env.Backend.ResolveIdentityExcerpt(id)
			if err != nil {
				return handleError(err)
			}
			var handle string
			if user.Login != "" {
				handle = user.Login
			} else {
				// "author:John Doe" does not work yet, so use the first name.
				handle = strings.Split(user.Name, " ")[0]
			}
			completions[i] = handle + "\t" + user.DisplayName()
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	}
}
