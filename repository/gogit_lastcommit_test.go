package repository

import (
	"bytes"
	"fmt"
	"io"
	"math/rand/v2"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// historyPaths are the directories of the history made by newHistoryRepo:
// the root, a directory changed by most commits, and a deep one rarely changed.
var historyPaths = []string{"", "internal/p3", "pkg/deep/nested/dir"}

// newHistoryRepo returns a repository with a history of 3000 steps on main,
// shaped like a real project, made with git fast-import, and its directory.
// Every 25 steps, a side branch of 3 commits is merged back while main gets 2
// commits of its own. Some files are never changed after the first commit, and
// third_party only once, far back. The history is the same on every call. With
// commitGraph, the repository also has a commit-graph file, written before the
// last 100 commits of main: the most recent commits are missing from it, as
// they are until the next git gc.
func newHistoryRepo(t testing.TB, commitGraph bool) (*GoGitRepo, string) {
	t.Helper()
	requireGitBinary(t)

	var files []string
	files = append(files, "LICENSE", ".gitignore", "README.md", "go.mod", "Makefile",
		"third_party/lib/a.go", "third_party/lib/b.go",
		"cmd/app/main.go", "cmd/app/flags.go",
		"docs/guide.md", "docs/old.md")
	for p := range 8 {
		for f := range 5 {
			files = append(files, fmt.Sprintf("internal/p%d/f%d.go", p, f))
		}
	}
	for f := range 10 {
		files = append(files, fmt.Sprintf("pkg/deep/nested/dir/f%d.go", f))
	}

	r := rand.New(rand.NewPCG(1, 2))
	pick := func() string {
		switch x := r.IntN(100); {
		case x < 88:
			return fmt.Sprintf("internal/p%d/f%d.go", r.IntN(8), r.IntN(5))
		case x < 93:
			return []string{"README.md", "go.mod", "Makefile"}[r.IntN(3)]
		case x < 97:
			return []string{"cmd/app/main.go", "cmd/app/flags.go"}[r.IntN(2)]
		case x < 98:
			return "docs/guide.md"
		default:
			return fmt.Sprintf("pkg/deep/nested/dir/f%d.go", r.IntN(10))
		}
	}

	var buf bytes.Buffer
	mark := 0
	rev := 0
	data := func(s string) {
		fmt.Fprintf(&buf, "data %d\n%s\n", len(s), s)
	}
	// commit writes a commit on branch, with from as first parent and merge as
	// the second one (0 for none), setting the files of changes, and returns
	// its mark.
	commit := func(branch string, from, merge int, changes []string) int {
		mark++
		ts := 1_700_000_000 + mark*60
		fmt.Fprintf(&buf, "commit refs/heads/%s\nmark :%d\n", branch, mark)
		fmt.Fprintf(&buf, "author test <test@example.com> %d +0000\n", ts)
		fmt.Fprintf(&buf, "committer test <test@example.com> %d +0000\n", ts)
		data(fmt.Sprintf("commit %d", mark))
		if from != 0 {
			fmt.Fprintf(&buf, "from :%d\n", from)
		}
		if merge != 0 {
			fmt.Fprintf(&buf, "merge :%d\n", merge)
		}
		for _, path := range changes {
			rev++
			fmt.Fprintf(&buf, "M 100644 inline %s\n", path)
			data(fmt.Sprintf("%s rev %d", path, rev))
		}
		return mark
	}
	some := func() []string {
		changes := make([]string, 1+r.IntN(3))
		for i := range changes {
			changes[i] = pick()
		}
		return changes
	}

	main := commit("main", 0, 0, files)
	for step := 1; step < 3000; step++ {
		if step == 701 {
			// The only change to third_party, far past the depth limit.
			main = commit("main", main, 0, []string{"third_party/lib/a.go"})
			continue
		}
		if step%25 != 0 {
			main = commit("main", main, 0, some())
			continue
		}
		side := main
		var sideChanges []string
		for range 3 {
			changes := some()
			side = commit("side", side, 0, changes)
			sideChanges = append(sideChanges, changes...)
		}
		for range 2 {
			main = commit("main", main, 0, some())
		}
		// fast-import doesn't merge trees: the merge commit sets again the
		// files of the side branch, which win over those of main.
		main = commit("main", main, side, sideChanges)
	}

	dir := t.TempDir()
	// gitInput runs git as runGit does, with input on its standard input.
	gitInput := func(input io.Reader, args ...string) {
		cmd := gitCommand(t, dir, args...)
		cmd.Stdin = input
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "%s: %s", cmd, out)
	}
	runGit(t, dir, "init", "-q", "-b", "main")
	gitInput(&buf, "fast-import", "--quiet")
	if commitGraph {
		older := runGit(t, dir, "rev-parse", "main~100")
		gitInput(strings.NewReader(older+"\n"), "commit-graph", "write", "--stdin-commits")
	}

	repo, err := OpenGoGitRepo(filepath.Join(dir, ".git"))
	require.NoError(t, err)
	return repo, dir
}

// historyEntries returns the names in the directory at path on main.
func historyEntries(t testing.TB, repo *GoGitRepo, path string) []string {
	t.Helper()

	h, err := repo.resolveRev("main")
	require.NoError(t, err)
	c, err := repo.r.CommitObject(h)
	require.NoError(t, err)
	_, entries, err := treeEntriesAtPath(c, path)
	require.NoError(t, err)
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	return names
}

func TestLastCommitForEntriesHistory(t *testing.T) {
	for _, commitGraph := range []bool{false, true} {
		repo, dir := newHistoryRepo(t, commitGraph)

		for _, path := range historyPaths {
			t.Run(fmt.Sprintf("graph=%v/path=%s", commitGraph, path), func(t *testing.T) {
				names := historyEntries(t, repo, path)
				got, err := repo.LastCommitForEntries("main", path, names)
				require.NoError(t, err)

				for _, name := range names {
					meta, ok := got[name]
					if !ok {
						// Past the depth limit, only a commit-graph lets
						// entries be found.
						require.False(t, commitGraph, "entry %s not found", name)
						continue
					}
					// Merge commits are compared to their first parent only.
					want := runGit(t, dir, "log", "-1", "--first-parent", "--format=%H", "main", "--", strings.TrimPrefix(path+"/"+name, "/"))
					require.Equal(t, want, string(meta.Hash), "entry %s", name)
				}
			})
		}
	}
}

func BenchmarkLastCommitForEntries(b *testing.B) {
	for _, commitGraph := range []bool{false, true} {
		repo, _ := newHistoryRepo(b, commitGraph)

		for _, path := range historyPaths {
			names := historyEntries(b, repo, path)
			b.Run(fmt.Sprintf("graph=%v/path=%s", commitGraph, path), func(b *testing.B) {
				var got map[string]CommitMeta
				for b.Loop() {
					repo.lastCommitCache.Purge()
					var err error
					got, err = repo.LastCommitForEntries("main", path, names)
					require.NoError(b, err)
				}
				b.ReportMetric(float64(len(names)-len(got)), "unresolved")
			})
		}
	}
}
