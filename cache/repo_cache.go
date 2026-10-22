package cache

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/git-bug/git-bug/entities/bug"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/repository"
	"github.com/git-bug/git-bug/util/multierr"
)

// 1: original format
// 2: added cache for identities with a reference in the bug cache
// 3: no more legacy identity
// 4: entities make their IDs from data, not git commit
// 5: record the commit each excerpt was built from
// 6: excerpts in JSON, in a SQLite DB
// TODO: if entities evolve independently, consider make that versioning per entity
const formatVersion = 6

// The maximum number of bugs loaded in memory. After that, eviction will be done.
const defaultMaxLoadedBugs = 1000

var _ repository.RepoConfig = &RepoCache{}
var _ repository.RepoKeyring = &RepoCache{}

// cacheMgmt is the expected interface for a sub-cache.
type cacheMgmt interface {
	Typename() string
	EnsureClocks() error
	// Load reads the cache, and brings it up to date with the repository.
	Load() <-chan BuildEvent
	SetCacheSize(size int)
	RemoveAll() error
	MergeAll(ctx context.Context, remote string) <-chan entity.MergeResult
	GetNamespace() string
	RegisterObserver(repoName string, observer Observer)
	UnregisterObserver(observer Observer)
	// requestSync asks for a sync of the given entities, or of every entity
	// if none is given, in the background.
	requestSync(ids ...entity.Id)
	Close() error
}

// RepoCache is a cache for a Repository. This cache has multiple functions:
//
//  1. After being loaded, a Bug is kept in memory in the cache, allowing for fast
//     access later.
//  2. The cache maintains in memory and on disk a pre-digested excerpt for each bug,
//     allowing for fast querying the whole set of bugs without having to load
//     them individually.
//  3. The cache keeps a single copy of each loaded entity when possible. A copy
//     that got evicted or went stale keeps working: its commits never overwrite
//     newer data, as the copy is reloaded and the operations written again on top.
type RepoCache struct {
	// the underlying repo
	repo repository.Repo

	// resolvers for all known entities and excerpts
	resolvers entity.Resolvers

	bugs       *RepoCacheBug
	identities *RepoCacheIdentity

	subcaches []cacheMgmt

	// the user identity's id, if known
	muUserIdentity sync.RWMutex
	userIdentityId entity.Id

	// changes, if not nil, is what tells the cache that the repository may
	// have changed outside of it, see followChanges.
	changes repository.ChangeSource
	// muChanges guards the state of the subscription below.
	muChanges           sync.Mutex
	changesClosed       bool
	stopChanges         context.CancelFunc
	changesFollowerDone sync.WaitGroup
}

// NewRepoCache create or open a cache on top of a raw repository.
// The caller is expected to read all returned events before the cache is considered
// ready to use.
//
// The cache follows the changes made to the repository outside of it, by
// another process or the git binary, as reported by changes. With a nil
// source, it only knows of the changes made through it, and of those made
// outside before it was loaded: a one-shot command needs nothing more, but a
// long-running process does.
func NewRepoCache(r repository.Repo, changes repository.ChangeSource) (*RepoCache, chan BuildEvent) {
	c := &RepoCache{
		repo:    r,
		changes: changes,
	}

	c.identities = NewRepoCacheIdentity(r, c.getResolvers, c.GetUserIdentity)
	c.subcaches = append(c.subcaches, c.identities)

	c.bugs = NewRepoCacheBug(r, c.getResolvers, c.GetUserIdentity)
	c.subcaches = append(c.subcaches, c.bugs)

	c.resolvers = entity.Resolvers{
		&IdentityCache{}: entity.ResolverFunc[*IdentityCache](c.identities.Resolve),
		&IdentityExcerpt{}: entity.ResolverFunc[*IdentityExcerpt](func(_ context.Context, id entity.Id) (*IdentityExcerpt, error) {
			return c.identities.ResolveExcerpt(id)
		}),
		&BugCache{}: entity.ResolverFunc[*BugCache](c.bugs.Resolve),
		&BugExcerpt{}: entity.ResolverFunc[*BugExcerpt](func(_ context.Context, id entity.Id) (*BugExcerpt, error) {
			return c.bugs.ResolveExcerpt(id)
		}),
	}

	// small buffer so that the functions below can emit an event without blocking
	events := make(chan BuildEvent)

	go func() {
		defer close(events)

		// The files of earlier versions, replaced by the index: the excerpts, and
		// the bleve indexes. Nothing is lost, the load fills the index from the
		// repository.
		// TODO: remove a few releases after this one, once the users upgraded.
		_ = c.repo.LocalStorage().RemoveAll("cache")
		_ = c.repo.LocalStorage().RemoveAll("indexes")

		// Reading entities doesn't create their clocks, and an identity version
		// records every clock: make sure they all exist before anything is written.
		for _, subcache := range c.subcaches {
			err := subcache.EnsureClocks()
			if err != nil {
				events <- BuildEvent{Err: err}
				return
			}
		}

		// Subscribed before loading, so that a change made while loading isn't
		// missed: the syncs it requests wait for the sync workers, started at
		// the end of the load.
		c.followChanges(events)
		if !c.load(events) {
			c.stopFollowingChanges()
		}
	}()

	return c, events
}

