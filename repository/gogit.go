package repository

// GoGitRepo works around go-git defects in several places. None is meant to
// remain: each goes once go-git fixes the defect and git-bug uses that version.
// The canary tests, in gogit_storage_test.go and gogit_refs_test.go, fail when
// go-git does, for the defects a test can show: not for concurrency, which
// crashes the whole test binary, the resource leak, or the transports.
//
//   - The configuration: go-git parses all of it whenever it reads it, opening
//     the repository included, and fails on what git accepts, such as a branch
//     merging a pull request's ref (https://github.com/go-git/go-git/issues/530)
//     or a negative fetch refspec (https://github.com/go-git/go-git/issues/1927).
//     gogitStorage.Config hides everything but core and extensions from go-git.
//     GoGitRepo then takes the configuration, read with gitconfig, only to build
//     the remotes of FetchRefs and PushRefs, and StoreSignedCommit writes no
//     author.
//   - The packfile index: go-git never notices the packs other processes add or
//     remove. gogitStorage rebuilds the index and retries a read when they
//     changed, keeps it built at all times, as go-git's pack writer crashes
//     otherwise, and refreshes it before iterating objects, which crashes on a
//     pack go-git hasn't indexed (fixed in v6 by
//     https://github.com/go-git/go-git/pull/2448).
//   - Concurrency: go-git isn't safe for concurrent use
//     (https://github.com/go-git/go-git/issues/208). rMutex serializes
//     GoGitRepo's reads, and gogitStorage's mu every access to the packfile
//     index, go-git's fetch and push included.
//   - The refs of git-bug's namespaces: go-git writes and lists them wrongly,
//     such as https://github.com/go-git/go-git/issues/2399. gogit_refs.go writes
//     and lists them itself, as git does.
//   - The remotes: go-git doesn't implement all of git's transports and their
//     configuration, such as parts of the SSH configuration, credential helpers
//     or remote.<name>.pushurl. git-bug.git-transport=git runs the git binary
//     for the remotes instead, see GitRepo, and FetchRefs and PushRefs suggest
//     it when they fail.
//   - A possible resource leak in gogit.Open, fixed in v6.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	fdiff "github.com/go-git/go-git/v5/plumbing/format/diff"
	"github.com/go-git/go-git/v5/plumbing/object"
	lru "github.com/hashicorp/golang-lru/v2"
	"resenje.org/singleflight"
)

// lastCommitDepthLimit is the maximum number of commits walked by
// LastCommitForEntries. Entries not found within this horizon are omitted from
// the result rather than stalling the caller indefinitely.
const lastCommitDepthLimit = 1000

// lastCommitCacheSize is the number of (resolvedHash, dirPath) pairs kept in
// the LRU cache for LastCommitForEntries. Each entry holds one CommitMeta per
// directory entry (≈ a few KB for a typical directory), so 256 slots ≈ a few
// MB of memory at most.
const lastCommitCacheSize = 256

var _ RepoData = &GoGitRepo{}
var _ RepoRemote = &GoGitRepo{}
var _ RepoBrowse = &GoGitRepo{}

// GoGitRepo is the git data of a Repo, in a git directory on disk, with go-git.
//
// It takes the repository's configuration, as read by gitconfig, only to work
// around go-git failing on configurations git accepts: see gogitStorage.Config.
// It's not meant to remain, and goes with that workaround.
type GoGitRepo struct {
	// Unfortunately, some parts of go-git are not thread-safe so we have to cover them with a big fat mutex here.
	// See https://github.com/go-git/go-git/issues/48
	// See https://github.com/go-git/go-git/issues/208
	// See https://github.com/go-git/go-git/pull/186
	rMutex sync.Mutex
	r      *gogit.Repository
	path   string
	config RepoConfig

	// lastCommitCache caches LastCommitForEntries results keyed by
	// "<treeHash>\x00<path>". Git trees are content-addressed and
	// immutable, so entries never need invalidation and can be shared
	// across refs that point to the same directory tree. The LRU bounds
	// memory to lastCommitCacheSize unique (treeHash, directory) pairs.
	lastCommitCache *lru.Cache[string, map[string]CommitMeta]
	// lastCommitSF deduplicates concurrent walks for the same cache key so
	// that a cold cache under parallel requests triggers only one history walk.
	lastCommitSF singleflight.Group[string, map[string]CommitMeta]
}

// OpenGoGitRepo opens the repository of the git directory gitDir, see
// FindGitDir, whose configuration is config. It fails with ErrNotARepo if
// gitDir isn't one.
func OpenGoGitRepo(gitDir string, config RepoConfig) (*GoGitRepo, error) {
	gitDir, err := filepath.Abs(gitDir)
	if err != nil {
		return nil, err
	}
	ok, err := isGitDir(gitDir)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNotARepo, gitDir)
	}

	// TODO: the possible resource leak is an existing issue in go-git v5
	// it's fixed in v6
	r, err := gogit.Open(newGogitStorage(gitDir, config), nil)
	if err != nil {
		return nil, err
	}

	return &GoGitRepo{
		r:               r,
		path:            gitDir,
		config:          config,
		lastCommitCache: must(lru.New[string, map[string]CommitMeta](lastCommitCacheSize)),
	}, nil
}

