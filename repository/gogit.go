package repository

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	stdpath "path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/osfs"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	storagefs "github.com/go-git/go-git/v5/storage/filesystem"

	"github.com/MichaelMure/git-bug/util/lamport"
)

var _ ClockedRepo = &GoGitRepo{}
var _ TestedRepo = &GoGitRepo{}

type GoGitRepo struct {
	r    *gogit.Repository
	path string

	clocksMutex sync.Mutex
	clocks      map[string]lamport.Clock

	keyring    Keyring
	filesystem billy.Filesystem
	RepoStorage
}

// NewGoGitRepo opens a git repository and returns a GoGitRepo object
// fs is the filesystem containing the git repo to be opened, or if nil, a billy.Filesystem
// is created which accesses the device filesystem (using go-billy osfs).
// path is relative the the root of the filesystem used, and may be the repo
// directory or a subdirectory of the git repo being opened.
func NewGoGitRepo(path string, clockLoaders []ClockLoader, fs billy.Filesystem) (*GoGitRepo, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	if fs == nil {
		fs = osfs.New(path)
	}

	repoPath, gitDirPath, err := detectRepoPaths(path, fs)
	if err != nil {
		return nil, err
	}

	dotGitFs, err := fs.Chroot(gitDirPath)
	if err != nil {
		return nil, err
	}

	wt, err := fs.Chroot(repoPath)
	if err != nil {
		return nil, err
	}

	r, err := gogit.Open(storagefs.NewStorage(dotGitFs, nil), wt)
	if err != nil {
		return nil, err
	}

	k, err := defaultKeyring()
	if err != nil {
		return nil, err
	}

	repo := &GoGitRepo{
		r:          r,
		path:       gitDirPath,
		clocks:     make(map[string]lamport.Clock),
		keyring:    k,
		filesystem: fs,
	}

	for _, loader := range clockLoaders {
		allExist := true
		for _, name := range loader.Clocks {
			if _, err := repo.getClock(name); err != nil {
				allExist = false
			}
		}

		if !allExist {
			err = loader.Witnesser(repo)
			if err != nil {
				return nil, err
			}
		}
	}

	return repo, nil
}

// Returns repo and git paths which has a plain or bare git repo
// starting from path and checking each parent
func detectRepoPaths(path string, fs billy.Filesystem) (string, string, error) {
	// normalize the path
	path, err := filepath.Abs(path)
	if err != nil {
		return "", "", err
	}

	for {
		var gitDirPath string
		if filepath.Base(path) == ".git" {
			gitDirPath = path
		} else {
			gitDirPath = stdpath.Join(path, ".git")
		}

		fi, err := fs.Stat(gitDirPath)
		if err == nil {
			if !fi.IsDir() {
				return "", "", fmt.Errorf(".git exist but is not a directory")
			}
			return stdpath.Dir(gitDirPath), gitDirPath, nil
		}
		if !os.IsNotExist(err) {
			// unknown error
			return "", "", err
		}

		// detect bare repo
		ok, err := isGitDir(path, fs)
		if err != nil {
			return "", "", err
		}
		if ok {
			return path, path, nil
		}

		if parent := filepath.Dir(path); parent == path {
			return "", "", fmt.Errorf(".git not found")
		} else {
			path = parent
		}
	}
}

