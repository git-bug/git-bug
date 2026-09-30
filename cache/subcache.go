package cache

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/pkg/errors"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/repository"
)

type Excerpt interface {
	Id() entity.Id
	setId(id entity.Id)
}

type CacheEntity interface {
	Id() entity.Id
	// LastCommit returns the commit holding the committed state of the entity.
	LastCommit() repository.Hash
	NeedCommit() bool
}

type getUserIdentityFunc func() (*IdentityCache, error)

// Actions expose a number of action functions on Entities, to give upper layers (cache) a way to normalize interactions.
// Note: ideally this wouldn't exist, the cache layer would assume that everything is an entity/dag, and directly use the
// functions from this package, but right now identities are not using that framework.
type Actions[EntityT entity.Interface] struct {
	ReadWithResolver    func(repo repository.ClockedRepo, resolvers entity.Resolvers, id entity.Id) (EntityT, error)
	ReadAllWithResolver func(repo repository.ClockedRepo, resolvers entity.Resolvers) <-chan entity.StreamedEntity[EntityT]
	Remove              func(repo repository.ClockedRepo, id entity.Id) error
	RemoveAll           func(repo repository.ClockedRepo) error
	MergeAll            func(repo repository.ClockedRepo, resolvers entity.Resolvers, remote string, mergeAuthor identity.Interface) <-chan entity.MergeResult
	EnsureClocks        func(repo repository.ClockedRepo) error
}

var _ cacheMgmt = &SubCache[entity.Interface, Excerpt, CacheEntity]{}

type SubCache[EntityT entity.Interface, ExcerptT Excerpt, CacheT CacheEntity] struct {
	repo      repository.ClockedRepo
	resolvers func() entity.Resolvers

	getUserIdentity getUserIdentityFunc
	makeCached      func(entity EntityT, onCommit func() error) CacheT
	makeExcerpt     func(CacheT) ExcerptT
	makeIndexData   func(CacheT) []string
	actions         Actions[EntityT]

	typename  string
	namespace string
	version   uint
	maxLoaded int

	// muDerived serializes building and publishing the derived state of the
	// entities (excerpt, index document, builtFrom), so that it is applied in
	// commit order.
	// Taken before muMaps.
	muDerived sync.Mutex

	// muMaps protects the in-memory maps below, and is only held briefly.
	muMaps   sync.RWMutex
	excerpts map[entity.Id]ExcerptT
	// builtFrom holds, for each entity, the commit its excerpt and index document
	// were built from. They are only built from the committed state of an entity,
	// never from pending changes, so they are up to date for exactly that commit.
	builtFrom map[entity.Id]repository.Hash
	cached    map[entity.Id]CacheT
	lru       lruIdCache

	muObservers sync.RWMutex
	observers   map[Observer]string // observer --> repo name
}

func NewSubCache[EntityT entity.Interface, ExcerptT Excerpt, CacheT CacheEntity](
	repo repository.ClockedRepo,
	resolvers func() entity.Resolvers, getUserIdentity getUserIdentityFunc,
	makeCached func(entity EntityT, onCommit func() error) CacheT,
	makeExcerpt func(CacheT) ExcerptT,
	makeIndexData func(CacheT) []string,
	actions Actions[EntityT],
	typename, namespace string,
	version uint, maxLoaded int) *SubCache[EntityT, ExcerptT, CacheT] {
	return &SubCache[EntityT, ExcerptT, CacheT]{
		repo:            repo,
		resolvers:       resolvers,
		getUserIdentity: getUserIdentity,
		makeCached:      makeCached,
		makeExcerpt:     makeExcerpt,
		makeIndexData:   makeIndexData,
		actions:         actions,
		typename:        typename,
		namespace:       namespace,
		version:         version,
		maxLoaded:       maxLoaded,
		excerpts:        make(map[entity.Id]ExcerptT),
		builtFrom:       make(map[entity.Id]repository.Hash),
		cached:          make(map[entity.Id]CacheT),
		lru:             newLRUIdCache(),
	}
}

func (sc *SubCache[EntityT, ExcerptT, CacheT]) Typename() string {
	return sc.typename
}

