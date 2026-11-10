package cache

import (
	"context"
	"errors"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/git-bug/git-bug/entity"
)

// syncInterval is the shortest time between two syncs of the worker, see
// syncWorker.start. It bounds the cost of a source reporting changes in quick
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

// syncRetryMaxDelay is the longest the worker waits before retrying a failed
// sync, see syncWorker.start. It bounds the cost of a failure that lasts.
const syncRetryMaxDelay = time.Minute

// syncWorker syncs a subcache in the background, one sync at a time, to bring
// it up to date with changes made to the references outside of it, by another
// process or the git binary.
type syncWorker struct {
	// ctx is the lifetime of the subcache: the worker stops when it's done
	ctx    context.Context
	cancel context.CancelFunc
	// sync brings the subcache up to date: every entity if all, or else those
	// ids. A syncEntitiesError has only the entities that failed synced again.
	sync func(all bool, ids []entity.Id) error

	// trigger asks the worker for a sync of what is pending, see request
	trigger chan struct{}
	done    sync.WaitGroup
	// mu guards the state below, and the cancellation of ctx, for the worker
	// not to start once closed. It is only held briefly.
	mu sync.Mutex
	// pendingAll and pendingIds are what the next sync covers: every entity,
	// or those ids.
	pendingAll bool
	pendingIds map[entity.Id]struct{}
	// err is the error of the latest sync, nil if it succeeded.
	err error
}

func newSyncWorker(ctx context.Context, cancel context.CancelFunc, sync func(all bool, ids []entity.Id) error) *syncWorker {
	return &syncWorker{
		ctx:     ctx,
		cancel:  cancel,
		sync:    sync,
		trigger: make(chan struct{}, 1),
	}
}

// request asks for a sync of the given entities, or of every entity if none is
// given. It never waits, and any number of requests collapse into one sync, of
// every entity they named.
func (w *syncWorker) request(ids ...entity.Id) {
	w.mu.Lock()
	w.addPendingLocked(ids...)
	w.mu.Unlock()

	select {
	case w.trigger <- struct{}{}:
	default:
	}
}

// addPendingLocked adds the given entities, or every entity if none is given,
// to what the next sync covers. mu must be held.
func (w *syncWorker) addPendingLocked(ids ...entity.Id) {
	switch {
	case len(ids) == 0:
		w.pendingAll = true
		w.pendingIds = nil
	case !w.pendingAll:
		if w.pendingIds == nil {
			w.pendingIds = make(map[entity.Id]struct{}, len(ids))
		}
		for _, id := range ids {
			w.pendingIds[id] = struct{}{}
		}
	}
}

// start starts the worker, until close. It syncs what request asked for. A
// request received less than syncInterval after the latest sync ended is
// postponed until then, not dropped.
//
// A failed sync is kept, see lastErr. The entities it was for stay pending, and
// are synced again with the next request, or else on their own, after a delay
// doubling with each failure, from syncInterval to syncRetryMaxDelay: with no
// source to report changes, nothing else would.
func (w *syncWorker) start() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.ctx.Err() != nil {
		return
	}

	w.done.Go(func() {
		// started right after the sync of the load
		lastSync := time.Now()
		// postponed fires when a request received too soon after the latest
		// sync can be served, or a failed sync retried, nil if there is none
		var postponed <-chan time.Time
		// retryDelay is the delay before the latest retry, 0 if the latest
		// sync succeeded
		var retryDelay time.Duration

		for {
			select {
			case <-w.ctx.Done():
				return
			case <-postponed:
			case <-w.trigger:
				if wait := syncInterval - time.Since(lastSync); wait > 0 {
					// replaces a retry, due later
					postponed = time.After(wait)
					continue
				}
			}

			// the sync serves any postponed request
			postponed = nil
			w.mu.Lock()
			all := w.pendingAll
			ids := slices.Collect(maps.Keys(w.pendingIds))
			w.pendingAll = false
			w.pendingIds = nil
			w.mu.Unlock()

			if !all && len(ids) == 0 {
				// served by the previous sync
				continue
			}
			err := w.sync(all, ids)
			lastSync = time.Now()

			w.mu.Lock()
			w.err = err
			if err != nil {
				// only the entities that failed, if the others were synced
				var failed syncEntitiesError
				if errors.As(err, &failed) {
					ids = slices.Collect(maps.Keys(failed))
				}
				// no ids for a sync of every entity
				w.addPendingLocked(ids...)
			}
			w.mu.Unlock()

			if err != nil {
				retryDelay = min(max(2*retryDelay, syncInterval), syncRetryMaxDelay)
				postponed = time.After(retryDelay)
			} else {
				retryDelay = 0
			}
		}
	})
}

// lastErr returns the error of the latest sync, nil if it succeeded.
func (w *syncWorker) lastErr() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.err
}

// close cancels ctx, which interrupts a sync in progress and stops the worker,
// and waits for it to end. It won't start again.
func (w *syncWorker) close() {
	w.mu.Lock()
	w.cancel()
	w.mu.Unlock()
	w.done.Wait()
}
