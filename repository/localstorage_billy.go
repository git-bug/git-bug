package repository

import (
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-billy/v5/util"
)

var _ LocalStorage = &billyLocalStorage{}

type billyLocalStorage struct {
	billy.Filesystem
}

// NewLocalStorage returns a LocalStorage of the directory at root.
func NewLocalStorage(root string) LocalStorage {
	return billyLocalStorage{Filesystem: osfs.New(root)}
}

func (b billyLocalStorage) RemoveAll(path string) error {
	return util.RemoveAll(b.Filesystem, path)
}

var _ RepoStorage = &repoStorage{}

type repoStorage struct {
	storage LocalStorage
}

// NewRepoStorage returns the RepoStorage of storage.
func NewRepoStorage(storage LocalStorage) RepoStorage {
	return &repoStorage{storage: storage}
}

func (r *repoStorage) LocalStorage() LocalStorage {
	return r.storage
}
