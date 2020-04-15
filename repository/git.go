// Package repository contains helper methods for working with the Git repo.
package repository

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"path"
	"strings"

	"github.com/pkg/errors"

	"github.com/MichaelMure/git-bug/util/git"
	"github.com/MichaelMure/git-bug/util/lamport"
)

const (
	createClockFile = "/git-bug/create-clock"
	editClockFile   = "/git-bug/edit-clock"
)

var (
	// ErrNotARepo is the error returned when the git repo root wan't be found
	ErrNotARepo = errors.New("not a git repository")
)

var _ ClockedRepo = &GitRepo{}

// GitRepo represents an instance of a (local) git repository.
type GitRepo struct {
	Path        string
	createClock *lamport.Persisted
	editClock   *lamport.Persisted
}

// LocalConfig give access to the repository scoped configuration
func (repo *GitRepo) LocalConfig() Config {
	return newGitConfig(repo, false)
}

// GlobalConfig give access to the git global configuration
func (repo *GitRepo) GlobalConfig() Config {
	return newGitConfig(repo, true)
}

// Run the given git command with the given I/O reader/writers, returning an error if it fails.
func (repo *GitRepo) runGitCommandWithIO(stdin io.Reader, stdout, stderr io.Writer, args ...string) error {
	repopath := repo.Path
	if repopath == ".git" {
		// seeduvax> trangely the git command sometimes fail for very unknown
		// reason wihtout this replacement.
		// observed with rev-list command when git-bug is called from git
		// hook script, even the same command with same args runs perfectly
		// when called directly from the same hook script.
		repopath = ""
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
	repo := &GitRepo{Path: path + "/.git"}
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
	return repo.storeCommitRaw(treeHash)
}

// StoreCommitWithParent will store a Git commit with the given Git tree
func (repo *GitRepo) StoreCommitWithParent(treeHash git.Hash, parent git.Hash) (git.Hash, error) {
	return repo.storeCommitRaw(treeHash, "-p", string(parent))
}

func (repo *GitRepo) storeCommitRaw(treeHash git.Hash, extraArgs ...string) (git.Hash, error) {
	args := []string{"commit-tree"}

	// `git commit-tree` uses user.signingkey and gpg.program, but not commit.gpgsign.
	// We read commit.gpgsign ourselves and simply pass -S to `git commit-tree`.
	config := repo.LocalConfig()
	gpgsign, err := config.ReadBool("commit.gpgsign")
	if err != nil && err != ErrNoConfigEntry {
		// There are more than one entries, or some other error.
		return "", errors.Wrap(err, "failed to read local commit.gpgsign")
	}
	if gpgsign {
		args = append(args, "-S")
	}

	args = append(args, extraArgs...)

	args = append(args, string(treeHash))

	stdout, err := repo.runGitCommand(args...)

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

// WitnessCreate witness another create time and increment the corresponding clock
// if needed.
func (repo *GitRepo) WitnessCreate(time lamport.Time) error {
	return repo.createClock.Witness(time)
}

// WitnessEdit witness another edition time and increment the corresponding clock
// if needed.
func (repo *GitRepo) WitnessEdit(time lamport.Time) error {
	return repo.editClock.Witness(time)
}