// EnsureClocks makes sure that the clocks of the entities are usable, rebuilding
// them if not. See dag.EnsureClocks.
func (sc *SubCache[EntityT, ExcerptT, CacheT]) EnsureClocks() error {
	return sc.actions.EnsureClocks(sc.repo)
}

// Load will try to read from the disk the entity cache file
func (sc *SubCache[EntityT, ExcerptT, CacheT]) Load() error {
	sc.muMaps.Lock()
	defer sc.muMaps.Unlock()

	var f billy.File
	err := retryOnWindows(func() (err error) {
		f, err = sc.repo.LocalStorage().Open(filepath.Join("cache", sc.namespace))
		return err
	})
	if err != nil {
		return err
	}

	aux := struct {
		Version   uint
		Excerpts  map[entity.Id]ExcerptT
		BuiltFrom map[entity.Id]repository.Hash
	}{}

	decoder := gob.NewDecoder(f)
	err = decoder.Decode(&aux)
	if err != nil {
		_ = f.Close()
		return err
	}

	err = f.Close()
	if err != nil {
		return err
	}

	if aux.Version != sc.version {
		return fmt.Errorf("unknown %s cache format version %v", sc.namespace, aux.Version)
	}

	// the id is not serialized in the excerpt itself (non-exported field in go, long story ...),
	// so we fix it here, which doubles as enforcing coherency.
	for id, excerpt := range aux.Excerpts {
		excerpt.setId(id)
	}

	sc.excerpts = aux.Excerpts
	sc.builtFrom = aux.BuiltFrom

	index, err := sc.repo.GetIndex(sc.namespace)
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

	// TODO: find a way to check lamport clocks

	return nil
}

// Write will serialize on disk the entity cache file
func (sc *SubCache[EntityT, ExcerptT, CacheT]) write() error {
	sc.muMaps.RLock()
	defer sc.muMaps.RUnlock()

	var data bytes.Buffer

	aux := struct {
		Version   uint
		Excerpts  map[entity.Id]ExcerptT
		BuiltFrom map[entity.Id]repository.Hash
	}{
		Version:   sc.version,
		Excerpts:  sc.excerpts,
		BuiltFrom: sc.builtFrom,
	}

	encoder := gob.NewEncoder(&data)

	err := encoder.Encode(aux)
	if err != nil {
		return err
	}

	// Written aside then renamed over the cache file, so that a concurrent reader
	// sees either the previous version or this one, never a partial write.
	storage := sc.repo.LocalStorage()

	f, err := storage.TempFile("cache", sc.namespace+".tmp-")
	if err != nil {
		return err
	}

	_, err = f.Write(data.Bytes())
	if err != nil {
		_ = f.Close()
		_ = storage.Remove(f.Name())
		return err
	}

	err = f.Close()
	if err != nil {
		_ = storage.Remove(f.Name())
		return err
	}

	err = retryOnWindows(func() error {
		return storage.Rename(f.Name(), filepath.Join("cache", sc.namespace))
	})
	if err != nil {
		_ = storage.Remove(f.Name())
		return err
	}

	return nil
}

