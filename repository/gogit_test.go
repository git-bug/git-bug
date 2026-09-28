package repository

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sync"
	"testing"

	"github.com/go-git/go-billy/v5/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/util/lamport"
)

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

	blobHash, err := repo.StoreData(randomData())
	require.NoError(t, err)
	treeHash, err := repo.StoreTree([]TreeEntry{{ObjectType: Blob, Hash: blobHash, Name: "blob"}})
	require.NoError(t, err)
	commit, err := repo.StoreCommit(treeHash)
	require.NoError(t, err)
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

func TestGoGitRepo_ConcurrentUpdateRef(t *testing.T) {
	repo := CreateGoGitTestRepo(t, false)

	commits := make([]Hash, 21)
	for i := range commits {
		blobHash, err := repo.StoreData(randomData())
		require.NoError(t, err)
		treeHash, err := repo.StoreTree([]TreeEntry{{ObjectType: Blob, Hash: blobHash, Name: "blob"}})
		require.NoError(t, err)
		commits[i], err = repo.StoreCommit(treeHash)
		require.NoError(t, err)
	}

	key := randomKey()
	require.NoError(t, repo.UpdateRef("concurrent", key, "", commits[0]))

	// every writer moves the ref from the same commit to its own: exactly one must succeed
	candidates := commits[1:]
	errs := make([]error, len(candidates))
	var wg sync.WaitGroup
	for i, commit := range candidates {
		wg.Go(func() { errs[i] = repo.UpdateRef("concurrent", key, commits[0], commit) })
	}
	wg.Wait()

	var winners []Hash
	for i, err := range errs {
		if err == nil {
			winners = append(winners, candidates[i])
			continue
		}
		require.ErrorIs(t, err, ErrRefChanged)
	}
	require.Len(t, winners, 1)

	h, err := repo.ResolveRef("concurrent", key)
	require.NoError(t, err)
	require.Equal(t, winners[0], h)
}

func TestGoGitRepo_Indexes(t *testing.T) {
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
}

func TestGoGit_DetectsSubmodules(t *testing.T) {
	repo := CreateGoGitTestRepo(t, false)
	expected := filepath.Join(goGitRepoDir(t, repo), "/.git")

	d := t.TempDir()
	err := os.WriteFile(filepath.Join(d, ".git"), []byte(fmt.Sprintf("gitdir: %s", expected)), 0600)
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
