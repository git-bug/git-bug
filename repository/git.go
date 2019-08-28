// Package repository contains helper methods for working with the Git repo.
package repository

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"path"
	"strconv"
	"strings"

	"github.com/blang/semver"
	"github.com/pkg/errors"

	"github.com/MichaelMure/git-bug/util/git"
	"github.com/MichaelMure/git-bug/util/lamport"
)

const createClockFile = "/git-bug/create-clock"
const editClockFile = "/git-bug/edit-clock"

// ErrNotARepo is the error returned when the git repo root wan't be found
var ErrNotARepo = errors.New("not a git repository")

var _ ClockedRepo = &GitRepo{}

// GitRepo represents an instance of a (local) git repository.
type GitRepo struct {
	Path        string
	createClock *lamport.Persisted
	editClock   *lamport.Persisted
}

// Run the given git command with the given I/O reader/writers, returning an error if it fails.
func (repo *GitRepo) runGitCommandWithIO(stdin io.Reader, stdout, stderr io.Writer, args ...string) error {
	repopath:=repo.Path
	if repopath==".git" {
		// seeduvax> trangely the git command sometimes fail for very unknown
		// reason wihtout this replacement.
		// observed with rev-list command when git-bug is called from git
		// hook script, even the same command with same args runs perfectly
		// when called directly from the same hook script. 
		repopath=""
	}
	// fmt.Printf("[%s] Running git %s\n", repopath, strings.Join(args, " "))

	cmd := exec.Command("git", args...)
	cmd.Dir = repopath
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	return cmd.Run()
}

// Run the given git command and return its stdout, or an error if the command fails.
func (repo *GitRepo) runGitCommandRaw(stdin io.Reader, args ...string) (string, string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := repo.runGitCommandWithIO(stdin, &stdout, &stderr, args...)
	return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), err
}

// Run the given git command and return its stdout, or an error if the command fails.
func (repo *GitRepo) runGitCommandWithStdin(stdin io.Reader, args ...string) (string, error) {
	stdout, stderr, err := repo.runGitCommandRaw(stdin, args...)
	if err != nil {
		if stderr == "" {
			stderr = "Error running git command: " + strings.Join(args, " ")
		}
		err = fmt.Errorf(stderr)
	}
	return stdout, err
}

// Run the given git command and return its stdout, or an error if the command fails.
func (repo *GitRepo) runGitCommand(args ...string) (string, error) {
	return repo.runGitCommandWithStdin(nil, args...)
}

// NewGitRepo determines if the given working directory is inside of a git repository,
// and returns the corresponding GitRepo instance if it is.
func NewGitRepo(path string, witnesser Witnesser) (*GitRepo, error) {
	repo := &GitRepo{Path: path}

	// Check the repo and retrieve the root path
	stdout, err := repo.runGitCommand("rev-parse", "--git-dir")

	// Now dir is fetched with "git rev-parse --git-dir". May be it can
	// still return nothing in some cases. Then empty stdout check is
	// kept.
	if err != nil || stdout == "" {
		return nil, ErrNotARepo
	}

	// Fix the path to be sure we are at the root
	repo.Path = stdout

	err = repo.LoadClocks()

	if err != nil {
		// No clock yet, trying to initialize them
		err = repo.createClocks()
		if err != nil {
			return nil, err
		}

		err = witnesser(repo)
		if err != nil {
			return nil, err
		}

		err = repo.WriteClocks()
		if err != nil {
			return nil, err
		}

		return repo, nil
	}

	return repo, nil
}

// InitGitRepo create a new empty git repo at the given path
func InitGitRepo(path string) (*GitRepo, error) {
	repo := &GitRepo{Path: path+"/.git"}
	err := repo.createClocks()
	if err != nil {
		return nil, err
	}

	_, err = repo.runGitCommand("init", path)
	if err != nil {
		return nil, err
	}

	return repo, nil
}

// InitBareGitRepo create a new --bare empty git repo at the given path
func InitBareGitRepo(path string) (*GitRepo, error) {
	repo := &GitRepo{Path: path}
	err := repo.createClocks()
	if err != nil {
		return nil, err
	}

	_, err = repo.runGitCommand("init", "--bare", path)
	if err != nil {
		return nil, err
	}

	return repo, nil
}

// GetPath returns the path to the repo.
func (repo *GitRepo) GetPath() string {
	return repo.Path
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
		elements := strings.Fields(line)
		if len(elements) != 3 {
			return nil, fmt.Errorf("unexpected output format: %s", line)
		}

		remotes[elements[0]] = elements[1]
	}

	return remotes, nil
}

// StoreConfig store a single key/value pair in the config of the repo
func (repo *GitRepo) StoreConfig(key string, value string) error {
	_, err := repo.runGitCommand("config", "--replace-all", key, value)

	return err
}