func (sc *SubCache[EntityT, ExcerptT, CacheT]) Build() <-chan BuildEvent {
	// value chosen experimentally as giving the fasted indexing, while
	// not driving the cache size on disk too high.
	//
	// | batchCount | bugIndex (MB) | idIndex (kB) | time (s) |
	// |:----------:|:-------------:|:------------:|:--------:|
	// |     10     |      24       |      84      |   1,59   |
	// |     30     |      26       |      84      |  1,388   |
	// |     50     |      26       |      84      |   1,44   |
	// |     60     |      26       |      80      |  1,377   |
	// |     68     |      27       |      80      |  1,385   |
	// |     75     |      26       |      84      |   1,32   |
	// |     80     |      26       |      80      |   1,37   |
	// |     85     |      27       |      80      |  1,317   |
	// |    100     |      26       |      80      |  1,455   |
	// |    150     |      26       |      80      |  2,066   |
	// |    200     |      28       |      80      |  2,885   |
	// |    250     |      30       |      72      |  3,555   |
	// |    300     |      31       |      72      |  4,787   |
	// |    500     |      23       |      72      |   5,4    |
	const maxBatchCount = 75

	out := make(chan BuildEvent)

	go func() {
		defer close(out)

		out <- BuildEvent{
			Typename: sc.typename,
			Event:    BuildEventStarted,
		}

		sc.muDerived.Lock()
		defer sc.muDerived.Unlock()

		sc.muMaps.Lock()
		sc.excerpts = make(map[entity.Id]ExcerptT)
		sc.builtFrom = make(map[entity.Id]repository.Hash)
		sc.muMaps.Unlock()

		allEntities := sc.actions.ReadAllWithResolver(sc.repo, sc.resolvers())

		index, err := sc.repo.GetIndex(sc.namespace)
		if err != nil {
			out <- BuildEvent{
				Typename: sc.typename,
				Err:      err,
			}
			return
		}

		// wipe the index just to be sure
		err = index.Clear()
		if err != nil {
			out <- BuildEvent{
				Typename: sc.typename,
				Err:      err,
			}
			return
		}

		indexer, indexEnd := index.IndexBatch()
		var batch []derived[ExcerptT]

		// the derived state of a batch is published once the batch is in the
		// index, so that the index is never behind it
		endBatch := func() error {
			if err := indexEnd(); err != nil {
				return err
			}
			sc.muMaps.Lock()
			for _, d := range batch {
				sc.excerpts[d.excerpt.Id()] = d.excerpt
				sc.builtFrom[d.excerpt.Id()] = d.commit
			}
			sc.muMaps.Unlock()
			batch = batch[:0]
			return nil
		}

		for e := range allEntities {
			if e.Err != nil {
				out <- BuildEvent{
					Typename: sc.typename,
					Err:      e.Err,
				}
				return
			}

			// The entity is only read to build its derived state, which only depends
			// on committed state: it's not kept in memory, whether a copy is already
			// loaded or not.
			d := sc.buildDerived(sc.newCached(e.Entity))

			if err := indexer(e.Entity.Id().String(), d.indexData); err != nil {
				out <- BuildEvent{
					Typename: sc.typename,
					Err:      err,
				}
				return
			}
			batch = append(batch, d)

			if len(batch) >= maxBatchCount {
				err = endBatch()
				if err != nil {
					out <- BuildEvent{
						Typename: sc.typename,
						Err:      err,
					}
					return
				}

				indexer, indexEnd = index.IndexBatch()
			}

			out <- BuildEvent{
				Typename: sc.typename,
				Event:    BuildEventProgress,
				Progress: e.CurrentEntity,
				Total:    e.TotalEntities,
			}
		}

		if len(batch) > 0 {
			err = endBatch()
			if err != nil {
				out <- BuildEvent{
					Typename: sc.typename,
					Err:      err,
				}
				return
			}
		}

		err = sc.write()
		if err != nil {
			out <- BuildEvent{
				Typename: sc.typename,
				Err:      err,
			}
			return
		}

		out <- BuildEvent{
			Typename: sc.typename,
			Event:    BuildEventFinished,
		}
	}()

	return out
}

func (sc *SubCache[EntityT, ExcerptT, CacheT]) SetCacheSize(size int) {
	sc.maxLoaded = size
	sc.evictIfNeeded()
}

func (sc *SubCache[EntityT, ExcerptT, CacheT]) Close() error {
	sc.muDerived.Lock()
	defer sc.muDerived.Unlock()
	sc.muMaps.Lock()
	defer sc.muMaps.Unlock()
	// nil maps mark the SubCache as closed, see updateDerived
	sc.excerpts = nil
	sc.builtFrom = nil
	sc.cached = make(map[entity.Id]CacheT)
	return nil
}

func (sc *SubCache[EntityT, ExcerptT, CacheT]) RegisterObserver(repoName string, observer Observer) {
	sc.muObservers.Lock()
	defer sc.muObservers.Unlock()
	if sc.observers == nil {
		sc.observers = make(map[Observer]string)
	}
	sc.observers[observer] = repoName
}

