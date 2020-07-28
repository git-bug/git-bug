// Package repository contains helper methods for working with a Git repo.
package repository

import (
	"bytes"
	"errors"
	"strings"

	"github.com/MichaelMure/git-bug/util/lamport"
)

var (
	// ErrNotARepo is the error returned when the git repo root wan't be found
	ErrNotARepo = errors.New("not a git repository")
	// ErrClockNotExist is the error returned when a clock can't be found
	ErrClockNotExist = errors.New("clock doesn't exist")
)

// RepoConfig access the configuration of a repository
type RepoConfig interface {
	// LocalConfig give access to the repository scoped configuration
	LocalConfig() Config
}

// RepoKeyring give access to a user-wide storage for secrets
type RepoKeyring interface {
	// Keyring give access to a user-wide storage for secrets
	Keyring() Keyring
}

// RepoCommon represent the common function the we want all the repo to implement
type RepoCommon interface {
	// GetPath returns the path to the repo.
	GetPath() string

	// GetUserName returns the name the the user has used to configure git
	GetUserName() (string, error)

	// GetUserEmail returns the email address that the user has used to configure git.
	GetUserEmail() (string, error)

	// GetCoreEditor returns the name of the editor that the user has used to configure git.
	GetCoreEditor() (string, error)

	// GetRemotes returns the configured remotes repositories.
	GetRemotes() (map[string]string, error)
}

// Repo represents a source code repository.
type Repo interface {
	RepoConfig
	RepoKeyring
	RepoCommon

	// FetchRefs fetch git refs from a remote
	FetchRefs(remote string, refSpec string) (string, error)

	// PushRefs push git refs to a remote
	PushRefs(remote string, refSpec string) (string, error)

	// StoreData will store arbitrary data and return the corresponding hash
	StoreData(data []byte) (Hash, error)

	// ReadData will attempt to read arbitrary data from the given hash
	ReadData(hash Hash) ([]byte, error)

	// StoreTree will store a mapping key-->Hash as a Git tree
	StoreTree(mapping []TreeEntry) (Hash, error)

	// ReadTree will return the list of entries in a Git tree
	ReadTree(hash Hash) ([]TreeEntry, error)

	// StoreCommit will store a Git commit with the given Git tree
	StoreCommit(treeHash Hash) (Hash, error)

	// StoreCommit will store a Git commit with the given Git tree
	StoreCommitWithParent(treeHash Hash, parent Hash) (Hash, error)

	// GetTreeHash return the git tree hash referenced in a commit
	GetTreeHash(commit Hash) (Hash, error)

	// FindCommonAncestor will return the last common ancestor of two chain of commit
	FindCommonAncestor(commit1 Hash, commit2 Hash) (Hash, error)

	// UpdateRef will create or update a Git reference
	UpdateRef(ref string, hash Hash) error

	// RemoveRef will remove a Git reference
	RemoveRef(ref string) error

	// ListRefs will return a list of Git ref matching the given refspec
	ListRefs(refspec string) ([]string, error)

	// RefExist will check if a reference exist in Git
	RefExist(ref string) (bool, error)

	// CopyRef will create a new reference with the same value as another one
	CopyRef(source string, dest string) error

	// ListCommits will return the list of tree hashes of a ref, in chronological order
	ListCommits(ref string) ([]Hash, error)
}

// ClockedRepo is a Repo that also has Lamport clocks
type ClockedRepo interface {
	Repo

	// GetOrCreateClock return a Lamport clock stored in the Repo.
	// If the clock doesn't exist, it's created.
	GetOrCreateClock(name string) (lamport.Clock, error)
}

// ClockLoader hold which logical clock need to exist for an entity and
// how to create them if they don't.
type ClockLoader struct {
	// Clocks hold the name of all the clocks this loader deal with.
	// Those clocks will be checked when the repo load. If not present or broken,
	// Witnesser will be used to create them.
	Clocks []string
	// Witnesser is a function that will initialize the clocks of a repo
	// from scratch
	Witnesser func(repo ClockedRepo) error
}

func prepareTreeEntries(entries []TreeEntry) bytes.Buffer {
	var buffer bytes.Buffer

	for _, entry := range entries {
		buffer.WriteString(entry.Format())
	}

	return buffer
}

func readTreeEntries(s string) ([]TreeEntry, error) {
	split := strings.Split(strings.TrimSpace(s), "\n")

	casted := make([]TreeEntry, len(split))
	for i, line := range split {
		if line == "" {
			continue
		}

		entry, err := ParseTreeEntry(line)

		if err != nil {
			return nil, err
		}

		casted[i] = entry
	}

	return casted, nil
}

// TestedRepo is an extended ClockedRepo with function for testing only
type TestedRepo interface {
	ClockedRepo

	// AddRemote add a new remote to the repository
	AddRemote(name string, url string) error
}
