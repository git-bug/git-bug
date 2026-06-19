package repository

import (
	"io/fs"
	"os"
	"sync"

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

// lockedFilesystem wraps a billy.Filesystem with a RWMutex for thread-safe access.
type lockedFilesystem struct {
	fs billy.Filesystem
	mu sync.RWMutex
}

// NewLockedFilesystem creates a new lockedFilesystem instance.
func NewLockedFilesystem(fs billy.Filesystem) billy.Filesystem {
	return &lockedFilesystem{fs: fs}
}

// Implement billy.Basic interface
func (l *lockedFilesystem) Create(filename string) (billy.File, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.fs.Create(filename)
}

func (l *lockedFilesystem) Open(filename string) (billy.File, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.fs.Open(filename)
}

func (l *lockedFilesystem) OpenFile(filename string, flag int, perm fs.FileMode) (billy.File, error) {
	// If write flag is present, use a write lock, otherwise a read lock.
	if flag&(os.O_WRONLY|os.O_RDWR|os.O_APPEND|os.O_CREATE|os.O_TRUNC) != 0 {
		l.mu.Lock()
		defer l.mu.Unlock()
	} else {
		l.mu.RLock()
		defer l.mu.RUnlock()
	}
	return l.fs.OpenFile(filename, flag, perm)
}

func (l *lockedFilesystem) Stat(filename string) (fs.FileInfo, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.fs.Stat(filename)
}

func (l *lockedFilesystem) Lstat(filename string) (fs.FileInfo, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.fs.Lstat(filename)
}

func (l *lockedFilesystem) Readlink(link string) (string, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.fs.Readlink(link)
}

func (l *lockedFilesystem) Symlink(oldname, newname string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.fs.Symlink(oldname, newname)
}

func (l *lockedFilesystem) Rename(oldpath, newpath string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.fs.Rename(oldpath, newpath)
}

func (l *lockedFilesystem) Remove(filename string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.fs.Remove(filename)
}

func (l *lockedFilesystem) Join(elem ...string) string {
	// Join does not modify state, no lock needed
	return l.fs.Join(elem...)
}

// Implement billy.Dir interface
func (l *lockedFilesystem) ReadDir(path string) ([]fs.FileInfo, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.fs.ReadDir(path)
}

func (l *lockedFilesystem) MkdirAll(path string, perm fs.FileMode) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.fs.MkdirAll(path, perm)
}

// Implement billy.Temp interface
func (l *lockedFilesystem) TempFile(dir, pattern string) (billy.File, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.fs.TempFile(dir, pattern)
}

// Implement billy.Capability interface
func (l *lockedFilesystem) Chroot(path string) (billy.Filesystem, error) {
	l.mu.Lock() // Chroot might change the root, so a write lock is appropriate for the parent FS
	defer l.mu.Unlock()
	// The returned filesystem should also be locked, so we wrap it.
	fs, err := l.fs.Chroot(path)
	if err != nil {
		return nil, err
	}
	return NewLockedFilesystem(fs), nil
}

func (l *lockedFilesystem) Root() string {
	// Root does not modify state, no lock needed
	return l.fs.Root()
}