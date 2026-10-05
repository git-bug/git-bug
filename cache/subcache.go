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

// Shared is the single loaded copy of an entity, holding its committed state and
// nothing else: what a caller stages lives in its view, see SubCache.Resolve.
type Shared interface {
	entity.Resolved
	// LastCommit returns the commit holding the committed state of the entity.
	LastCommit() repository.Hash
}

type getUserIdentityFunc func() (*IdentityCache, error)

// Actions expose a number of action functions on Entities, to give upper layers (cache) a way to normalize interactions.
// Note: ideally this wouldn't exist, the cache layer would assume that everything is an entity/dag, and directly use the
// functions from this package, but right now identities are not using that framework.
type Actions[SharedT Shared] struct {
	// ReadWithResolver reads an entity from the repository
	ReadWithResolver func(repo repository.ClockedRepo, resolvers entity.Resolvers, id entity.Id) (SharedT, error)
	// Refresh brings a loaded entity up to date with its reference, either in place
	// or by returning a replacement, which the cache then holds instead
	Refresh      func(repo repository.ClockedRepo, resolvers entity.Resolvers, shared SharedT) (SharedT, error)
	Remove       func(repo repository.ClockedRepo, id entity.Id) error
	RemoveAll    func(repo repository.ClockedRepo) error
	MergeAll     func(repo repository.ClockedRepo, resolvers entity.Resolvers, remote string, mergeAuthor identity.Interface) <-chan entity.MergeResult
	EnsureClocks func(repo repository.ClockedRepo) error
}

var _ cacheMgmt = &SubCache[Shared, Excerpt, entity.Resolved]{}

// SubCache holds the entities of one type: their derived state, and a single
// loaded copy of each entity in use, see Shared. Callers get views of that copy,
// see Resolve.
type SubCache[SharedT Shared, ExcerptT Excerpt, ViewT entity.Resolved] struct {
	repo      repository.ClockedRepo
	resolvers func() entity.Resolvers

	getUserIdentity getUserIdentityFunc
	makeView        func(shared SharedT, onCommit func() error) ViewT
	makeExcerpt     func(SharedT) ExcerptT
	makeIndexData   func(SharedT) []string
	actions         Actions[SharedT]

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
	// cached holds the single loaded copy of each entity in use, with its
	// committed state only, see Shared. Any of them can be evicted: a view keeps
	// working with the copy it holds.
	cached map[entity.Id]SharedT
	lru    lruIdCache

	muObservers sync.RWMutex
	observers   map[Observer]string // observer --> repo name

	// syncTrigger asks the worker for a sync, see requestSync and startSyncWorker.
	syncTrigger chan struct{}
	// stopSyncWorker is closed by Close, to stop the worker.
	stopSyncWorker chan struct{}
	syncWorkerDone sync.WaitGroup
	// muSyncWorker guards the state of the worker below. It is only held briefly.
	muSyncWorker     sync.Mutex
	syncWorkerClosed bool
	// syncWorkerErr is the error of the latest sync of the worker, nil if it
	// succeeded.
	syncWorkerErr error
}

func NewSubCache[SharedT Shared, ExcerptT Excerpt, ViewT entity.Resolved](
	repo repository.ClockedRepo,
	resolvers func() entity.Resolvers, getUserIdentity getUserIdentityFunc,
	makeView func(shared SharedT, onCommit func() error) ViewT,
	makeExcerpt func(SharedT) ExcerptT,
	makeIndexData func(SharedT) []string,
	actions Actions[SharedT],
	typename, namespace string,
	version uint, maxLoaded int) *SubCache[SharedT, ExcerptT, ViewT] {
	return &SubCache[SharedT, ExcerptT, ViewT]{
		repo:            repo,
		resolvers:       resolvers,
		getUserIdentity: getUserIdentity,
		makeView:        makeView,
		makeExcerpt:     makeExcerpt,
		makeIndexData:   makeIndexData,
		actions:         actions,
		typename:        typename,
		namespace:       namespace,
		version:         version,
		maxLoaded:       maxLoaded,
		excerpts:        make(map[entity.Id]ExcerptT),
		builtFrom:       make(map[entity.Id]repository.Hash),
		cached:          make(map[entity.Id]SharedT),
		syncTrigger:     make(chan struct{}, 1),
		stopSyncWorker:  make(chan struct{}),
		lru:             newLRUIdCache(),
	}
}

