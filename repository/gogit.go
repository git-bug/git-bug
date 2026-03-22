package repository

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/go-git/go-billy/v5/osfs"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	fdiff "github.com/go-git/go-git/v5/plumbing/format/diff"
	"github.com/go-git/go-git/v5/plumbing/object"
	lru "github.com/hashicorp/golang-lru/v2"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sys/execabs"

	"github.com/git-bug/git-bug/util/lamport"
)

const clockPath = "clocks"
const indexPath = "indexes"

// lastCommitDepthLimit is the maximum number of commits walked by
// LastCommitForEntries. Entries not found within this horizon are omitted from
// the result rather than stalling the caller indefinitely.
const lastCommitDepthLimit = 1000

// lastCommitCacheSize is the number of (resolvedHash, dirPath) pairs kept in
// the LRU cache for LastCommitForEntries. Each entry holds one CommitMeta per
// directory entry (≈ a few KB for a typical directory), so 256 slots ≈ a few
// MB of memory at most.
const lastCommitCacheSize = 256

var _ ClockedRepo = &GoGitRepo{}
var _ TestedRepo = &GoGitRepo{}

type GoGitRepo struct {
	// Unfortunately, some parts of go-git are not thread-safe so we have to cover them with a big fat mutex here.
	// See https://github.com/go-git/go-git/issues/48
	// See https://github.com/go-git/go-git/issues/208
	// See https://github.com/go-git/go-git/pull/186
	rMutex sync.Mutex
	r      *gogit.Repository
	path   string

	clocksMutex sync.Mutex
	clocks      map[string]lamport.Clock

	indexesMutex sync.Mutex
	indexes      map[string]Index

	// lastCommitCache caches LastCommitForEntries results keyed by
	// "<treeHash>\x00<path>". Git trees are content-addressed and
	// immutable, so entries never need invalidation and can be shared
	// across refs that point to the same directory tree. The LRU bounds
	// memory to lastCommitCacheSize unique (treeHash, directory) pairs.
	lastCommitCache *lru.Cache[string, map[string]CommitMeta]

	keyring      Keyring
	localStorage LocalStorage
}

// OpenGoGitRepo opens an already existing repo at the given path and
// with the specified LocalStorage namespace.  Given a repository path
// of "~/myrepo" and a namespace of "git-bug", local storage for the
// GoGitRepo will be configured at "~/myrepo/.git/git-bug".
func OpenGoGitRepo(path, namespace string, clockLoaders []ClockLoader) (*GoGitRepo, error) {
	path, err := detectGitPath(path, 0)
	if err != nil {
		return nil, err
	}

	r, err := gogit.PlainOpen(path)
	if err != nil {
		return nil, err
	}

	k, err := defaultKeyring()
	if err != nil {
		return nil, err
	}

	repo := &GoGitRepo{
		r:               r,
		path:            path,
		clocks:          make(map[string]lamport.Clock),
		indexes:         make(map[string]Index),
		lastCommitCache: must(lru.New[string, map[string]CommitMeta](lastCommitCacheSize)),
		keyring:         k,
		localStorage:    billyLocalStorage{Filesystem: osfs.New(filepath.Join(path, namespace))},
	}

	loaderToRun := make([]ClockLoader, 0, len(clockLoaders))
	for _, loader := range clockLoaders {
		loader := loader
		allExist := true
		for _, name := range loader.Clocks {
			if _, err := repo.getClock(name); err != nil {
				allExist = false
			}
		}

		if !allExist {
			loaderToRun = append(loaderToRun, loader)
		}
	}

	var errG errgroup.Group
	for _, loader := range loaderToRun {
		loader := loader
		errG.Go(func() error {
			return loader.Witnesser(repo)
		})
	}
	err = errG.Wait()
	if err != nil {
		return nil, err
	}

	return repo, nil
}