func (sc *SubCache[EntityT, ExcerptT, CacheT]) UnregisterObserver(observer Observer) {
	sc.muObservers.Lock()
	defer sc.muObservers.Unlock()
	delete(sc.observers, observer)
}

// AllIds return all known bug ids
func (sc *SubCache[EntityT, ExcerptT, CacheT]) AllIds() []entity.Id {
	sc.muMaps.RLock()
	defer sc.muMaps.RUnlock()

	result := make([]entity.Id, len(sc.excerpts))

	i := 0
	for _, excerpt := range sc.excerpts {
		result[i] = excerpt.Id()
		i++
	}

	return result
}

// Resolve retrieve an entity matching the exact given id
func (sc *SubCache[EntityT, ExcerptT, CacheT]) Resolve(id entity.Id) (CacheT, error) {
	sc.muMaps.RLock()
	cached, ok := sc.cached[id]
	if ok {
		// mark as recently used
		sc.lru.Get(id)
		sc.muMaps.RUnlock()
		return cached, nil
	}
	sc.muMaps.RUnlock()

	e, err := sc.actions.ReadWithResolver(sc.repo, sc.resolvers(), id)
	if err != nil {
		return *new(CacheT), err
	}

	cached = sc.newCached(e)

	sc.muMaps.Lock()
	if existing, ok := sc.cached[id]; ok {
		// loaded concurrently in the meantime, keep a single copy in memory
		// and mark it as recently used
		sc.lru.Get(id)
		sc.muMaps.Unlock()
		return existing, nil
	}
	sc.cached[id] = cached
	sc.lru.Add(id)
	sc.muMaps.Unlock()

	sc.evictIfNeeded()

	return cached, nil
}

// ResolvePrefix retrieve an entity matching an id prefix. It fails if multiple
// entities match.
func (sc *SubCache[EntityT, ExcerptT, CacheT]) ResolvePrefix(prefix string) (CacheT, error) {
	return sc.ResolveMatcher(func(excerpt ExcerptT) bool {
		return excerpt.Id().HasPrefix(prefix)
	})
}

func (sc *SubCache[EntityT, ExcerptT, CacheT]) ResolveMatcher(f func(ExcerptT) bool) (CacheT, error) {
	id, err := sc.resolveMatcher(f)
	if err != nil {
		return *new(CacheT), err
	}
	return sc.Resolve(id)
}

// ResolveExcerpt retrieves an Excerpt matching the exact given id
func (sc *SubCache[EntityT, ExcerptT, CacheT]) ResolveExcerpt(id entity.Id) (ExcerptT, error) {
	sc.muMaps.RLock()
	defer sc.muMaps.RUnlock()

	excerpt, ok := sc.excerpts[id]
	if !ok {
		return *new(ExcerptT), entity.NewErrNotFound(sc.typename)
	}

	return excerpt, nil
}

// ResolveExcerptPrefix retrieves an Excerpt matching an id prefix. It fails if multiple
// entities match.
func (sc *SubCache[EntityT, ExcerptT, CacheT]) ResolveExcerptPrefix(prefix string) (ExcerptT, error) {
	return sc.ResolveExcerptMatcher(func(excerpt ExcerptT) bool {
		return excerpt.Id().HasPrefix(prefix)
	})
}

// ResolveExcerptMatcher retrieves an Excerpt selected by the given matcher function.
func (sc *SubCache[EntityT, ExcerptT, CacheT]) ResolveExcerptMatcher(f func(ExcerptT) bool) (ExcerptT, error) {
	id, err := sc.resolveMatcher(f)
	if err != nil {
		return *new(ExcerptT), err
	}
	return sc.ResolveExcerpt(id)
}

