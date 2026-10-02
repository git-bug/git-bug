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

	// muDerived serializes the changes to the derived state of the entities
	// (excerpts, index documents, builtFrom), see sync.
	// Taken before muMaps.
	muDerived sync.Mutex
	// muMaps protects the in-memory maps below, and is only held briefly.
	// excerpts and builtFrom are only written holding muDerived too, so holding
	// either is enough to read them.
	muMaps   sync.RWMutex
	excerpts map[entity.Id]ExcerptT
	// builtFrom holds, for each entity, the commit its excerpt was built from.
	// Excerpts are only built from the committed state of an entity, never from
	// pending changes, so they are up to date for exactly that commit. The index
	// keeps its own record, see repository.Index.
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

// Load reads the entity cache file, and brings the cache up to date with the
// repository. The caller must read all the returned events.
func (sc *SubCache[EntityT, ExcerptT, CacheT]) Load() <-chan BuildEvent {
	out := make(chan BuildEvent)

	go func() {
		defer close(out)

		// A missing, unreadable or outdated file is an empty store, brought up
		// to date like any other.
		_ = sc.read()

		err := sc.syncAll(func(event BuildEvent) { out <- event })
		if err != nil {
			out <- BuildEvent{Typename: sc.typename, Err: err}
		}
	}()

	return out
}

// read reads the entity cache file. The maps are left empty if it fails.
func (sc *SubCache[EntityT, ExcerptT, CacheT]) read() error {
	sc.muDerived.Lock()
	defer sc.muDerived.Unlock()
	sc.muMaps.Lock()
	defer sc.muMaps.Unlock()

	sc.excerpts = make(map[entity.Id]ExcerptT)
	sc.builtFrom = make(map[entity.Id]repository.Hash)

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
		sc.excerpts[id] = excerpt
	}
	// A commit without its excerpt would have sync see the excerpt as up to date,
	// and never build it: it's dropped, which has sync build the excerpt.
	for id, commit := range aux.BuiltFrom {
		if _, ok := sc.excerpts[id]; ok {
			sc.builtFrom[id] = commit
		}
	}

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
	id := e.Id()
	cached := sc.newCached(e)

	// Under muDerived until the copy is registered, so that a concurrent
	// removal comes either before, and the entity is not added, or after.
	changes, err := func() (map[entity.Id]EntityEventType, error) {
		sc.muDerived.Lock()
		defer sc.muDerived.Unlock()

		sc.muMaps.RLock()
		_, has := sc.cached[id]
		sc.muMaps.RUnlock()
		if has {
			return nil, fmt.Errorf("entity %s already exist in the cache", id)
		}

		changes, err := sc.syncOneLocked(id)
		if err != nil {
			return changes, err
		}
		if _, ok := sc.builtFrom[id]; !ok {
			return changes, fmt.Errorf("%s %s got removed, or the cache closed", sc.typename, id)
		}

		sc.muMaps.Lock()
		sc.cached[id] = cached
		sc.lru.Add(id)
		sc.muMaps.Unlock()
		return changes, nil
	}()
	sc.notifyChanges(changes)
	if err != nil {
		return *new(CacheT), err
	}

	sc.evictIfNeeded()

	return cached, nil
}

func (sc *SubCache[EntityT, ExcerptT, CacheT]) Remove(prefix string) error {
	e, err := sc.ResolvePrefix(prefix)
	if err != nil {
		return err
	}

	err = sc.actions.Remove(sc.repo, e.Id())
	if err != nil {
		return err
	}

	return sc.sync(e.Id())
}

