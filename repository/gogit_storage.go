package repository

import (
	"errors"
	"io"
	"slices"
	"sort"
	"sync"

	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/go-git/go-git/v5/storage/filesystem"
)

// reindexingStorage is the storer behind every GoGitRepo. It heals go-git's
// packfile index when other processes change the repository.
//
// go-git parses the .idx of every packfile on the first packed lookup and never
// looks at objects/pack again, so a pack written by another process (a native
// `git fetch` or `git gc`) is invisible, and a pack deleted by `git gc` leaves
// the index pointing at a file that no longer opens. When an object read fails
// and the set of packs on disk has changed since the index was built, the index
// is rebuilt and the read retried once.
//
// This is a storer decorator rather than a wrapper around GoGitRepo's own
// methods because go-git resolves objects internally (commit walks, tree
// decoding) through whatever storer the Repository holds.
//
// go-git's ObjectStorage is not safe for concurrent use: its index map is built
// lazily, even by reads, and written by the packfile writer when a fetch
// completes. GoGitRepo's rMutex serializes its own reads, but not go-git's
// fetch and push, which run without it. mu therefore guards every access to the
// index made through this storer.
type reindexingStorage struct {
	*filesystem.Storage

	mu sync.Mutex
	// packs is the list of packfiles on disk before the index was last built.
	packs []plumbing.Hash
	// stale is set when the last index build failed, so the index must be
	// rebuilt on the next miss whatever packs says.
	stale bool
}

// newReindexingStorage opens the storage for the git directory at dotGitPath,
// configured as gogit.PlainOpen would.
func newReindexingStorage(dotGitPath string) *reindexingStorage {
	s := &reindexingStorage{
		Storage: filesystem.NewStorage(osfs.New(dotGitPath), cache.NewObjectLRUDefault()),
	}
	s.mu.Lock()
	s.loadIndex()
	s.mu.Unlock()
	return s
}

// loadIndex builds go-git's packfile index now rather than lazily. Keeping the
// index built at all times is required for correctness, not just speed: the
// packfile writer adds its new pack to the index map when it closes, and would
// panic on a nil map if the index had been dropped in the meantime.
//
// Must be called with mu held.
func (s *reindexingStorage) loadIndex() {
	// The pack list is taken before the index is built, so a pack that lands
	// in between is seen as a change on the next miss.
	packs, err := s.listPacks()
	if err != nil {
		s.stale = true
		return
	}

	// ObjectStorage has no exported way to build the index; looking up an
	// object that cannot exist builds it and reports any failure to do so.
	err = s.Storage.HasEncodedObject(plumbing.ZeroHash)
	if err != nil && !errors.Is(err, plumbing.ErrObjectNotFound) {
		// A pack could not be indexed, typically because another process
		// was in the middle of writing or deleting it. Mark the index as
		// stale so the next miss rebuilds it.
		s.stale = true
		return
	}
	s.packs = packs
	s.stale = false
}

// refresh rebuilds the index if the packs on disk changed since it was last
// built, and reports whether it did.
//
// Must be called with mu held.
func (s *reindexingStorage) refresh() bool {
	if !s.stale {
		packs, err := s.listPacks()
		if err != nil || slices.Equal(packs, s.packs) {
			return false
		}
	}
	s.Storage.Reindex()
	s.loadIndex()
	return true
}

// listPacks returns the packfiles on disk, in a stable order.
func (s *reindexingStorage) listPacks() ([]plumbing.Hash, error) {
	packs, err := s.Storage.ObjectPacks()
	if err != nil {
		return nil, err
	}
	sort.Sort(plumbing.HashSlice(packs))
	return packs, nil
}

// Reindex rebuilds the packfile index unconditionally.
func (s *reindexingStorage) Reindex() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Storage.Reindex()
	s.loadIndex()
}

// withRetry runs a read under mu and, if it fails and the packs changed, runs
// it once more. Any error is retried, not only ErrObjectNotFound: a pack
// deleted by `git gc` surfaces as an I/O error.
func withRetry[T any](s *reindexingStorage, read func() (T, error)) (T, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := read()
	if err != nil && s.refresh() {
		return read()
	}
	return res, err
}

