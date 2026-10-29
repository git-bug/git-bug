package repository

import (
	"path/filepath"
	"testing"

	"github.com/git-bug/gitconfig"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/stretchr/testify/require"
)

// newTestGoGitRepo creates a GoGitRepo in a directory removed with the test.
func newTestGoGitRepo(t testing.TB, bare bool) *GoGitRepo {
	t.Helper()

	dir := t.TempDir()
	create, gitDir := InitGoGitRepo, filepath.Join(dir, ".git")
	if bare {
		create, gitDir = InitBareGoGitRepo, dir
	}
	require.NoError(t, create(dir))
	config := NewRepoConfig(NewGitConfig(gitconfig.Env{
		GitDir:       gitDir,
		GlobalConfig: []string{filepath.Join(t.TempDir(), ".gitconfig")},
	}))
	repo, err := OpenGoGitRepo(gitDir, config)
	require.NoError(t, err)
	return repo
}

// TestGoGitRepo_Specifics tests what the shared suite, run by TestGoGitRepo,
// doesn't.
func TestGoGitRepo_Specifics(t *testing.T) {
	t.Run("OpenGoGitRepo doesn't search", func(t *testing.T) {
		opened := newTestGoGitRepo(t, false)
		gitDir := opened.path

		_, err := OpenGoGitRepo(filepath.Dir(gitDir), opened.config)
		require.ErrorIs(t, err, ErrNotARepo)

		repo, err := OpenGoGitRepo(gitDir, opened.config)
		require.NoError(t, err)
		require.Equal(t, gitDir, repo.path)
	})

	t.Run("StoreTree sorts as git", func(t *testing.T) {
		requireGitBinary(t)

		repo := newTestGoGitRepo(t, false)
		blob, err := repo.StoreData(t.Context(), []byte("content"))
		require.NoError(t, err)
		sub, err := repo.StoreTree(t.Context(), []TreeEntry{{ObjectType: Blob, Hash: blob, Name: "x"}})
		require.NoError(t, err)

		// git compares a directory's name as if it ended with a slash: "a" goes
		// after "a-c" and "a.b", not before
		tree, err := repo.StoreTree(t.Context(), []TreeEntry{
			{ObjectType: Tree, Hash: sub, Name: "a"},
			{ObjectType: Blob, Hash: blob, Name: "a.b"},
			{ObjectType: Blob, Hash: blob, Name: "a-c"},
		})
		require.NoError(t, err)

		// git fsck rejects a tree out of git's order
		commit, err := repo.StoreCommit(t.Context(), tree)
		require.NoError(t, err)
		dir := filepath.Dir(repo.path)
		runGit(t, dir, "update-ref", "refs/test/tree", string(commit))
		runGit(t, dir, "fsck", "--strict")
	})

	t.Run("Head", func(t *testing.T) {
		repo := newTestGoGitRepo(t, false)

		// a new repository's HEAD points to refs/heads/master, which doesn't exist yet
		_, err := repo.Head(t.Context())
		require.ErrorIs(t, err, ErrNotFound)

		commit := storeTestCommits(t, repo, 1)[0]
		require.NoError(t, repo.r.Storer.SetReference(plumbing.NewHashReference(
			plumbing.NewBranchReferenceName("master"), plumbing.NewHash(commit.String()))))

		meta, err := repo.Head(t.Context())
		require.NoError(t, err)
		require.Equal(t, RefMeta{
			Name:      "refs/heads/master",
			ShortName: "master",
			Type:      GitRefTypeBranch,
			Hash:      string(commit),
		}, meta)
	})
}
