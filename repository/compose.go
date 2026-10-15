package repository

import (
	"fmt"
	"path/filepath"

	"github.com/git-bug/gitconfig"
)

var _ Repo = &ComposedRepo{}

// ComposedRepo is a Repo assembled from independent parts.
type ComposedRepo struct {
	RepoConfig
	RepoKeyring
	RepoStorage
	RepoIndex
	RepoData
	RepoBrowse
	RepoClock
}

// OpenDefault opens the repository of the git directory gitDir, see
// FindGitDir, with the parts git-bug uses outside of tests: the configuration
// as git finds it, the user's keyring, and the local storage, clocks and
// indexes in <gitDir>/<namespace>.
func OpenDefault(gitDir string, namespace string) (Repo, error) {
	git, err := OpenGoGitRepo(gitDir)
	if err != nil {
		return nil, err
	}
	return newDefault(git, namespace)
}

// InitDefault creates an empty git repository at path, and returns it with the
// same parts as OpenDefault.
func InitDefault(path string, namespace string) (Repo, error) {
	git, err := InitGoGitRepo(path)
	if err != nil {
		return nil, err
	}
	return newDefault(git, namespace)
}

// newDefault returns the repository of git with the parts of OpenDefault.
func newDefault(git *GoGitRepo, namespace string) (Repo, error) {
	env, err := gitconfig.Discover(git.path)
	if err != nil {
		return nil, err
	}

	keyring, err := DefaultKeyring()
	if err != nil {
		return nil, err
	}

	storage := NewLocalStorage(filepath.Join(git.path, namespace))

	return &ComposedRepo{
		RepoConfig:  NewRepoConfig(NewGitConfig(env)),
		RepoKeyring: NewRepoKeyring(keyring),
		RepoStorage: NewRepoStorage(storage),
		RepoIndex:   NewSqliteIndex(filepath.Join(storage.Root(), fmt.Sprintf("index-v%d.db", sqliteLayout))),
		RepoData:    git,
		RepoBrowse:  git,
		RepoClock:   NewPersistedClocks(storage),
	}, nil
}

// Close closes the parts holding resources: the indexes.
func (r *ComposedRepo) Close() error {
	return r.RepoIndex.Close()
}