// ReadConfigs read all key/value pair matching the key prefix
func (repo *GitRepo) ReadConfigs(keyPrefix string) (map[string]string, error) {
	stdout, err := repo.runGitCommand("config", "--get-regexp", keyPrefix)

	//   / \
	//  / ! \
	// -------
	//
	// There can be a legitimate error here, but I see no portable way to
	// distinguish them from the git error that say "no matching value exist"
	if err != nil {
		return nil, nil
	}

	lines := strings.Split(stdout, "\n")

	result := make(map[string]string, len(lines))

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) != 2 {
			return nil, fmt.Errorf("bad git config: %s", line)
		}

		result[parts[0]] = parts[1]
	}

	return result, nil
}

func (repo *GitRepo) ReadConfigBool(key string) (bool, error) {
	val, err := repo.ReadConfigString(key)
	if err != nil {
		return false, err
	}

	return strconv.ParseBool(val)
}

func (repo *GitRepo) ReadConfigString(key string) (string, error) {
	stdout, err := repo.runGitCommand("config", "--get-all", key)

	//   / \
	//  / ! \
	// -------
	//
	// There can be a legitimate error here, but I see no portable way to
	// distinguish them from the git error that say "no matching value exist"
	if err != nil {
		return "", ErrNoConfigEntry
	}

	lines := strings.Split(stdout, "\n")

	if len(lines) == 0 {
		return "", ErrNoConfigEntry
	}
	if len(lines) > 1 {
		return "", ErrMultipleConfigEntry
	}

	return lines[0], nil
}

func (repo *GitRepo) rmSection(keyPrefix string) error {
	_, err := repo.runGitCommand("config", "--remove-section", keyPrefix)
	return err
}

func (repo *GitRepo) unsetAll(keyPrefix string) error {
	_, err := repo.runGitCommand("config", "--unset-all", keyPrefix)
	return err
}

// return keyPrefix section
// example: sectionFromKey(a.b.c.d) return a.b.c
func sectionFromKey(keyPrefix string) string {
	s := strings.Split(keyPrefix, ".")
	if len(s) == 1 {
		return keyPrefix
	}

	return strings.Join(s[:len(s)-1], ".")
}

// rmConfigs with git version lesser than 2.18
func (repo *GitRepo) rmConfigsGitVersionLT218(keyPrefix string) error {
	// try to remove key/value pair by key
	err := repo.unsetAll(keyPrefix)
	if err != nil {
		return repo.rmSection(keyPrefix)
	}

	m, err := repo.ReadConfigs(sectionFromKey(keyPrefix))
	if err != nil {
		return err
	}

	// if section doesn't have any left key/value remove the section
	if len(m) == 0 {
		return repo.rmSection(sectionFromKey(keyPrefix))
	}

	return nil
}

// RmConfigs remove all key/value pair matching the key prefix
func (repo *GitRepo) RmConfigs(keyPrefix string) error {
	// starting from git 2.18.0 sections are automatically deleted when the last existing
	// key/value is removed. Before 2.18.0 we should remove the section
	// see https://github.com/git/git/blob/master/Documentation/RelNotes/2.18.0.txt#L379
	lt218, err := repo.gitVersionLT218()
	if err != nil {
		return errors.Wrap(err, "getting git version")
	}

	if lt218 {
		return repo.rmConfigsGitVersionLT218(keyPrefix)
	}

	err = repo.unsetAll(keyPrefix)
	if err != nil {
		return repo.rmSection(keyPrefix)
	}

	return nil
}

