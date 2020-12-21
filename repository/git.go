// Package repository contains helper methods for working with the Git repo.
package repository

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/blevesearch/bleve"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/osfs"

	"github.com/MichaelMure/git-bug/util/lamport"
)

var _ ClockedRepo = &GitRepo{}
var _ TestedRepo = &GitRepo{}

// GitRepo represents an instance of a (local) git repository.
type GitRepo struct {
	gitCli
	path string

	clocksMutex sync.Mutex
	clocks      map[string]lamport.Clock

	indexesMutex sync.Mutex
	indexes      map[string]bleve.Index

	keyring      Keyring
	localStorage billy.Filesystem
}

func (repo *GitRepo) ReadCommit(hash Hash) (Commit, error) {
	panic("implement me")
}

func (repo *GitRepo) ResolveRef(ref string) (Hash, error) {
	panic("implement me")
}

// OpenGitRepo determines if the given working directory is inside of a git repository,
// and returns the corresponding GitRepo instance if it is.
func OpenGitRepo(path string, clockLoaders []ClockLoader) (*GitRepo, error) {
	k, err := defaultKeyring()
	if err != nil {
		return nil, err
	}

	repo := &GitRepo{
		gitCli:  gitCli{path: path},
		path:    path,
		clocks:  make(map[string]lamport.Clock),
		indexes: make(map[string]bleve.Index),
		keyring: k,
	}

	// Check the repo and retrieve the root path
	stdout, err := repo.runGitCommand("rev-parse", "--absolute-git-dir")

	// Now dir is fetched with "git rev-parse --git-dir". May be it can
	// still return nothing in some cases. Then empty stdout check is
	// kept.
	if err != nil || stdout == "" {
		return nil, ErrNotARepo
	}

	// Fix the path to be sure we are at the root
	repo.path = stdout
	repo.gitCli.path = stdout
	repo.localStorage = osfs.New(filepath.Join(path, "git-bug"))

	for _, loader := range clockLoaders {
		allExist := true
		for _, name := range loader.Clocks {
			if _, err := repo.getClock(name); err != nil {
				allExist = false
			}
		}

		if !allExist {
			err = loader.Witnesser(repo)
			if err != nil {
				return nil, err
			}
		}
	}

	return repo, nil
}

// InitGitRepo create a new empty git repo at the given path
func InitGitRepo(path string) (*GitRepo, error) {
	k, err := defaultKeyring()
	if err != nil {
		return nil, err
	}

	repo := &GitRepo{
		gitCli:       gitCli{path: path},
		path:         filepath.Join(path, ".git"),
		clocks:       make(map[string]lamport.Clock),
		indexes:      make(map[string]bleve.Index),
		keyring:      k,
		localStorage: osfs.New(filepath.Join(path, ".git", "git-bug")),
	}

	_, err = repo.runGitCommand("init", path)
	if err != nil {
		return nil, err
	}

	return repo, nil
}

// InitBareGitRepo create a new --bare empty git repo at the given path
func InitBareGitRepo(path string) (*GitRepo, error) {
	k, err := defaultKeyring()
	if err != nil {
		return nil, err
	}

	repo := &GitRepo{
		gitCli:       gitCli{path: path},
		path:         path,
		clocks:       make(map[string]lamport.Clock),
		indexes:      make(map[string]bleve.Index),
		keyring:      k,
		localStorage: osfs.New(filepath.Join(path, "git-bug")),
	}

	_, err = repo.runGitCommand("init", "--bare", path)
	if err != nil {
		return nil, err
	}

	return repo, nil
}

