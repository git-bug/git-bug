package cache

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/repository"
	"resenje.org/singleflight"
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

type getUserIdentityFunc func(ctx context.Context) (*IdentityCache, error)

// Actions expose a number of action functions on Entities, to give upper layers (cache) a way to normalize interactions.
// Note: ideally this wouldn't exist, the cache layer would assume that everything is an entity/dag, and directly use the
// functions from this package, but right now identities are not using that framework.
type Actions[SharedT Shared] struct {
	// ReadWithResolver reads an entity from the repository
	ReadWithResolver func(ctx context.Context, repo repository.Repo, resolvers entity.Resolvers, id entity.Id) (SharedT, error)
	// Refresh brings a loaded entity up to date with its reference, either in place
	// or by returning a replacement, which the cache then holds instead
	Refresh      func(ctx context.Context, repo repository.Repo, resolvers entity.Resolvers, shared SharedT) (SharedT, error)
	Remove       func(repo repository.Repo, id entity.Id) error
	RemoveAll    func(repo repository.Repo) error
	MergeAll     func(ctx context.Context, repo repository.Repo, resolvers entity.Resolvers, remote string, mergeAuthor identity.Interface) <-chan entity.MergeResult
	EnsureClocks func(repo repository.Repo) error
}

var _ cacheMgmt = &SubCache[Shared, Excerpt, entity.Resolved]{}

// SubCache holds the entities of one type: their derived state, and a single
// loaded copy of each entity in use, see Shared. Callers get views of that copy,
// see Resolve.
type SubCache[SharedT Shared, ExcerptT Excerpt, ViewT entity.Resolved] struct {
	repo      repository.Repo
	resolvers func() entity.Resolvers

	// ctx is the lifetime of the SubCache, cancelled by Close. The work done
	// for the SubCache itself rather than for a caller runs with it: Load, and
	// every sync, including the one following a commit, as the derived state
	// must follow the references whatever happens to the caller.
	ctx    context.Context
	cancel context.CancelFunc

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
	// records the commits too, for every process: this is what this one has
	// seen, and notified.
	builtFrom map[entity.Id]repository.Hash
	// cached holds the single loaded copy of each entity in use, with its
	// committed state only, see Shared. Any of them can be evicted: a view keeps
	// working with the copy it holds.
	cached map[entity.Id]SharedT
	lru    lruIdCache
	// loading deduplicates the concurrent reads of an entity missing from
	// cached, see Resolve.
	loading singleflight.Group[entity.Id, SharedT]

	muObservers sync.RWMutex
	observers   map[Observer]string // observer --> repo name

	// syncTrigger asks the worker for a sync of what is pending, see
	// requestSync and startSyncWorker.
	syncTrigger    chan struct{}
	syncWorkerDone sync.WaitGroup
	// muSyncWorker guards the state of the worker below, and the cancellation
	// of ctx, for the worker not to start once closed. It is only held briefly.
	muSyncWorker sync.Mutex
	// syncWorkerPendingAll and syncWorkerPendingIds are what the next sync of
	// the worker covers: every entity, or those ids.
	syncWorkerPendingAll bool
	syncWorkerPendingIds map[entity.Id]struct{}
	// syncWorkerErr is the error of the latest sync of the worker, nil if it
	// succeeded.
	syncWorkerErr error
}