func isGitDir(path string, fs billy.Filesystem) (bool, error) {
	markers := []string{"HEAD", "objects", "refs"}

	for _, marker := range markers {
		_, err := fs.Stat(stdpath.Join(path, marker))
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

// InitGoGitRepo create a new empty git repo at the given path
// fs is the filesystem and if nil a go-billy/osfs will be used (local filesystem)
func InitGoGitRepo(path string, fs billy.Filesystem) (*GoGitRepo, error) {
	println("InitGoGitRepo()... at:", path)

	if fs == nil {
		fs = osfs.New(path)
	}
	dotGitFs, err := fs.Chroot(".git")
	if err != nil {
		return nil, err
	}

	r, err := gogit.Init(storagefs.NewStorage(dotGitFs, nil), fs)
	if err != nil {
		return nil, err
	}

	k, err := defaultKeyring()
	if err != nil {
		return nil, err
	}

	return &GoGitRepo{
		r:          r,
		path:       path + "/.git",
		clocks:     make(map[string]lamport.Clock),
		keyring:    k,
		filesystem: fs,
	}, nil
}

// InitBareGoGitRepo create a new --bare empty git repo at the given path
// fs is the filesystem and if nil a go-billy/osfs will be used (local filesystem)
func InitBareGoGitRepo(path string, fs billy.Filesystem) (*GoGitRepo, error) {
	println("InitBareGoGitRepo()... at:", path)

	if fs == nil {
		fs = osfs.New(path)
	}
	r, err := gogit.Init(storagefs.NewStorage(fs, nil), nil)
	if err != nil {
		return nil, err
	}

	k, err := defaultKeyring()
	if err != nil {
		return nil, err
	}

	return &GoGitRepo{
		r:          r,
		path:       path,
		clocks:     make(map[string]lamport.Clock),
		keyring:    k,
		filesystem: fs,
	}, nil
}

// LocalConfig give access to the repository scoped configuration
func (repo *GoGitRepo) LocalConfig() Config {
	return newGoGitLocalConfig(repo.r)
}

// GlobalConfig give access to the global scoped configuration
func (repo *GoGitRepo) GlobalConfig() Config {
	// TODO: replace that with go-git native implementation once it's supported
	// see: https://github.com/go-git/go-git
	// see: https://github.com/src-d/go-git/issues/760
	return newGoGitGlobalConfig(repo.r)
}

// AnyConfig give access to a merged local/global configuration
func (repo *GoGitRepo) AnyConfig() ConfigRead {
	return mergeConfig(repo.LocalConfig(), repo.GlobalConfig())
}

// Keyring give access to a user-wide storage for secrets
func (repo *GoGitRepo) Keyring() Keyring {
	return repo.keyring
}

// GetUserName returns the name the the user has used to configure git
func (repo *GoGitRepo) GetUserName() (string, error) {
	return repo.AnyConfig().ReadString("user.name")
}

// GetUserEmail returns the email address that the user has used to configure git.
func (repo *GoGitRepo) GetUserEmail() (string, error) {
	return repo.AnyConfig().ReadString("user.email")
}

// GitDirPath returns the full path to the repo git directory (e.g. on the local device)
func (repo *GoGitRepo) GitDirPath() string {
	return repo.path
}

// LocalStorage return a billy.Filesystem giving access to $RepoPath/.git/git-bug
func (repo *GoGitRepo) LocalStorage() billy.Filesystem {
	return repo.filesystem
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
	if err != nil && err != ErrNoConfigEntry {
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
		if _, err = exec.LookPath(cmd); err == nil {
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

// FetchRefs fetch git refs from a remote
func (repo *GoGitRepo) FetchRefs(remote string, refSpec string) (string, error) {
	buf := bytes.NewBuffer(nil)

	err := repo.r.Fetch(&gogit.FetchOptions{
		RemoteName: remote,
		RefSpecs:   []config.RefSpec{config.RefSpec(refSpec)},
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

// PushRefs push git refs to a remote
func (repo *GoGitRepo) PushRefs(remote string, refSpec string) (string, error) {
	buf := bytes.NewBuffer(nil)

	err := repo.r.Push(&gogit.PushOptions{
		RemoteName: remote,
		RefSpecs:   []config.RefSpec{config.RefSpec(refSpec)},
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
	obj, err := repo.r.BlobObject(plumbing.NewHash(hash.String()))
	if err != nil {
		return nil, err
	}

	r, err := obj.Reader()
	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(r)
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
	h := plumbing.NewHash(hash.String())

	// the given hash could be a tree or a commit
	obj, err := repo.r.Storer.EncodedObject(plumbing.AnyObject, h)
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
func (repo *GoGitRepo) StoreCommit(treeHash Hash) (Hash, error) {
	return repo.StoreCommitWithParent(treeHash, "")
}

// StoreCommit will store a Git commit with the given Git tree
func (repo *GoGitRepo) StoreCommitWithParent(treeHash Hash, parent Hash) (Hash, error) {
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

	if parent != "" {
		commit.ParentHashes = []plumbing.Hash{plumbing.NewHash(parent.String())}
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

// GetTreeHash return the git tree hash referenced in a commit
func (repo *GoGitRepo) GetTreeHash(commit Hash) (Hash, error) {
	obj, err := repo.r.CommitObject(plumbing.NewHash(commit.String()))
	if err != nil {
		return "", err
	}

	return Hash(obj.TreeHash.String()), nil
}

// FindCommonAncestor will return the last common ancestor of two chain of commit
func (repo *GoGitRepo) FindCommonAncestor(commit1 Hash, commit2 Hash) (Hash, error) {
	obj1, err := repo.r.CommitObject(plumbing.NewHash(commit1.String()))
	if err != nil {
		return "", err
	}
	obj2, err := repo.r.CommitObject(plumbing.NewHash(commit2.String()))
	if err != nil {
		return "", err
	}

	commits, err := obj1.MergeBase(obj2)
	if err != nil {
		return "", err
	}

	return Hash(commits[0].Hash.String()), nil
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
	if err != nil {
		return err
	}
	return repo.r.Storer.SetReference(plumbing.NewHashReference(plumbing.ReferenceName(dest), r.Hash()))
}

// ListCommits will return the list of tree hashes of a ref, in chronological order
func (repo *GoGitRepo) ListCommits(ref string) ([]Hash, error) {
	println("ListCommits() for ref: ", ref)
	r, err := repo.r.Reference(plumbing.ReferenceName(ref), false)
	if err != nil {
		return nil, err
	}
	println("repo.r.CommitObject(r.Hash()) hash:", r.Hash().String())
	commit, err := repo.r.CommitObject(r.Hash()) // <<crashes here for test TestQueries
	println("DONE repo.r.CommitObject(r.Hash())")
	if err != nil {
		return nil, err
	}
	hashes := []Hash{Hash(commit.Hash.String())}

	for {
		commit, err = commit.Parent(0)
		if err == object.ErrParentNotFound {
			break
		}
		if err != nil {
			return nil, err
		}

		if commit.NumParents() > 1 {
			return nil, fmt.Errorf("multiple parents")
		}

		hashes = append([]Hash{Hash(commit.Hash.String())}, hashes...)
	}

	return hashes, nil
}

// GetOrCreateClock return a Lamport clock stored in the Repo.
// If the clock doesn't exist, it's created.
func (repo *GoGitRepo) GetOrCreateClock(name string) (lamport.Clock, error) {
	c, err := repo.getClock(name)
	if err == nil {
		return c, nil
	}
	if err != ErrClockNotExist {
		return nil, err
	}

	repo.clocksMutex.Lock()
	defer repo.clocksMutex.Unlock()

	p := stdpath.Join(repo.path, clockPath, name+"-clock")

	c, err = lamport.NewPersistedClock(p)
	if err != nil {
		return nil, err
	}

	repo.clocks[name] = c
	return c, nil
}

func (repo *GoGitRepo) getClock(name string) (lamport.Clock, error) {
	repo.clocksMutex.Lock()
	defer repo.clocksMutex.Unlock()

	if c, ok := repo.clocks[name]; ok {
		return c, nil
	}

	p := stdpath.Join(repo.path, clockPath, name+"-clock")

	c, err := lamport.LoadPersistedClock(p)
	if err == nil {
		repo.clocks[name] = c
		return c, nil
	}
	if err == lamport.ErrClockNotExist {
		return nil, ErrClockNotExist
	}
	return nil, err
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
	// FIXME path := filepath.Clean(strings.TrimSuffix(repo.path, string(filepath.Separator)+".git"))

	// fmt.Println("Cleaning repo:", path)
	return nil //FIXME repo.LocalStorage().RemoveAll(path)
}
