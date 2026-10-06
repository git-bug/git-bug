package cache

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/git-bug/git-bug/entities/bug"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/repository"
	"github.com/git-bug/git-bug/util/multierr"
	"github.com/git-bug/git-bug/util/process"
)

// 1: original format
// 2: added cache for identities with a reference in the bug cache
// 3: no more legacy identity
// 4: entities make their IDs from data, not git commit
// 5: record the commit each excerpt was built from
const formatVersion = 5

// The maximum number of bugs loaded in memory. After that, eviction will be done.
const defaultMaxLoadedBugs = 1000

var _ repository.RepoCommon = &RepoCache{}
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
	MergeAll(remote string) <-chan entity.MergeResult
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
//     that got evicted keeps working, and if it goes stale, its commits are
//     rejected by the repository rather than overwriting newer data.
//
// The cache also protects the on-disk data by locking the git repository for its
// own usage, by writing a lock file. Of course, normal git operations are not
// affected, only git-bug related one.
type RepoCache struct {
	// the underlying repo
	repo repository.ClockedRepo

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
func NewRepoCache(r repository.ClockedRepo, changes repository.ChangeSource) (*RepoCache, chan BuildEvent) {
	c := &RepoCache{
		repo:    r,
		changes: changes,
	}

	c.identities = NewRepoCacheIdentity(r, c.getResolvers, c.GetUserIdentity)
	c.subcaches = append(c.subcaches, c.identities)

	c.bugs = NewRepoCacheBug(r, c.getResolvers, c.GetUserIdentity)
	c.subcaches = append(c.subcaches, c.bugs)

	c.resolvers = entity.Resolvers{
		&IdentityCache{}:   entity.ResolverFunc[*IdentityCache](c.identities.Resolve),
		&IdentityExcerpt{}: entity.ResolverFunc[*IdentityExcerpt](c.identities.ResolveExcerpt),
		&BugCache{}:        entity.ResolverFunc[*BugCache](c.bugs.Resolve),
		&BugExcerpt{}:      entity.ResolverFunc[*BugExcerpt](c.bugs.ResolveExcerpt),
	}

	// small buffer so that the functions below can emit an event without blocking
	events := make(chan BuildEvent)

	go func() {
		defer close(events)

		err := c.lock(events)
		if err != nil {
			events <- BuildEvent{Err: err}
			return
		}

		// Reading entities doesn't create their clocks, and an identity version
		// records every clock: make sure they all exist before anything is written.
		for _, subcache := range c.subcaches {
			err = subcache.EnsureClocks()
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

func NewRepoCacheNoEvents(r repository.ClockedRepo, changes repository.ChangeSource) (*RepoCache, error) {
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

// load reads the cache files, and brings them up to date with the repository.
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

func (c *RepoCache) lock(events chan BuildEvent) error {
	err := repoIsAvailable(c.repo, events)
	if err != nil {
		return err
	}

	f, err := c.repo.LocalStorage().Create(lockfile)
	if err != nil {
		return err
	}

	pid := fmt.Sprintf("%d", os.Getpid())
	_, err = f.Write([]byte(pid))
	if err != nil {
		_ = f.Close()
		return err
	}

	return f.Close()
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

	err = c.repo.Close()
	if err != nil {
		return err
	}

	return c.repo.LocalStorage().Remove(lockfile)
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

// repoIsAvailable check is the given repository is locked by a Cache.
// Note: this is a smart function that will clean the lock file if the
// corresponding process is not there anymore.
// If no error is returned, the repo is free to edit.
func repoIsAvailable(repo repository.RepoStorage, events chan BuildEvent) error {
	// Todo: this leave way for a racey access to the repo between the test
	// if the file exist and the actual write. It's probably not a problem in
	// practice because using a repository will be done from user interaction
	// or in a context where a single instance of git-bug is already guaranteed
	// (say, a server with the web UI running). But still, that might be nice to
	// have a mutex or something to guard that.

	// Todo: this will fail if somehow the filesystem is shared with another
	// computer. Should add a configuration that prevent the cleaning of the
	// lock file

	f, err := repo.LocalStorage().Open(lockfile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if err == nil {
		// lock file already exist
		buf, err := io.ReadAll(io.LimitReader(f, 10))
		if err != nil {
			_ = f.Close()
			return err
		}

		err = f.Close()
		if err != nil {
			return err
		}

		if len(buf) >= 10 {
			return fmt.Errorf("the lock file should be < 10 bytes")
		}

		pid, err := strconv.Atoi(string(buf))
		if err != nil {
			return err
		}

		if process.IsRunning(pid) {
			return fmt.Errorf("the repository you want to access is already locked by the process pid %d", pid)
		}

		// The lock file is just laying there after a crash, clean it

		events <- BuildEvent{Event: BuildEventRemoveLock}

		err = repo.LocalStorage().Remove(lockfile)
		if err != nil {
			return err
		}
	}

	return nil
}