func NewSubCache[SharedT Shared, ExcerptT Excerpt, ViewT entity.Resolved](
	repo repository.Repo,
	resolvers func() entity.Resolvers, getUserIdentity getUserIdentityFunc,
	makeView func(shared SharedT, onCommit func() error) ViewT,
	makeExcerpt func(SharedT) ExcerptT,
	makeIndexData func(SharedT) []string,
	actions Actions[SharedT],
	typename, namespace string,
	version uint, maxLoaded int) *SubCache[SharedT, ExcerptT, ViewT] {
	ctx, cancel := context.WithCancel(context.Background())
	return &SubCache[SharedT, ExcerptT, ViewT]{
		repo:            repo,
		ctx:             ctx,
		cancel:          cancel,
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

// Load reads the excerpts of the index, and brings them up to date with the
// repository. The caller must read all the returned events.
func (sc *SubCache[SharedT, ExcerptT, ViewT]) Load() <-chan BuildEvent {
	out := make(chan BuildEvent)

	go func() {
		defer close(out)

		err := sc.read(sc.ctx)
		if err == nil {
			err = sc.syncAll(func(event BuildEvent) { out <- event })
		}
		if err != nil {
			out <- BuildEvent{Typename: sc.typename, Err: err}
			return
		}
		sc.startSyncWorker()
	}()

	return out
}

// read fills the excerpts with those of the index, as left by the latest sync
// of any process. Each excerpt comes with the commit it was built from, read
// together: another process can replace a document with the one of an older
// commit, and an excerpt recorded under a commit it wasn't built from would
// never be brought up to date.
// An excerpt that can't be decoded is left out, for the sync to build it again.
func (sc *SubCache[SharedT, ExcerptT, ViewT]) read(ctx context.Context) error {
	sc.muDerived.Lock()
	defer sc.muDerived.Unlock()

	index, err := sc.repo.GetIndex(sc.namespace, sc.version)
	if err != nil {
		return err
	}
	builtFrom, raws, err := index.Excerpts(ctx)
	if err != nil {
		return err
	}

	sc.muMaps.Lock()
	defer sc.muMaps.Unlock()
	for key, raw := range raws {
		id := entity.Id(key)
		excerpt, ok := sc.decodeExcerpt(id, raw)
		if !ok {
			continue
		}
		sc.excerpts[id] = excerpt
		sc.builtFrom[id] = builtFrom[key]
	}
	return nil
}

// decodeExcerpt decodes the excerpt of an entity as the index stores it. It
// returns false if it can't.
func (sc *SubCache[SharedT, ExcerptT, ViewT]) decodeExcerpt(id entity.Id, raw []byte) (ExcerptT, bool) {
	var excerpt ExcerptT
	if json.Unmarshal(raw, &excerpt) != nil {
		return excerpt, false
	}
	// the id is the key of the excerpt, not part of it
	excerpt.setId(id)
	return excerpt, true
}

// excerptFromIndex returns the excerpt of an entity as the index stores it, if
// it was built from ref. It returns false if the document is gone, was replaced
// by another process since it was listed, or can't be decoded.
func (sc *SubCache[SharedT, ExcerptT, ViewT]) excerptFromIndex(ctx context.Context, index repository.Index, id entity.Id, ref repository.Hash) (*derived[ExcerptT], bool) {
	commit, raw, err := index.Excerpt(ctx, id.String())
	if err != nil || commit != ref {
		return nil, false
	}
	excerpt, ok := sc.decodeExcerpt(id, raw)
	if !ok {
		return nil, false
	}
	return &derived[ExcerptT]{commit: commit, excerpt: excerpt}, true
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
func (sc *SubCache[SharedT, ExcerptT, ViewT]) Resolve(ctx context.Context, id entity.Id) (ViewT, error) {
	sc.muMaps.RLock()
	shared, ok := sc.cached[id]
	if ok {
		// mark as recently used
		sc.lru.Get(id)
		sc.muMaps.RUnlock()
		return sc.newView(shared), nil
	}
	sc.muMaps.RUnlock()

	shared, _, err := sc.loading.Do(ctx, id, func(ctx context.Context) (SharedT, error) {
		return sc.load(ctx, id)
	})
	if err != nil {
		return *new(ViewT), err
	}
	return sc.newView(shared), nil
}

// load reads an entity from the repository and keeps it as its loaded copy,
// unless there is one already.
func (sc *SubCache[SharedT, ExcerptT, ViewT]) load(ctx context.Context, id entity.Id) (SharedT, error) {
	shared, err := sc.actions.ReadWithResolver(ctx, sc.repo, sc.resolvers(), id)
	if err != nil {
		return *new(SharedT), err
	}

	sc.muMaps.Lock()
	if existing, ok := sc.cached[id]; ok {
		// loaded in the meantime, keep a single copy in memory and mark it as
		// recently used
		sc.lru.Get(id)
		sc.muMaps.Unlock()
		return existing, nil
	}
	// This copy can already be stale: if the reference moves while it's read,
	// a sync running meanwhile doesn't see it, as it's not loaded yet. The next
	// sync of the entity brings it up to date (loadedBehind). Until then it's
	// still safe to use: a view works on any copy, and its commits reach the
	// repository all the same.
	sc.cached[id] = shared
	sc.lru.Add(id)
	sc.muMaps.Unlock()

	sc.evictIfNeeded()

	return shared, nil
}

// ResolvePrefix retrieve an entity matching an id prefix. It fails if multiple
// entities match.
func (sc *SubCache[SharedT, ExcerptT, ViewT]) ResolvePrefix(ctx context.Context, prefix string) (ViewT, error) {
	return sc.ResolveMatcher(ctx, func(excerpt ExcerptT) bool {
		return excerpt.Id().HasPrefix(prefix)
	})
}

func (sc *SubCache[SharedT, ExcerptT, ViewT]) ResolveMatcher(ctx context.Context, f func(ExcerptT) bool) (ViewT, error) {
	id, err := sc.resolveMatcher(f)
	if err != nil {
		return *new(ViewT), err
	}
	return sc.Resolve(ctx, id)
}

// ResolveExcerpt retrieves an Excerpt matching the exact given id
func (sc *SubCache[SharedT, ExcerptT, ViewT]) ResolveExcerpt(id entity.Id) (ExcerptT, error) {
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

func (sc *SubCache[SharedT, ExcerptT, ViewT]) MergeAll(ctx context.Context, remote string) <-chan entity.MergeResult {
	out := make(chan entity.MergeResult)

	// Intercept merge results to update the cache properly
	go func() {
		defer close(out)

		// the author is only needed for merge commits, so a user identity is optional
		user, err := sc.getUserIdentity(ctx)
		if err != nil && !errors.Is(err, identity.ErrNoIdentitySet) {
			out <- entity.NewMergeError(err, "")
			return
		}
		var author identity.Interface
		if err == nil {
			author = user
		}

		results := sc.actions.MergeAll(ctx, sc.repo, sc.resolvers(), remote, author)
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

// newView wraps a copy of an entity for a caller, and has its commits go through
// onCommit.
func (sc *SubCache[SharedT, ExcerptT, ViewT]) newView(shared SharedT) ViewT {
	return sc.makeView(shared, func() error { return sc.onCommit(shared) })
}

// onCommit refreshes the derived state after an entity has been committed from
// the given copy. With no copy of it loaded, as for a new entity or an evicted
// one, that copy becomes the loaded one if it holds what was synced.
func (sc *SubCache[SharedT, ExcerptT, ViewT]) onCommit(shared SharedT) error {
	// the id is only read now: the copy of a new entity has none before its
	// first commit
	id := shared.Id()

	// Under muDerived until the copy is registered, so that a concurrent
	// removal comes either before, and the copy is not loaded, or after.
	changes, err := func() (map[entity.Id]EntityEventType, error) {
		sc.muDerived.Lock()
		defer sc.muDerived.Unlock()

		changes, err := sc.syncIdsLocked(sc.ctx, []entity.Id{id})
		if err != nil {
			return changes, err
		}
		// The copy can be behind what was synced: another writer moved the
		// reference since, or the view committed a copy of its own, as an
		// identity view does. The next Resolve reads it again then.
		if built, ok := sc.builtFrom[id]; !ok || shared.LastCommit() != built {
			return changes, nil
		}

		sc.muMaps.Lock()
		if _, has := sc.cached[id]; !has {
			sc.cached[id] = shared
			sc.lru.Add(id)
		}
		sc.muMaps.Unlock()
		return changes, nil
	}()
	sc.notifyChanges(changes)
	if err != nil {
		return err
	}

	sc.evictIfNeeded()
	return nil
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
// syncIdsLocked, and notifies the observers.
func (sc *SubCache[SharedT, ExcerptT, ViewT]) sync(id entity.Id) error {
	return sc.syncIds([]entity.Id{id})
}

// syncIds brings the derived state of the given entities up to date with their
// references, see syncIdsLocked, and notifies the observers.
func (sc *SubCache[SharedT, ExcerptT, ViewT]) syncIds(ids []entity.Id) error {
	sc.muDerived.Lock()
	changes, err := sc.syncIdsLocked(sc.ctx, ids)
	sc.muDerived.Unlock()
	sc.notifyChanges(changes)
	return err
}

// syncAll brings the derived state of every entity up to date with the
// references, see syncAllLocked, and notifies the observers. progress, if not nil,
// receives the build events.
func (sc *SubCache[SharedT, ExcerptT, ViewT]) syncAll(progress func(BuildEvent)) error {
	sc.muDerived.Lock()
	changes, err := sc.syncAllLocked(sc.ctx, progress)
	sc.muDerived.Unlock()
	sc.notifyChanges(changes)

	return err
}

// syncInterval is the shortest time between two syncs of the worker, see
// startSyncWorker. It bounds the cost of a source reporting changes in quick
// succession.
//
// TODO: shrink it, or drop it, once a sync that finds nothing to do is cheap.
// With 10k bugs it takes ~60ms and allocates ~19MB: ~30ms in ListRefs, which
// reads every loose ref file, on a worker per CPU, and ~25ms reading the
// commit of every document of the index. Levers, by expected gain:
//   - go-git keeping the refs it lists, and reading them again only when the
//     stat of their directory or of packed-refs changed: no read at all when
//     nothing changed;
//   - packing the refs: listing 10k packed refs takes ~3ms instead of ~30ms,
//     until new writes make them loose again;
//   - keeping the record of the index in memory rather than reading it on each
//     sync.
const syncInterval = time.Second

// requestSync asks the worker for a sync of the given entities, or of every
// entity if none is given, to bring the cache up to date with changes made to
// the references outside of it, by another process or the git binary. It never
// waits, and any number of requests collapse into one sync, of every entity
// they named.
func (sc *SubCache[SharedT, ExcerptT, ViewT]) requestSync(ids ...entity.Id) {
	sc.muSyncWorker.Lock()
	sc.addPendingLocked(ids...)
	sc.muSyncWorker.Unlock()

	select {
	case sc.syncTrigger <- struct{}{}:
	default:
	}
}

// addPendingLocked adds the given entities, or every entity if none is given,
// to what the next sync of the worker covers. muSyncWorker must be held.
func (sc *SubCache[SharedT, ExcerptT, ViewT]) addPendingLocked(ids ...entity.Id) {
	switch {
	case len(ids) == 0:
		sc.syncWorkerPendingAll = true
		sc.syncWorkerPendingIds = nil
	case !sc.syncWorkerPendingAll:
		if sc.syncWorkerPendingIds == nil {
			sc.syncWorkerPendingIds = make(map[entity.Id]struct{}, len(ids))
		}
		for _, id := range ids {
			sc.syncWorkerPendingIds[id] = struct{}{}
		}
	}
}

// startSyncWorker starts the worker, until Close. It syncs in the background,
// one sync at a time, what requestSync asked for. A request received less than
// syncInterval after the latest sync ended is postponed until then, not
// dropped.
//
// A failed sync is kept in syncWorkerErr. It is not retried on its own: the
// entities it was for stay pending, and are synced again with the next request,
// at the latest from the periodic source backing the others.
func (sc *SubCache[SharedT, ExcerptT, ViewT]) startSyncWorker() {
	sc.muSyncWorker.Lock()
	defer sc.muSyncWorker.Unlock()
	if sc.ctx.Err() != nil {
		return
	}

	sc.syncWorkerDone.Go(func() {
		// started right after the sync of Load
		lastSync := time.Now()
		// postponed fires when a request received too soon after the latest
		// sync can be served, nil if there is none
		var postponed <-chan time.Time

		for {
			select {
			case <-sc.ctx.Done():
				return
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
			sc.muSyncWorker.Lock()
			all := sc.syncWorkerPendingAll
			ids := slices.Collect(maps.Keys(sc.syncWorkerPendingIds))
			sc.syncWorkerPendingAll = false
			sc.syncWorkerPendingIds = nil
			sc.muSyncWorker.Unlock()

			var err error
			switch {
			case all:
				err = sc.syncAll(nil)
			case len(ids) > 0:
				err = sc.syncIds(ids)
			default:
				// served by the previous sync
				continue
			}
			lastSync = time.Now()

			sc.muSyncWorker.Lock()
			sc.syncWorkerErr = err
			if err != nil {
				// no ids for a sync of every entity
				sc.addPendingLocked(ids...)
			}
			sc.muSyncWorker.Unlock()
		}
	})
}

// closeSyncWorker cancels ctx, which interrupts a sync in progress and stops the
// worker, and waits for it to end. It won't start again.
func (sc *SubCache[SharedT, ExcerptT, ViewT]) closeSyncWorker() {
	sc.muSyncWorker.Lock()
	sc.cancel()
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

// syncIdsLocked brings the derived state of the given entities up to date with
// their references, see syncEntitiesLocked.
func (sc *SubCache[SharedT, ExcerptT, ViewT]) syncIdsLocked(ctx context.Context, ids []entity.Id) (map[entity.Id]EntityEventType, error) {
	// nil maps mark the SubCache as closed
	if sc.excerpts == nil {
		return nil, nil
	}

	index, err := sc.repo.GetIndex(sc.namespace, sc.version)
	if err != nil {
		return nil, err
	}
	refs := make(map[string]repository.Hash, len(ids))
	indexBuiltFrom := make(map[string]repository.Hash, len(ids))
	candidates := make(map[entity.Id]struct{}, len(ids))
	for _, id := range ids {
		ref, err := sc.repo.ResolveRef(sc.namespace, id.String())
		switch {
		case err == nil:
			refs[id.String()] = ref
		case !errors.Is(err, repository.ErrNotFound):
			return nil, err
		}
		commit, err := index.BuiltFrom(ctx, id.String())
		switch {
		case err == nil:
			indexBuiltFrom[id.String()] = commit
		case !errors.Is(err, repository.ErrNotFound):
			return nil, err
		}
		candidates[id] = struct{}{}
	}

	return sc.syncEntitiesLocked(ctx, index, candidates, refs, indexBuiltFrom, nil)
}

// syncAllLocked brings the derived state of every entity up to date with the
// references, see syncEntitiesLocked: those with a reference, with derived state
// in memory or in the index, or with a loaded copy.
func (sc *SubCache[SharedT, ExcerptT, ViewT]) syncAllLocked(ctx context.Context, progress func(BuildEvent)) (map[entity.Id]EntityEventType, error) {
	// nil maps mark the SubCache as closed
	if sc.excerpts == nil {
		return nil, nil
	}

	index, err := sc.repo.GetIndex(sc.namespace, sc.version)
	if err != nil {
		return nil, err
	}
	indexBuiltFrom, err := index.AllBuiltFrom(ctx)
	if err != nil {
		return nil, err
	}

	refs, err := sc.repo.ListRefs(sc.namespace)
	if err != nil {
		return nil, err
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

	return sc.syncEntitiesLocked(ctx, index, candidates, refs, indexBuiltFrom, progress)
}

// syncEntitiesLocked brings the derived state of the candidates up to date with
// their references, given in refs, a candidate without one being removed. The
// excerpts and the index each record the commit they were built from, the
// latter given in indexBuiltFrom. An entity that is behind in either store, or
// whose loaded copy is behind its reference, is brought to its reference once,
// see bringToRef, then applied to each store that is behind, or removed from
// them if its reference is gone, along with its loaded copy. An entity only
// behind in memory, the index being up to date from another process or an
// earlier sync, has its excerpt read from the index instead. Nothing is read if
// everything is up to date. Changes are applied in batches.
//
// Entities are brought up to date under muDerived, which every change to the
// derived state holds, so that each sync applies a state at least as recent as
// the previous one. A reference moving after that is followed by a sync of its
// own, from the commit callback.
//
// It returns the changes of excerpts that were applied, even on error, for the
// caller to notify once muDerived is released. progress, if not nil, receives
// the build events.
func (sc *SubCache[SharedT, ExcerptT, ViewT]) syncEntitiesLocked(ctx context.Context, index repository.Index, candidates map[entity.Id]struct{}, refs map[string]repository.Hash, indexBuiltFrom map[string]repository.Hash, progress func(BuildEvent)) (map[entity.Id]EntityEventType, error) {
	if progress == nil {
		progress = func(BuildEvent) {}
	}

	var behind []entity.Id
	// the entities only behind in memory
	inIndex := make(map[entity.Id]bool)
	sc.muMaps.RLock()
	for id := range candidates {
		ref := refs[id.String()]
		loaded, isLoaded := sc.cached[id]
		loadedBehind := isLoaded && loaded.LastCommit() != ref
		if sc.builtFrom[id] != ref || indexBuiltFrom[id.String()] != ref || loadedBehind {
			behind = append(behind, id)
			inIndex[id] = ref != "" && indexBuiltFrom[id.String()] == ref && !loadedBehind
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
			err := batch.Apply(ctx)
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

	// fromEntity builds the derived state of an entity from the entity itself,
	// see bringToRef, adding it to the current batch
	fromEntity := func(id entity.Id, ref repository.Hash) error {
		shared, err := sc.bringToRef(ctx, id, ref)
		if entity.IsErrNotFound(err) {
			// not found can also come from a missing dependency, such as the
			// author identity: only a missing reference is a removal
			_, refErr := sc.repo.ResolveRef(sc.namespace, id.String())
			if !errors.Is(refErr, repository.ErrNotFound) {
				return err
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
			return err
		default:
			d := sc.buildDerived(shared)
			if indexBuiltFrom[id.String()] != d.commit {
				excerpt, err := json.Marshal(d.excerpt)
				if err != nil {
					return err
				}
				err = batch.Set(id.String(), d.commit, excerpt, d.indexData)
				if err != nil {
					return err
				}
				batchIndexed++
			}
			if sc.builtFrom[id] != d.commit {
				pending[id] = &d
			}
		}
		return nil
	}

	for i, id := range behind {
		ref := refs[id.String()]
		var d *derived[ExcerptT]
		fromIndex := false
		if inIndex[id] {
			d, fromIndex = sc.excerptFromIndex(ctx, index, id, ref)
			if !fromIndex {
				// gone, replaced or unreadable since: built from the entity instead
				delete(indexBuiltFrom, id.String())
			}
		}
		if fromIndex {
			pending[id] = d
		} else if err := fromEntity(id, ref); err != nil {
			return changes, err
		}

		batchSize++
		if batchSize >= syncBatchSize {
			err := endBatch()
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

	progress(BuildEvent{Typename: sc.typename, Event: BuildEventFinished})

	return changes, nil
}

// bringToRef returns the entity at its reference ref, for its derived state to
// be built: the loaded copy, brought up to date if it's behind, see
// Actions.Refresh, or a read from git otherwise, which is not kept in memory. A
// refresh that returns a replacement has it loaded instead of the previous copy.
func (sc *SubCache[SharedT, ExcerptT, ViewT]) bringToRef(ctx context.Context, id entity.Id, ref repository.Hash) (SharedT, error) {
	sc.muMaps.RLock()
	loaded, isLoaded := sc.cached[id]
	sc.muMaps.RUnlock()

	if !isLoaded {
		return sc.actions.ReadWithResolver(ctx, sc.repo, sc.resolvers(), id)
	}
	if loaded.LastCommit() == ref {
		return loaded, nil
	}

	refreshed, err := sc.actions.Refresh(ctx, sc.repo, sc.resolvers(), loaded)
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
