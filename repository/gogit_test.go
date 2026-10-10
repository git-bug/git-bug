package repository

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-git/go-billy/v5/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/util/lamport"
)

// requireGitBinary skips the test when the git binary is not available.
func requireGitBinary(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available")
	}
}

// runGit runs the git binary in dir, which can be a worktree or a git
// directory, and returns its trimmed output.
//
// git is isolated from the environment running the tests: no inherited GIT_*
// variable redirecting it (a GIT_DIR set by a hook would win over -C), no
// system or user config, which could run hooks (core.hooksPath) or change its
// behavior, and no automatic gc or maintenance changing the object store
// behind the test's back. Commits get a fixed identity.
func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()

	home := t.TempDir()
	env := []string{
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=" + os.DevNull,
		"HOME=" + home,
		"XDG_CONFIG_HOME=" + home,
		"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com",
	}
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "GIT_") && !strings.HasPrefix(kv, "HOME=") && !strings.HasPrefix(kv, "XDG_CONFIG_HOME=") {
			env = append(env, kv)
		}
	}

	args = append([]string{"-C", dir, "-c", "gc.auto=0", "-c", "maintenance.auto=false"}, args...)
	cmd := exec.Command("git", args...)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %s: %s", strings.Join(args, " "), out)
	return strings.TrimSpace(string(out))
}

func TestNewGoGitRepo(t *testing.T) {
	// Plain
	plainRepo := CreateGoGitTestRepo(t, false)
	plainRoot := goGitRepoDir(t, plainRepo)
	require.NoError(t, plainRepo.Close())
	plainGitDir := filepath.Join(plainRoot, ".git")

	// Bare
	bareRepo := CreateGoGitTestRepo(t, true)
	bareRoot := goGitRepoDir(t, bareRepo)
	require.NoError(t, bareRepo.Close())
	bareGitDir := bareRoot

	tests := []struct {
		inPath  string
		outPath string
		err     bool
	}{
		// errors
		{"/", "", true},
		// parent dir of a repo
		{filepath.Dir(plainRoot), "", true},

		// Plain repo
		{plainRoot, plainGitDir, false},
		{plainGitDir, plainGitDir, false},
		{path.Join(plainGitDir, "objects"), plainGitDir, false},

		// Bare repo
		{bareRoot, bareGitDir, false},
		{bareGitDir, bareGitDir, false},
		{path.Join(bareGitDir, "objects"), bareGitDir, false},
	}

	for i, tc := range tests {
		r, err := OpenGoGitRepo(tc.inPath, namespace)

		if tc.err {
			require.Error(t, err, i)
		} else {
			require.NoError(t, err, i)
			assert.Equal(t, filepath.ToSlash(tc.outPath), filepath.ToSlash(r.path), i)
			require.NoError(t, r.Close())
		}
	}
}

func TestGoGitRepo(t *testing.T) {
	RepoTest(t, CreateGoGitTestRepo)
}

func TestGoGitRepo_Head(t *testing.T) {
	repo := CreateGoGitTestRepo(t, false)

	// a new repository's HEAD points to refs/heads/master, which doesn't exist yet
	_, err := repo.Head()
	require.ErrorIs(t, err, ErrNotFound)

	commit := storeTestCommits(t, repo, 1)[0]
	require.NoError(t, repo.SetBranch("master", commit))

	meta, err := repo.Head()
	require.NoError(t, err)
	require.Equal(t, RefMeta{
		Name:      "refs/heads/master",
		ShortName: "master",
		Type:      GitRefTypeBranch,
		Hash:      string(commit),
	}, meta)
}