// InitGoGitRepo creates a new empty git repo at the given path, to open with
// OpenGoGitRepo.
func InitGoGitRepo(path string) error {
	_, err := gogit.PlainInit(path, false)
	return err
}

// InitBareGoGitRepo creates a new --bare empty git repo at the given path, to
// open with OpenGoGitRepo.
func InitBareGoGitRepo(path string) error {
	_, err := gogit.PlainInit(path, true)
	return err
}

// FetchRefs retrieves the refs of the given namespaces (and the data they
// point to) from a remote, and stores them as that remote's tracking refs.
// Local refs are left untouched.
// The equivalent git refspec for namespace="foo" is "refs/foo/*:refs/remotes/<remote>/foo/*"
func (repo *GoGitRepo) FetchRefs(ctx context.Context, remote string, namespaces ...string) (string, error) {
	refSpecs := make([]config.RefSpec, len(namespaces))

	for i, namespace := range namespaces {
		refSpecs[i] = config.RefSpec(refPrefix(namespace) + "*:" + trackingRefPrefix(remote, namespace) + "*")
	}

	remo, err := repo.remote(remote, true, refSpecs)
	if err != nil {
		return "", err
	}

	buf := bytes.NewBuffer(nil)

	err = remo.FetchContext(ctx, &gogit.FetchOptions{
		RemoteName: remote,
		RefSpecs:   refSpecs,
		Progress:   buf,
	})
	if errors.Is(err, gogit.NoErrAlreadyUpToDate) {
		return "already up-to-date", nil
	}
	if err != nil {
		// might be one of go-git's shortcomings
		return "", fmt.Errorf("%w\nif git itself can reach this remote, setting %s=git uses it instead of go-git", err, gitTransportConfigKey)
	}

	return buf.String(), nil
}

// PushRefs sends the local refs of the given namespaces (and the data they
// point to) to a remote, and updates that remote's tracking refs to match.
// The equivalent git refspec for namespace="foo" is "refs/foo/*:refs/foo/*"
func (repo *GoGitRepo) PushRefs(ctx context.Context, remote string, namespaces ...string) (string, error) {
	refSpecs := make([]config.RefSpec, len(namespaces))
	// go-git updates the tracking ref of a pushed ref only if a fetch refspec
	// of the remote maps it
	fetchRefSpecs := make([]config.RefSpec, len(namespaces))

	for i, namespace := range namespaces {
		refSpecs[i] = config.RefSpec(refPrefix(namespace) + "*:" + refPrefix(namespace) + "*")
		fetchRefSpecs[i] = config.RefSpec(refPrefix(namespace) + "*:" + trackingRefPrefix(remote, namespace) + "*")
	}

	remo, err := repo.remote(remote, false, fetchRefSpecs)
	if err != nil {
		return "", err
	}

	buf := bytes.NewBuffer(nil)

	err = remo.PushContext(ctx, &gogit.PushOptions{
		RemoteName: remote,
		RefSpecs:   refSpecs,
		Progress:   buf,
	})
	if errors.Is(err, gogit.NoErrAlreadyUpToDate) {
		return "already up-to-date", nil
	}
	if err != nil {
		// might be one of go-git's shortcomings
		return "", fmt.Errorf("%w\nif git itself can reach this remote, setting %s=git uses it instead of go-git", err, gitTransportConfigKey)
	}

	return buf.String(), nil
}

// remote returns the go-git remote for the configured remote name, with fetch
// as its fetch refspecs. Its URL is the first configured one to fetch, the
// last one to push, rewritten by the longest matching url.<base>.insteadOf,
// as go-git would. go-git never reads the remotes itself: see
// gogitStorage.Config.
func (repo *GoGitRepo) remote(name string, fetching bool, fetch []config.RefSpec) (*gogit.Remote, error) {
	cfg, err := repo.config.Config().Read()
	if err != nil {
		return nil, err
	}

	urls := cfg.GetAll("remote." + name + ".url")
	if len(urls) == 0 {
		return nil, fmt.Errorf("unknown remote %q", name)
	}
	url := urls[len(urls)-1].Value
	if fetching {
		url = urls[0].Value
	}

	base, prefix := "", ""
	for _, sub := range cfg.Subsections("url") {
		for _, insteadOf := range cfg.GetAll("url." + sub + ".insteadOf") {
			if strings.HasPrefix(url, insteadOf.Value) && len(insteadOf.Value) > len(prefix) {
				base, prefix = sub, insteadOf.Value
			}
		}
	}
	if prefix != "" {
		url = base + url[len(prefix):]
	}

	return gogit.NewRemote(repo.r.Storer, &config.RemoteConfig{
		Name:  name,
		URLs:  []string{url},
		Fetch: fetch,
	}), nil
}

