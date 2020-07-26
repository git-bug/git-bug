package repository

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	stdpath "path"
	"path/filepath"
	"sync"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"

	"github.com/MichaelMure/git-bug/util/lamport"
)

var _ ClockedRepo = &GoGitRepo{}

type GoGitRepo struct {
	r    *gogit.Repository
	path string

	clocksMutex sync.Mutex
	clocks      map[string]lamport.Clock

	keyring Keyring
}

func NewGoGitRepo(path string, clockLoaders []ClockLoader) (*GoGitRepo, error) {
	path, err := detectGitPath(path)
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
		r:       r,
		path:    path,
		clocks:  make(map[string]lamport.Clock),
		keyring: k,
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

func detectGitPath(path string) (string, error) {
	// normalize the path
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	for {
		fi, err := os.Stat(stdpath.Join(path, ".git"))
		if err == nil {
			if !fi.IsDir() {
				return "", fmt.Errorf(".git exist but is not a directory")
			}
			return stdpath.Join(path, ".git"), nil
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
		_, err := os.Stat(stdpath.Join(path, marker))
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
func InitGoGitRepo(path string) (*GoGitRepo, error) {
	r, err := gogit.PlainInit(path, false)
	if err != nil {
		return nil, err
	}

	return &GoGitRepo{
		r:      r,
		path:   path + "/.git",
		clocks: make(map[string]lamport.Clock),
	}, nil
}

// InitBareGoGitRepo create a new --bare empty git repo at the given path
func InitBareGoGitRepo(path string) (*GoGitRepo, error) {
	r, err := gogit.PlainInit(path, true)
	if err != nil {
		return nil, err
	}

	return &GoGitRepo{
		r:      r,
		path:   path,
		clocks: make(map[string]lamport.Clock),
	}, nil
}

func (repo *GoGitRepo) LocalConfig() Config {
	return newGoGitConfig(repo.r)
}

func (repo *GoGitRepo) GlobalConfig() Config {
	panic("go-git doesn't support writing global config")
}

func (repo *GoGitRepo) Keyring() Keyring {
	return repo.keyring
}

// GetPath returns the path to the repo.
func (repo *GoGitRepo) GetPath() string {
	return repo.path
}

// GetUserName returns the name the the user has used to configure git
func (repo *GoGitRepo) GetUserName() (string, error) {
	cfg, err := repo.r.Config()
	if err != nil {
		return "", err
	}

	return cfg.User.Name, nil
}

// GetUserEmail returns the email address that the user has used to configure git.
func (repo *GoGitRepo) GetUserEmail() (string, error) {
	cfg, err := repo.r.Config()
	if err != nil {
		return "", err
	}

	return cfg.User.Email, nil
}

// GetCoreEditor returns the name of the editor that the user has used to configure git.
func (repo *GoGitRepo) GetCoreEditor() (string, error) {

	panic("implement me")
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

func (repo *GoGitRepo) StoreTree(mapping []TreeEntry) (Hash, error) {
	panic("implement me")
}

func (repo *GoGitRepo) ReadTree(hash Hash) ([]TreeEntry, error) {
	panic("implement me")
}

func (repo *GoGitRepo) StoreCommit(treeHash Hash) (Hash, error) {
	panic("implement me")
}

func (repo *GoGitRepo) StoreCommitWithParent(treeHash Hash, parent Hash) (Hash, error) {
	panic("implement me")
}

func (repo *GoGitRepo) GetTreeHash(commit Hash) (Hash, error) {
	panic("implement me")
}

func (repo *GoGitRepo) FindCommonAncestor(commit1 Hash, commit2 Hash) (Hash, error) {
	panic("implement me")
}

func (repo *GoGitRepo) UpdateRef(ref string, hash Hash) error {
	panic("implement me")
}

func (repo *GoGitRepo) RemoveRef(ref string) error {
	panic("implement me")
}

func (repo *GoGitRepo) ListRefs(refspec string) ([]string, error) {
	panic("implement me")
}

func (repo *GoGitRepo) RefExist(ref string) (bool, error) {
	panic("implement me")
}

func (repo *GoGitRepo) CopyRef(source string, dest string) error {
	panic("implement me")
}

func (repo *GoGitRepo) ListCommits(ref string) ([]Hash, error) {
	panic("implement me")
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

	p := clockPath + name + "-clock"

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

	p := clockPath + name + "-clock"

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
