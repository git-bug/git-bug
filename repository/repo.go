// Package repository contains helper methods for working with a Git repo.
package repository

import (
	"errors"
	"io"

	"github.com/blevesearch/bleve"
	"github.com/go-git/go-billy/v5"
	"golang.org/x/crypto/openpgp"

	"github.com/MichaelMure/git-bug/util/lamport"
)

var (
	// ErrNotARepo is the error returned when the git repo root can't be found
	ErrNotARepo = errors.New("not a git repository")
	// ErrClockNotExist is the error returned when a clock can't be found
	ErrClockNotExist = errors.New("clock doesn't exist")
)

// Repo represents a source code repository.
type Repo interface {
	RepoConfig
	RepoKeyring
	RepoCommon
	RepoStorage
	RepoBleve
	RepoData

	Close() error
}

type RepoCommonStorage interface {
	RepoCommon
	RepoStorage
}

// ClockedRepo is a Repo that also has Lamport clocks
type ClockedRepo interface {
	Repo
	RepoClock
}

// RepoConfig access the configuration of a repository
type RepoConfig interface {
	// LocalConfig give access to the repository scoped configuration
	LocalConfig() Config

	// GlobalConfig give access to the global scoped configuration
	GlobalConfig() Config

	// AnyConfig give access to a merged local/global configuration
	AnyConfig() ConfigRead
}

// RepoKeyring give access to a user-wide storage for secrets
type RepoKeyring interface {
	// Keyring give access to a user-wide storage for secrets
	Keyring() Keyring
}

// RepoCommon represent the common function the we want all the repo to implement
type RepoCommon interface {
	// GetUserName returns the name the the user has used to configure git
	GetUserName() (string, error)

	// GetUserEmail returns the email address that the user has used to configure git.
	GetUserEmail() (string, error)

	// GetCoreEditor returns the name of the editor that the user has used to configure git.
	GetCoreEditor() (string, error)

	// GetRemotes returns the configured remotes repositories.
	GetRemotes() (map[string]string, error)
}

// RepoStorage give access to the filesystem
type RepoStorage interface {
	// LocalStorage return a billy.Filesystem giving access to $RepoPath/.git/git-bug
	LocalStorage() billy.Filesystem
}

// RepoBleve give access to Bleve to implement full-text search indexes.
type RepoBleve interface {
	// GetBleveIndex return a bleve.Index that can be used to index documents
	GetBleveIndex(name string) (bleve.Index, error)

	// ClearBleveIndex will wipe the given index
	ClearBleveIndex(name string) error
}

type Commit struct {
	Hash       Hash
	Parents    []Hash    // hashes of the parents, if any
	TreeHash   Hash      // hash of the git Tree
	SignedData io.Reader // if signed, reader for the signed data (likely, the serialized commit)
	Signature  io.Reader // if signed, reader for the (non-armored) signature
}

// RepoData give access to the git data storage
type RepoData interface {
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
	// The given hash could be from either a commit or a tree
	ReadTree(hash Hash) ([]TreeEntry, error)

	// StoreCommit will store a Git commit with the given Git tree
	StoreCommit(treeHash Hash, parents ...Hash) (Hash, error)

	// StoreCommit will store a Git commit with the given Git tree. If signKey is not nil, the commit
	// will be signed accordingly.
	StoreSignedCommit(treeHash Hash, signKey *openpgp.Entity, parents ...Hash) (Hash, error)

	// ReadCommit read a Git commit and returns some of its characteristic
	ReadCommit(hash Hash) (Commit, error)

	// GetTreeHash return the git tree hash referenced in a commit
	GetTreeHash(commit Hash) (Hash, error)

	// ResolveRef returns the hash of the target commit of the given ref
	ResolveRef(ref string) (Hash, error)

	// UpdateRef will create or update a Git reference
	UpdateRef(ref string, hash Hash) error

	// // MergeRef merge other into ref and update the reference
	// // If the update is not fast-forward, the callback treeHashFn will be called for the caller to generate
	// // the Tree to store in the merge commit.
	// MergeRef(ref string, otherRef string, treeHashFn func() Hash) error

	// RemoveRef will remove a Git reference
	RemoveRef(ref string) error

	// ListRefs will return a list of Git ref matching the given refspec
	ListRefs(refPrefix string) ([]string, error)

	// RefExist will check if a reference exist in Git
	RefExist(ref string) (bool, error)

	// CopyRef will create a new reference with the same value as another one
	CopyRef(source string, dest string) error

	// FindCommonAncestor will return the last common ancestor of two chain of commit
	// Deprecated
	FindCommonAncestor(commit1 Hash, commit2 Hash) (Hash, error)

	// ListCommits will return the list of tree hashes of a ref, in chronological order
	ListCommits(ref string) ([]Hash, error)
}

// RepoClock give access to Lamport clocks
type RepoClock interface {
	// AllClocks return all the known clocks
	AllClocks() (map[string]lamport.Clock, error)

	// GetOrCreateClock return a Lamport clock stored in the Repo.
	// If the clock doesn't exist, it's created.
	GetOrCreateClock(name string) (lamport.Clock, error)

	// Increment is equivalent to c = GetOrCreateClock(name) + c.Increment()
	Increment(name string) (lamport.Time, error)

	// Witness is equivalent to c = GetOrCreateClock(name) + c.Witness(time)
	Witness(name string, time lamport.Time) error
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

// TestedRepo is an extended ClockedRepo with function for testing only
type TestedRepo interface {
	ClockedRepo
	repoTest
}

// repoTest give access to test only functions
type repoTest interface {
	// AddRemote add a new remote to the repository
	AddRemote(name string, url string) error

	// GetLocalRemote return the URL to use to add this repo as a local remote
	GetLocalRemote() string

	// EraseFromDisk delete this repository entirely from the disk
	EraseFromDisk() error
}
