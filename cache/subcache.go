package cache

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"sync"

	"github.com/pkg/errors"

	"github.com/MichaelMure/git-bug/entities/bug"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/repository"
)

type Excerpt interface {
	Id() entity.Id
}

type CacheEntity interface {
	NeedCommit() bool
}

type cacheMgmt interface {
	Load() error
	Write() error
	Build() error
	Close() error
}

type getUserIdentityFunc func() (*IdentityCache, error)

type SubCache[ExcerptT Excerpt, CacheT CacheEntity, EntityT entity.Interface] struct {
	repo      repository.ClockedRepo
	resolvers func() entity.Resolvers

	getUserIdentity  getUserIdentityFunc
	readWithResolver func(repository.ClockedRepo, entity.Resolvers, entity.Id) (EntityT, error)
	makeCached       func(*SubCache[ExcerptT, CacheT, EntityT], getUserIdentityFunc, EntityT) CacheT
	makeExcerpt      func() Excerpt

	typename  string
	namespace string
	version   uint
	maxLoaded int

	mu       sync.RWMutex
	excerpts map[entity.Id]ExcerptT
	cached   map[entity.Id]CacheT
	lru      *lruIdCache
}

func NewSubCache[ExcerptT Excerpt, CacheT CacheEntity, EntityT entity.Interface](
	repo repository.ClockedRepo,
	resolvers func() entity.Resolvers,
	getUserIdentity getUserIdentityFunc,
	typename, namespace string,
	version uint, maxLoaded int) *SubCache[ExcerptT, CacheT, EntityT] {
	return &SubCache[ExcerptT, CacheT, EntityT]{
		repo:            repo,
		resolvers:       resolvers,
		getUserIdentity: getUserIdentity,
		typename:        typename,
		namespace:       namespace,
		version:         version,
		maxLoaded:       maxLoaded,
		excerpts:        make(map[entity.Id]ExcerptT),
		cached:          make(map[entity.Id]CacheT),
		lru:             newLRUIdCache(),
	}
}

// Load will try to read from the disk the entity cache file
func (sc *SubCache[ExcerptT, CacheT, EntityT]) Load() error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	f, err := sc.repo.LocalStorage().Open(sc.namespace + "-file")
	if err != nil {
		return err
	}

	decoder := gob.NewDecoder(f)

	aux := struct {
		Version  uint
		Excerpts map[entity.Id]ExcerptT
	}{}

	err = decoder.Decode(&aux)
	if err != nil {
		return err
	}

	if aux.Version != sc.version {
		return fmt.Errorf("unknown %s cache format version %v", sc.namespace, aux.Version)
	}

	sc.excerpts = aux.Excerpts

	index, err := sc.repo.GetBleveIndex("bug")
	if err != nil {
		return err
	}

	// simple heuristic to detect a mismatch between the index and the entities
	count, err := index.DocCount()
	if err != nil {
		return err
	}
	if count != uint64(len(sc.excerpts)) {
		return fmt.Errorf("count mismatch between bleve and %s excerpts", sc.namespace)
	}

	return nil
}

// Write will serialize on disk the entity cache file
func (sc *SubCache[ExcerptT, CacheT, EntityT]) Write() error {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	var data bytes.Buffer

	aux := struct {
		Version  uint
		Excerpts map[entity.Id]ExcerptT
	}{
		Version:  sc.version,
		Excerpts: sc.excerpts,
	}

	encoder := gob.NewEncoder(&data)

	err := encoder.Encode(aux)
	if err != nil {
		return err
	}

	f, err := sc.repo.LocalStorage().Create(sc.namespace + "-file")
	if err != nil {
		return err
	}

	_, err = f.Write(data.Bytes())
	if err != nil {
		return err
	}

	return f.Close()
}

func (sc *SubCache[ExcerptT, CacheT, EntityT]) Build() error {

}

func (sc *SubCache[ExcerptT, CacheT, EntityT]) Close() error {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.excerpts = nil
	sc.cached = make(map[entity.Id]CacheT)
	return nil
}

// AllIds return all known bug ids
func (sc *SubCache[ExcerptT, CacheT, EntityT]) AllIds() []entity.Id {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	result := make([]entity.Id, len(sc.excerpts))

	i := 0
	for _, excerpt := range sc.excerpts {
		result[i] = excerpt.Id()
		i++
	}

	return result
}

// Resolve retrieve an entity matching the exact given id
func (sc *SubCache[ExcerptT, CacheT, EntityT]) Resolve(id entity.Id) (CacheT, error) {
	sc.mu.RLock()
	cached, ok := sc.cached[id]
	if ok {
		sc.lru.Get(id)
		sc.mu.RUnlock()
		return cached, nil
	}
	sc.mu.RUnlock()

	b, err := sc.readWithResolver(sc.repo, sc.resolvers(), id)
	if err != nil {
		return nil, err
	}

	cached = sc.makeCached(sc, sc.getUserIdentity, b)

	sc.mu.Lock()
	sc.cached[id] = cached
	sc.lru.Add(id)
	sc.mu.Unlock()

	sc.evictIfNeeded()

	return cached, nil
}