// EncodedObject returns the object of type t (or of any type, for
// plumbing.AnyObject) with hash h, retrying once against a rebuilt index if the
// packs on disk changed.
func (s *reindexingStorage) EncodedObject(t plumbing.ObjectType, h plumbing.Hash) (plumbing.EncodedObject, error) {
	return withRetry(s, func() (plumbing.EncodedObject, error) {
		return s.Storage.EncodedObject(t, h)
	})
}

// DeltaObject is EncodedObject, except that an object stored as a delta in a
// pack is returned as that delta rather than resolved, so that building a pack
// to push can reuse it instead of computing a new one.
func (s *reindexingStorage) DeltaObject(t plumbing.ObjectType, h plumbing.Hash) (plumbing.EncodedObject, error) {
	return withRetry(s, func() (plumbing.EncodedObject, error) {
		return s.Storage.DeltaObject(t, h)
	})
}

// HasEncodedObject returns nil if the object with hash h exists, without
// reading it, or ErrObjectNotFound if it still does not after the retry.
func (s *reindexingStorage) HasEncodedObject(h plumbing.Hash) error {
	_, err := withRetry(s, func() (struct{}, error) {
		return struct{}{}, s.Storage.HasEncodedObject(h)
	})
	return err
}

// EncodedObjectSize returns the uncompressed size of the object with hash h,
// without reading its content.
func (s *reindexingStorage) EncodedObjectSize(h plumbing.Hash) (int64, error) {
	return withRetry(s, func() (int64, error) {
		return s.Storage.EncodedObjectSize(h)
	})
}

// HashesWithPrefix returns the hashes of every object starting with prefix,
// for resolving abbreviated hashes. The index is refreshed first rather than
// on a miss: a stale index can return some matches but not all, and an
// ambiguous abbreviation would then look unique.
func (s *reindexingStorage) HashesWithPrefix(prefix []byte) ([]plumbing.Hash, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.refresh()
	return s.Storage.HashesWithPrefix(prefix)
}

// IterEncodedObjects returns an iterator over every object of type t, loose and
// packed. It covers the packs on disk when it is created, opening each one as
// it reaches it: a pack added meanwhile is not visited, and one removed
// meanwhile makes the iteration fail, with no retry.
func (s *reindexingStorage) IterEncodedObjects(t plumbing.ObjectType) (storer.EncodedObjectIter, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	iter, err := s.Storage.IterEncodedObjects(t)
	if err != nil {
		return nil, err
	}
	// go-git fixed the iterator's list of packs above, but reads their
	// indexes only later, in Next, and crashes on a pack it doesn't know.
	// Refreshing now makes the index cover every pack the iterator can
	// visit. If rebuilding it failed, typically on a pack whose .idx is still
	// being written, some may be missing: fail now rather than crash later.
	// Fix proposed upstream: https://github.com/go-git/go-git/pull/2448
	s.refresh()
	if s.stale {
		iter.Close()
		return nil, errors.New("can't index every pack: retry once concurrent git operations are done")
	}
	// The iterator reads the index lazily as it moves from one pack to the
	// next, so each step needs the lock too.
	return &lockedObjectIter{mu: &s.mu, iter: iter}, nil
}

// PackfileWriter returns a writer for a new pack, which go-git uses to store
// what a fetch receives. When the writer is closed, the pack is renamed into
// place and added to the index.
func (s *reindexingStorage) PackfileWriter() (io.WriteCloser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	w, err := s.Storage.PackfileWriter()
	if err != nil {
		return nil, err
	}
	return &lockedPackWriter{mu: &s.mu, WriteCloser: w}, nil
}

// lockedObjectIter takes the storage lock around each step of an object
// iterator, but not around the ForEach callback, which may read objects.
type lockedObjectIter struct {
	mu   *sync.Mutex
	iter storer.EncodedObjectIter
}

func (i *lockedObjectIter) Next() (plumbing.EncodedObject, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.iter.Next()
}

func (i *lockedObjectIter) ForEach(cb func(plumbing.EncodedObject) error) error {
	return storer.ForEachIterator(i, cb)
}

func (i *lockedObjectIter) Close() {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.iter.Close()
}

// lockedPackWriter takes the storage lock on Close, which is when go-git adds
// the new pack to the index map. Writes only touch the pack's own temp file.
type lockedPackWriter struct {
	io.WriteCloser
	mu *sync.Mutex
}

func (w *lockedPackWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.WriteCloser.Close()
}