// InitGoGitRepo creates a new empty git repo at the given path and
// with the specified LocalStorage namespace.  Given a repository path
// of "~/myrepo" and a namespace of "git-bug", local storage for the
// GoGitRepo will be configured at "~/myrepo/.git/git-bug".
func InitGoGitRepo(path, namespace string) (*GoGitRepo, error) {
	r, err := gogit.PlainInit(path, false)
	if err != nil {
		return nil, err
	}

	k, err := defaultKeyring()
	if err != nil {
		return nil, err
	}

	return &GoGitRepo{
		r:               r,
		path:            filepath.Join(path, ".git"),
		clocks:          make(map[string]lamport.Clock),
		indexes:         make(map[string]Index),
		lastCommitCache: must(lru.New[string, map[string]CommitMeta](lastCommitCacheSize)),
		keyring:         k,
		localStorage:    billyLocalStorage{Filesystem: osfs.New(filepath.Join(path, ".git", namespace))},
	}, nil
}

// InitBareGoGitRepo creates a new --bare empty git repo at the given
// path and with the specified LocalStorage namespace.  Given a repository
// path of "~/myrepo" and a namespace of "git-bug", local storage for the
// GoGitRepo will be configured at "~/myrepo/.git/git-bug".
func InitBareGoGitRepo(path, namespace string) (*GoGitRepo, error) {
	r, err := gogit.PlainInit(path, true)
	if err != nil {
		return nil, err
	}

	k, err := defaultKeyring()
	if err != nil {
		return nil, err
	}

	return &GoGitRepo{
		r:               r,
		path:            path,
		clocks:          make(map[string]lamport.Clock),
		indexes:         make(map[string]Index),
		lastCommitCache: must(lru.New[string, map[string]CommitMeta](lastCommitCacheSize)),
		keyring:         k,
		localStorage:    billyLocalStorage{Filesystem: osfs.New(filepath.Join(path, namespace))},
	}, nil
}

func detectGitPath(path string, depth int) (string, error) {
	if depth >= 10 {
		return "", fmt.Errorf("gitdir loop detected")
	}

	// normalize the path
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	for {
		fi, err := os.Stat(filepath.Join(path, ".git"))
		if err == nil {
			if !fi.IsDir() {
				// See if our .git item is a dotfile that holds a submodule reference
				dotfile, err := os.Open(filepath.Join(path, fi.Name()))
				if err != nil {
					// Can't open error
					return "", fmt.Errorf(".git exists but is not a directory or a readable file: %w", err)
				}
				// We aren't going to defer the dotfile.Close, because we might keep looping, so we have to be sure to
				// clean up before returning an error
				reader := bufio.NewReader(io.LimitReader(dotfile, 2048))
				line, _, err := reader.ReadLine()
				_ = dotfile.Close()
				if err != nil {
					return "", fmt.Errorf(".git exists but is not a directory and cannot be read: %w", err)
				}
				dotContent := string(line)
				if strings.HasPrefix(dotContent, "gitdir:") {
					// This is a submodule parent path link. Strip the prefix, clean the string of whitespace just to
					// be safe, and return
					dotContent = strings.TrimSpace(strings.TrimPrefix(dotContent, "gitdir: "))
					p, err := detectGitPath(dotContent, depth+1)
					if err != nil {
						return "", fmt.Errorf(".git gitdir error: %w", err)
					}
					return p, nil
				}
				return "", fmt.Errorf(".git exist but is not a directory or module/workspace file")
			}
			return filepath.Join(path, ".git"), nil
		}
		if !os.IsNotExist(err) {
			// unknown error
			return "", err
		}

		// detect bare repo
		ok, err := isGitDir(path)
		if err != nil {
			return "", err
		}
		if ok {
			return path, nil
		}

		if parent := filepath.Dir(path); parent == path {
			return "", fmt.Errorf(".git not found")
		} else {
			path = parent
		}
	}
}

func isGitDir(path string) (bool, error) {
	markers := []string{"HEAD", "objects", "refs"}

	for _, marker := range markers {
		_, err := os.Stat(filepath.Join(path, marker))
		if err == nil {
			continue
		}
		if !os.IsNotExist(err) {
			// unknown error
			return false, err
		} else {
			return false, nil
		}
	}

	return true, nil
}

