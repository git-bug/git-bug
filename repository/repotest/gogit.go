package repotest

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/99designs/keyring"
	"github.com/git-bug/gitconfig"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/repository"
)

const namespace = "git-bug"

var _ Repo = &GoGitRepo{}

// GoGitRepo is a GoGitRepo created for a test.
type GoGitRepo struct {
	*repository.GoGitRepo

	// GitDir is the git directory of the repository, to use it as a remote.
	GitDir string
}

// NewGoGitRepo creates a GoGitRepo in a directory removed with the test. It
// doesn't touch the configuration or the keyring of the user or the system,
// and has a git user set.
func NewGoGitRepo(t testing.TB, bare bool) *GoGitRepo {
	t.Helper()

	dir := t.TempDir()
	create, gitDir := repository.InitGoGitRepo, filepath.Join(dir, ".git")
	if bare {
		create, gitDir = repository.InitBareGoGitRepo, dir
	}
	repo, err := create(dir, namespace,
		repository.WithConfigEnv(gitconfig.Env{
			GlobalConfig: []string{filepath.Join(t.TempDir(), ".gitconfig")},
		}),
		repository.WithKeyring(keyring.NewArrayKeyring(nil)),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := repo.Close(); err != nil {
			t.Error(err)
		}
	})

	err = repo.Config().Update(func(f *gitconfig.File) error {
		return errors.Join(
			f.Set("user.name", "testuser"),
			f.Set("user.email", "testuser@example.com"),
		)
	})
	require.NoError(t, err)

	return &GoGitRepo{GoGitRepo: repo, GitDir: gitDir}
}

// SetBranch points the branch name to commit, creating it if needed.
func (r *GoGitRepo) SetBranch(name string, commit repository.Hash) error {
	return r.setRef(plumbing.NewBranchReferenceName(name), commit)
}

// SetTag points the lightweight tag name to commit, creating it if needed.
func (r *GoGitRepo) SetTag(name string, commit repository.Hash) error {
	return r.setRef(plumbing.NewTagReferenceName(name), commit)
}

func (r *GoGitRepo) setRef(name plumbing.ReferenceName, commit repository.Hash) error {
	storage := filesystem.NewStorage(osfs.New(r.GitDir), cache.NewObjectLRUDefault())
	return storage.SetReference(plumbing.NewHashReference(name, plumbing.NewHash(commit.String())))
}

// SetupGoGitReposAndRemote creates two repositories, and a bare one as their
// remote "origin".
func SetupGoGitReposAndRemote(t *testing.T) (repoA, repoB, remote *GoGitRepo) {
	t.Helper()

	repoA = NewGoGitRepo(t, false)
	repoB = NewGoGitRepo(t, false)
	remote = NewGoGitRepo(t, true)

	AddRemote(t, repoA, "origin", remote.GitDir)
	AddRemote(t, repoB, "origin", remote.GitDir)

	return repoA, repoB, remote
}

// AddRemote adds a remote to repo, as git remote add does.
func AddRemote(t testing.TB, repo repository.RepoConfig, name, url string) {
	t.Helper()

	err := repo.Config().Update(func(f *gitconfig.File) error {
		return errors.Join(
			f.Set("remote."+name+".url", url),
			f.Set("remote."+name+".fetch", "+refs/heads/*:refs/remotes/"+name+"/*"),
		)
	})
	require.NoError(t, err)
}
