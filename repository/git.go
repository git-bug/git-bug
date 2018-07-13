// Package repository contains helper methods for working with the Git repo.
package repository

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"github.com/MichaelMure/git-bug/util"
	"io"
	"os/exec"
	"strings"
)

// This is used to have a different staging area than the regular git index
// when creating data in git
const gitEnvConfig = "GIT_INDEX_FILE=BUG_STAGING_INDEX"

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

	//cmd.Env = append(cmd.Env, gitEnvConfig)

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

// PullRefs pull git refs from a remote
func (repo *GitRepo) PullRefs(remote string, refPattern string) error {
	fetchRefSpec := fmt.Sprintf("+%s:%s", refPattern, refPattern)
	_, err := repo.runGitCommand("fetch", remote, fetchRefSpec)

	// TODO: merge new data

	return err
}

// PushRefs push git refs to a remote
func (repo *GitRepo) PushRefs(remote string, refPattern string) error {
	// The push is liable to fail if the user forgot to do a pull first, so
	// we treat errors as user errors rather than fatal errors.
	_, err := repo.runGitCommand("push", remote, refPattern)
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

// StoreTree will store a mapping key-->Hash as a Git tree
func (repo *GitRepo) StoreTree(mapping map[string]util.Hash) (util.Hash, error) {
	var buffer bytes.Buffer

	for key, hash := range mapping {
		buffer.WriteString(fmt.Sprintf("100644 blob %s\t%s\n", hash, key))
	}

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