func (sc *SubCache[EntityT, ExcerptT, CacheT]) resolveMatcher(f func(ExcerptT) bool) (entity.Id, error) {
	sc.muMaps.RLock()
	defer sc.muMaps.RUnlock()

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

func (sc *SubCache[EntityT, ExcerptT, CacheT]) add(e EntityT) (CacheT, error) {
	cached := sc.newCached(e)

	// Under muDerived until the copy is registered, so that a concurrent
	// removal comes either before, and the entity is not added, or after.
	err := func() error {
		sc.muDerived.Lock()
		defer sc.muDerived.Unlock()

		sc.muMaps.RLock()
		_, has := sc.cached[e.Id()]
		sc.muMaps.RUnlock()
		if has {
			return fmt.Errorf("entity %s already exist in the cache", e.Id())
		}

		// derive before the copy is shared, while it can't have pending changes
		event, err := sc.publishDerivedLocked(cached)
		if err != nil {
			return err
		}
		if event == 0 {
			return fmt.Errorf("%s %s got removed, or the cache closed", sc.typename, e.Id())
		}

		sc.muMaps.Lock()
		sc.cached[e.Id()] = cached
		sc.lru.Add(e.Id())
		sc.muMaps.Unlock()
		return nil
	}()
	if err != nil {
		return *new(CacheT), err
	}

	err = sc.write()
	if err != nil {
		return *new(CacheT), err
	}

	sc.evictIfNeeded()

	sc.notifyObservers(EntityEventCreated, e.Id())

	return cached, nil
}

func (sc *SubCache[EntityT, ExcerptT, CacheT]) Remove(prefix string) error {
	e, err := sc.ResolvePrefix(prefix)
	if err != nil {
		return err
	}

	// under muDerived, so that no derived state is published for the entity
	// once removed
	err = func() error {
		sc.muDerived.Lock()
		defer sc.muDerived.Unlock()
		sc.muMaps.Lock()
		defer sc.muMaps.Unlock()

		err := sc.actions.Remove(sc.repo, e.Id())
		if err != nil {
			return err
		}

		delete(sc.cached, e.Id())
		delete(sc.excerpts, e.Id())
		delete(sc.builtFrom, e.Id())
		sc.lru.Remove(e.Id())

		index, err := sc.repo.GetIndex(sc.namespace)
		if err != nil {
			return err
		}
		return index.Remove(e.Id().String())
	}()
	if err != nil {
		return err
	}

	// defer to notify after the write
	defer sc.notifyObservers(EntityEventRemoved, e.Id())

	return sc.write()
}

func (sc *SubCache[EntityT, ExcerptT, CacheT]) RemoveAll() error {
	ids := make(map[entity.Id]struct{})

	// under muDerived, so that no derived state is published for the entities
	// once removed
	err := func() error {
		sc.muDerived.Lock()
		defer sc.muDerived.Unlock()
		sc.muMaps.Lock()
		defer sc.muMaps.Unlock()

		err := sc.actions.RemoveAll(sc.repo)
		if err != nil {
			return err
		}

		for id, _ := range sc.cached {
			delete(sc.cached, id)
			sc.lru.Remove(id)
			ids[id] = struct{}{}
		}
		for id, _ := range sc.excerpts {
			delete(sc.excerpts, id)
			ids[id] = struct{}{}
		}
		clear(sc.builtFrom)

		index, err := sc.repo.GetIndex(sc.namespace)
		if err != nil {
			return err
		}
		return index.Clear()
	}()
	if err != nil {
		return err
	}

	// defer to notify after the write
	defer func() {
		for id := range ids {
			sc.notifyObservers(EntityEventRemoved, id)
		}
	}()

	return sc.write()
}

func (sc *SubCache[EntityT, ExcerptT, CacheT]) MergeAll(remote string) <-chan entity.MergeResult {
	out := make(chan entity.MergeResult)

	// Intercept merge results to update the cache properly
	go func() {
		defer close(out)

		// the author is only needed for merge commits, so a user identity is optional
		user, err := sc.getUserIdentity()
		if err != nil && !errors.Is(err, identity.ErrNoIdentitySet) {
			out <- entity.NewMergeError(err, "")
			return
		}
		var author identity.Interface
		if err == nil {
			author = user
		}

		// merge a single entity in the cache and its derived state. The excerpt file is
		// not written here, it's written once for all the merged entities.
		updateCache := func(result entity.MergeResult) error {
			e := result.Entity.(EntityT)
			cached := sc.newCached(e)

			// derive before the copy is shared, while it can't have pending
			// changes, and before notifying, so that an observer can already
			// search it
			_, err := sc.publishDerived(cached)
			if err != nil {
				return err
			}

			sc.muMaps.Lock()
			// If the entity is already loaded, replace it with the merged version,
			// otherwise the loaded copy would be outdated.
			// If it's not loaded, don't load it: a merge is not a use of the entity,
			// and adding it to the LRU could evict entities that are actually in use.
			// The downside is that the entity is read again from git when needed.
			if _, loaded := sc.cached[result.Id]; loaded {
				sc.cached[result.Id] = cached
			}
			sc.muMaps.Unlock()

			return nil
		}

		results := sc.actions.MergeAll(sc.repo, sc.resolvers(), remote, author)
		for result := range results {
			out <- result

			if result.Err != nil {
				continue
			}

			switch result.Status {
			case entity.MergeStatusNew:
				if err := updateCache(result); err != nil {
					out <- entity.NewMergeError(err, result.Id)
					continue
				}
				sc.notifyObservers(EntityEventCreated, result.Id)

			case entity.MergeStatusUpdated:
				// TODO: can that result in multiple copy of the same entity?
				if err := updateCache(result); err != nil {
					out <- entity.NewMergeError(err, result.Id)
					continue
				}
				sc.notifyObservers(EntityEventUpdated, result.Id)
			}
		}

		err = sc.write()
		if err != nil {
			out <- entity.NewMergeError(err, "")
			return
		}
	}()

	return out

}

// GetNamespace expose the namespace in git where entities are located.
func (sc *SubCache[EntityT, ExcerptT, CacheT]) GetNamespace() string {
	return sc.namespace
}

// newCached wraps an entity for the cache, and has its commits refresh its
// derived state.
func (sc *SubCache[EntityT, ExcerptT, CacheT]) newCached(e EntityT) CacheT {
	// read the id now, while the entity is not shared
	id := e.Id()
	return sc.makeCached(e, func() error { return sc.onCommit(id) })
}

// onCommit refreshes the derived state after an entity has been committed.
func (sc *SubCache[EntityT, ExcerptT, CacheT]) onCommit(id entity.Id) error {
	event, err := sc.refresh(id)
	if err != nil || event == 0 {
		return err
	}

	// defer to notify after the write
	defer sc.notifyObservers(event, id)

	return sc.write()
}

// notifyObservers notifies all the observers when something happening for an entity
func (sc *SubCache[EntityT, ExcerptT, CacheT]) notifyObservers(event EntityEventType, id entity.Id) {
	sc.muObservers.RLock()
	for observer, repoName := range sc.observers {
		observer.EntityEvent(event, repoName, sc.typename, id)
	}
	sc.muObservers.RUnlock()
}

// derived is the state the cache keeps about an entity, built from its committed
// state: its excerpt, its index document, and the commit they were built from.
type derived[ExcerptT Excerpt] struct {
	commit    repository.Hash
	excerpt   ExcerptT
	indexData []string
}

// buildDerived derives the state the cache keeps about an entity. The entity
// must not be shared, and have no pending changes.
func (sc *SubCache[EntityT, ExcerptT, CacheT]) buildDerived(fresh CacheT) derived[ExcerptT] {
	return derived[ExcerptT]{
		commit:    fresh.LastCommit(),
		excerpt:   sc.makeExcerpt(fresh),
		indexData: sc.makeIndexData(fresh),
	}
}

// refresh brings the derived state of an entity up to date with its reference:
// derived from a fresh read of the entity if the reference moved, dropped if the
// reference is gone. It reports the change as an event, or 0 if there was none.
// The excerpt file is not written.
func (sc *SubCache[EntityT, ExcerptT, CacheT]) refresh(id entity.Id) (EntityEventType, error) {
	e, err := sc.actions.ReadWithResolver(sc.repo, sc.resolvers(), id)
	if entity.IsErrNotFound(err) {
		return sc.dropDerived(id)
	}
	if err != nil {
		return 0, err
	}
	return sc.publishDerived(sc.newCached(e))
}

// publishDerived publishes the derived state of an entity, unless it's already
// up to date. The entity must not be shared, and have no pending changes. It
// reports the change as an event, or 0 if there was none. The excerpt file is
// not written.
func (sc *SubCache[EntityT, ExcerptT, CacheT]) publishDerived(fresh CacheT) (EntityEventType, error) {
	sc.muDerived.Lock()
	defer sc.muDerived.Unlock()
	return sc.publishDerivedLocked(fresh)
}

// publishDerivedLocked is publishDerived, for a caller already holding muDerived.
func (sc *SubCache[EntityT, ExcerptT, CacheT]) publishDerivedLocked(fresh CacheT) (EntityEventType, error) {
	id := fresh.Id()

	sc.muMaps.RLock()
	closed := sc.excerpts == nil
	builtFrom, known := sc.builtFrom[id]
	sc.muMaps.RUnlock()
	if closed || fresh.LastCommit() == builtFrom {
		return 0, nil
	}

	// Only publish if the reference still points to that commit: the entity
	// can have been read before a newer commit, or before its removal.
	ref, err := sc.repo.ResolveRef(sc.namespace, id.String())
	if errors.Is(err, repository.ErrNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if ref != fresh.LastCommit() {
		return 0, nil
	}

	d := sc.buildDerived(fresh)

	index, err := sc.repo.GetIndex(sc.namespace)
	if err != nil {
		return 0, err
	}

	// index first, so that the index is never behind what builtFrom records.
	// Readers of the index must tolerate it being ahead of the excerpts.
	err = index.IndexOne(id.String(), d.indexData)
	if err != nil {
		return 0, err
	}

	sc.muMaps.Lock()
	sc.excerpts[id] = d.excerpt
	sc.builtFrom[id] = d.commit
	sc.muMaps.Unlock()

	if known {
		return EntityEventUpdated, nil
	}
	return EntityEventCreated, nil
}

// dropDerived drops the derived state of an entity whose reference is gone,
// along with its loaded copy. It reports the change as an event, or 0 if there
// was none. The excerpt file is not written.
func (sc *SubCache[EntityT, ExcerptT, CacheT]) dropDerived(id entity.Id) (EntityEventType, error) {
	sc.muDerived.Lock()
	defer sc.muDerived.Unlock()

	sc.muMaps.RLock()
	_, known := sc.builtFrom[id]
	sc.muMaps.RUnlock()
	if !known {
		return 0, nil
	}

	// the reference could have come back since it was found missing
	_, err := sc.repo.ResolveRef(sc.namespace, id.String())
	if err == nil {
		return 0, nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return 0, err
	}

	index, err := sc.repo.GetIndex(sc.namespace)
	if err != nil {
		return 0, err
	}
	err = index.Remove(id.String())
	if err != nil {
		return 0, err
	}

	sc.muMaps.Lock()
	delete(sc.excerpts, id)
	delete(sc.builtFrom, id)
	delete(sc.cached, id)
	sc.lru.Remove(id)
	sc.muMaps.Unlock()

	return EntityEventRemoved, nil
}

// evictIfNeeded will evict an entity from the cache if needed
func (sc *SubCache[EntityT, ExcerptT, CacheT]) evictIfNeeded() {
	sc.muMaps.Lock()
	defer sc.muMaps.Unlock()
	if sc.lru.Len() <= sc.maxLoaded {
		return
	}

	for _, id := range sc.lru.GetOldestToNewest() {
		b := sc.cached[id]
		if b.NeedCommit() {
			continue
		}

		sc.lru.Remove(id)
		delete(sc.cached, id)

		if sc.lru.Len() <= sc.maxLoaded {
			return
		}
	}
}

// On Windows, a file can't be replaced while open, nor opened while being
// replaced. Both are short-lived, so retry for a bit.
func retryOnWindows(fn func() error) error {
	for i := 0; ; i++ {
		err := fn()
		if err == nil || runtime.GOOS != "windows" || os.IsNotExist(err) || i == 100 {
			return err
		}
		time.Sleep(20 * time.Millisecond)
	}
}
