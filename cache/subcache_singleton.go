package cache

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/repository"
	"github.com/git-bug/git-bug/util/lamport"
)

// SingletonActions are the functions a SingletonCache needs from its entity type,
// see Actions.
type SingletonActions[SnapT dag.Snapshot, OpT dag.Operation] struct {
	// New returns a new, empty entity
	New func() dag.Tracked[SnapT, OpT]
	// Head returns the commit the reference of the entity points to, or
	// repository.ErrNotFound if there is none
	Head func(ctx context.Context, repo repository.Repo) (repository.Hash, error)
	// ReadWithResolver reads the entity from the repository, or returns an
	// entity.ErrNotFound if there is none
	ReadWithResolver func(ctx context.Context, repo repository.Repo, resolvers entity.Resolvers) (dag.Tracked[SnapT, OpT], error)
	// NewCreate returns the operation creating the entity, see refTracked
	NewCreate    func(repo repository.RepoClock, author identity.Interface, unixTime int64) (OpT, error)
	RemoveAll    func(ctx context.Context, repo repository.Repo) error
	MergeAll     func(ctx context.Context, repo repository.Repo, resolvers entity.Resolvers, remote string, mergeAuthor identity.Interface) <-chan entity.MergeResult
	EnsureClocks func(ctx context.Context, repo repository.Repo) error
}

// SingletonCache holds the singleton entity of a type, see
// dag.Definition.SingletonKey: there is at most one, in
// refs/<namespace>/<SingletonKey>, not addressed by id. SubCache doesn't fit it:
// its index, excerpts, prefix resolution and search all exist to serve many
// entities found by id.
//
// It holds a single loaded copy, see refTracked, whichever entity the
// reference holds, or none. Callers get views of that copy, see View: what they
// stage is their own until committed.
type SingletonCache[SnapT cloneable[SnapT], OpT dag.OperationWithApply[SnapT], ViewT any] struct {
	repo      repository.Repo
	resolvers func() entity.Resolvers

	// ctx is the lifetime of the SingletonCache, cancelled by Close, see
	// SubCache.ctx.
	ctx context.Context

	getUserIdentity getUserIdentityFunc
	makeView        func(shared *sharedEntity[SnapT, OpT], onCommit func()) ViewT
	actions         SingletonActions[SnapT, OpT]

	typename  string
	namespace string

	// shared is the single loaded copy, for the whole life of the cache: it
	// follows the reference, see refTracked.
	shared *sharedEntity[SnapT, OpT]

	// muSync serializes the syncs, and guards the state below.
	muSync sync.Mutex
	// notifiedId and notifiedCommit are the entity and commit the observers
	// were last told of, empty for none. The loaded copy can't tell: a commit
	// through a view moves it before the sync that notifies, see onCommit.
	notifiedId     entity.Id
	notifiedCommit repository.Hash

	muObservers sync.RWMutex
	observers   map[Observer]string // observer --> repo name

	// worker syncs in the background, see requestSync and startSyncWorker
	worker *syncWorker
}

func NewSingletonCache[SnapT cloneable[SnapT], OpT dag.OperationWithApply[SnapT], ViewT any](
	repo repository.Repo,
	resolvers func() entity.Resolvers, getUserIdentity getUserIdentityFunc,
	makeView func(shared *sharedEntity[SnapT, OpT], onCommit func()) ViewT,
	actions SingletonActions[SnapT, OpT],
	typename, namespace string) *SingletonCache[SnapT, OpT, ViewT] {
	ctx, cancel := context.WithCancel(context.Background())
	sc := &SingletonCache[SnapT, OpT, ViewT]{
		repo:            repo,
		resolvers:       resolvers,
		ctx:             ctx,
		getUserIdentity: getUserIdentity,
		makeView:        makeView,
		actions:         actions,
		typename:        typename,
		namespace:       namespace,
		shared: &sharedEntity[SnapT, OpT]{
			Tracked: &refTracked[SnapT, OpT]{
				typename: typename,
				actions:  actions,
				current:  actions.New(),
			},
		},
	}
	// a singleton has nothing to sync by id: every request syncs it
	sc.worker = newSyncWorker(ctx, cancel, func(bool, []entity.Id) error {
		return sc.sync()
	})
	return sc
}

func (sc *SingletonCache[SnapT, OpT, ViewT]) Typename() string {
	return sc.typename
}

// GetNamespace expose the namespace in git where the entity is located.
func (sc *SingletonCache[SnapT, OpT, ViewT]) GetNamespace() string {
	return sc.namespace
}

// EnsureClocks makes sure that the clocks of the entity are usable, rebuilding
// them if not. See dag.EnsureClocks.
func (sc *SingletonCache[SnapT, OpT, ViewT]) EnsureClocks() error {
	return sc.actions.EnsureClocks(sc.ctx, sc.repo)
}