func TestGoGitRepo_Indexes(t *testing.T) {
	const commit1 = Hash("1111111111111111111111111111111111111111")

	t.Run("created on disk", func(t *testing.T) {
		repo := CreateGoGitTestRepo(t, false)
		plainRoot := goGitRepoDir(t, repo)

		// Can create indices
		indexA, err := repo.GetIndex("a")
		require.NoError(t, err)
		require.NotZero(t, indexA)
		require.FileExists(t, filepath.Join(plainRoot, ".git", namespace, "indexes", "a", "index_meta.json"))
		require.FileExists(t, filepath.Join(plainRoot, ".git", namespace, "indexes", "a", "store", "root.bolt"))

		indexB, err := repo.GetIndex("b")
		require.NoError(t, err)
		require.NotZero(t, indexB)
		require.DirExists(t, filepath.Join(plainRoot, ".git", namespace, "indexes", "b"))

		// Can get an existing index
		indexA, err = repo.GetIndex("a")
		require.NoError(t, err)
		require.NotZero(t, indexA)
	})

	t.Run("documents and their record survive a reopen", func(t *testing.T) {
		repo := CreateGoGitTestRepo(t, false)

		idx, err := repo.GetIndex("a")
		require.NoError(t, err)
		b := idx.NewBatch()
		require.NoError(t, b.Set("id1", []string{"marker"}, commit1))
		require.NoError(t, b.Apply())

		require.NoError(t, repo.Close())

		idx, err = repo.GetIndex("a")
		require.NoError(t, err)
		builtFrom, err := idx.BuiltFrom()
		require.NoError(t, err)
		require.Equal(t, map[string]Hash{"id1": commit1}, builtFrom)
		res, err := idx.Search([]string{"marker"})
		require.NoError(t, err)
		require.Equal(t, []string{"id1"}, res)
	})

	t.Run("search", func(t *testing.T) {
		repo := CreateGoGitTestRepo(t, false)

		idx, err := repo.GetIndex("a")
		require.NoError(t, err)
		b := idx.NewBatch()
		require.NoError(t, b.Set("id1", []string{"the server crashes on login"}, commit1))
		require.NoError(t, b.Set("id2", []string{"login works, the server crashes later"}, commit1))
		require.NoError(t, b.Set("id3", []string{"-x +y title:z (abc"}, commit1))
		require.NoError(t, b.Set("id4", []string{"see file.go, and foo-bar"}, commit1))
		require.NoError(t, b.Set("id5", []string{"数据库连接失败"}, commit1))
		require.NoError(t, b.Set("id6", []string{"库存数据"}, commit1))
		require.NoError(t, b.Apply())

		requireSearch := func(t *testing.T, terms []string, expected ...string) {
			t.Helper()
			res, err := idx.Search(terms)
			require.NoError(t, err)
			require.ElementsMatch(t, expected, res)
		}

		// words are stemmed
		requireSearch(t, []string{"crash"}, "id1", "id2")
		// a term of several words is a phrase
		requireSearch(t, []string{"crashes on login"}, "id1")
		requireSearch(t, []string{"login crashes"})
		// so is a single word that splits into several
		requireSearch(t, []string{"foo-bar"}, "id4")
		requireSearch(t, []string{"bar-foo"})
		requireSearch(t, []string{"file.go"}, "id4")
		requireSearch(t, []string{"数据库"}, "id5")
		requireSearch(t, []string{"数据"}, "id5", "id6")
		// the caller's terms are left untouched
		terms := []string{"crashes on login"}
		requireSearch(t, terms, "id1")
		require.Equal(t, []string{"crashes on login"}, terms)
		// a stop word alone matches nothing, and doesn't prevent other matches
		requireSearch(t, []string{"the"})
		requireSearch(t, []string{"the", "later"}, "id2")
		// the query syntax of bleve is not interpreted
		requireSearch(t, []string{"-x"}, "id3")
		requireSearch(t, []string{"+y"}, "id3")
		requireSearch(t, []string{"title:z"}, "id3")
		requireSearch(t, []string{"(abc"}, "id3")
		requireSearch(t, []string{`"server`}, "id1", "id2")
	})

	t.Run("a document without text is recorded, not indexed", func(t *testing.T) {
		repo := CreateGoGitTestRepo(t, false)

		idx, err := repo.GetIndex("a")
		require.NoError(t, err)
		b := idx.NewBatch()
		require.NoError(t, b.Set("id1", []string{"marker"}, commit1))
		require.NoError(t, b.Set("id2", nil, commit1))
		require.NoError(t, b.Set("id3", []string{"", "  "}, commit1))
		require.NoError(t, b.Apply())

		builtFrom, err := idx.BuiltFrom()
		require.NoError(t, err)
		require.Equal(t, map[string]Hash{"id1": commit1, "id2": commit1, "id3": commit1}, builtFrom)
		count, err := idx.(*bleveIndex).index.DocCount()
		require.NoError(t, err)
		require.Equal(t, uint64(1), count)

		// a document losing its text is dropped
		b = idx.NewBatch()
		require.NoError(t, b.Set("id1", nil, commit1))
		require.NoError(t, b.Apply())
		count, err = idx.(*bleveIndex).index.DocCount()
		require.NoError(t, err)
		require.Equal(t, uint64(0), count)
		res, err := idx.Search([]string{"marker"})
		require.NoError(t, err)
		require.Empty(t, res)
	})

	t.Run("an index with another layout is replaced by an empty one", func(t *testing.T) {
		repo := CreateGoGitTestRepo(t, false)

		idx, err := repo.GetIndex("a")
		require.NoError(t, err)
		b := idx.NewBatch()
		require.NoError(t, b.Set("id1", []string{"marker"}, commit1))
		require.NoError(t, b.Apply())
		require.NoError(t, idx.(*bleveIndex).index.SetInternal(layoutVersionKey, []byte("0")))

		require.NoError(t, repo.Close())

		idx, err = repo.GetIndex("a")
		require.NoError(t, err)
		builtFrom, err := idx.BuiltFrom()
		require.NoError(t, err)
		require.Empty(t, builtFrom)
		res, err := idx.Search([]string{"marker"})
		require.NoError(t, err)
		require.Empty(t, res)

		// and the new one keeps its layout across a reopen
		b = idx.NewBatch()
		require.NoError(t, b.Set("id1", []string{"marker"}, commit1))
		require.NoError(t, b.Apply())
		require.NoError(t, repo.Close())
		idx, err = repo.GetIndex("a")
		require.NoError(t, err)
		res, err = idx.Search([]string{"marker"})
		require.NoError(t, err)
		require.Equal(t, []string{"id1"}, res)
	})

	t.Run("a malformed record is rejected", func(t *testing.T) {
		repo := CreateGoGitTestRepo(t, false)

		idx, err := repo.GetIndex("a")
		require.NoError(t, err)
		require.NoError(t, idx.(*bleveIndex).index.SetInternal(builtFromKey, []byte("not a record")))

		_, err = idx.BuiltFrom()
		require.Error(t, err)
		require.Error(t, idx.NewBatch().Apply())
	})

	t.Run("a batch that fails leaves neither documents nor record", func(t *testing.T) {
		repo := CreateGoGitTestRepo(t, false)

		idx, err := repo.GetIndex("a")
		require.NoError(t, err)
		b := idx.NewBatch()
		require.NoError(t, b.Set("id1", []string{"marker"}, commit1))

		// the index is closed underneath the batch
		require.NoError(t, repo.Close())
		require.Error(t, b.Apply())

		idx, err = repo.GetIndex("a")
		require.NoError(t, err)
		builtFrom, err := idx.BuiltFrom()
		require.NoError(t, err)
		require.Empty(t, builtFrom)
		res, err := idx.Search([]string{"marker"})
		require.NoError(t, err)
		require.Empty(t, res)
	})
}