func (repo *GitRepo) Close() error {
	var firstErr error
	for _, index := range repo.indexes {
		err := index.Close()
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// LocalConfig give access to the repository scoped configuration
func (repo *GitRepo) LocalConfig() Config {
	return newGitConfig(repo.gitCli, false)
}

// GlobalConfig give access to the global scoped configuration
func (repo *GitRepo) GlobalConfig() Config {
	return newGitConfig(repo.gitCli, true)
}

// AnyConfig give access to a merged local/global configuration
func (repo *GitRepo) AnyConfig() ConfigRead {
	return mergeConfig(repo.LocalConfig(), repo.GlobalConfig())
}

// Keyring give access to a user-wide storage for secrets
func (repo *GitRepo) Keyring() Keyring {
	return repo.keyring
}

// GetPath returns the path to the repo.
func (repo *GitRepo) GetPath() string {
	return repo.path
}

// GetUserName returns the name the the user has used to configure git
func (repo *GitRepo) GetUserName() (string, error) {
	return repo.runGitCommand("config", "user.name")
}

// GetUserEmail returns the email address that the user has used to configure git.
func (repo *GitRepo) GetUserEmail() (string, error) {
	return repo.runGitCommand("config", "user.email")
}

// GetCoreEditor returns the name of the editor that the user has used to configure git.
func (repo *GitRepo) GetCoreEditor() (string, error) {
	return repo.runGitCommand("var", "GIT_EDITOR")
}

// GetRemotes returns the configured remotes repositories.
func (repo *GitRepo) GetRemotes() (map[string]string, error) {
	stdout, err := repo.runGitCommand("remote", "--verbose")
	if err != nil {
		return nil, err
	}

	lines := strings.Split(stdout, "\n")
	remotes := make(map[string]string, len(lines))

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		elements := strings.Fields(line)
		if len(elements) != 3 {
			return nil, fmt.Errorf("git remote: unexpected output format: %s", line)
		}

		remotes[elements[0]] = elements[1]
	}

	return remotes, nil
}

// LocalStorage return a billy.Filesystem giving access to $RepoPath/.git/git-bug
func (repo *GitRepo) LocalStorage() billy.Filesystem {
	return repo.localStorage
}

// GetBleveIndex return a bleve.Index that can be used to index documents
func (repo *GitRepo) GetBleveIndex(name string) (bleve.Index, error) {
	repo.indexesMutex.Lock()
	defer repo.indexesMutex.Unlock()

	if index, ok := repo.indexes[name]; ok {
		return index, nil
	}

	path := filepath.Join(repo.path, "indexes", name)

	index, err := bleve.Open(path)
	if err == nil {
		repo.indexes[name] = index
		return index, nil
	}

	err = os.MkdirAll(path, os.ModeDir)
	if err != nil {
		return nil, err
	}

	mapping := bleve.NewIndexMapping()
	mapping.DefaultAnalyzer = "en"

	index, err = bleve.New(path, mapping)
	if err != nil {
		return nil, err
	}

	repo.indexes[name] = index

	return index, nil
}

// ClearBleveIndex will wipe the given index
func (repo *GitRepo) ClearBleveIndex(name string) error {
	repo.indexesMutex.Lock()
	defer repo.indexesMutex.Unlock()

	path := filepath.Join(repo.path, "indexes", name)

	err := os.RemoveAll(path)
	if err != nil {
		return err
	}

	delete(repo.indexes, name)

	return nil
}

// FetchRefs fetch git refs from a remote
func (repo *GitRepo) FetchRefs(remote, refSpec string) (string, error) {
	stdout, err := repo.runGitCommand("fetch", remote, refSpec)

	if err != nil {
		return stdout, fmt.Errorf("failed to fetch from the remote '%s': %v", remote, err)
	}

	return stdout, err
}

// PushRefs push git refs to a remote
func (repo *GitRepo) PushRefs(remote string, refSpec string) (string, error) {
	stdout, stderr, err := repo.runGitCommandRaw(nil, "push", remote, refSpec)

	if err != nil {
		return stdout + stderr, fmt.Errorf("failed to push to the remote '%s': %v", remote, stderr)
	}
	return stdout + stderr, nil
}

// StoreData will store arbitrary data and return the corresponding hash
func (repo *GitRepo) StoreData(data []byte) (Hash, error) {
	var stdin = bytes.NewReader(data)

	stdout, err := repo.runGitCommandWithStdin(stdin, "hash-object", "--stdin", "-w")

	return Hash(stdout), err
}

// ReadData will attempt to read arbitrary data from the given hash
func (repo *GitRepo) ReadData(hash Hash) ([]byte, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := repo.runGitCommandWithIO(nil, &stdout, &stderr, "cat-file", "-p", string(hash))

	if err != nil {
		return []byte{}, err
	}

	return stdout.Bytes(), nil
}

// StoreTree will store a mapping key-->Hash as a Git tree
func (repo *GitRepo) StoreTree(entries []TreeEntry) (Hash, error) {
	buffer := prepareTreeEntries(entries)

	stdout, err := repo.runGitCommandWithStdin(&buffer, "mktree")

	if err != nil {
		return "", err
	}

	return Hash(stdout), nil
}

// StoreCommit will store a Git commit with the given Git tree
func (repo *GitRepo) StoreCommit(treeHash Hash) (Hash, error) {
	stdout, err := repo.runGitCommand("commit-tree", string(treeHash))

	if err != nil {
		return "", err
	}

	return Hash(stdout), nil
}

// StoreCommitWithParent will store a Git commit with the given Git tree
func (repo *GitRepo) StoreCommitWithParent(treeHash Hash, parent Hash) (Hash, error) {
	stdout, err := repo.runGitCommand("commit-tree", string(treeHash),
		"-p", string(parent))

	if err != nil {
		return "", err
	}

	return Hash(stdout), nil
}

// UpdateRef will create or update a Git reference
func (repo *GitRepo) UpdateRef(ref string, hash Hash) error {
	_, err := repo.runGitCommand("update-ref", ref, string(hash))

	return err
}

// RemoveRef will remove a Git reference
func (repo *GitRepo) RemoveRef(ref string) error {
	_, err := repo.runGitCommand("update-ref", "-d", ref)

	return err
}

// ListRefs will return a list of Git ref matching the given refspec
func (repo *GitRepo) ListRefs(refPrefix string) ([]string, error) {
	stdout, err := repo.runGitCommand("for-each-ref", "--format=%(refname)", refPrefix)

	if err != nil {
		return nil, err
	}

	split := strings.Split(stdout, "\n")

	if len(split) == 1 && split[0] == "" {
		return []string{}, nil
	}

	return split, nil
}

// RefExist will check if a reference exist in Git
func (repo *GitRepo) RefExist(ref string) (bool, error) {
	stdout, err := repo.runGitCommand("for-each-ref", ref)

	if err != nil {
		return false, err
	}

	return stdout != "", nil
}

// CopyRef will create a new reference with the same value as another one
func (repo *GitRepo) CopyRef(source string, dest string) error {
	_, err := repo.runGitCommand("update-ref", dest, source)

	return err
}

// ListCommits will return the list of commit hashes of a ref, in chronological order
func (repo *GitRepo) ListCommits(ref string) ([]Hash, error) {
	stdout, err := repo.runGitCommand("rev-list", "--first-parent", "--reverse", ref)

	if err != nil {
		return nil, err
	}

	split := strings.Split(stdout, "\n")

	casted := make([]Hash, len(split))
	for i, line := range split {
		casted[i] = Hash(line)
	}

	return casted, nil

}

// ReadTree will return the list of entries in a Git tree
func (repo *GitRepo) ReadTree(hash Hash) ([]TreeEntry, error) {
	stdout, err := repo.runGitCommand("ls-tree", string(hash))

	if err != nil {
		return nil, err
	}

	return readTreeEntries(stdout)
}

// FindCommonAncestor will return the last common ancestor of two chain of commit
func (repo *GitRepo) FindCommonAncestor(hash1 Hash, hash2 Hash) (Hash, error) {
	stdout, err := repo.runGitCommand("merge-base", string(hash1), string(hash2))

	if err != nil {
		return "", err
	}

	return Hash(stdout), nil
}

// GetTreeHash return the git tree hash referenced in a commit
func (repo *GitRepo) GetTreeHash(commit Hash) (Hash, error) {
	stdout, err := repo.runGitCommand("rev-parse", string(commit)+"^{tree}")

	if err != nil {
		return "", err
	}

	return Hash(stdout), nil
}

func (repo *GitRepo) AllClocks() (map[string]lamport.Clock, error) {
	repo.clocksMutex.Lock()
	defer repo.clocksMutex.Unlock()

	result := make(map[string]lamport.Clock)

	files, err := ioutil.ReadDir(filepath.Join(repo.path, "git-bug", clockPath))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		name := file.Name()
		if c, ok := repo.clocks[name]; ok {
			result[name] = c
		} else {
			c, err := lamport.LoadPersistedClock(repo.LocalStorage(), filepath.Join(clockPath, name))
			if err != nil {
				return nil, err
			}
			repo.clocks[name] = c
			result[name] = c
		}
	}

	return result, nil
}