// Load reads the entity, if any. The caller must read all the returned events,
// and start the sync worker once every subcache loaded, see startSyncWorker.
//
// An entity that can't be read, for instance because the identity of an author
// is missing, doesn't fail the load: it's reported as a warning, the cache works
// without it, and the worker retries.
func (sc *SingletonCache[SnapT, OpT, ViewT]) Load() <-chan BuildEvent {
	out := make(chan BuildEvent)

	go func() {
		defer close(out)

		err := sc.sync()
		switch {
		case err == nil:
		case sc.ctx.Err() != nil:
			out <- BuildEvent{Typename: sc.typename, Err: err}
		default:
			out <- BuildEvent{
				Typename: sc.typename,
				Event:    BuildEventWarning,
				Warning:  fmt.Errorf("%s: %w", sc.typename, err),
			}
			// served once the worker is started
			sc.requestSync()
		}
	}()

	return out
}

// SetCacheSize does nothing: there is a single entity, always loaded.
func (sc *SingletonCache[SnapT, OpT, ViewT]) SetCacheSize(int) {}

func (sc *SingletonCache[SnapT, OpT, ViewT]) Close() error {
	sc.worker.close()
	return nil
}

func (sc *SingletonCache[SnapT, OpT, ViewT]) RegisterObserver(repoName string, observer Observer) {
	sc.muObservers.Lock()
	defer sc.muObservers.Unlock()
	if sc.observers == nil {
		sc.observers = make(map[Observer]string)
	}
	sc.observers[observer] = repoName
}

func (sc *SingletonCache[SnapT, OpT, ViewT]) UnregisterObserver(observer Observer) {
	sc.muObservers.Lock()
	defer sc.muObservers.Unlock()
	delete(sc.observers, observer)
}

// View returns a new view of the entity: what a caller stages on it is its own
// until committed, while the committed state is shared. With no entity yet, the
// view starts empty, and its first commit creates it, see refTracked.
func (sc *SingletonCache[SnapT, OpT, ViewT]) View() ViewT {
	return sc.makeView(sc.shared, sc.onCommit)
}

// Snapshot returns the committed state of the entity, empty if there is none. It
// is shared with every caller, and must not be modified.
func (sc *SingletonCache[SnapT, OpT, ViewT]) Snapshot() SnapT {
	return sc.shared.Compile()
}

func (sc *SingletonCache[SnapT, OpT, ViewT]) RemoveAll(ctx context.Context) error {
	err := sc.actions.RemoveAll(ctx, sc.repo)
	// git is changed already: a failed sync is left to the worker, as in onCommit
	sc.onCommit()
	return err
}

func (sc *SingletonCache[SnapT, OpT, ViewT]) MergeAll(ctx context.Context, remote string) <-chan entity.MergeResult {
	out := make(chan entity.MergeResult)

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

		for result := range sc.actions.MergeAll(ctx, sc.repo, sc.resolvers(), remote, author) {
			out <- result
		}

		// the merge is in git already, and reported: a failed sync is left to
		// the worker, as in onCommit
		sc.onCommit()
	}()

	return out
}

// refsChanged asks the worker for a sync, whatever the keys: the namespace holds
// the entity only.
func (sc *SingletonCache[SnapT, OpT, ViewT]) refsChanged([]string) {
	sc.requestSync()
}

// requestSync asks the worker for a sync, see syncWorker.request. The ids don't
// matter: the entity is not addressed by id.
func (sc *SingletonCache[SnapT, OpT, ViewT]) requestSync(...entity.Id) {
	sc.worker.request()
}

// startSyncWorker starts the worker, until Close, see syncWorker.start.
func (sc *SingletonCache[SnapT, OpT, ViewT]) startSyncWorker() {
	sc.worker.start()
}

// onCommit syncs after the reference was changed through the cache. The change
// is in git already, so a failed sync doesn't fail it: it's left to the worker.
func (sc *SingletonCache[SnapT, OpT, ViewT]) onCommit() {
	if err := sc.sync(); err != nil {
		sc.requestSync()
	}
}

// sync brings the loaded copy up to date with the reference, and notifies the
// observers of the change since the previous sync: the entity created, updated,
// or removed, or replaced by another one, which is a removal and a creation.
func (sc *SingletonCache[SnapT, OpT, ViewT]) sync() error {
	sc.muSync.Lock()
	if sc.ctx.Err() != nil {
		// closed
		sc.muSync.Unlock()
		return nil
	}

	err := sc.shared.Repair(sc.ctx, sc.repo, sc.resolvers())
	if err != nil {
		sc.muSync.Unlock()
		return err
	}

	id, commit := sc.shared.Id(), sc.shared.LastCommit()
	if commit == "" {
		id = ""
	}
	previousId, previousCommit := sc.notifiedId, sc.notifiedCommit
	sc.notifiedId, sc.notifiedCommit = id, commit
	sc.muSync.Unlock()

	switch {
	case commit == previousCommit:
	case previousId != id:
		if previousId != "" {
			sc.notifyObservers(EntityEventRemoved, previousId)
		}
		if id != "" {
			sc.notifyObservers(EntityEventCreated, id)
		}
	default:
		sc.notifyObservers(EntityEventUpdated, id)
	}
	return nil
}