func TestGoGit_DetectsSubmodules(t *testing.T) {
	repo := CreateGoGitTestRepo(t, false)
	expected := filepath.Join(goGitRepoDir(t, repo), "/.git")

	d := t.TempDir()
	rel, err := filepath.Rel(d, expected)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(d, ".git"), []byte(fmt.Sprintf("gitdir: %s", rel)), 0600)
	require.NoError(t, err)

	result, err := detectGitPath(d, 0)
	assert.Empty(t, err)
	assert.Equal(t, expected, result)
}

// An empty clock file is one being created, or whose creation was interrupted:
// not a clock yet, and not a reason to fail listing the others.
func TestGoGitRepo_AllClocksSkipsEmpty(t *testing.T) {
	repo := CreateGoGitTestRepo(t, false)

	foo, err := repo.GetOrCreateClock("foo", 1)
	require.NoError(t, err)

	f, err := repo.LocalStorage().Create(filepath.Join(clockPath, "empty"))
	require.NoError(t, err)
	require.NoError(t, f.Close())

	allClocks, err := repo.AllClocks()
	require.NoError(t, err)
	require.Equal(t, map[string]lamport.Clock{"foo": foo}, allClocks)
}

// A corrupted clock is listed and handed out, but can't be used until it's
// created again, at a value the caller vouches for.
func TestGoGitRepo_CorruptedClock(t *testing.T) {
	repo := CreateGoGitTestRepo(t, false)

	clockFile := filepath.Join(clockPath, "foo")
	require.NoError(t, util.WriteFile(repo.LocalStorage(), clockFile, []byte("garbage"), 0644))

	// listed, for the caller to decide what to do with it
	allClocks, err := repo.AllClocks()
	require.NoError(t, err)
	require.Contains(t, allClocks, "foo")
	_, err = allClocks["foo"].Time()
	require.ErrorIs(t, err, lamport.ErrClockCorrupt)

	_, err = repo.Increment("foo")
	require.ErrorIs(t, err, lamport.ErrClockCorrupt)
	require.ErrorIs(t, repo.Witness("foo", 10), lamport.ErrClockCorrupt)

	clock, err := repo.GetOrCreateClock("foo", 10)
	require.NoError(t, err)
	time, err := clock.Time()
	require.NoError(t, err)
	require.Equal(t, lamport.Time(10), time)

	// the clock listed earlier is the same one
	time, err = allClocks["foo"].Time()
	require.NoError(t, err)
	require.Equal(t, lamport.Time(10), time)

	time, err = repo.Increment("foo")
	require.NoError(t, err)
	require.Equal(t, lamport.Time(11), time)
}