func (repo *GoGitRepo) Close() error {
	var firstErr error
	for name, index := range repo.indexes {
		err := index.Close()
		if err != nil && firstErr == nil {
			firstErr = err
		}
		delete(repo.indexes, name)
	}
	return firstErr
}

// LocalConfig give access to the repository scoped configuration
func (repo *GoGitRepo) LocalConfig() Config {
	return newGoGitLocalConfig(repo.r)
}

// GlobalConfig give access to the global scoped configuration
func (repo *GoGitRepo) GlobalConfig() Config {
	return newGoGitGlobalConfig()
}

// AnyConfig give access to a merged local/global configuration
func (repo *GoGitRepo) AnyConfig() ConfigRead {
	return mergeConfig(repo.LocalConfig(), repo.GlobalConfig())
}

// Keyring give access to a user-wide storage for secrets
func (repo *GoGitRepo) Keyring() Keyring {
	return repo.keyring
}

// GetUserName returns the name the user has used to configure git
func (repo *GoGitRepo) GetUserName() (string, error) {
	return repo.AnyConfig().ReadString("user.name")
}

// GetUserEmail returns the email address that the user has used to configure git.
func (repo *GoGitRepo) GetUserEmail() (string, error) {
	return repo.AnyConfig().ReadString("user.email")
}

// GetCoreEditor returns the name of the editor that the user has used to configure git.
func (repo *GoGitRepo) GetCoreEditor() (string, error) {
	// See https://git-scm.com/docs/git-var
	// The order of preference is the $GIT_EDITOR environment variable, then core.editor configuration, then $VISUAL, then $EDITOR, and then the default chosen at compile time, which is usually vi.

	if val, ok := os.LookupEnv("GIT_EDITOR"); ok {
		return val, nil
	}

	val, err := repo.AnyConfig().ReadString("core.editor")
	if err == nil && val != "" {
		return val, nil
	}
	if err != nil && !errors.Is(err, ErrNoConfigEntry) {
		return "", err
	}

	if val, ok := os.LookupEnv("VISUAL"); ok {
		return val, nil
	}

	if val, ok := os.LookupEnv("EDITOR"); ok {
		return val, nil
	}

	priorities := []string{
		"editor",
		"nano",
		"vim",
		"vi",
		"emacs",
	}

	for _, cmd := range priorities {
		if _, err = execabs.LookPath(cmd); err == nil {
			return cmd, nil
		}

	}

	return "ed", nil
}

// GetRemotes returns the configured remotes repositories.
func (repo *GoGitRepo) GetRemotes() (map[string]string, error) {
	cfg, err := repo.r.Config()
	if err != nil {
		return nil, err
	}

	result := make(map[string]string, len(cfg.Remotes))
	for name, remote := range cfg.Remotes {
		if len(remote.URLs) > 0 {
			result[name] = remote.URLs[0]
		}
	}

	return result, nil
}

// LocalStorage returns a billy.Filesystem giving access to
// $RepoPath/.git/$Namespace.
func (repo *GoGitRepo) LocalStorage() LocalStorage {
	return repo.localStorage
}

func (repo *GoGitRepo) GetIndex(name string) (Index, error) {
	repo.indexesMutex.Lock()
	defer repo.indexesMutex.Unlock()

	if index, ok := repo.indexes[name]; ok {
		return index, nil
	}

	path := filepath.Join(repo.localStorage.Root(), indexPath, name)

	index, err := openBleveIndex(path)
	if err == nil {
		repo.indexes[name] = index
	}
	return index, err
}

