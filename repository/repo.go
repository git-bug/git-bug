// Package repository contains helper methods for working with a Git repo.
package repository

import (
	"errors"
	"io"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/go-git/go-billy/v5"

	"github.com/git-bug/git-bug/util/lamport"
)

var (
	// ErrNotARepo is the error returned when the git repo root can't be found
	ErrNotARepo = errors.New("not a git repository")
	// ErrClockNotExist is the error returned when a clock can't be found
	ErrClockNotExist = lamport.ErrClockNotExist
	// ErrNotFound is the error returned when a git object can't be found
	ErrNotFound = errors.New("ref not found")
	// ErrRefChanged is the error returned when a ref doesn't have the expected value anymore
	ErrRefChanged = errors.New("ref changed")
)

// Repo represents a source code repository.
type Repo interface {
	RepoConfig
	RepoKeyring
	RepoCommon
	RepoStorage
	RepoIndex
	RepoData
	RepoBrowse

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

// RepoCommon represent the common function we want all repos to implement
type RepoCommon interface {
	// GetUserName returns the name the user has used to configure git
	GetUserName() (string, error)

	// GetUserEmail returns the email address that the user has used to configure git.
	GetUserEmail() (string, error)

	// GetCoreEditor returns the name of the editor that the user has used to configure git.
	GetCoreEditor() (string, error)

	// GetRemotes returns the configured remotes repositories.
	GetRemotes() (map[string]string, error)
}

type LocalStorage interface {
	billy.Filesystem
	RemoveAll(path string) error
}

// RepoStorage give access to the filesystem
type RepoStorage interface {
	// LocalStorage return a billy.Filesystem giving access to $RepoPath/.git/git-bug
	LocalStorage() LocalStorage
}

// RepoIndex gives access to full-text search indexes
type RepoIndex interface {
	GetIndex(name string) (Index, error)
}

// Index is a full-text search index
type Index interface {
	// IndexOne indexes one document, for the given ID. If the document already exist,
	// it replaces it.
	IndexOne(id string, texts []string) error

	// IndexBatch start a batch indexing. The returned indexer function is used the same
	// way as IndexOne, and the closer function complete the batch insertion.
	IndexBatch() (indexer func(id string, texts []string) error, closer func() error)

	// Search returns the list of IDs matching the given terms.
	Search(terms []string) (ids []string, err error)

	// DocCount returns the number of document in the index.
	DocCount() (uint64, error)

	// Remove delete one document in the index.
	Remove(id string) error

	// Clear empty the index.
	Clear() error

	// Close closes the index and make sure everything is safely written. After this call
	// the index can't be used anymore.
	Close() error
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
	// FetchRefs retrieves the refs of the given namespaces (and the data they
	// point to) from a remote, and stores them as that remote's tracking refs.
	// Local refs are left untouched.
	FetchRefs(remote string, namespaces ...string) (string, error)

	// PushRefs sends the local refs of the given namespaces (and the data they
	// point to) to a remote, and updates that remote's tracking refs to match.
	PushRefs(remote string, namespaces ...string) (string, error)

	// StoreData will store arbitrary data and return the corresponding hash
	StoreData(data []byte) (Hash, error)

	// ReadData returns a reader for arbitrary data associated with the given hash.
	// Returns ErrNotFound if not found.
	// The caller must close the reader.
	ReadData(hash Hash) (io.ReadCloser, error)

	// StoreTree will store a mapping key-->Hash as a Git tree
	StoreTree(mapping []TreeEntry) (Hash, error)

	// ReadTree will return the list of entries in a Git tree
	// The given hash could be from either a commit or a tree
	// Returns ErrNotFound if not found.
	ReadTree(hash Hash) ([]TreeEntry, error)

	// StoreCommit will store a Git commit with the given Git tree
	StoreCommit(treeHash Hash, parents ...Hash) (Hash, error)

	// StoreSignedCommit will store a Git commit with the given Git tree. If signKey is not nil, the commit
	// will be signed accordingly.
	StoreSignedCommit(treeHash Hash, signKey *openpgp.Entity, parents ...Hash) (Hash, error)

	// ReadCommit read a Git commit and returns some of its characteristic
	// Returns ErrNotFound if not found.
	ReadCommit(hash Hash) (Commit, error)

	// ListRefs returns every local ref of a namespace, by key.
	ListRefs(namespace string) (map[string]Hash, error)

	// ResolveRef returns the commit a local ref points to.
	// Returns ErrNotFound if it doesn't exist.
	ResolveRef(namespace string, key string) (Hash, error)

	// UpdateRef sets a local ref to hash, only if it currently is old.
	// An empty old means that the ref must not exist yet; that check is not
	// atomic, two concurrent creations of the same ref can both succeed.
	// Returns ErrRefChanged otherwise, and the ref is left unchanged.
	UpdateRef(namespace string, key string, old Hash, hash Hash) error

	// RemoveRef deletes a local ref.
	// RemoveRef is idempotent.
	RemoveRef(namespace string, key string) error

	// A remote's tracking refs are the local record of that remote's refs, as
	// of the last fetch or push. Reading or removing them never contacts the
	// remote.

	// ListTrackingRefs returns every tracking ref of a namespace for a remote,
	// by key.
	ListTrackingRefs(remote string, namespace string) (map[string]Hash, error)

	// ResolveTrackingRef returns the commit a tracking ref points to.
	// Returns ErrNotFound if it doesn't exist.
	ResolveTrackingRef(remote string, namespace string, key string) (Hash, error)

	// RemoveTrackingRef deletes a tracking ref.
	// RemoveTrackingRef is idempotent.
	RemoveTrackingRef(remote string, namespace string, key string) error

	// ListCommits returns the hashes of commit and all its ancestors, in
	// chronological order.
	ListCommits(commit Hash) ([]Hash, error)
}

// RepoClock give access to Lamport clocks
type RepoClock interface {
	// AllClocks return all the known clocks
	AllClocks() (map[string]lamport.Clock, error)

	// GetClock return a Lamport clock stored in the Repo, or ErrClockNotExist
	// if there is none.
	GetClock(name string) (lamport.Clock, error)

	// GetOrCreateClock return a Lamport clock stored in the Repo, creating it
	// at the initial time if there is none, or if the existing one is corrupted.
	// The initial time must be at least every time already issued or witnessed
	// for that clock: see lamport.GetOrCreatePersistedClock.
	GetOrCreateClock(name string, initial lamport.Time) (lamport.Clock, error)

	// Increment is equivalent to c = GetClock(name) + c.Increment()
	Increment(name string) (lamport.Time, error)

	// Witness is equivalent to c = GetClock(name) + c.Witness(time)
	Witness(name string, time lamport.Time) error
}

// RepoBrowse is implemented by all Repo implementations and provides
// code-browsing endpoints (file tree, history, diffs).
//
// All methods accepting a ref parameter resolve it in order:
// refs/heads/<ref>, refs/tags/<ref>, full ref name, raw commit hash.
type RepoBrowse interface {
	// Branches returns all local branches (refs/heads/*).
	// All other ref namespaces — including git-bug's internal refs
	// (refs/bugs/, refs/identities/, …) — are excluded.
	Branches() ([]BranchInfo, error)

	// Tags returns all tags (refs/tags/*).
	// All other ref namespaces are excluded.
	Tags() ([]TagInfo, error)

	// TreeAtPath returns the entries of the directory at path under ref.
	// An empty path returns the root tree.
	// Returns ErrNotFound if ref or path does not exist, or if path
	// resolves to a blob rather than a tree.
	// Symlinks appear as entries with ObjectType Symlink; they are not followed.
	TreeAtPath(ref, path string) ([]TreeEntry, error)

	// BlobAtPath returns the raw content, byte size, and git object hash of
	// the file at path under ref. Returns ErrNotFound if ref or path does
	// not exist, or if path resolves to a tree. Symlinks are not followed.
	// The caller must close the reader.
	BlobAtPath(ref, path string) (io.ReadCloser, int64, Hash, error)

	// CommitLog returns at most limit commits reachable from ref, filtered
	// to those touching path (empty = unrestricted). after is an exclusive
	// cursor; pass Hash("") for no cursor. since and until bound the author
	// date (inclusive); pass nil for no bound. Merge commits appear once,
	// compared against the first parent only.
	CommitLog(ref, path string, limit int, after Hash, since, until *time.Time) ([]CommitMeta, error)

	// LastCommitForEntries returns the most recent commit that touched each
	// name in the directory at path under ref. Entries not resolved within
	// the implementation's depth limit are silently absent from the result.
	LastCommitForEntries(ref, path string, names []string) (map[string]CommitMeta, error)

	// CommitDetail returns the full metadata and changed-file list for a
	// single commit identified by its hash. Diffs against the first parent
	// only; the initial commit is diffed against the empty tree.
	CommitDetail(hash Hash) (CommitDetail, error)

	// CommitFileDiff returns the unified diff for a single file in a commit
	// identified by its hash. Diffs against the first parent only; the
	// initial commit is diffed against the empty tree.
	CommitFileDiff(hash Hash, filePath string) (FileDiff, error)

	// Head returns the commit that HEAD currently points to.
	// Returns ErrNotFound if HEAD cannot be resolved to a commit, including
	// for an empty (unborn) repository.
	Head() (RefMeta, error)
}

// TestedRepo is an extended ClockedRepo with functions for testing only
type TestedRepo interface {
	ClockedRepo
	repoTest
}

// repoTest give access to test-only functions
type repoTest interface {
	// AddRemote add a new remote to the repository
	AddRemote(name string, url string) error

	// GetLocalRemote return the URL to use to add this repo as a local remote
	GetLocalRemote() string

	// EraseFromDisk delete this repository entirely from the disk
	EraseFromDisk() error

	// SetBranch points the branch name to commit, creating it if needed.
	SetBranch(name string, commit Hash) error

	// SetTag points the lightweight tag name to commit, creating it if needed.
	SetTag(name string, commit Hash) error
}