// StoreData will store arbitrary data and return the corresponding hash
func (repo *GoGitRepo) StoreData(data []byte) (Hash, error) {
	obj := repo.r.Storer.NewEncodedObject()
	obj.SetType(plumbing.BlobObject)

	w, err := obj.Writer()
	if err != nil {
		return "", err
	}

	_, err = w.Write(data)
	if err != nil {
		return "", err
	}

	h, err := repo.r.Storer.SetEncodedObject(obj)
	if err != nil {
		return "", err
	}

	return Hash(h.String()), nil
}

// ReadData will attempt to read arbitrary data from the given hash
func (repo *GoGitRepo) ReadData(hash Hash) (io.ReadCloser, error) {
	repo.rMutex.Lock()
	defer repo.rMutex.Unlock()

	obj, err := repo.r.BlobObject(plumbing.NewHash(hash.String()))
	if errors.Is(err, plumbing.ErrObjectNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return obj.Reader()
}

// StoreTree will store a mapping key-->Hash as a Git tree
func (repo *GoGitRepo) StoreTree(mapping []TreeEntry) (Hash, error) {
	var tree object.Tree

	for _, entry := range mapping {
		mode := filemode.Regular
		if entry.ObjectType == Tree {
			mode = filemode.Dir
		}

		tree.Entries = append(tree.Entries, object.TreeEntry{
			Name: entry.Name,
			Mode: mode,
			Hash: plumbing.NewHash(entry.Hash.String()),
		})
	}
	// in git's order, which Encode requires
	sort.Sort(object.TreeEntrySorter(tree.Entries))

	obj := repo.r.Storer.NewEncodedObject()
	obj.SetType(plumbing.TreeObject)
	err := tree.Encode(obj)
	if err != nil {
		return "", err
	}

	hash, err := repo.r.Storer.SetEncodedObject(obj)
	if err != nil {
		return "", err
	}

	return Hash(hash.String()), nil
}

// ReadTree will return the list of entries in a Git tree
func (repo *GoGitRepo) ReadTree(hash Hash) ([]TreeEntry, error) {
	repo.rMutex.Lock()
	defer repo.rMutex.Unlock()

	h := plumbing.NewHash(hash.String())

	// the given hash could be a tree or a commit
	obj, err := repo.r.Storer.EncodedObject(plumbing.AnyObject, h)
	if err == plumbing.ErrObjectNotFound {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	var tree *object.Tree
	switch obj.Type() {
	case plumbing.TreeObject:
		tree, err = object.DecodeTree(repo.r.Storer, obj)
	case plumbing.CommitObject:
		var commit *object.Commit
		commit, err = object.DecodeCommit(repo.r.Storer, obj)
		if err != nil {
			return nil, err
		}
		tree, err = commit.Tree()
	default:
		return nil, fmt.Errorf("given hash is not a tree")
	}
	if err != nil {
		return nil, err
	}

	treeEntries := make([]TreeEntry, len(tree.Entries))
	for i, entry := range tree.Entries {
		objType := Blob
		if entry.Mode == filemode.Dir {
			objType = Tree
		}

		treeEntries[i] = TreeEntry{
			ObjectType: objType,
			Hash:       Hash(entry.Hash.String()),
			Name:       entry.Name,
		}
	}

	return treeEntries, nil
}

// StoreCommit will store a Git commit with the given Git tree
func (repo *GoGitRepo) StoreCommit(treeHash Hash, parents ...Hash) (Hash, error) {
	return repo.StoreSignedCommit(treeHash, nil, parents...)
}

// StoreSignedCommit will store a Git commit with the given Git tree. If signKey is not nil, the commit
// will be signed accordingly.
func (repo *GoGitRepo) StoreSignedCommit(treeHash Hash, signKey *openpgp.Entity, parents ...Hash) (Hash, error) {
	// no name nor email: the operations carry their own authors
	commit := object.Commit{
		Author:    object.Signature{When: time.Now()},
		Committer: object.Signature{When: time.Now()},
		Message:   "",
		TreeHash:  plumbing.NewHash(treeHash.String()),
	}

	for _, parent := range parents {
		commit.ParentHashes = append(commit.ParentHashes, plumbing.NewHash(parent.String()))
	}

	// Compute the signature if needed
	if signKey != nil {
		// first get the serialized commit
		encoded := &plumbing.MemoryObject{}
		if err := commit.Encode(encoded); err != nil {
			return "", err
		}
		r, err := encoded.Reader()
		if err != nil {
			return "", err
		}

		// sign the data
		var sig bytes.Buffer
		if err := openpgp.ArmoredDetachSign(&sig, signKey, r, nil); err != nil {
			return "", err
		}
		commit.PGPSignature = sig.String()
	}

	obj := repo.r.Storer.NewEncodedObject()
	obj.SetType(plumbing.CommitObject)
	err := commit.Encode(obj)
	if err != nil {
		return "", err
	}

	hash, err := repo.r.Storer.SetEncodedObject(obj)
	if err != nil {
		return "", err
	}

	return Hash(hash.String()), nil
}

// ListRefs, UpdateRef, RemoveRef and ListTrackingRefs are in gogit_refs.go:
// go-git gets them wrong in ways git-bug depends on.

// ResolveRef returns the commit a local ref points to.
// Returns ErrNotFound if it doesn't exist.
func (repo *GoGitRepo) ResolveRef(namespace string, key string) (Hash, error) {
	return repo.lookupRef(refPrefix(namespace) + key)
}

// ResolveTrackingRef returns the commit a tracking ref points to.
// Returns ErrNotFound if it doesn't exist.
func (repo *GoGitRepo) ResolveTrackingRef(remote string, namespace string, key string) (Hash, error) {
	return repo.lookupRef(trackingRefPrefix(remote, namespace) + key)
}

// RemoveTrackingRef deletes a tracking ref.
// RemoveTrackingRef is idempotent.
func (repo *GoGitRepo) RemoveTrackingRef(remote string, namespace string, key string) error {
	return repo.r.Storer.RemoveReference(plumbing.ReferenceName(trackingRefPrefix(remote, namespace) + key))
}

func (repo *GoGitRepo) lookupRef(name string) (Hash, error) {
	r, err := repo.r.Reference(plumbing.ReferenceName(name), false)
	if err == plumbing.ErrReferenceNotFound {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return Hash(r.Hash().String()), nil
}

// ListCommits returns the hashes of commit and all its ancestors, in
// chronological order.
func (repo *GoGitRepo) ListCommits(ctx context.Context, commit Hash) ([]Hash, error) {
	return nonNativeListCommits(ctx, repo, commit)
}

func (repo *GoGitRepo) ReadCommit(hash Hash) (Commit, error) {
	repo.rMutex.Lock()
	defer repo.rMutex.Unlock()

	commit, err := repo.r.CommitObject(plumbing.NewHash(hash.String()))
	if err == plumbing.ErrObjectNotFound {
		return Commit{}, ErrNotFound
	}
	if err != nil {
		return Commit{}, err
	}

	parents := make([]Hash, len(commit.ParentHashes))
	for i, parentHash := range commit.ParentHashes {
		parents[i] = Hash(parentHash.String())
	}

	result := Commit{
		Hash:     hash,
		Parents:  parents,
		TreeHash: Hash(commit.TreeHash.String()),
	}

	if commit.PGPSignature != "" {
		// I can't find a way to just remove the signature when reading the encoded commit so we need to
		// re-encode the commit without signature.

		encoded := &plumbing.MemoryObject{}
		err := commit.EncodeWithoutSignature(encoded)
		if err != nil {
			return Commit{}, err
		}

		result.SignedData, err = encoded.Reader()
		if err != nil {
			return Commit{}, err
		}

		result.Signature, err = deArmorSignature(strings.NewReader(commit.PGPSignature))
		if err != nil {
			return Commit{}, err
		}
	}

	return result, nil
}

// commitToMeta converts a go-git Commit to a CommitMeta.
func commitToMeta(c *object.Commit) CommitMeta {
	h := Hash(c.Hash.String())
	parents := make([]Hash, len(c.ParentHashes))
	for i, p := range c.ParentHashes {
		parents[i] = Hash(p.String())
	}
	// Use first line of message as the short message.
	msg := strings.TrimSpace(c.Message)
	if idx := strings.Index(msg, "\n"); idx >= 0 {
		msg = msg[:idx]
	}
	return CommitMeta{
		Hash:        h,
		Message:     msg,
		AuthorName:  c.Author.Name,
		AuthorEmail: c.Author.Email,
		Date:        c.Author.When,
		Parents:     parents,
	}
}

// peelToCommit follows tag objects until it reaches a commit hash.
// This is necessary for annotated tags, whose ref hash points to a tag object
// rather than directly to a commit.
func (repo *GoGitRepo) peelToCommit(h plumbing.Hash) (plumbing.Hash, error) {
	for {
		if _, err := repo.r.CommitObject(h); err == nil {
			return h, nil
		}
		tagObj, err := repo.r.TagObject(h)
		if err != nil {
			return plumbing.ZeroHash, ErrNotFound
		}
		h = tagObj.Target
	}
}

// resolveRev resolves a branch/tag name or raw hash to a commit hash.
// Resolution order: refs/heads/<rev>, refs/tags/<rev>, full ref name, raw commit hash.
// Annotated tags are peeled to their target commit.
func (repo *GoGitRepo) resolveRev(rev string) (plumbing.Hash, error) {
	for _, prefix := range []string{"refs/heads/", "refs/tags/"} {
		r, err := repo.r.Reference(plumbing.ReferenceName(prefix+rev), true)
		if err == nil {
			return repo.peelToCommit(r.Hash())
		}
	}
	// try as a full ref name
	r, err := repo.r.Reference(plumbing.ReferenceName(rev), true)
	if err == nil {
		return repo.peelToCommit(r.Hash())
	}
	// try as a raw commit hash
	h := plumbing.NewHash(rev)
	if h != plumbing.ZeroHash {
		if _, err := repo.r.CommitObject(h); err == nil {
			return h, nil
		}
	}
	return plumbing.ZeroHash, ErrNotFound
}

// Branches returns all local branches (refs/heads/*).
func (repo *GoGitRepo) Branches(ctx context.Context) ([]BranchInfo, error) {
	repo.rMutex.Lock()
	defer repo.rMutex.Unlock()

	refs, err := repo.r.References()
	if err != nil {
		return nil, err
	}

	var branches []BranchInfo
	err = refs.ForEach(func(r *plumbing.Reference) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !r.Name().IsBranch() {
			return nil
		}
		branches = append(branches, BranchInfo{
			Name: r.Name().Short(),
			Hash: Hash(r.Hash().String()),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	if branches == nil {
		branches = []BranchInfo{}
	}
	return branches, nil
}

// Tags returns all tags. For annotated tags the hash is dereferenced to the
// target commit; for lightweight tags it is the commit hash directly.
func (repo *GoGitRepo) Tags(ctx context.Context) ([]TagInfo, error) {
	repo.rMutex.Lock()
	defer repo.rMutex.Unlock()

	refs, err := repo.r.References()
	if err != nil {
		return nil, err
	}

	var tags []TagInfo
	err = refs.ForEach(func(r *plumbing.Reference) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !r.Name().IsTag() {
			return nil
		}
		// Peel to the target commit hash, handling arbitrarily nested tag objects.
		commit, err := repo.peelToCommit(r.Hash())
		if err != nil {
			// Skip refs that don't resolve to a commit (shouldn't happen for tags).
			return nil
		}
		tags = append(tags, TagInfo{
			Name: r.Name().Short(),
			Hash: Hash(commit.String()),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	if tags == nil {
		tags = []TagInfo{}
	}
	return tags, nil
}

// TreeAtPath returns the entries of the directory at path under rev.
func (repo *GoGitRepo) TreeAtPath(ctx context.Context, rev, path string) ([]TreeEntry, error) {
	path = strings.Trim(path, "/")

	repo.rMutex.Lock()
	defer repo.rMutex.Unlock()

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	startHash, err := repo.resolveRev(rev)
	if err != nil {
		return nil, ErrNotFound
	}
	commit, err := repo.r.CommitObject(startHash)
	if err != nil {
		return nil, err
	}
	tree, err := commit.Tree()
	if err != nil {
		return nil, err
	}
	if path != "" {
		subtree, err := tree.Tree(path)
		if err != nil {
			return nil, ErrNotFound
		}
		tree = subtree
	}

	entries := make([]TreeEntry, len(tree.Entries))
	for i, e := range tree.Entries {
		entries[i] = TreeEntry{
			Name:       e.Name,
			Hash:       Hash(e.Hash.String()),
			ObjectType: objectTypeFromFileMode(e.Mode),
		}
	}
	return entries, nil
}

// objectTypeFromFileMode maps a go-git filemode to the repository ObjectType.
func objectTypeFromFileMode(m filemode.FileMode) ObjectType {
	switch m {
	case filemode.Dir:
		return Tree
	case filemode.Regular:
		return Blob
	case filemode.Executable:
		return Executable
	case filemode.Symlink:
		return Symlink
	case filemode.Submodule:
		return Submodule
	default:
		return Unknown
	}
}

// BlobAtPath returns the content, size, and git object hash of the file at
// path under rev. rMutex is held for the entire function, covering all
// shared-Scanner access (CommitObject, Tree, File). The returned reader is
// safe to use without the mutex: small blobs are already materialized into a
// MemoryObject (bytes.Reader) by the time File() returns; large blobs come
// back as an FSObject whose Reader() opens its own independent file handle and
// Scanner and then reads via ReadAt — no shared state is touched after this
// function returns. Callers must Close the reader.
func (repo *GoGitRepo) BlobAtPath(ctx context.Context, rev, path string) (io.ReadCloser, int64, Hash, error) {
	path = strings.Trim(path, "/")
	if path == "" {
		return nil, 0, "", ErrNotFound
	}

	repo.rMutex.Lock()
	defer repo.rMutex.Unlock()

	if err := ctx.Err(); err != nil {
		return nil, 0, "", err
	}

	startHash, err := repo.resolveRev(rev)
	if err != nil {
		return nil, 0, "", ErrNotFound
	}
	commit, err := repo.r.CommitObject(startHash)
	if err != nil {
		return nil, 0, "", err
	}
	tree, err := commit.Tree()
	if err != nil {
		return nil, 0, "", err
	}
	f, err := tree.File(path)
	if err != nil {
		return nil, 0, "", ErrNotFound
	}
	r, err := f.Reader()
	if err != nil {
		return nil, 0, "", err
	}

	return r, f.Blob.Size, Hash(f.Blob.Hash.String()), nil
}

// CommitLog returns at most limit commits reachable from rev, optionally
// filtered to those that touched path, starting after the given cursor hash,
// and bounded by the since/until author-date range.
func (repo *GoGitRepo) CommitLog(ctx context.Context, rev, path string, limit int, after Hash, since, until *time.Time) ([]CommitMeta, error) {
	repo.rMutex.Lock()
	defer repo.rMutex.Unlock()

	startHash, err := repo.resolveRev(rev)
	if err != nil {
		return nil, err
	}

	// Normalize path: strip leading/trailing slashes so prefix matching works.
	path = strings.Trim(path, "/")

	opts := &gogit.LogOptions{
		From:  startHash,
		Order: gogit.LogOrderCommitterTime,
	}
	if path != "" {
		opts.PathFilter = func(p string) bool {
			return p == path || strings.HasPrefix(p, path+"/")
		}
	}

	iter, err := repo.r.Log(opts)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var result []CommitMeta
	skipping := after != ""
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		c, err := iter.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		h := Hash(c.Hash.String())
		if skipping {
			if h == after {
				skipping = false
			}
			continue
		}
		if since != nil && c.Author.When.Before(*since) {
			continue
		}
		if until != nil && c.Author.When.After(*until) {
			continue
		}
		result = append(result, commitToMeta(c))
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result, nil
}

// treeEntriesAtPath returns the tree hash and a name→entry-hash map for the
// directory at dirPath inside the given commit. An empty dirPath means the
// root tree. The tree hash is content-addressed and can be used as a stable
// cache key regardless of which rev was resolved.
func treeEntriesAtPath(c *object.Commit, dirPath string) (plumbing.Hash, map[string]plumbing.Hash, error) {
	tree, err := c.Tree()
	if err != nil {
		return plumbing.ZeroHash, nil, err
	}
	if dirPath != "" {
		subtree, err := tree.Tree(dirPath)
		if err != nil {
			return plumbing.ZeroHash, nil, err
		}
		tree = subtree
	}
	result := make(map[string]plumbing.Hash, len(tree.Entries))
	for _, e := range tree.Entries {
		result[e.Name] = e.Hash
	}
	return tree.Hash, result, nil
}

// LastCommitForEntries performs a single history walk to find, for each name,
// the most recent commit that changed that entry in the directory at path.
//
// Results are cached by (dirTreeHash, path). Because git trees are
// content-addressed, two revs that resolve to the same directory tree share one
// cache entry, and the cache never needs invalidation: a changed directory
// produces a new tree hash, which becomes a new key.
func (repo *GoGitRepo) LastCommitForEntries(ctx context.Context, rev, path string, names []string) (map[string]CommitMeta, error) {
	// Normalize path up front so the cache key is canonical.
	path = strings.Trim(path, "/")

	// Resolve rev and load the current directory tree in one brief lock.
	// We need the tree hash for the cache key and we keep the entries to
	// seed the parent-reuse optimisation in the walk below.
	repo.rMutex.Lock()
	startHash, err := repo.resolveRev(rev)
	if err != nil {
		repo.rMutex.Unlock()
		return nil, err
	}
	startCommit, err := repo.r.CommitObject(startHash)
	if err != nil {
		repo.rMutex.Unlock()
		return nil, err
	}
	treeHash, startEntries, err := treeEntriesAtPath(startCommit, path)
	repo.rMutex.Unlock()
	if err != nil {
		// path doesn't exist at HEAD — nothing to return.
		return map[string]CommitMeta{}, nil
	}

	// The cache is keyed by the directory's tree hash (content-addressed)
	// plus the path so two directories with identical content but different
	// locations don't collide.
	cacheKey := treeHash.String() + "\x00" + path

	// Concurrent calls for the same directory share one history walk.
	all, _, err := repo.lastCommitSF.Do(ctx, cacheKey, func(ctx context.Context) (map[string]CommitMeta, error) {
		if cached, ok := repo.lastCommitCache.Get(cacheKey); ok {
			return cached, nil
		}
		// The cached map holds every entry of the directory, not just the
		// requested names, for any later call on it to be served from the
		// cache. Callers only ever get a filtered copy of it.
		walked, err := repo.walkLastCommits(ctx, startHash, startEntries, path)
		if err != nil {
			return nil, err
		}
		repo.lastCommitCache.Add(cacheKey, walked)
		return walked, nil
	})
	if err != nil {
		return nil, err
	}

	result := make(map[string]CommitMeta, len(names))
	for _, n := range names {
		if m, ok := all[n]; ok {
			result[n] = m
		}
	}
	return result, nil
}

// walkLastCommits walks the history from start, for at most
// lastCommitDepthLimit commits, and returns the most recent commit that changed
// each of the entries of the directory at path, which are startEntries at start.
func (repo *GoGitRepo) walkLastCommits(ctx context.Context, start plumbing.Hash, startEntries map[string]plumbing.Hash, path string) (map[string]CommitMeta, error) {
	remaining := make(map[string]bool, len(startEntries))
	for name := range startEntries {
		remaining[name] = true
	}
	result := make(map[string]CommitMeta, len(remaining))

	repo.rMutex.Lock()
	defer repo.rMutex.Unlock()

	iter, err := repo.r.Log(&gogit.LogOptions{
		From:  start,
		Order: gogit.LogOrderCommitterTime,
	})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	// Seed the parent-reuse cache with the entries already known at start, so
	// the first iteration's current-tree read is skipped for free.
	// In a linear history this halves tree reads for every subsequent step:
	// the parent fetched at depth D is the current commit at depth D+1.
	cachedParentHash := start
	cachedParentEntries := startEntries

	for depth := 0; len(remaining) > 0 && depth < lastCommitDepthLimit; depth++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		c, err := iter.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		var currentEntries map[string]plumbing.Hash
		if c.Hash == cachedParentHash && cachedParentEntries != nil {
			currentEntries = cachedParentEntries
		} else {
			_, currentEntries, err = treeEntriesAtPath(c, path)
			if err != nil {
				// path may not exist in this commit; treat as empty
				currentEntries = map[string]plumbing.Hash{}
			}
		}

		var parentEntries map[string]plumbing.Hash
		cachedParentHash = plumbing.ZeroHash
		cachedParentEntries = nil
		if len(c.ParentHashes) > 0 {
			if parent, err := c.Parents().Next(); err == nil {
				_, parentEntries, _ = treeEntriesAtPath(parent, path)
				cachedParentHash = c.ParentHashes[0]
				cachedParentEntries = parentEntries
			}
		}

		meta := commitToMeta(c)
		for name := range remaining {
			curHash, inCurrent := currentEntries[name]
			parentHash, inParent := parentEntries[name]
			if inCurrent != inParent || (inCurrent && curHash != parentHash) {
				result[name] = meta
				delete(remaining, name)
			}
		}
	}

	return result, nil
}

// firstParentChanges returns the commit of hash, and its changes from its
// first parent, or from the empty tree for an initial commit. rMutex must be
// held.
func (repo *GoGitRepo) firstParentChanges(ctx context.Context, hash Hash) (*object.Commit, object.Changes, error) {
	c, err := repo.r.CommitObject(plumbing.NewHash(hash.String()))
	if errors.Is(err, plumbing.ErrObjectNotFound) {
		return nil, nil, ErrNotFound
	}
	if err != nil {
		return nil, nil, err
	}

	toTree, err := c.Tree()
	if err != nil {
		return nil, nil, err
	}

	var fromTree *object.Tree
	if len(c.ParentHashes) > 0 {
		parent, err := repo.r.CommitObject(c.ParentHashes[0])
		if err != nil {
			return nil, nil, fmt.Errorf("loading parent commit: %w", err)
		}
		fromTree, err = parent.Tree()
		if err != nil {
			return nil, nil, fmt.Errorf("loading parent tree: %w", err)
		}
	}

	changes, err := object.DiffTreeContext(ctx, fromTree, toTree)
	if errors.Is(err, object.ErrCanceled) {
		return nil, nil, ctx.Err()
	}
	if err != nil {
		return nil, nil, err
	}
	return c, changes, nil
}

// CommitDetail returns the full commit metadata and list of changed files.
func (repo *GoGitRepo) CommitDetail(ctx context.Context, hash Hash) (CommitDetail, error) {
	repo.rMutex.Lock()
	defer repo.rMutex.Unlock()

	c, changes, err := repo.firstParentChanges(ctx, hash)
	if err != nil {
		return CommitDetail{}, err
	}

	// Use ch.From.Name / ch.To.Name directly — these come from the tree
	// metadata and do not require reading any blob content.
	files := make([]ChangedFile, 0, len(changes))
	for _, ch := range changes {
		files = append(files, changedFileFromChange(ch.From.Name, ch.To.Name))
	}

	return CommitDetail{
		CommitMeta:  commitToMeta(c),
		FullMessage: c.Message,
		Files:       files,
	}, nil
}

func changedFileFromChange(fromName, toName string) ChangedFile {
	switch {
	case fromName == "":
		return ChangedFile{Path: toName, Status: ChangeStatusAdded}
	case toName == "":
		return ChangedFile{Path: fromName, Status: ChangeStatusDeleted}
	case fromName != toName:
		op := fromName
		return ChangedFile{Path: toName, OldPath: &op, Status: ChangeStatusRenamed}
	default:
		return ChangedFile{Path: toName, Status: ChangeStatusModified}
	}
}

// CommitFileDiff returns the unified diff for a single file in a commit,
// relative to the first parent.
func (repo *GoGitRepo) CommitFileDiff(ctx context.Context, hash Hash, filePath string) (FileDiff, error) {
	repo.rMutex.Lock()
	defer repo.rMutex.Unlock()

	_, changes, err := repo.firstParentChanges(ctx, hash)
	if err != nil {
		return FileDiff{}, err
	}

	for _, ch := range changes {
		name := ch.To.Name
		if name == "" {
			name = ch.From.Name
		}
		// match on either new or old path
		if name != filePath && ch.From.Name != filePath {
			continue
		}

		from, to, err := ch.Files()
		if err != nil {
			return FileDiff{}, err
		}

		patch, err := ch.PatchContext(ctx)
		if errors.Is(err, object.ErrCanceled) {
			return FileDiff{}, ctx.Err()
		}
		if err != nil {
			return FileDiff{}, err
		}

		fd := FileDiff{
			IsNew:    from == nil,
			IsDelete: to == nil,
		}
		if to != nil {
			fd.Path = to.Name
		}
		if from != nil {
			if fd.Path == "" {
				fd.Path = from.Name
			} else if from.Name != fd.Path {
				op := from.Name
				fd.OldPath = &op
			}
		}

		fps := patch.FilePatches()
		if len(fps) > 0 {
			fp := fps[0]
			fd.IsBinary = fp.IsBinary()
			if !fd.IsBinary {
				fd.Hunks = buildDiffHunks(fp)
			}
		}
		return fd, nil
	}
	return FileDiff{}, ErrNotFound
}

// buildDiffHunks converts a go-git FilePatch into DiffHunks with line numbers
// and context grouping.
func buildDiffHunks(fp fdiff.FilePatch) []DiffHunk {
	type pendingLine struct {
		typ     DiffLineType
		content string
		oldLine int
		newLine int
	}

	var allLines []pendingLine
	oldLine, newLine := 1, 1
	for _, chunk := range fp.Chunks() {
		lines := strings.Split(chunk.Content(), "\n")
		// strip trailing empty element produced by a trailing newline
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}
		switch chunk.Type() {
		case fdiff.Equal:
			for _, l := range lines {
				allLines = append(allLines, pendingLine{DiffLineContext, l, oldLine, newLine})
				oldLine++
				newLine++
			}
		case fdiff.Add:
			for _, l := range lines {
				allLines = append(allLines, pendingLine{DiffLineAdded, l, 0, newLine})
				newLine++
			}
		case fdiff.Delete:
			for _, l := range lines {
				allLines = append(allLines, pendingLine{DiffLineDeleted, l, oldLine, 0})
				oldLine++
			}
		}
	}
	if len(allLines) == 0 {
		return nil
	}

	const ctx = 3 // context lines around each changed block

	// find spans of changed lines
	type span struct{ start, end int }
	var spans []span
	for i, l := range allLines {
		if l.typ == DiffLineContext {
			continue
		}
		if len(spans) == 0 || i > spans[len(spans)-1].end+1 {
			spans = append(spans, span{i, i})
		} else {
			spans[len(spans)-1].end = i
		}
	}

	// expand each span by ctx lines and merge overlapping ones
	var merged []span
	for _, s := range spans {
		s.start = max(0, s.start-ctx)
		s.end = min(len(allLines)-1, s.end+ctx)
		if len(merged) > 0 && s.start <= merged[len(merged)-1].end+1 {
			merged[len(merged)-1].end = s.end
		} else {
			merged = append(merged, s)
		}
	}

	hunks := make([]DiffHunk, 0, len(merged))
	for _, s := range merged {
		segment := allLines[s.start : s.end+1]
		dl := make([]DiffLine, len(segment))
		var oldStart, newStart, oldCount, newCount int
		for i, l := range segment {
			dl[i] = DiffLine{Type: l.typ, Content: l.content, OldLine: l.oldLine, NewLine: l.newLine}
			if l.oldLine > 0 {
				if oldStart == 0 {
					oldStart = l.oldLine
				}
				oldCount++
			}
			if l.newLine > 0 {
				if newStart == 0 {
					newStart = l.newLine
				}
				newCount++
			}
		}
		hunks = append(hunks, DiffHunk{
			OldStart: oldStart,
			OldLines: oldCount,
			NewStart: newStart,
			NewLines: newCount,
			Lines:    dl,
		})
	}
	return hunks
}

// Head returns the ref that HEAD currently points to.
func (repo *GoGitRepo) Head(ctx context.Context) (RefMeta, error) {
	repo.rMutex.Lock()
	defer repo.rMutex.Unlock()

	ref, err := repo.r.Head()
	if err == plumbing.ErrReferenceNotFound {
		return RefMeta{}, ErrNotFound
	}
	if err != nil {
		return RefMeta{}, err
	}

	var refType GitRefType
	switch {
	case ref.Name().IsBranch():
		refType = GitRefTypeBranch
	case ref.Name().IsTag():
		refType = GitRefTypeTag
	default:
		refType = GitRefTypeCommit
	}

	return RefMeta{
		Name:      ref.Name().String(),
		ShortName: ref.Name().Short(),
		Type:      refType,
		Hash:      ref.Hash().String(),
	}, nil
}