// FetchRefs fetch git refs matching a directory prefix to a remote
// Ex: prefix="foo" will fetch any remote refs matching "refs/foo/*" locally.
// The equivalent git refspec would be "refs/foo/*:refs/remotes/<remote>/foo/*"
func (repo *GoGitRepo) FetchRefs(remote string, prefixes ...string) (string, error) {
	refSpecs := make([]config.RefSpec, len(prefixes))

	for i, prefix := range prefixes {
		refSpecs[i] = config.RefSpec(fmt.Sprintf("refs/%s/*:refs/remotes/%s/%s/*", prefix, remote, prefix))
	}

	buf := bytes.NewBuffer(nil)

	remoteUrl, err := repo.resolveRemote(remote, true)
	if err != nil {
		return "", err
	}

	err = repo.r.Fetch(&gogit.FetchOptions{
		RemoteName: remote,
		RemoteURL:  remoteUrl,
		RefSpecs:   refSpecs,
		Progress:   buf,
	})
	if err == gogit.NoErrAlreadyUpToDate {
		return "already up-to-date", nil
	}
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// resolveRemote returns the URI for a given remote
func (repo *GoGitRepo) resolveRemote(remote string, fetch bool) (string, error) {
	cfg, err := repo.r.ConfigScoped(config.SystemScope)
	if err != nil {
		return "", fmt.Errorf("unable to load system-scoped git config: %v", err)
	}

	var url string
	for _, re := range cfg.Remotes {
		if remote == re.Name {
			// url is set matching the default logic in go-git's repository.Push
			// and repository.Fetch logic as of go-git v5.12.1.
			//
			// we do this because the push and fetch methods can only take one
			// remote for both option structs, even though the push method
			// _should_ push to all of the URLs defined for a given remote.
			url = re.URLs[len(re.URLs)-1]
			if fetch {
				url = re.URLs[0]
			}

			for _, u := range cfg.URLs {
				if strings.HasPrefix(url, u.InsteadOf) {
					url = u.ApplyInsteadOf(url)
					break
				}
			}
		}
	}

	if url == "" {
		return "", fmt.Errorf("unable to resolve URL for remote: %v", err)
	}

	return url, nil
}

// PushRefs push git refs matching a directory prefix to a remote
// Ex: prefix="foo" will push any local refs matching "refs/foo/*" to the remote.
// The equivalent git refspec would be "refs/foo/*:refs/foo/*"
//
// Additionally, PushRefs will update the local references in refs/remotes/<remote>/foo to match
// the remote state.
func (repo *GoGitRepo) PushRefs(remote string, prefixes ...string) (string, error) {
	remo, err := repo.r.Remote(remote)
	if err != nil {
		return "", err
	}

	refSpecs := make([]config.RefSpec, len(prefixes))

	for i, prefix := range prefixes {
		refspec := fmt.Sprintf("refs/%s/*:refs/%s/*", prefix, prefix)

		// to make sure that the push also create the corresponding refs/remotes/<remote>/... references,
		// we need to have a default fetch refspec configured on the remote, to make our refs "track" the remote ones.
		// This does not change the config on disk, only on memory.
		hasCustomFetch := false
		fetchRefspec := fmt.Sprintf("refs/%s/*:refs/remotes/%s/%s/*", prefix, remote, prefix)
		for _, r := range remo.Config().Fetch {
			if string(r) == fetchRefspec {
				hasCustomFetch = true
				break
			}
		}

		if !hasCustomFetch {
			remo.Config().Fetch = append(remo.Config().Fetch, config.RefSpec(fetchRefspec))
		}

		refSpecs[i] = config.RefSpec(refspec)
	}

	buf := bytes.NewBuffer(nil)

	remoteUrl, err := repo.resolveRemote(remote, false)
	if err != nil {
		return "", err
	}

	err = remo.Push(&gogit.PushOptions{
		RemoteName: remote,
		RemoteURL:  remoteUrl,
		RefSpecs:   refSpecs,
		Progress:   buf,
	})
	if err == gogit.NoErrAlreadyUpToDate {
		return "already up-to-date", nil
	}
	if err != nil {
		return "", err
	}

	return buf.String(), nil
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
func (repo *GoGitRepo) ReadData(hash Hash) ([]byte, error) {
	repo.rMutex.Lock()
	defer repo.rMutex.Unlock()

	obj, err := repo.r.BlobObject(plumbing.NewHash(hash.String()))
	if err == plumbing.ErrObjectNotFound {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	r, err := obj.Reader()
	if err != nil {
		return nil, err
	}

	// TODO: return a io.Reader instead
	return io.ReadAll(r)
}

// StoreTree will store a mapping key-->Hash as a Git tree
func (repo *GoGitRepo) StoreTree(mapping []TreeEntry) (Hash, error) {
	var tree object.Tree

	// TODO: can be removed once https://github.com/go-git/go-git/issues/193 is resolved
	sorted := make([]TreeEntry, len(mapping))
	copy(sorted, mapping)
	sort.Slice(sorted, func(i, j int) bool {
		nameI := sorted[i].Name
		if sorted[i].ObjectType == Tree {
			nameI += "/"
		}
		nameJ := sorted[j].Name
		if sorted[j].ObjectType == Tree {
			nameJ += "/"
		}
		return nameI < nameJ
	})

	for _, entry := range sorted {
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
	cfg, err := repo.r.Config()
	if err != nil {
		return "", err
	}

	commit := object.Commit{
		Author: object.Signature{
			Name:  cfg.Author.Name,
			Email: cfg.Author.Email,
			When:  time.Now(),
		},
		Committer: object.Signature{
			Name:  cfg.Committer.Name,
			Email: cfg.Committer.Email,
			When:  time.Now(),
		},
		Message:  "",
		TreeHash: plumbing.NewHash(treeHash.String()),
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
	err = commit.Encode(obj)
	if err != nil {
		return "", err
	}

	hash, err := repo.r.Storer.SetEncodedObject(obj)
	if err != nil {
		return "", err
	}

	return Hash(hash.String()), nil
}

func (repo *GoGitRepo) ResolveRef(ref string) (Hash, error) {
	r, err := repo.r.Reference(plumbing.ReferenceName(ref), false)
	if err == plumbing.ErrReferenceNotFound {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return Hash(r.Hash().String()), nil
}

// UpdateRef will create or update a Git reference
func (repo *GoGitRepo) UpdateRef(ref string, hash Hash) error {
	return repo.r.Storer.SetReference(plumbing.NewHashReference(plumbing.ReferenceName(ref), plumbing.NewHash(hash.String())))
}

// RemoveRef will remove a Git reference
func (repo *GoGitRepo) RemoveRef(ref string) error {
	return repo.r.Storer.RemoveReference(plumbing.ReferenceName(ref))
}

// ListRefs will return a list of Git ref matching the given refspec
func (repo *GoGitRepo) ListRefs(refPrefix string) ([]string, error) {
	refIter, err := repo.r.References()
	if err != nil {
		return nil, err
	}

	refs := make([]string, 0)

	err = refIter.ForEach(func(ref *plumbing.Reference) error {
		if strings.HasPrefix(ref.Name().String(), refPrefix) {
			refs = append(refs, ref.Name().String())
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return refs, nil
}

// RefExist will check if a reference exist in Git
func (repo *GoGitRepo) RefExist(ref string) (bool, error) {
	_, err := repo.r.Reference(plumbing.ReferenceName(ref), false)
	if err == nil {
		return true, nil
	} else if err == plumbing.ErrReferenceNotFound {
		return false, nil
	}
	return false, err
}

// CopyRef will create a new reference with the same value as another one
func (repo *GoGitRepo) CopyRef(source string, dest string) error {
	r, err := repo.r.Reference(plumbing.ReferenceName(source), false)
	if err == plumbing.ErrReferenceNotFound {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	return repo.r.Storer.SetReference(plumbing.NewHashReference(plumbing.ReferenceName(dest), r.Hash()))
}

// ListCommits will return the list of tree hashes of a ref, in chronological order
func (repo *GoGitRepo) ListCommits(ref string) ([]Hash, error) {
	return nonNativeListCommits(repo, ref)
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

func (repo *GoGitRepo) AllClocks() (map[string]lamport.Clock, error) {
	repo.clocksMutex.Lock()
	defer repo.clocksMutex.Unlock()

	result := make(map[string]lamport.Clock)

	files, err := os.ReadDir(filepath.Join(repo.localStorage.Root(), clockPath))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		name := file.Name()
		if c, ok := repo.clocks[name]; ok {
			result[name] = c
		} else {
			c, err := lamport.LoadPersistedClock(repo.LocalStorage(), filepath.Join(clockPath, name))
			if err != nil {
				return nil, err
			}
			repo.clocks[name] = c
			result[name] = c
		}
	}

	return result, nil
}

// GetOrCreateClock return a Lamport clock stored in the Repo.
// If the clock doesn't exist, it's created.
func (repo *GoGitRepo) GetOrCreateClock(name string) (lamport.Clock, error) {
	repo.clocksMutex.Lock()
	defer repo.clocksMutex.Unlock()

	c, err := repo.getClock(name)
	if err == nil {
		return c, nil
	}
	if err != ErrClockNotExist {
		return nil, err
	}

	c, err = lamport.NewPersistedClock(repo.LocalStorage(), filepath.Join(clockPath, name))
	if err != nil {
		return nil, err
	}

	repo.clocks[name] = c
	return c, nil
}

func (repo *GoGitRepo) getClock(name string) (lamport.Clock, error) {
	if c, ok := repo.clocks[name]; ok {
		return c, nil
	}

	c, err := lamport.LoadPersistedClock(repo.LocalStorage(), filepath.Join(clockPath, name))
	if err == nil {
		repo.clocks[name] = c
		return c, nil
	}
	if err == lamport.ErrClockNotExist {
		return nil, ErrClockNotExist
	}
	return nil, err
}

// Increment is equivalent to c = GetOrCreateClock(name) + c.Increment()
func (repo *GoGitRepo) Increment(name string) (lamport.Time, error) {
	c, err := repo.GetOrCreateClock(name)
	if err != nil {
		return lamport.Time(0), err
	}
	return c.Increment()
}

// Witness is equivalent to c = GetOrCreateClock(name) + c.Witness(time)
func (repo *GoGitRepo) Witness(name string, time lamport.Time) error {
	c, err := repo.GetOrCreateClock(name)
	if err != nil {
		return err
	}
	return c.Witness(time)
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

// resolveRefToHash resolves a branch/tag name or raw hash to a commit hash.
// Resolution order: refs/heads/<ref>, refs/tags/<ref>, full ref name, raw commit hash.
// Annotated tags are peeled to their target commit.
func (repo *GoGitRepo) resolveRefToHash(ref string) (plumbing.Hash, error) {
	for _, prefix := range []string{"refs/heads/", "refs/tags/"} {
		r, err := repo.r.Reference(plumbing.ReferenceName(prefix+ref), true)
		if err == nil {
			return repo.peelToCommit(r.Hash())
		}
	}
	// try as a full ref name
	r, err := repo.r.Reference(plumbing.ReferenceName(ref), true)
	if err == nil {
		return repo.peelToCommit(r.Hash())
	}
	// try as a raw commit hash
	h := plumbing.NewHash(ref)
	if h != plumbing.ZeroHash {
		if _, err := repo.r.CommitObject(h); err == nil {
			return h, nil
		}
	}
	return plumbing.ZeroHash, ErrNotFound
}

// defaultBranchName returns the short name of the default branch.
func (repo *GoGitRepo) defaultBranchName() string {
	repo.rMutex.Lock()
	defer repo.rMutex.Unlock()

	// refs/remotes/origin/HEAD is a symbolic ref set by git clone that points
	// to the remote's default branch (e.g. refs/remotes/origin/main). It is
	// the most reliable signal for "what does the upstream consider default".
	ref, err := repo.r.Reference("refs/remotes/origin/HEAD", false)
	if err == nil && ref.Type() == plumbing.SymbolicReference {
		const prefix = "refs/remotes/origin/"
		if target := ref.Target().String(); strings.HasPrefix(target, prefix) {
			return strings.TrimPrefix(target, prefix)
		}
	}
	// Fall back to well-known names for repos without a configured remote.
	for _, name := range []string{"main", "master", "trunk", "develop"} {
		_, err := repo.r.Reference(plumbing.NewBranchReferenceName(name), false)
		if err == nil {
			return name
		}
	}
	return ""
}

// Branches returns all local branches. IsDefault marks the upstream's default
// branch, determined in order:
//  1. refs/remotes/origin/HEAD (set by git clone, reflects the server default)
//  2. First match among: main, master, trunk, develop
//  3. No branch marked if none of the above resolve
func (repo *GoGitRepo) Branches() ([]BranchInfo, error) {
	defaultBranch := repo.defaultBranchName()

	repo.rMutex.Lock()
	defer repo.rMutex.Unlock()

	refs, err := repo.r.References()
	if err != nil {
		return nil, err
	}

	var branches []BranchInfo
	err = refs.ForEach(func(r *plumbing.Reference) error {
		if !r.Name().IsBranch() {
			return nil
		}
		branches = append(branches, BranchInfo{
			Name:      r.Name().Short(),
			Hash:      Hash(r.Hash().String()),
			IsDefault: r.Name().Short() == defaultBranch,
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
func (repo *GoGitRepo) Tags() ([]TagInfo, error) {
	repo.rMutex.Lock()
	defer repo.rMutex.Unlock()

	refs, err := repo.r.References()
	if err != nil {
		return nil, err
	}

	var tags []TagInfo
	err = refs.ForEach(func(r *plumbing.Reference) error {
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

// TreeAtPath returns the entries of the directory at path under ref.
func (repo *GoGitRepo) TreeAtPath(ref, path string) ([]TreeEntry, error) {
	path = strings.Trim(path, "/")

	repo.rMutex.Lock()
	defer repo.rMutex.Unlock()

	startHash, err := repo.resolveRefToHash(ref)
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
// path under ref. rMutex is held for the entire function, covering all
// shared-Scanner access (CommitObject, Tree, File). The returned reader is
// safe to use without the mutex: small blobs are already materialized into a
// MemoryObject (bytes.Reader) by the time File() returns; large blobs come
// back as an FSObject whose Reader() opens its own independent file handle and
// Scanner and then reads via ReadAt — no shared state is touched after this
// function returns. Callers must Close the reader.
func (repo *GoGitRepo) BlobAtPath(ref, path string) (io.ReadCloser, int64, Hash, error) {
	path = strings.Trim(path, "/")
	if path == "" {
		return nil, 0, "", ErrNotFound
	}

	repo.rMutex.Lock()
	defer repo.rMutex.Unlock()

	startHash, err := repo.resolveRefToHash(ref)
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

// CommitLog returns at most limit commits reachable from ref, optionally
// filtered to those that touched path, starting after the given cursor hash,
// and bounded by the since/until author-date range.
func (repo *GoGitRepo) CommitLog(ref, path string, limit int, after Hash, since, until *time.Time) ([]CommitMeta, error) {
	repo.rMutex.Lock()
	defer repo.rMutex.Unlock()

	startHash, err := repo.resolveRefToHash(ref)
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
// cache key regardless of which branch or ref was resolved.
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
// content-addressed, two refs that point to the same directory tree share one
// cache entry, and the cache never needs invalidation: a changed directory
// produces a new tree hash, which becomes a new key.
func (repo *GoGitRepo) LastCommitForEntries(ref, path string, names []string) (map[string]CommitMeta, error) {
	// Normalize path up front so the cache key is canonical.
	path = strings.Trim(path, "/")

	// Resolve ref and load the current directory tree in one brief lock.
	// We need the tree hash for the cache key and we keep the entries to
	// seed the parent-reuse optimisation in the walk below.
	repo.rMutex.Lock()
	startHash, err := repo.resolveRefToHash(ref)
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

	// Cache hit: filter the stored result down to the requested names.
	if cached, ok := repo.lastCommitCache.Get(cacheKey); ok {
		result := make(map[string]CommitMeta, len(names))
		for _, n := range names {
			if m, found := cached[n]; found {
				result[n] = m
			}
		}
		return result, nil
	}

	// Cache miss: walk history for ALL entries in this directory so the
	// cached result is complete and valid for any future name subset.
	remaining := make(map[string]bool, len(startEntries))
	for name := range startEntries {
		remaining[name] = true
	}
	result := make(map[string]CommitMeta, len(remaining))

	repo.rMutex.Lock()

	iter, err := repo.r.Log(&gogit.LogOptions{
		From:  startHash,
		Order: gogit.LogOrderCommitterTime,
	})
	if err != nil {
		repo.rMutex.Unlock()
		return nil, err
	}

	// Seed the parent-reuse cache with the entries we already fetched above
	// so the first iteration's current-tree read is skipped for free.
	// In a linear history this halves tree reads for every subsequent step:
	// the parent fetched at depth D is the current commit at depth D+1.
	cachedParentHash := startHash
	cachedParentEntries := startEntries

	for depth := 0; len(remaining) > 0 && depth < lastCommitDepthLimit; depth++ {
		c, err := iter.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			iter.Close()
			repo.rMutex.Unlock()
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

	iter.Close()
	repo.rMutex.Unlock()

	// Store a defensive copy so that callers cannot mutate cached entries.
	// The cached map contains all directory entries, not just the requested
	// names, so future calls for the same directory are fully served from
	// cache regardless of which names they request.
	cached := make(map[string]CommitMeta, len(result))
	for k, v := range result {
		cached[k] = v
	}
	repo.lastCommitCache.Add(cacheKey, cached)

	// Return only the entries that were requested.
	filtered := make(map[string]CommitMeta, len(names))
	for _, n := range names {
		if m, ok := result[n]; ok {
			filtered[n] = m
		}
	}
	return filtered, nil
}

// CommitDetail returns the full commit metadata and list of changed files.
func (repo *GoGitRepo) CommitDetail(hash Hash) (CommitDetail, error) {
	repo.rMutex.Lock()
	defer repo.rMutex.Unlock()

	c, err := repo.r.CommitObject(plumbing.NewHash(hash.String()))
	if err == plumbing.ErrObjectNotFound {
		return CommitDetail{}, ErrNotFound
	}
	if err != nil {
		return CommitDetail{}, err
	}

	toTree, err := c.Tree()
	if err != nil {
		return CommitDetail{}, err
	}

	var fromTree *object.Tree
	if len(c.ParentHashes) > 0 {
		parent, err := repo.r.CommitObject(c.ParentHashes[0])
		if err != nil {
			return CommitDetail{}, fmt.Errorf("loading parent commit: %w", err)
		}
		fromTree, err = parent.Tree()
		if err != nil {
			return CommitDetail{}, fmt.Errorf("loading parent tree: %w", err)
		}
	}

	changes, err := object.DiffTree(fromTree, toTree)
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
func (repo *GoGitRepo) CommitFileDiff(hash Hash, filePath string) (FileDiff, error) {
	repo.rMutex.Lock()
	defer repo.rMutex.Unlock()

	c, err := repo.r.CommitObject(plumbing.NewHash(hash.String()))
	if err == plumbing.ErrObjectNotFound {
		return FileDiff{}, ErrNotFound
	}
	if err != nil {
		return FileDiff{}, err
	}

	toTree, err := c.Tree()
	if err != nil {
		return FileDiff{}, err
	}

	var fromTree *object.Tree
	if len(c.ParentHashes) > 0 {
		parent, err := repo.r.CommitObject(c.ParentHashes[0])
		if err != nil {
			return FileDiff{}, fmt.Errorf("loading parent commit: %w", err)
		}
		fromTree, err = parent.Tree()
		if err != nil {
			return FileDiff{}, fmt.Errorf("loading parent tree: %w", err)
		}
	}

	changes, err := object.DiffTree(fromTree, toTree)
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

		patch, err := ch.Patch()
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

// AddRemote add a new remote to the repository
// Not in the interface because it's only used for testing
func (repo *GoGitRepo) AddRemote(name string, url string) error {
	_, err := repo.r.CreateRemote(&config.RemoteConfig{
		Name: name,
		URLs: []string{url},
	})

	return err
}

// GetLocalRemote return the URL to use to add this repo as a local remote
func (repo *GoGitRepo) GetLocalRemote() string {
	return repo.path
}

// EraseFromDisk delete this repository entirely from the disk
func (repo *GoGitRepo) EraseFromDisk() error {
	err := repo.Close()
	if err != nil {
		return err
	}

	path := filepath.Clean(strings.TrimSuffix(repo.path, string(filepath.Separator)+".git"))

	// fmt.Println("Cleaning repo:", path)
	return os.RemoveAll(path)
}
