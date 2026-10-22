// Package repository contains helper methods for working with a Git repo.
package repository

import (
	"context"
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

// Repo represents a source code repository. See ComposedRepo to assemble one
// from its parts.
type Repo interface {
	RepoConfig
	RepoKeyring
	RepoStorage
	RepoIndex
	RepoData
	RepoRemote
	RepoBrowse
	RepoClock

	// Close closes every part of the repository that holds resources. The
	// repository can't be used anymore afterward.
	Close() error
}

type RepoCommonStorage interface {
	RepoConfig
	RepoStorage
}

// RepoConfig access the configuration of a repository, and the values of it
// that git-bug uses.
type RepoConfig interface {
	// Config give access to the git configuration
	Config() Config

	// GetUserName returns the name the user has used to configure git
	GetUserName() (string, error)

	// GetUserEmail returns the email address that the user has used to configure git.
	GetUserEmail() (string, error)

	// GetCoreEditor returns the name of the editor that the user has used to configure git.
	GetCoreEditor() (string, error)

	// GetRemotes returns the configured remotes repositories.
	GetRemotes() (map[string]string, error)
}

// RepoKeyring give access to a user-wide storage for secrets
type RepoKeyring interface {
	// Keyring give access to a user-wide storage for secrets
	Keyring() Keyring
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

// RepoIndex gives access to the indexes of the entities, one per namespace.
// They can be shared by several processes.
type RepoIndex interface {
	// GetIndex returns the index of a namespace, whose documents have the format
	// of the given version. Each version of a namespace has its own documents,
	// for processes of different versions to never read those of each other.
	GetIndex(namespace string, version uint) (Index, error)

	// Close releases the indexes opened by GetIndex.
	Close() error
}

// Index stores the excerpts of entities, and searches their texts. Each
// document records the commit it was built from, so that the index can tell
// what it is up to date with.
type Index interface {
	// NewBatch starts a batch of changes, applied as a whole by its Apply.
	NewBatch() IndexBatch

	// AllBuiltFrom returns, for each document, the commit it was built from.
	AllBuiltFrom(ctx context.Context) (map[string]Hash, error)

	// BuiltFrom returns the commit a document was built from, or ErrNotFound.
	BuiltFrom(ctx context.Context, id string) (Hash, error)

	// Excerpt returns the excerpt of a document, see IndexBatch.Set, and the
	// commit it was built from, or ErrNotFound.
	Excerpt(ctx context.Context, id string) (Hash, []byte, error)

	// Excerpts returns the excerpt of every document, and the commit each was
	// built from.
	Excerpts(ctx context.Context) (map[string]Hash, map[string][]byte, error)

	// Search returns the ids of the documents having any of the phrases in one
	// of their texts. A phrase is a sequence of words, matched in that order
	// regardless of case and inflection: "crash on login" matches "Crashes on
	// login". A single word is a phrase of one word.
	Search(ctx context.Context, phrases []string) ([]string, error)
}

// IndexBatch is a set of changes to an Index. Nothing is visible until Apply,
// which applies every change or none of them.
type IndexBatch interface {
	// Set stores the document of an entity, built from the given commit,
	// replacing any previous version:
	//   - excerpt is the excerpt of the entity, a JSON object, returned by
	//     Index.Excerpt with the same content, not necessarily the same bytes.
	//     Anything else fails Apply;
	//   - texts are what the document is found by in Search, such as a title
	//     and comments. They are searched separately: a phrase never spans two
	//     of them.
	Set(id string, commit Hash, excerpt []byte, texts []string) error

	// Remove deletes the document of an ID, if any.
	Remove(id string)

	// Apply applies the batch. A batch can't be used anymore afterward.
	Apply(ctx context.Context) error
}

type Commit struct {
	Hash       Hash
	Parents    []Hash    // hashes of the parents, if any
	TreeHash   Hash      // hash of the git Tree
	SignedData io.Reader // if signed, reader for the signed data (likely, the serialized commit)
	Signature  io.Reader // if signed, reader for the (non-armored) signature
}

// RepoRemote gives access to the network operations on a git remote, the
// only ones of a repository. They stop when ctx is done.
type RepoRemote interface {
	// FetchRefs retrieves the refs of the given namespaces (and the data they
	// point to) from a remote, and stores them as that remote's tracking refs.
	// Local refs are left untouched.
	FetchRefs(ctx context.Context, remote string, namespaces ...string) (string, error)

	// PushRefs sends the local refs of the given namespaces (and the data they
	// point to) to a remote, and updates that remote's tracking refs to match.
	PushRefs(ctx context.Context, remote string, namespaces ...string) (string, error)
}

// RepoData give access to the git data storage
type RepoData interface {
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

	// UpdateRef points a local ref to commit, only if it currently points to old.
	// An empty old means that the ref must not exist yet.
	// Returns ErrRefChanged otherwise, and the ref is left unchanged.
	UpdateRef(namespace string, key string, old Hash, commit Hash) error

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
	ListCommits(ctx context.Context, commit Hash) ([]Hash, error)
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
// All methods accepting a rev parameter resolve it in order:
// refs/heads/<rev>, refs/tags/<rev>, full ref name, raw commit hash.
type RepoBrowse interface {
	// Branches returns all local branches (refs/heads/*).
	// All other ref namespaces — including git-bug's internal refs
	// (refs/bugs/, refs/identities/, …) — are excluded.
	Branches(ctx context.Context) ([]BranchInfo, error)

	// Tags returns all tags (refs/tags/*).
	// All other ref namespaces are excluded.
	Tags(ctx context.Context) ([]TagInfo, error)

	// TreeAtPath returns the entries of the directory at path under rev.
	// An empty path returns the root tree.
	// Returns ErrNotFound if rev or path does not exist, or if path
	// resolves to a blob rather than a tree.
	// Symlinks appear as entries with ObjectType Symlink; they are not followed.
	TreeAtPath(ctx context.Context, rev, path string) ([]TreeEntry, error)

	// BlobAtPath returns the raw content, byte size, and git object hash of
	// the file at path under rev. Returns ErrNotFound if rev or path does
	// not exist, or if path resolves to a tree. Symlinks are not followed.
	// The caller must close the reader.
	BlobAtPath(ctx context.Context, rev, path string) (io.ReadCloser, int64, Hash, error)

	// CommitLog returns at most limit commits reachable from rev, filtered
	// to those touching path (empty = unrestricted). after is an exclusive
	// cursor; pass Hash("") for no cursor. since and until bound the author
	// date (inclusive); pass nil for no bound. Merge commits appear once,
	// compared against the first parent only.
	CommitLog(ctx context.Context, rev, path string, limit int, after Hash, since, until *time.Time) ([]CommitMeta, error)

	// LastCommitForEntries returns the most recent commit that touched each
	// name in the directory at path under rev. Entries not resolved within
	// the implementation's depth limit are silently absent from the result.
	LastCommitForEntries(ctx context.Context, rev, path string, names []string) (map[string]CommitMeta, error)

	// CommitDetail returns the full metadata and changed-file list for a
	// single commit identified by its hash. Diffs against the first parent
	// only; the initial commit is diffed against the empty tree.
	CommitDetail(ctx context.Context, hash Hash) (CommitDetail, error)

	// CommitFileDiff returns the unified diff for a single file in a commit
	// identified by its hash. Diffs against the first parent only; the
	// initial commit is diffed against the empty tree.
	CommitFileDiff(ctx context.Context, hash Hash, filePath string) (FileDiff, error)

	// Head returns the commit that HEAD currently points to.
	// Returns ErrNotFound if HEAD cannot be resolved to a commit, including
	// for an empty (unborn) repository.
	Head(ctx context.Context) (RefMeta, error)
}