func (sc *SubCache[SharedT, ExcerptT, ViewT]) Typename() string {
	return sc.typename
}

// EnsureClocks makes sure that the clocks of the entities are usable, rebuilding
// them if not. See dag.EnsureClocks.
func (sc *SubCache[SharedT, ExcerptT, ViewT]) EnsureClocks() error {
	return sc.actions.EnsureClocks(sc.repo)
}

// Load reads the entity cache file, and brings the cache up to date with the
// repository. The caller must read all the returned events.
func (sc *SubCache[SharedT, ExcerptT, ViewT]) Load() <-chan BuildEvent {
	out := make(chan BuildEvent)

	go func() {
		defer close(out)

		// A missing, unreadable or outdated file is an empty store, brought up
		// to date like any other.
		_ = sc.read()

		err := sc.syncAll(func(event BuildEvent) { out <- event })
		if err != nil {
			out <- BuildEvent{Typename: sc.typename, Err: err}
			return
		}
		sc.startSyncWorker()
	}()

	return out
}

// read reads the entity cache file. The maps are left empty if it fails.
func (sc *SubCache[SharedT, ExcerptT, ViewT]) read() error {
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
func (sc *SubCache[SharedT, ExcerptT, ViewT]) write() error {
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

func (sc *SubCache[SharedT, ExcerptT, ViewT]) SetCacheSize(size int) {
	sc.maxLoaded = size
	sc.evictIfNeeded()
}

func (sc *SubCache[SharedT, ExcerptT, ViewT]) Close() error {
	sc.closeSyncWorker()

	sc.muDerived.Lock()
	defer sc.muDerived.Unlock()
	sc.muMaps.Lock()
	defer sc.muMaps.Unlock()
	// nil maps mark the SubCache as closed, see updateDerived
	sc.excerpts = nil
	sc.builtFrom = nil
	sc.cached = make(map[entity.Id]SharedT)
	return nil
}

func (sc *SubCache[SharedT, ExcerptT, ViewT]) RegisterObserver(repoName string, observer Observer) {
	sc.muObservers.Lock()
	defer sc.muObservers.Unlock()
	if sc.observers == nil {
		sc.observers = make(map[Observer]string)
	}
	sc.observers[observer] = repoName
}

func (sc *SubCache[SharedT, ExcerptT, ViewT]) UnregisterObserver(observer Observer) {
	sc.muObservers.Lock()
	defer sc.muObservers.Unlock()
	delete(sc.observers, observer)
}

// AllIds return all known bug ids
func (sc *SubCache[SharedT, ExcerptT, ViewT]) AllIds() []entity.Id {
	sc.requestSync()
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

// Resolve retrieve an entity matching the exact given id. Each call returns a new
// view of the single loaded copy of the entity: what a caller stages on it is its
// own until committed, while the committed state is shared.
func (sc *SubCache[SharedT, ExcerptT, ViewT]) Resolve(id entity.Id) (ViewT, error) {
	sc.requestSync()
	sc.muMaps.RLock()
	shared, ok := sc.cached[id]
	if ok {
		// mark as recently used
		sc.lru.Get(id)
		sc.muMaps.RUnlock()
		return sc.newView(shared), nil
	}
	sc.muMaps.RUnlock()

	shared, err := sc.actions.ReadWithResolver(sc.repo, sc.resolvers(), id)
	if err != nil {
		return *new(ViewT), err
	}

	sc.muMaps.Lock()
	if existing, ok := sc.cached[id]; ok {
		// loaded concurrently in the meantime, keep a single copy in memory
		// and mark it as recently used
		sc.lru.Get(id)
		sc.muMaps.Unlock()
		return sc.newView(existing), nil
	}
	sc.cached[id] = shared
	sc.lru.Add(id)
	sc.muMaps.Unlock()

	sc.evictIfNeeded()

	return sc.newView(shared), nil
}

// ResolvePrefix retrieve an entity matching an id prefix. It fails if multiple
// entities match.
func (sc *SubCache[SharedT, ExcerptT, ViewT]) ResolvePrefix(prefix string) (ViewT, error) {
	return sc.ResolveMatcher(func(excerpt ExcerptT) bool {
		return excerpt.Id().HasPrefix(prefix)
	})
}

func (sc *SubCache[SharedT, ExcerptT, ViewT]) ResolveMatcher(f func(ExcerptT) bool) (ViewT, error) {
	id, err := sc.resolveMatcher(f)
	if err != nil {
		return *new(ViewT), err
	}
	return sc.Resolve(id)
}

// ResolveExcerpt retrieves an Excerpt matching the exact given id
func (sc *SubCache[SharedT, ExcerptT, ViewT]) ResolveExcerpt(id entity.Id) (ExcerptT, error) {
	sc.requestSync()
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
func (sc *SubCache[SharedT, ExcerptT, ViewT]) ResolveExcerptPrefix(prefix string) (ExcerptT, error) {
	return sc.ResolveExcerptMatcher(func(excerpt ExcerptT) bool {
		return excerpt.Id().HasPrefix(prefix)
	})
}

// ResolveExcerptMatcher retrieves an Excerpt selected by the given matcher function.
func (sc *SubCache[SharedT, ExcerptT, ViewT]) ResolveExcerptMatcher(f func(ExcerptT) bool) (ExcerptT, error) {
	id, err := sc.resolveMatcher(f)
	if err != nil {
		return *new(ExcerptT), err
	}
	return sc.ResolveExcerpt(id)
}

func (sc *SubCache[SharedT, ExcerptT, ViewT]) resolveMatcher(f func(ExcerptT) bool) (entity.Id, error) {
	sc.requestSync()
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

// add registers a freshly committed entity, and returns a view of it.
func (sc *SubCache[SharedT, ExcerptT, ViewT]) add(shared SharedT) (ViewT, error) {
	id := shared.Id()

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
		sc.cached[id] = shared
		sc.lru.Add(id)
		sc.muMaps.Unlock()
		return changes, nil
	}()
	sc.notifyChanges(changes)
	if err != nil {
		return *new(ViewT), err
	}

	sc.evictIfNeeded()

	return sc.newView(shared), nil
}

func (sc *SubCache[SharedT, ExcerptT, ViewT]) Remove(prefix string) error {
	id, err := sc.resolveMatcher(func(excerpt ExcerptT) bool {
		return excerpt.Id().HasPrefix(prefix)
	})
	if err != nil {
		return err
	}

	err = sc.actions.Remove(sc.repo, id)
	if err != nil {
		return err
	}

	return sc.sync(id)
}

func (sc *SubCache[SharedT, ExcerptT, ViewT]) RemoveAll() error {
	err := sc.actions.RemoveAll(sc.repo)
	if err != nil {
		return err
	}

	return sc.syncAll(nil)
}

func (sc *SubCache[SharedT, ExcerptT, ViewT]) MergeAll(remote string) <-chan entity.MergeResult {
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
		}

		// One listing of the refs rather than a lookup per merged entity, which
		// also picks up any change made outside since the last sync. Loaded
		// copies that are behind their reference are brought up to date there,
		// like any other, and nothing is loaded for a merge alone: a merge is not
		// a use of the entity.
		err = sc.syncAll(nil)
		if err != nil {
			out <- entity.NewMergeError(err, "")
		}
	}()

	return out
}

// GetNamespace expose the namespace in git where entities are located.
func (sc *SubCache[SharedT, ExcerptT, ViewT]) GetNamespace() string {
	return sc.namespace
}

// newView wraps the loaded copy of an entity for a caller, and has its commits
// refresh the derived state.
func (sc *SubCache[SharedT, ExcerptT, ViewT]) newView(shared SharedT) ViewT {
	id := shared.Id()
	return sc.makeView(shared, func() error { return sc.onCommit(id) })
}

// onCommit refreshes the derived state after an entity has been committed.
func (sc *SubCache[SharedT, ExcerptT, ViewT]) onCommit(id entity.Id) error {
	return sc.sync(id)
}

// notifyObservers notifies all the observers when something happening for an entity
func (sc *SubCache[SharedT, ExcerptT, ViewT]) notifyObservers(event EntityEventType, id entity.Id) {
	sc.muObservers.RLock()
	for observer, repoName := range sc.observers {
		observer.EntityEvent(event, repoName, sc.typename, id)
	}
	sc.muObservers.RUnlock()
}

func (sc *SubCache[SharedT, ExcerptT, ViewT]) notifyChanges(changes map[entity.Id]EntityEventType) {
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

// buildDerived derives the state the cache keeps about an entity from its
// committed state. The commit is recorded before the rest is read: should a
// commit land in between, the derived state is newer than its record says, and
// the sync that commit triggers rebuilds it. The other way around would record
// it as up to date while it is not.
func (sc *SubCache[SharedT, ExcerptT, ViewT]) buildDerived(shared SharedT) derived[ExcerptT] {
	commit := shared.LastCommit()
	return derived[ExcerptT]{
		commit:    commit,
		excerpt:   sc.makeExcerpt(shared),
		indexData: sc.makeIndexData(shared),
	}
}

// sync brings the derived state of an entity up to date with its reference, see
// syncOneLocked, and notifies the observers.
func (sc *SubCache[SharedT, ExcerptT, ViewT]) sync(id entity.Id) error {
	sc.muDerived.Lock()
	changes, err := sc.syncOneLocked(id)
	sc.muDerived.Unlock()
	sc.notifyChanges(changes)
	return err
}

// syncAll brings the derived state of every entity up to date with the
// references, see syncAllLocked, and notifies the observers. progress, if not nil,
// receives the build events.
func (sc *SubCache[SharedT, ExcerptT, ViewT]) syncAll(progress func(BuildEvent)) error {
	sc.muDerived.Lock()
	changes, err := sc.syncAllLocked(progress)
	sc.muDerived.Unlock()
	sc.notifyChanges(changes)

	return err
}

// syncInterval is how long after a sync of the worker a read can start another,
// see startSyncWorker.
//
// TODO: shrink it, or drop it, once a sync that finds nothing to do is cheap.
// With 10k bugs it takes ~110ms and allocates ~20MB, 93% of the CPU and 80% of
// the allocations in ListRefs, which opens and reads every loose ref file; the
// rest mostly decodes the index record. Levers, by expected gain:
//   - go-git keeping the refs it lists, and reading them again only when the
//     stat of their directory or of packed-refs changed: no read at all when
//     nothing changed;
//   - packing the refs: listing 10k packed refs takes ~3ms instead of ~100ms,
//     until new writes make them loose again;
//   - keeping the decoded index record in memory rather than decoding it on
//     each sync.
const syncInterval = time.Second

// syncPeriod is how often the worker syncs, whether anything reads or not, see
// startSyncWorker.
const syncPeriod = time.Minute

// requestSync asks the worker for a sync of every entity, to bring the cache up
// to date with changes made to the references outside of it, by another process
// or the git binary. Every read calls it first. It never waits: the read is
// served from the cache as it is, and a change made outside shows up shortly
// after. Any number of requests collapse into one.
func (sc *SubCache[SharedT, ExcerptT, ViewT]) requestSync() {
	select {
	case sc.syncTrigger <- struct{}{}:
	default:
	}
}

// startSyncWorker starts the worker, until Close. It syncs every entity in the
// background, one sync at a time: every syncPeriod, which keeps the cache close
// to the references while nothing reads so that the observers learn of changes
// made outside, and on request, see requestSync. A request received less than
// syncInterval after the latest sync ended is postponed until then, not
// dropped: a change made outside then shows up at most syncInterval after a
// read. When nothing changed, a sync only lists the references, which is why
// it isn't done more often.
//
// A failed sync is kept in syncWorkerErr, and retried like any other.
func (sc *SubCache[SharedT, ExcerptT, ViewT]) startSyncWorker() {
	sc.muSyncWorker.Lock()
	defer sc.muSyncWorker.Unlock()
	if sc.syncWorkerClosed {
		return
	}

	sc.syncWorkerDone.Add(1)
	go func() {
		defer sc.syncWorkerDone.Done()

		ticker := time.NewTicker(syncPeriod)
		defer ticker.Stop()
		// started right after the sync of Load
		lastSync := time.Now()
		// postponed fires when a request received too soon after the latest
		// sync can be served, nil if there is none
		var postponed <-chan time.Time

		for {
			select {
			case <-sc.stopSyncWorker:
				return
			case <-ticker.C:
			case <-postponed:
			case <-sc.syncTrigger:
				if wait := syncInterval - time.Since(lastSync); wait > 0 {
					if postponed == nil {
						postponed = time.After(wait)
					}
					continue
				}
			}

			// the sync serves any postponed request
			postponed = nil
			err := sc.syncAll(nil)
			lastSync = time.Now()

			sc.muSyncWorker.Lock()
			sc.syncWorkerErr = err
			sc.muSyncWorker.Unlock()
		}
	}()
}

// closeSyncWorker stops the worker, and waits for it to end. It won't start
// again.
func (sc *SubCache[SharedT, ExcerptT, ViewT]) closeSyncWorker() {
	sc.muSyncWorker.Lock()
	if !sc.syncWorkerClosed {
		sc.syncWorkerClosed = true
		close(sc.stopSyncWorker)
	}
	sc.muSyncWorker.Unlock()
	sc.syncWorkerDone.Wait()
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
func (sc *SubCache[SharedT, ExcerptT, ViewT]) syncOneLocked(id entity.Id) (map[entity.Id]EntityEventType, error) {
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
func (sc *SubCache[SharedT, ExcerptT, ViewT]) syncAllLocked(progress func(BuildEvent)) (map[entity.Id]EntityEventType, error) {
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
// latter given in indexBuiltFrom. An entity that is behind in either store, or
// whose loaded copy is behind its reference, is brought to its reference once,
// see bringToRef, then applied to each store that is behind, or removed from
// them if its reference is gone, along with its loaded copy. Nothing is read if
// everything is up to date. Changes are applied in batches, and the excerpt
// file is written once, if the excerpts changed.
//
// Entities are brought up to date under muDerived, which every change to the
// derived state holds, so that each sync applies a state at least as recent as
// the previous one. A reference moving after that is followed by a sync of its
// own, from the commit callback.
//
// It returns the changes of excerpts that were applied, even on error, for the
// caller to notify once muDerived is released. progress, if not nil, receives
// the build events.
func (sc *SubCache[SharedT, ExcerptT, ViewT]) syncEntitiesLocked(index repository.Index, candidates map[entity.Id]struct{}, refs map[string]repository.Hash, indexBuiltFrom map[string]repository.Hash, progress func(BuildEvent)) (map[entity.Id]EntityEventType, error) {
	if progress == nil {
		progress = func(BuildEvent) {}
	}

	var behind []entity.Id
	sc.muMaps.RLock()
	for id := range candidates {
		ref := refs[id.String()]
		loaded, isLoaded := sc.cached[id]
		if sc.builtFrom[id] != ref || indexBuiltFrom[id.String()] != ref || (isLoaded && loaded.LastCommit() != ref) {
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
		shared, err := sc.bringToRef(id, refs[id.String()])
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
			d := sc.buildDerived(shared)
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

// bringToRef returns the entity at its reference ref, for its derived state to
// be built: the loaded copy, brought up to date if it's behind, see
// Actions.Refresh, or a read from git otherwise, which is not kept in memory. A
// refresh that returns a replacement has it loaded instead of the previous copy.
func (sc *SubCache[SharedT, ExcerptT, ViewT]) bringToRef(id entity.Id, ref repository.Hash) (SharedT, error) {
	sc.muMaps.RLock()
	loaded, isLoaded := sc.cached[id]
	sc.muMaps.RUnlock()

	if !isLoaded {
		return sc.actions.ReadWithResolver(sc.repo, sc.resolvers(), id)
	}
	if loaded.LastCommit() == ref {
		return loaded, nil
	}

	refreshed, err := sc.actions.Refresh(sc.repo, sc.resolvers(), loaded)
	if err != nil {
		return refreshed, err
	}
	sc.muMaps.Lock()
	// unless evicted in the meantime, and maybe loaded again since
	if current, still := sc.cached[id]; still && any(current) == any(loaded) {
		sc.cached[id] = refreshed
	}
	sc.muMaps.Unlock()
	return refreshed, nil
}

// evictIfNeeded drops the least recently used loaded copies until the cache is
// under its size. A loaded copy holds committed state only, so any of them can
// go: a view keeps working with the copy it holds, and its commits reach the
// repository all the same.
func (sc *SubCache[SharedT, ExcerptT, ViewT]) evictIfNeeded() {
	sc.muMaps.Lock()
	defer sc.muMaps.Unlock()

	for sc.lru.Len() > sc.maxLoaded {
		id, ok := sc.lru.GetOldest()
		if !ok {
			return
		}
		sc.lru.Remove(id)
		delete(sc.cached, id)
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