// notifyObservers notifies all the observers when something happening for the
// entity
func (sc *SingletonCache[SnapT, OpT, ViewT]) notifyObservers(event EntityEventType, id entity.Id) {
	sc.muObservers.RLock()
	defer sc.muObservers.RUnlock()
	for observer, repoName := range sc.observers {
		observer.EntityEvent(event, repoName, sc.typename, id)
	}
}

var _ dag.Tracked[dag.Snapshot, dag.Operation] = &refTracked[dag.Snapshot, dag.Operation]{}

// refTracked wraps the entity that the reference of a singleton holds: it
// delegates to that entity, and switches to another one when the reference holds
// another, see Repair. On top of that, a commit with no entity creates it, see
// CommitOperations. It is held by the sharedEntity of the SingletonCache, which
// serializes the changes to its committed state.
//
// The reference of any other entity is named after its id, so it only ever holds
// that entity: it moves to newer commits of it, which dag.Entity.Repair reads in
// place, or is removed. Its loaded copy is always the same dag.Entity, and the
// views can hold it as is. The reference of a singleton has a fixed name, so it
// differs in three ways:
//
//   - it can hold another entity: one created first by another process or clone,
//     or after a removal. The views hold the loaded copy, and their commits must
//     land on whatever entity is there, so the switch happens underneath them: a
//     Repair that finds another entity, where dag.Entity.Repair returns an
//     entity.ErrNotFound, switches to it
//   - it can be absent while the cache holds a copy: views are handed out even
//     with no entity yet, so there is then a new empty one, which a Repair
//     switches from when an entity appears, and back to when it's removed
//   - the creation is implicit: nobody creates the entity, the first change does.
//     So it is never staged: a commit with no entity creates it first. If another
//     one was created meanwhile, the commit fails as the reference moved, and the
//     retry after the Repair commits on top of that one
type refTracked[SnapT dag.Snapshot, OpT dag.Operation] struct {
	typename string
	actions  SingletonActions[SnapT, OpT]

	// mu guards current, the entity followed
	mu      sync.RWMutex
	current dag.Tracked[SnapT, OpT]
}

// followed returns the entity followed.
func (s *refTracked[SnapT, OpT]) followed() dag.Tracked[SnapT, OpT] {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.current
}

func (s *refTracked[SnapT, OpT]) Id() entity.Id               { return s.followed().Id() }
func (s *refTracked[SnapT, OpT]) Validate() error             { return s.followed().Validate() }
func (s *refTracked[SnapT, OpT]) Compile() SnapT              { return s.followed().Compile() }
func (s *refTracked[SnapT, OpT]) LastCommit() repository.Hash { return s.followed().LastCommit() }
func (s *refTracked[SnapT, OpT]) CreateLamportTime() lamport.Time {
	return s.followed().CreateLamportTime()
}
func (s *refTracked[SnapT, OpT]) EditLamportTime() lamport.Time {
	return s.followed().EditLamportTime()
}

// CommitOperations writes the given operations on top of the committed state,
// see dag.Tracked. With no entity, the operation creating it is written first, by
// the author of the first operation, at its time.
func (s *refTracked[SnapT, OpT]) CommitOperations(ctx context.Context, repo repository.Repo, ops []OpT) error {
	current := s.followed()
	if len(ops) > 0 && current.LastCommit() == "" {
		create, err := s.actions.NewCreate(repo, ops[0].Author(), ops[0].Time().Unix())
		if err != nil {
			return err
		}
		ops = append([]OpT{create}, ops...)
	}
	return current.CommitOperations(ctx, repo, ops)
}

// Repair reloads the committed state from the repository, see dag.Tracked,
// switching to the entity the reference holds if it's another one. With no
// reference, it switches to a new empty entity and succeeds: having no entity is a
// state like any other for a singleton, and a commit retried after it creates the
// entity again.
//
// Only a missing reference means there is no entity. A read can also return an
// entity.ErrNotFound for a missing dependency, such as the identity of an author:
// that is returned as any other failure, the entity followed being left as it is.
func (s *refTracked[SnapT, OpT]) Repair(ctx context.Context, repo repository.Repo, resolvers entity.Resolvers) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.actions.Head(ctx, repo)
	if errors.Is(err, repository.ErrNotFound) {
		if s.current.LastCommit() != "" {
			s.current = s.actions.New()
		}
		return nil
	}
	if err != nil {
		return err
	}

	if s.current.LastCommit() != "" {
		err := s.current.Repair(ctx, repo, resolvers)
		if !entity.IsErrNotFound(err) {
			return err
		}
		// the reference holds another entity, or a dependency is missing: the
		// read below tells
	}

	next, err := s.actions.ReadWithResolver(ctx, repo, resolvers)
	if err != nil {
		return err
	}
	s.current = next
	return nil
}