func (sc *SubCache[EntityT, ExcerptT, CacheT]) RemoveAll() error {
	err := sc.actions.RemoveAll(sc.repo)
	if err != nil {
		return err
	}

	return sc.syncAll(nil)
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

		results := sc.actions.MergeAll(sc.repo, sc.resolvers(), remote, author)
		for result := range results {
			out <- result

			if result.Err != nil {
				continue
			}

			switch result.Status {
			case entity.MergeStatusNew, entity.MergeStatusUpdated:
				// If the entity is already loaded, replace it with the merged version,
				// otherwise the loaded copy would be outdated.
				// If it's not loaded, don't load it: a merge is not a use of the entity,
				// and adding it to the LRU could evict entities that are actually in use.
				// The downside is that the entity is read again from git when needed.
				// TODO: can that result in multiple copy of the same entity?
				sc.muMaps.Lock()
				if _, loaded := sc.cached[result.Id]; loaded {
					sc.cached[result.Id] = sc.newCached(result.Entity.(EntityT))
				}
				sc.muMaps.Unlock()
			}
		}

		// one listing of the refs rather than a lookup per merged entity, which
		// also picks up any change made outside since the last sync
		err = sc.syncAll(nil)
		if err != nil {
			out <- entity.NewMergeError(err, "")
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
	return sc.sync(id)
}

// notifyObservers notifies all the observers when something happening for an entity
func (sc *SubCache[EntityT, ExcerptT, CacheT]) notifyObservers(event EntityEventType, id entity.Id) {
	sc.muObservers.RLock()
	for observer, repoName := range sc.observers {
		observer.EntityEvent(event, repoName, sc.typename, id)
	}
	sc.muObservers.RUnlock()
}

func (sc *SubCache[EntityT, ExcerptT, CacheT]) notifyChanges(changes map[entity.Id]EntityEventType) {
	for id, event := range changes {
		sc.notifyObservers(event, id)
	}
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

// sync brings the derived state of an entity up to date with its reference, see
// syncOneLocked, and notifies the observers.
func (sc *SubCache[EntityT, ExcerptT, CacheT]) sync(id entity.Id) error {
	sc.muDerived.Lock()
	changes, err := sc.syncOneLocked(id)
	sc.muDerived.Unlock()
	sc.notifyChanges(changes)
	return err
}

// syncAll brings the derived state of every entity up to date with the
// references, see syncAllLocked, and notifies the observers. progress, if not nil,
// receives the build events.
func (sc *SubCache[EntityT, ExcerptT, CacheT]) syncAll(progress func(BuildEvent)) error {
	sc.muDerived.Lock()
	changes, err := sc.syncAllLocked(progress)
	sc.muDerived.Unlock()
	sc.notifyChanges(changes)
	return err
}

// syncBatchSize is the number of entities applied to the index at once.
//
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
const syncBatchSize = 75

// syncOneLocked brings the derived state of an entity up to date with its
// reference, see syncEntitiesLocked. An unreadable index record can't be
// repaired entry by entry: it has every entity synced instead, see
// syncAllLocked.
func (sc *SubCache[EntityT, ExcerptT, CacheT]) syncOneLocked(id entity.Id) (map[entity.Id]EntityEventType, error) {
	// nil maps mark the SubCache as closed
	if sc.excerpts == nil {
		return nil, nil
	}

	index, err := sc.repo.GetIndex(sc.namespace)
	if err != nil {
		return nil, err
	}
	indexBuiltFrom, err := index.BuiltFrom()
	if err != nil {
		return sc.syncAllLocked(nil)
	}

	refs := make(map[string]repository.Hash, 1)
	ref, err := sc.repo.ResolveRef(sc.namespace, id.String())
	switch {
	case err == nil:
		refs[id.String()] = ref
	case !errors.Is(err, repository.ErrNotFound):
		return nil, err
	}

	candidates := map[entity.Id]struct{}{id: {}}
	return sc.syncEntitiesLocked(index, candidates, refs, indexBuiltFrom, nil)
}

// syncAllLocked brings the derived state of every entity up to date with the
// references, see syncEntitiesLocked: those with a reference, with derived state
// in either store, or with a loaded copy. An unreadable index record is cleared,
// and the index rebuilt.
func (sc *SubCache[EntityT, ExcerptT, CacheT]) syncAllLocked(progress func(BuildEvent)) (map[entity.Id]EntityEventType, error) {
	// nil maps mark the SubCache as closed
	if sc.excerpts == nil {
		return nil, nil
	}

	index, err := sc.repo.GetIndex(sc.namespace)
	if err != nil {
		return nil, err
	}
	indexBuiltFrom, recordErr := index.BuiltFrom()

	refs, err := sc.repo.ListRefs(sc.namespace)
	if err != nil {
		return nil, err
	}
	// cleared once the refs are known, so that failing to list them leaves the
	// index as it is
	if recordErr != nil {
		err = index.Clear()
		if err != nil {
			return nil, err
		}
		indexBuiltFrom = nil
	}

	candidates := make(map[entity.Id]struct{}, len(refs))
	for key := range refs {
		candidates[entity.Id(key)] = struct{}{}
	}
	for id := range sc.builtFrom {
		candidates[id] = struct{}{}
	}
	for key := range indexBuiltFrom {
		candidates[entity.Id(key)] = struct{}{}
	}
	// A loaded copy can exist without derived state, for an entity created
	// outside and resolved before any sync: it is dropped too if its reference
	// is gone.
	sc.muMaps.RLock()
	for id := range sc.cached {
		candidates[id] = struct{}{}
	}
	sc.muMaps.RUnlock()

	return sc.syncEntitiesLocked(index, candidates, refs, indexBuiltFrom, progress)
}

// syncEntitiesLocked brings the derived state of the candidates up to date with
// their references, given in refs, a candidate without one being removed. The
// excerpts and the index each record the commit they were built from, the
// latter given in indexBuiltFrom: an entity that is behind in either, or that
// is loaded while its reference is gone, is read once from git, and applied to
// each store that is behind, or removed from them if its reference is gone,
// along with its loaded copy. Nothing is read if every store is up to date.
// Changes are applied in batches, and the excerpt file is written once, if the
// excerpts changed.
//
// Entities are read under muDerived, which every change to the derived state
// holds, so that each sync applies a state at least as recent as the previous
// one. A reference moving after its entity is read is followed by a sync of
// its own, from the commit callback.
//
// Loaded copies are left as they are, other than removed ones. It returns the
// changes of excerpts that were applied, even on error, for the caller to
// notify once muDerived is released. progress, if not nil, receives the build
// events.
func (sc *SubCache[EntityT, ExcerptT, CacheT]) syncEntitiesLocked(index repository.Index, candidates map[entity.Id]struct{}, refs map[string]repository.Hash, indexBuiltFrom map[string]repository.Hash, progress func(BuildEvent)) (map[entity.Id]EntityEventType, error) {
	if progress == nil {
		progress = func(BuildEvent) {}
	}

	var behind []entity.Id
	sc.muMaps.RLock()
	for id := range candidates {
		ref := refs[id.String()]
		_, isLoaded := sc.cached[id]
		if sc.builtFrom[id] != ref || indexBuiltFrom[id.String()] != ref || (ref == "" && isLoaded) {
			behind = append(behind, id)
		}
	}
	sc.muMaps.RUnlock()
	if len(behind) == 0 {
		return nil, nil
	}

	total := int64(len(behind))
	progress(BuildEvent{
		Typename: sc.typename,
		Event:    BuildEventStarted,
		Total:    total,
	})

	changes := make(map[entity.Id]EntityEventType)

	// excerpt changes of the current batch, a nil derived being a removal
	pending := make(map[entity.Id]*derived[ExcerptT])
	batch := index.NewBatch()
	// entities read for the current batch, and how many of them changed the index
	batchSize, batchIndexed := 0, 0

	// the excerpts of a batch are published once the batch is in the index
	endBatch := func() error {
		if batchIndexed > 0 {
			err := batch.Apply()
			if err != nil {
				return err
			}
			batch = index.NewBatch()
		}
		sc.muMaps.Lock()
		for id, d := range pending {
			_, known := sc.builtFrom[id]
			switch {
			case d == nil:
				delete(sc.excerpts, id)
				delete(sc.builtFrom, id)
				delete(sc.cached, id)
				sc.lru.Remove(id)
				if known {
					changes[id] = EntityEventRemoved
				}
			case known:
				sc.excerpts[id] = d.excerpt
				sc.builtFrom[id] = d.commit
				changes[id] = EntityEventUpdated
			default:
				sc.excerpts[id] = d.excerpt
				sc.builtFrom[id] = d.commit
				changes[id] = EntityEventCreated
			}
		}
		sc.muMaps.Unlock()
		clear(pending)
		batchSize, batchIndexed = 0, 0
		return nil
	}

	for i, id := range behind {
		e, err := sc.actions.ReadWithResolver(sc.repo, sc.resolvers(), id)
		if entity.IsErrNotFound(err) {
			// not found can also come from a missing dependency, such as the
			// author identity: only a missing reference is a removal
			_, refErr := sc.repo.ResolveRef(sc.namespace, id.String())
			if !errors.Is(refErr, repository.ErrNotFound) {
				return changes, err
			}
		}
		switch {
		case entity.IsErrNotFound(err):
			if _, ok := indexBuiltFrom[id.String()]; ok {
				batch.Remove(id.String())
				batchIndexed++
			}
			pending[id] = nil
		case err != nil:
			return changes, err
		default:
			// The entity is only read to build its derived state, which only
			// depends on committed state: it's not kept in memory, whether a copy
			// is already loaded or not.
			d := sc.buildDerived(sc.newCached(e))
			if indexBuiltFrom[id.String()] != d.commit {
				err = batch.Set(id.String(), d.indexData, d.commit)
				if err != nil {
					return changes, err
				}
				batchIndexed++
			}
			if sc.builtFrom[id] != d.commit {
				pending[id] = &d
			}
		}

		batchSize++
		if batchSize >= syncBatchSize {
			err = endBatch()
			if err != nil {
				return changes, err
			}
		}

		progress(BuildEvent{
			Typename: sc.typename,
			Event:    BuildEventProgress,
			Progress: int64(i + 1),
			Total:    total,
		})
	}

	if batchSize > 0 {
		err := endBatch()
		if err != nil {
			return changes, err
		}
	}

	if len(changes) > 0 {
		err := sc.write()
		if err != nil {
			return changes, err
		}
	}

	progress(BuildEvent{Typename: sc.typename, Event: BuildEventFinished})

	return changes, nil
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