func NewRepoCacheNoEvents(r repository.Repo, changes repository.ChangeSource) (*RepoCache, error) {
	cache, events := NewRepoCache(r, changes)
	for event := range events {
		if event.Err != nil {
			for range events {
			}
			return nil, event.Err
		}
	}
	return cache, nil
}

// Bugs gives access to the Bug entities
func (c *RepoCache) Bugs() *RepoCacheBug {
	return c.bugs
}

// Identities gives access to the Identity entities
func (c *RepoCache) Identities() *RepoCacheIdentity {
	return c.identities
}

func (c *RepoCache) getResolvers() entity.Resolvers {
	return c.resolvers
}

// setCacheSize change the maximum number of loaded bugs
func (c *RepoCache) setCacheSize(size int) {
	for _, subcache := range c.subcaches {
		subcache.SetCacheSize(size)
	}
}

// load reads the excerpts of the index, and brings them up to date with the
// repository.
// It returns whether it succeeded.
func (c *RepoCache) load(events chan BuildEvent) bool {
	// announced once, before the first subcache starts building
	var announce sync.Once
	var failed atomic.Bool

	var wg sync.WaitGroup
	for _, subcache := range c.subcaches {
		wg.Add(1)
		go func(subcache cacheMgmt) {
			defer wg.Done()
			for event := range subcache.Load() {
				if event.Event == BuildEventStarted {
					announce.Do(func() { events <- BuildEvent{Event: BuildEventCacheIsBuilt} })
				}
				if event.Err != nil {
					failed.Store(true)
				}
				events <- event
			}
		}(subcache)
	}
	wg.Wait()

	return !failed.Load()
}

// followChanges subscribes to the change source, if any, until
// stopFollowingChanges: each change it reports has the subcaches sync the
// entities it names, or every entity if it names none. A source that can't work
// here is reported as a warning: the cache works without it, only knowing less
// of the changes made outside.
func (c *RepoCache) followChanges(events chan BuildEvent) {
	if c.changes == nil {
		return
	}

	namespaces := make([]string, len(c.subcaches))
	for i, subcache := range c.subcaches {
		namespaces[i] = subcache.GetNamespace()
	}

	c.muChanges.Lock()
	if c.changesClosed {
		c.muChanges.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	changes, err := c.changes.Subscribe(ctx, namespaces)
	if changes == nil {
		cancel()
	} else {
		c.stopChanges = cancel
		c.changesFollowerDone.Go(func() {
			for change := range changes {
				if len(change.Keys) == 0 {
					for _, subcache := range c.subcaches {
						subcache.requestSync()
					}
					continue
				}
				for _, subcache := range c.subcaches {
					keys := change.Keys[subcache.GetNamespace()]
					ids := make([]entity.Id, 0, len(keys))
					for _, key := range keys {
						// not a ref of an entity: nothing to sync
						if id := entity.Id(key); id.Validate() == nil {
							ids = append(ids, id)
						}
					}
					if len(ids) > 0 {
						subcache.requestSync(ids...)
					}
				}
			}
		})
	}
	c.muChanges.Unlock()

	if err != nil {
		events <- BuildEvent{Event: BuildEventWarning, Warning: fmt.Errorf("following changes made outside: %w", err)}
	}
}

// stopFollowingChanges ends the subscription to the change source, and waits
// for the changes it reported to be passed on. It won't start again.
func (c *RepoCache) stopFollowingChanges() {
	c.muChanges.Lock()
	c.changesClosed = true
	if c.stopChanges != nil {
		c.stopChanges()
	}
	c.muChanges.Unlock()
	c.changesFollowerDone.Wait()
}

func (c *RepoCache) Close() error {
	c.stopFollowingChanges()

	var errWait multierr.ErrWaitGroup
	for _, mgmt := range c.subcaches {
		errWait.Go(mgmt.Close)
	}
	err := errWait.Wait()
	if err != nil {
		return err
	}

	return c.repo.Close()
}

func (c *RepoCache) registerObserver(repoName string, typename string, observer Observer) error {
	switch typename {
	case bug.Typename:
		c.bugs.RegisterObserver(repoName, observer)
	case identity.Typename:
		c.identities.RegisterObserver(repoName, observer)
	default:
		var allTypenames []string
		for _, subcache := range c.subcaches {
			allTypenames = append(allTypenames, subcache.Typename())
		}
		return fmt.Errorf("unknown typename `%s`, available types are [%s]", typename, strings.Join(allTypenames, ", "))
	}
	return nil
}

func (c *RepoCache) registerAllObservers(repoName string, observer Observer) {
	for _, subcache := range c.subcaches {
		subcache.RegisterObserver(repoName, observer)
	}
}

func (c *RepoCache) unregisterAllObservers(observer Observer) {
	for _, subcache := range c.subcaches {
		subcache.UnregisterObserver(observer)
	}
}
