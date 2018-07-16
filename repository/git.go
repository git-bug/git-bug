// Package repository contains helper methods for working with the Git repo.
package repository

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"github.com/MichaelMure/git-bug/util"
	"io"
	"os"
	"os/exec"
	"strings"
)

// GitRepo represents an instance of a (local) git repository.
type GitRepo struct {
	Path string
}

// Run the given git command with the given I/O reader/writers, returning an error if it fails.
func (repo *GitRepo) runGitCommandWithIO(stdin io.Reader, stdout, stderr io.Writer, args ...string) error {
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
func NewGitRepo(path string) (*GitRepo, error) {
	repo := &GitRepo{Path: path}
	_, err := repo.runGitCommand("rev-parse")
	if err == nil {
		return repo, nil
	}
	if _, ok := err.(*exec.ExitError); ok {
		return nil, err
	}
	return nil, err
}

// GetPath returns the path to the repo.
func (repo *GitRepo) GetPath() string {
	return repo.Path
}

// GetRepoStateHash returns a hash which embodies the entire current state of a repository.
func (repo *GitRepo) GetRepoStateHash() (string, error) {
	stateSummary, err := repo.runGitCommand("show-ref")
	return fmt.Sprintf("%x", sha1.Sum([]byte(stateSummary))), err
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
func (repo *GitRepo) FetchRefs(remote, refPattern, remoteRefPattern string) error {
	remoteRefSpec := fmt.Sprintf(remoteRefPattern, remote)
	fetchRefSpec := fmt.Sprintf("%s:%s", refPattern, remoteRefSpec)
	err := repo.runGitCommandInline("fetch", remote, fetchRefSpec)

	if err != nil {
		return fmt.Errorf("failed to fetch from the remote '%s': %v", remote, err)
	}

	return err
}

// PushRefs push git refs to a remote
func (repo *GitRepo) PushRefs(remote string, refPattern string) error {
	err := repo.runGitCommandInline("push", remote, refPattern)

	if err != nil {
		return fmt.Errorf("failed to push to the remote '%s': %v", remote, err)
	}
	return nil
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
	// the format option will strip the ref name to keep only the last part (ie, the bug id)
	stdout, err := repo.runGitCommand("for-each-ref", "--format=%(refname:lstrip=-1)", refspec)

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

// Return the git tree hash referenced in a commit
func (repo *GitRepo) GetTreeHash(commit util.Hash) (util.Hash, error) {
	stdout, err := repo.runGitCommand("rev-parse", string(commit)+"^{tree}")

	if err != nil {
		return "", nil
	}

	return util.Hash(stdout), nil
}
