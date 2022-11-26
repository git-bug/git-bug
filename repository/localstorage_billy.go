package repository

import (
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/util"
)

var _ LocalStorage = &billyLocalStorage{}

type billyLocalStorage struct {
	billy.Filesystem
}

func (b billyLocalStorage) RemoveAll(path string) error {
	return util.RemoveAll(b.Filesystem, path)
}