// ResolvePrefix retrieve an entity matching an id prefix. It fails if multiple
// entity match.
func (sc *SubCache[ExcerptT, CacheT, EntityT]) ResolvePrefix(prefix string) (CacheT, error) {
	return sc.ResolveMatcher(func(excerpt ExcerptT) bool {
		return excerpt.Id().HasPrefix(prefix)
	})
}

func (sc *SubCache[ExcerptT, CacheT, EntityT]) ResolveMatcher(f func(ExcerptT) bool) (CacheT, error) {
	id, err := sc.resolveMatcher(f)
	if err != nil {
		return nil, err
	}
	return sc.Resolve(id)
}

// ResolveExcerpt retrieve an Excerpt matching the exact given id
func (sc *SubCache[ExcerptT, CacheT, EntityT]) ResolveExcerpt(id entity.Id) (ExcerptT, error) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	excerpt, ok := sc.excerpts[id]
	if !ok {
		return nil, entity.NewErrNotFound(sc.typename)
	}

	return excerpt, nil
}

// ResolveExcerptPrefix retrieve an Excerpt matching an id prefix. It fails if multiple
// entity match.
func (sc *SubCache[ExcerptT, CacheT, EntityT]) ResolveExcerptPrefix(prefix string) (ExcerptT, error) {
	return sc.ResolveExcerptMatcher(func(excerpt ExcerptT) bool {
		return excerpt.Id().HasPrefix(prefix)
	})
}

func (sc *SubCache[ExcerptT, CacheT, EntityT]) ResolveExcerptMatcher(f func(ExcerptT) bool) (ExcerptT, error) {
	id, err := sc.resolveMatcher(f)
	if err != nil {
		return nil, err
	}
	return sc.ResolveExcerpt(id)
}

func (sc *SubCache[ExcerptT, CacheT, EntityT]) resolveMatcher(f func(ExcerptT) bool) (entity.Id, error) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	// preallocate but empty
	matching := make([]entity.Id, 0, 5)

	for _, excerpt := range sc.excerpts {
		if f(excerpt) {
			matching = append(matching, excerpt.Id())
		}
	}

	if len(matching) > 1 {
		return entity.UnsetId, entity.NewErrMultipleMatch(sc.typename, matching)
	}

	if len(matching) == 0 {
		return entity.UnsetId, entity.NewErrNotFound(sc.typename)
	}

	return matching[0], nil
}

var errNotInCache = errors.New("entity missing from cache")

func (sc *SubCache[ExcerptT, CacheT, EntityT]) add(e EntityT) (CacheT, error) {
	sc.mu.Lock()
	if _, has := sc.cached[e.Id()]; has {
		sc.mu.Unlock()
		return nil, fmt.Errorf("entity %s already exist in the cache", e.Id())
	}

	cached := sc.makeCached(sc, sc.getUserIdentity, e)
	sc.cached[e.Id()] = cached
	sc.lru.Add(e.Id())
	sc.mu.Unlock()

	sc.evictIfNeeded()

	// force the write of the excerpt
	err := sc.entityUpdated(e.Id())
	if err != nil {
		return nil, err
	}

	return cached, nil
}

func (sc *SubCache[ExcerptT, CacheT, EntityT]) Remove(prefix string) error {
	e, err := sc.ResolvePrefix(prefix)
	if err != nil {
		return err
	}

	sc.mu.Lock()

	err = bug.Remove(c.repo, b.Id())
	if err != nil {
		c.muBug.Unlock()

		return err
	}

	delete(c.bugs, b.Id())
	delete(c.bugExcerpts, b.Id())
	c.loadedBugs.Remove(b.Id())

	c.muBug.Unlock()

	return c.writeBugCache()
}

// entityUpdated is a callback to trigger when the excerpt of an entity changed
func (sc *SubCache[ExcerptT, CacheT, EntityT]) entityUpdated(id entity.Id) error {
	sc.mu.Lock()
	b, ok := sc.cached[id]
	if !ok {
		sc.mu.Unlock()

		// if the bug is not loaded at this point, it means it was loaded before
		// but got evicted. Which means we potentially have multiple copies in
		// memory and thus concurrent write.
		// Failing immediately here is the simple and safe solution to avoid
		// complicated data loss.
		return errNotInCache
	}
	sc.lru.Get(id)
	// sc.excerpts[id] = bug2.NewBugExcerpt(b.bug, b.Snapshot())
	sc.excerpts[id] = bug2.NewBugExcerpt(b.bug, b.Snapshot())
	sc.mu.Unlock()

	if err := sc.addBugToSearchIndex(b.Snapshot()); err != nil {
		return err
	}

	// we only need to write the bug cache
	return sc.Write()
}

// evictIfNeeded will evict an entity from the cache if needed
func (sc *SubCache[ExcerptT, CacheT, EntityT]) evictIfNeeded() {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if sc.lru.Len() <= sc.maxLoaded {
		return
	}

	for _, id := range sc.lru.GetOldestToNewest() {
		b := sc.cached[id]
		if b.NeedCommit() {
			continue
		}

		b.Lock()
		sc.lru.Remove(id)
		delete(sc.cached, id)

		if sc.lru.Len() <= sc.maxLoaded {
			return
		}
	}
}
