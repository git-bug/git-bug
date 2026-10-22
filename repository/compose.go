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
	RepoRemote
	RepoBrowse
	RepoClock
}

// gitTransportConfigKey chooses what talks to the remotes: go-git (the
// default), or the git binary, for what go-git can't do.
const gitTransportConfigKey = "git-bug.git-transport"

// OpenDefault opens the repository of the git directory gitDir, see
// FindGitDir, with the parts git-bug uses outside of tests: the configuration
// as git finds it, the user's keyring, the local storage, clocks and indexes
// in <gitDir>/<namespace>, and the remote operations of go-git or the git
// binary, as git-bug.git-transport says.
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

	config := NewRepoConfig(NewGitConfig(env))

	cfg, err := config.Config().Read()
	if err != nil {
		return nil, err
	}
	var remote RepoRemote
	switch transport := cfg.Value(gitTransportConfigKey, "go-git"); transport {
	case "go-git":
		remote = git
	case "git":
		remote, err = NewGitRepo(git.path, config)
		if err != nil {
			return nil, fmt.Errorf("%s=git: %w", gitTransportConfigKey, err)
		}
	default:
		return nil, fmt.Errorf("invalid %s %q: expected go-git or git", gitTransportConfigKey, transport)
	}

	storage := NewLocalStorage(filepath.Join(git.path, namespace))

	return &ComposedRepo{
		RepoConfig:  config,
		RepoKeyring: NewRepoKeyring(keyring),
		RepoStorage: NewRepoStorage(storage),
		RepoIndex:   NewSqliteIndex(filepath.Join(storage.Root(), fmt.Sprintf("index-v%d.db", sqliteLayout))),
		RepoData:    git,
		RepoRemote:  remote,
		RepoBrowse:  git,
		RepoClock:   NewPersistedClocks(storage),
	}, nil
}

// Close closes the parts holding resources: the indexes.
func (r *ComposedRepo) Close() error {
	return r.RepoIndex.Close()
}