// GetOrCreateClock return a Lamport clock stored in the Repo.
// If the clock doesn't exist, it's created.
func (repo *GitRepo) GetOrCreateClock(name string) (lamport.Clock, error) {
	repo.clocksMutex.Lock()
	defer repo.clocksMutex.Unlock()

	c, err := repo.getClock(name)
	if err == nil {
		return c, nil
	}
	if err != ErrClockNotExist {
		return nil, err
	}

	c, err = lamport.NewPersistedClock(repo.LocalStorage(), filepath.Join(clockPath, name))
	if err != nil {
		return nil, err
	}

	repo.clocks[name] = c
	return c, nil
}

func (repo *GitRepo) getClock(name string) (lamport.Clock, error) {
	if c, ok := repo.clocks[name]; ok {
		return c, nil
	}

	c, err := lamport.LoadPersistedClock(repo.LocalStorage(), filepath.Join(clockPath, name))
	if err == nil {
		repo.clocks[name] = c
		return c, nil
	}
	if err == lamport.ErrClockNotExist {
		return nil, ErrClockNotExist
	}
	return nil, err
}

// Increment is equivalent to c = GetOrCreateClock(name) + c.Increment()
func (repo *GitRepo) Increment(name string) (lamport.Time, error) {
	c, err := repo.GetOrCreateClock(name)
	if err != nil {
		return lamport.Time(0), err
	}
	return c.Increment()
}

// Witness is equivalent to c = GetOrCreateClock(name) + c.Witness(time)
func (repo *GitRepo) Witness(name string, time lamport.Time) error {
	c, err := repo.GetOrCreateClock(name)
	if err != nil {
		return err
	}
	return c.Witness(time)
}

// AddRemote add a new remote to the repository
// Not in the interface because it's only used for testing
func (repo *GitRepo) AddRemote(name string, url string) error {
	_, err := repo.runGitCommand("remote", "add", name, url)

	return err
}

// GetLocalRemote return the URL to use to add this repo as a local remote
func (repo *GitRepo) GetLocalRemote() string {
	return repo.path
}

// EraseFromDisk delete this repository entirely from the disk
func (repo *GitRepo) EraseFromDisk() error {
	err := repo.Close()
	if err != nil {
		return err
	}

	path := filepath.Clean(strings.TrimSuffix(repo.path, string(filepath.Separator)+".git"))

	// fmt.Println("Cleaning repo:", path)
	return os.RemoveAll(path)
}