func (repo *GitRepo) gitVersionLT218() (bool, error) {
	versionOut, err := repo.runGitCommand("version")
	if err != nil {
		return false, err
	}

	versionString := strings.Fields(versionOut)[2]
	version, err := semver.Make(versionString)
	if err != nil {
		return false, err
	}

	version218string := "2.18.0"
	gitVersion218, err := semver.Make(version218string)
	if err != nil {
		return false, err
	}

	return version.LT(gitVersion218), nil
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
func (repo *GitRepo) StoreData(data []byte) (git.Hash, error) {
	var stdin = bytes.NewReader(data)

	stdout, err := repo.runGitCommandWithStdin(stdin, "hash-object", "--stdin", "-w")

	return git.Hash(stdout), err
}

// ReadData will attempt to read arbitrary data from the given hash
func (repo *GitRepo) ReadData(hash git.Hash) ([]byte, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := repo.runGitCommandWithIO(nil, &stdout, &stderr, "cat-file", "-p", string(hash))

	if err != nil {
		return []byte{}, err
	}

	return stdout.Bytes(), nil
}

// StoreTree will store a mapping key-->Hash as a Git tree
func (repo *GitRepo) StoreTree(entries []TreeEntry) (git.Hash, error) {
	buffer := prepareTreeEntries(entries)

	stdout, err := repo.runGitCommandWithStdin(&buffer, "mktree")

	if err != nil {
		return "", err
	}

	return git.Hash(stdout), nil
}

// StoreCommit will store a Git commit with the given Git tree
func (repo *GitRepo) StoreCommit(treeHash git.Hash) (git.Hash, error) {
	stdout, err := repo.runGitCommand("commit-tree", string(treeHash))

	if err != nil {
		return "", err
	}

	return git.Hash(stdout), nil
}

// StoreCommitWithParent will store a Git commit with the given Git tree
func (repo *GitRepo) StoreCommitWithParent(treeHash git.Hash, parent git.Hash) (git.Hash, error) {
	stdout, err := repo.runGitCommand("commit-tree", string(treeHash),
		"-p", string(parent))

	if err != nil {
		return "", err
	}

	return git.Hash(stdout), nil
}

// UpdateRef will create or update a Git reference
func (repo *GitRepo) UpdateRef(ref string, hash git.Hash) error {
	_, err := repo.runGitCommand("update-ref", ref, string(hash))

	return err
}

// ListRefs will return a list of Git ref matching the given refspec
func (repo *GitRepo) ListRefs(refspec string) ([]string, error) {
	stdout, err := repo.runGitCommand("for-each-ref", "--format=%(refname)", refspec)

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
func (repo *GitRepo) ListCommits(ref string) ([]git.Hash, error) {
	stdout, err := repo.runGitCommand("rev-list", "--first-parent", "--reverse", ref)

	if err != nil {
		return nil, err
	}

	split := strings.Split(stdout, "\n")

	casted := make([]git.Hash, len(split))
	for i, line := range split {
		casted[i] = git.Hash(line)
	}

	return casted, nil

}

// ListEntries will return the list of entries in a Git tree
func (repo *GitRepo) ListEntries(hash git.Hash) ([]TreeEntry, error) {
	stdout, err := repo.runGitCommand("ls-tree", string(hash))

	if err != nil {
		return nil, err
	}

	return readTreeEntries(stdout)
}

// FindCommonAncestor will return the last common ancestor of two chain of commit
func (repo *GitRepo) FindCommonAncestor(hash1 git.Hash, hash2 git.Hash) (git.Hash, error) {
	stdout, err := repo.runGitCommand("merge-base", string(hash1), string(hash2))

	if err != nil {
		return "", err
	}

	return git.Hash(stdout), nil
}

// GetTreeHash return the git tree hash referenced in a commit
func (repo *GitRepo) GetTreeHash(commit git.Hash) (git.Hash, error) {
	stdout, err := repo.runGitCommand("rev-parse", string(commit)+"^{tree}")

	if err != nil {
		return "", err
	}

	return git.Hash(stdout), nil
}

// AddRemote add a new remote to the repository
// Not in the interface because it's only used for testing
func (repo *GitRepo) AddRemote(name string, url string) error {
	_, err := repo.runGitCommand("remote", "add", name, url)

	return err
}

func (repo *GitRepo) createClocks() error {
	createPath := path.Join(repo.Path, createClockFile)
	createClock, err := lamport.NewPersisted(createPath)
	if err != nil {
		return err
	}

	editPath := path.Join(repo.Path, editClockFile)
	editClock, err := lamport.NewPersisted(editPath)
	if err != nil {
		return err
	}

	repo.createClock = createClock
	repo.editClock = editClock

	return nil
}

// LoadClocks read the clocks values from the on-disk repo
func (repo *GitRepo) LoadClocks() error {
	createClock, err := lamport.LoadPersisted(repo.GetPath() + createClockFile)
	if err != nil {
		return err
	}

	editClock, err := lamport.LoadPersisted(repo.GetPath() + editClockFile)
	if err != nil {
		return err
	}

	repo.createClock = createClock
	repo.editClock = editClock
	return nil
}

// WriteClocks write the clocks values into the repo
func (repo *GitRepo) WriteClocks() error {
	err := repo.createClock.Write()
	if err != nil {
		return err
	}

	err = repo.editClock.Write()
	if err != nil {
		return err
	}

	return nil
}

// CreateTime return the current value of the creation clock
func (repo *GitRepo) CreateTime() lamport.Time {
	return repo.createClock.Time()
}

// CreateTimeIncrement increment the creation clock and return the new value.
func (repo *GitRepo) CreateTimeIncrement() (lamport.Time, error) {
	return repo.createClock.Increment()
}

// EditTime return the current value of the edit clock
func (repo *GitRepo) EditTime() lamport.Time {
	return repo.editClock.Time()
}

// EditTimeIncrement increment the edit clock and return the new value.
func (repo *GitRepo) EditTimeIncrement() (lamport.Time, error) {
	return repo.editClock.Increment()
}

// CreateWitness witness another create time and increment the corresponding clock
// if needed.
func (repo *GitRepo) CreateWitness(time lamport.Time) error {
	return repo.createClock.Witness(time)
}

// EditWitness witness another edition time and increment the corresponding clock
// if needed.
func (repo *GitRepo) EditWitness(time lamport.Time) error {
	return repo.editClock.Witness(time)
}
