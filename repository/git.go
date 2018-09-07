// Package repository contains helper methods for working with the Git repo.
package repository

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/MichaelMure/git-bug/util"
)

const createClockFile = "/.git/git-bug/create-clock"
const editClockFile = "/.git/git-bug/edit-clock"

// ErrNotARepo is the error returned when the git repo root wan't be found
var ErrNotARepo = errors.New("not a git repository")

// GitRepo represents an instance of a (local) git repository.
type GitRepo struct {
	Path        string
	createClock *util.PersistedLamport
	editClock   *util.PersistedLamport
}

// Run the given git command with the given I/O reader/writers, returning an error if it fails.
func (repo *GitRepo) runGitCommandWithIO(stdin io.Reader, stdout, stderr io.Writer, args ...string) error {
	//fmt.Println("Running git", strings.Join(args, " "))

	cmd := exec.Command("git", args...)
	cmd.Dir = repo.Path
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

// Run the given git command using the same stdin, stdout, and stderr as the review tool.
func (repo *GitRepo) runGitCommandInline(args ...string) error {
	return repo.runGitCommandWithIO(os.Stdin, os.Stdout, os.Stderr, args...)
}

// NewGitRepo determines if the given working directory is inside of a git repository,
// and returns the corresponding GitRepo instance if it is.
func NewGitRepo(path string, witnesser func(repo *GitRepo) error) (*GitRepo, error) {
	repo := &GitRepo{Path: path}

	// Check the repo and retrieve the root path
	stdout, err := repo.runGitCommand("rev-parse", "--show-toplevel")

	if err != nil {
		return nil, ErrNotARepo
	}

	// Fix the path to be sure we are at the root
	repo.Path = stdout

	err = repo.LoadClocks()

	if err != nil {
		// No clock yet, trying to initialize them
		repo.createClocks()

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
	repo := &GitRepo{Path: path}
	repo.createClocks()

	_, err := repo.runGitCommand("init", path)
	if err != nil {
		return nil, err
	}

	return repo, nil
}

// InitBareGitRepo create a new --bare empty git repo at the given path
func InitBareGitRepo(path string) (*GitRepo, error) {
	repo := &GitRepo{Path: path}
	repo.createClocks()

	_, err := repo.runGitCommand("init", "--bare", path)
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
		return stdout + stderr, fmt.Errorf("failed to push to the remote '%s': %v", remote, err)
	}
	return stdout + stderr, nil
}

// StoreData will store arbitrary data and return the corresponding hash
func (repo *GitRepo) StoreData(data []byte) (util.Hash, error) {
	var stdin = bytes.NewReader(data)

	stdout, err := repo.runGitCommandWithStdin(stdin, "hash-object", "--stdin", "-w")

	return util.Hash(stdout), err
}

// ReadData will attempt to read arbitrary data from the given hash
func (repo *GitRepo) ReadData(hash util.Hash) ([]byte, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := repo.runGitCommandWithIO(nil, &stdout, &stderr, "cat-file", "-p", string(hash))

	if err != nil {
		return []byte{}, err
	}

	return stdout.Bytes(), nil
}

// StoreTree will store a mapping key-->Hash as a Git tree
func (repo *GitRepo) StoreTree(entries []TreeEntry) (util.Hash, error) {
	buffer := prepareTreeEntries(entries)

	stdout, err := repo.runGitCommandWithStdin(&buffer, "mktree")

	if err != nil {
		return "", err
	}

	return util.Hash(stdout), nil
}

// StoreCommit will store a Git commit with the given Git tree
func (repo *GitRepo) StoreCommit(treeHash util.Hash) (util.Hash, error) {
	stdout, err := repo.runGitCommand("commit-tree", string(treeHash))

	if err != nil {
		return "", err
	}

	return util.Hash(stdout), nil
}

// StoreCommitWithParent will store a Git commit with the given Git tree
func (repo *GitRepo) StoreCommitWithParent(treeHash util.Hash, parent util.Hash) (util.Hash, error) {
	stdout, err := repo.runGitCommand("commit-tree", string(treeHash),
		"-p", string(parent))

	if err != nil {
		return "", err
	}

	return util.Hash(stdout), nil
}

// UpdateRef will create or update a Git reference
func (repo *GitRepo) UpdateRef(ref string, hash util.Hash) error {
	_, err := repo.runGitCommand("update-ref", ref, string(hash))

	return err
}

// ListRefs will return a list of Git ref matching the given refspec
func (repo *GitRepo) ListRefs(refspec string) ([]string, error) {
	stdout, err := repo.runGitCommand("for-each-ref", "--format=%(refname)", refspec)

	if err != nil {
		return nil, err
	}

	splitted := strings.Split(stdout, "\n")

	if len(splitted) == 1 && splitted[0] == "" {
		return []string{}, nil
	}

	return splitted, nil
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
func (repo *GitRepo) ListCommits(ref string) ([]util.Hash, error) {
	stdout, err := repo.runGitCommand("rev-list", "--first-parent", "--reverse", ref)

	if err != nil {
		return nil, err
	}

	splitted := strings.Split(stdout, "\n")

	casted := make([]util.Hash, len(splitted))
	for i, line := range splitted {
		casted[i] = util.Hash(line)
	}

	return casted, nil

}

// ListEntries will return the list of entries in a Git tree
func (repo *GitRepo) ListEntries(hash util.Hash) ([]TreeEntry, error) {
	stdout, err := repo.runGitCommand("ls-tree", string(hash))

	if err != nil {
		return nil, err
	}

	return readTreeEntries(stdout)
}

// FindCommonAncestor will return the last common ancestor of two chain of commit
func (repo *GitRepo) FindCommonAncestor(hash1 util.Hash, hash2 util.Hash) (util.Hash, error) {
	stdout, err := repo.runGitCommand("merge-base", string(hash1), string(hash2))

	if err != nil {
		return "", nil
	}

	return util.Hash(stdout), nil
}

// GetTreeHash return the git tree hash referenced in a commit
func (repo *GitRepo) GetTreeHash(commit util.Hash) (util.Hash, error) {
	stdout, err := repo.runGitCommand("rev-parse", string(commit)+"^{tree}")

	if err != nil {
		return "", nil
	}

	return util.Hash(stdout), nil
}

// AddRemote add a new remote to the repository
// Not in the interface because it's only used for testing
func (repo *GitRepo) AddRemote(name string, url string) error {
	_, err := repo.runGitCommand("remote", "add", name, url)

	return err
}

func (repo *GitRepo) createClocks() {
	createPath := path.Join(repo.Path, createClockFile)
	repo.createClock = util.NewPersistedLamport(createPath)

	editPath := path.Join(repo.Path, editClockFile)
	repo.editClock = util.NewPersistedLamport(editPath)
}

func (repo *GitRepo) LoadClocks() error {
	createClock, err := util.LoadPersistedLamport(repo.GetPath() + createClockFile)
	if err != nil {
		return err
	}

	editClock, err := util.LoadPersistedLamport(repo.GetPath() + editClockFile)
	if err != nil {
		return err
	}

	repo.createClock = createClock
	repo.editClock = editClock
	return nil
}

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

func (repo *GitRepo) CreateTimeIncrement() (util.LamportTime, error) {
	return repo.createClock.Increment()
}

func (repo *GitRepo) EditTimeIncrement() (util.LamportTime, error) {
	return repo.editClock.Increment()
}

func (repo *GitRepo) CreateWitness(time util.LamportTime) error {
	return repo.createClock.Witness(time)
}

func (repo *GitRepo) EditWitness(time util.LamportTime) error {
	return repo.editClock.Witness(time)
}

func (repo *GitRepo) GC() error {
	_, err := repo.runGitCommand("gc")

	return err
}

func (repo *GitRepo) GCAggressive() error {
	_, err := repo.runGitCommand("gc", "--aggressive")

	return err
}
