package cache

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/entities/bug"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/internal/test"
	"github.com/git-bug/git-bug/query"
	"github.com/git-bug/git-bug/repository"
)

type observerEvent struct {
	typename string
	id       entity.Id
}

var _ Observer = &observer{}

type observer struct {
	created []observerEvent
	updated []observerEvent
	removed []observerEvent
}

func (o *observer) EntityEvent(event EntityEventType, _ string, typename string, id entity.Id) {
	switch event {
	case EntityEventCreated:
		o.created = append(o.created, observerEvent{typename, id})
	case EntityEventUpdated:
		o.updated = append(o.updated, observerEvent{typename, id})
	case EntityEventRemoved:
		o.removed = append(o.removed, observerEvent{typename, id})
	}
}

func TestCache(t *testing.T) {
	f := test.NewFlaky(t, &test.FlakyOptions{
		MaxAttempts: 5,
	})

	f.Run(func(t testing.TB) {
		repo := repository.CreateGoGitTestRepo(t, false)

		indexCount := func(t testing.TB, name string) uint64 {
			t.Helper()
			idx, err := repo.GetIndex(name)
			require.NoError(t, err)
			count, err := idx.DocCount()
			require.NoError(t, err)
			return count
		}
		assertOberserverEvent := func(obs observer, created, updated, removed int) {
			t.Helper()
			require.Len(t, obs.created, created)
			require.Len(t, obs.updated, updated)
			require.Len(t, obs.removed, removed)
		}

		cache, err := NewRepoCacheNoEvents(repo)
		require.NoError(t, err)

		var obsIdentity, obsBug observer
		require.NoError(t, cache.registerObserver("repotest", identity.Typename, &obsIdentity))
		require.NoError(t, cache.registerObserver("repotest", bug.Typename, &obsBug))

		// Create, set and get user identity
		iden1, err := cache.Identities().New("René Descartes", "rene@descartes.fr")
		require.NoError(t, err)
		assertOberserverEvent(obsIdentity, 1, 0, 0)
		assertOberserverEvent(obsBug, 0, 0, 0)
		err = cache.SetUserIdentity(iden1)
		require.NoError(t, err)
		userIden, err := cache.GetUserIdentity()
		require.NoError(t, err)
		require.Equal(t, iden1.Id(), userIden.Id())

		// it's possible to create two identical identities
		iden2, err := cache.Identities().New("René Descartes", "rene@descartes.fr")
		require.NoError(t, err)
		assertOberserverEvent(obsIdentity, 2, 0, 0)
		assertOberserverEvent(obsBug, 0, 0, 0)

		// Two identical identities yield a different id
		require.NotEqual(t, iden1.Id(), iden2.Id())

		// There are now two identities in the cache
		require.Len(t, cache.Identities().AllIds(), 2)
		require.Len(t, cache.identities.excerpts, 2)
		require.Len(t, cache.identities.cached, 2)
		require.Equal(t, uint64(2), indexCount(t, identity.Namespace))
		require.Equal(t, uint64(0), indexCount(t, bug.Namespace))

		// Create a bug
		bug1, _, err := cache.Bugs().New("title", "message")
		require.NoError(t, err)
		assertOberserverEvent(obsIdentity, 2, 0, 0)
		assertOberserverEvent(obsBug, 1, 0, 0)

		// It's possible to create two identical bugs
		bug2, _, err := cache.Bugs().New("title", "marker")
		require.NoError(t, err)
		assertOberserverEvent(obsIdentity, 2, 0, 0)
		assertOberserverEvent(obsBug, 2, 0, 0)

		// two identical bugs yield a different id
		require.NotEqual(t, bug1.Id(), bug2.Id())

		// There is now two bugs in the cache
		require.Len(t, cache.Bugs().AllIds(), 2)
		require.Len(t, cache.bugs.excerpts, 2)
		require.Len(t, cache.bugs.cached, 2)
		require.Equal(t, uint64(2), indexCount(t, identity.Namespace))
		require.Equal(t, uint64(2), indexCount(t, bug.Namespace))

		// Resolving
		_, err = cache.Identities().Resolve(iden1.Id())
		require.NoError(t, err)
		_, err = cache.Identities().ResolveExcerpt(iden1.Id())
		require.NoError(t, err)
		_, err = cache.Identities().ResolvePrefix(iden1.Id().String()[:10])
		require.NoError(t, err)

		_, err = cache.Bugs().Resolve(bug1.Id())
		require.NoError(t, err)
		_, err = cache.Bugs().ResolveExcerpt(bug1.Id())
		require.NoError(t, err)
		_, err = cache.Bugs().ResolvePrefix(bug1.Id().String()[:10])
		require.NoError(t, err)

		// Querying
		q, err := query.Parse("status:open author:descartes sort:edit-asc")
		require.NoError(t, err)
		res, err := cache.Bugs().Query(q)
		require.NoError(t, err)
		require.Len(t, res, 2)

		q, err = query.Parse("status:open marker") // full-text search
		require.NoError(t, err)
		res, err = cache.Bugs().Query(q)
		require.NoError(t, err)
		require.Len(t, res, 1)

		// Updating
		_, _, err = bug1.AddComment("new comment")
		require.NoError(t, err)
		assertOberserverEvent(obsIdentity, 2, 0, 0)
		assertOberserverEvent(obsBug, 2, 1, 0)

		// Close
		require.NoError(t, cache.Close())
		require.Empty(t, cache.bugs.cached)
		require.Empty(t, cache.bugs.excerpts)
		require.Empty(t, cache.identities.cached)
		require.Empty(t, cache.identities.excerpts)

		// Reload, only excerpt are loaded, but as we need to load the identities used in the bugs
		// to check the signatures, we also load the identity used above
		cache, err = NewRepoCacheNoEvents(repo)
		require.NoError(t, err)
		require.NoError(t, cache.registerObserver("repotest", identity.Typename, &obsIdentity))
		require.NoError(t, cache.registerObserver("repotest", bug.Typename, &obsBug))

		require.Len(t, cache.bugs.cached, 0)
		require.Len(t, cache.bugs.excerpts, 2)
		require.Len(t, cache.identities.cached, 0)
		require.Len(t, cache.identities.excerpts, 2)
		require.Equal(t, uint64(2), indexCount(t, identity.Namespace))
		require.Equal(t, uint64(2), indexCount(t, bug.Namespace))

		// Resolving load from the disk
		_, err = cache.Identities().Resolve(iden1.Id())
		require.NoError(t, err)
		_, err = cache.Identities().ResolveExcerpt(iden1.Id())
		require.NoError(t, err)
		_, err = cache.Identities().ResolvePrefix(iden1.Id().String()[:10])
		require.NoError(t, err)

		_, err = cache.Bugs().Resolve(bug1.Id())
		require.NoError(t, err)
		_, err = cache.Bugs().ResolveExcerpt(bug1.Id())
		require.NoError(t, err)
		_, err = cache.Bugs().ResolvePrefix(bug1.Id().String()[:10])
		require.NoError(t, err)

		require.Len(t, cache.bugs.cached, 1)
		require.Len(t, cache.bugs.excerpts, 2)
		require.Len(t, cache.identities.cached, 1)
		require.Len(t, cache.identities.excerpts, 2)
		require.Equal(t, uint64(2), indexCount(t, identity.Namespace))
		require.Equal(t, uint64(2), indexCount(t, bug.Namespace))

		// Remove + RemoveAll
		err = cache.Identities().Remove(iden1.Id().String()[:10])
		require.NoError(t, err)
		assertOberserverEvent(obsIdentity, 2, 0, 1)
		assertOberserverEvent(obsBug, 2, 1, 0)
		err = cache.Bugs().Remove(bug1.Id().String()[:10])
		require.NoError(t, err)
		assertOberserverEvent(obsIdentity, 2, 0, 1)
		assertOberserverEvent(obsBug, 2, 1, 1)
		require.Len(t, cache.bugs.cached, 0)
		require.Len(t, cache.bugs.excerpts, 1)
		require.Len(t, cache.identities.cached, 0)
		require.Len(t, cache.identities.excerpts, 1)
		require.Equal(t, uint64(1), indexCount(t, identity.Namespace))
		require.Equal(t, uint64(1), indexCount(t, bug.Namespace))

		_, err = cache.Identities().New("René Descartes", "rene@descartes.fr")
		require.NoError(t, err)
		assertOberserverEvent(obsIdentity, 3, 0, 1)
		assertOberserverEvent(obsBug, 2, 1, 1)
		_, _, err = cache.Bugs().NewRaw(iden2, time.Now().Unix(), "title", "message", nil, nil)
		require.NoError(t, err)
		assertOberserverEvent(obsIdentity, 3, 0, 1)
		assertOberserverEvent(obsBug, 3, 1, 1)

		err = cache.RemoveAll()
		require.NoError(t, err)
		assertOberserverEvent(obsIdentity, 3, 0, 3)
		assertOberserverEvent(obsBug, 3, 1, 3)
		require.Len(t, cache.bugs.cached, 0)
		require.Len(t, cache.bugs.excerpts, 0)
		require.Len(t, cache.identities.cached, 0)
		require.Len(t, cache.identities.excerpts, 0)
		require.Equal(t, uint64(0), indexCount(t, identity.Namespace))
		require.Equal(t, uint64(0), indexCount(t, bug.Namespace))

		// Close
		require.NoError(t, cache.Close())
		require.Empty(t, cache.bugs.cached)
		require.Empty(t, cache.bugs.excerpts)
		require.Empty(t, cache.identities.cached)
		require.Empty(t, cache.identities.excerpts)
	})
}

func TestCachePushPull(t *testing.T) {
	repoA, repoB, _ := repository.SetupGoGitReposAndRemote(t)

	cacheA := createTestRepoCacheNoEvents(t, repoA)
	cacheB := createTestRepoCacheNoEvents(t, repoB)

	// Create, set and get user identity
	reneA, err := cacheA.Identities().New("René Descartes", "rene@descartes.fr")
	require.NoError(t, err)
	err = cacheA.SetUserIdentity(reneA)
	require.NoError(t, err)
	isaacB, err := cacheB.Identities().New("Isaac Newton", "isaac@newton.uk")
	require.NoError(t, err)
	err = cacheB.SetUserIdentity(isaacB)
	require.NoError(t, err)

	// distribute the identity
	_, err = cacheA.Push("origin")
	require.NoError(t, err)
	err = cacheB.Pull("origin")
	require.NoError(t, err)

	// Create a bug in A
	_, _, err = cacheA.Bugs().New("bug1", "message")
	require.NoError(t, err)

	// A --> remote --> B
	_, err = cacheA.Push("origin")
	require.NoError(t, err)

	err = cacheB.Pull("origin")
	require.NoError(t, err)

	require.Len(t, cacheB.Bugs().AllIds(), 1)

	// retrieve and set identity
	reneB, err := cacheB.Identities().Resolve(reneA.Id())
	require.NoError(t, err)

	err = cacheB.SetUserIdentity(reneB)
	require.NoError(t, err)

	// B --> remote --> A
	_, _, err = cacheB.Bugs().New("bug2", "message")
	require.NoError(t, err)

	_, err = cacheB.Push("origin")
	require.NoError(t, err)

	err = cacheA.Pull("origin")
	require.NoError(t, err)

	require.Len(t, cacheA.Bugs().AllIds(), 2)
}

func TestRemove(t *testing.T) {
	repo := repository.CreateGoGitTestRepo(t, false)
	remoteA := repository.CreateGoGitTestRepo(t, true)
	remoteB := repository.CreateGoGitTestRepo(t, true)

	err := repo.AddRemote("remoteA", remoteA.GetLocalRemote())
	require.NoError(t, err)

	err = repo.AddRemote("remoteB", remoteB.GetLocalRemote())
	require.NoError(t, err)

	repoCache := createTestRepoCacheNoEvents(t, repo)

	rene, err := repoCache.Identities().New("René Descartes", "rene@descartes.fr")
	require.NoError(t, err)

	err = repoCache.SetUserIdentity(rene)
	require.NoError(t, err)

	_, _, err = repoCache.Bugs().New("title", "message")
	require.NoError(t, err)

	// and one more for testing
	b1, _, err := repoCache.Bugs().New("title", "message")
	require.NoError(t, err)

	_, err = repoCache.Push("remoteA")
	require.NoError(t, err)

	_, err = repoCache.Push("remoteB")
	require.NoError(t, err)

	_, err = repoCache.Fetch("remoteA")
	require.NoError(t, err)

	_, err = repoCache.Fetch("remoteB")
	require.NoError(t, err)

	err = repoCache.Bugs().Remove(b1.Id().String())
	require.NoError(t, err)
	assert.Len(t, repoCache.bugs.cached, 1)
	assert.Len(t, repoCache.bugs.excerpts, 1)

	_, err = repoCache.Bugs().Resolve(b1.Id())
	assert.ErrorAs(t, entity.ErrNotFound{}, err)
}

func TestCacheEviction(t *testing.T) {
	repo := repository.CreateGoGitTestRepo(t, false)
	repoCache := createTestRepoCacheNoEvents(t, repo)
	repoCache.setCacheSize(2)

	require.Equal(t, 2, repoCache.bugs.maxLoaded)
	require.Len(t, repoCache.bugs.cached, 0)
	require.Equal(t, repoCache.bugs.lru.Len(), 0)

	// Generating some bugs
	rene, err := repoCache.Identities().New("René Descartes", "rene@descartes.fr")
	require.NoError(t, err)
	err = repoCache.SetUserIdentity(rene)
	require.NoError(t, err)

	bug1, _, err := repoCache.Bugs().New("title", "message")
	require.NoError(t, err)

	checkBugPresence(t, repoCache, bug1, true)
	require.Len(t, repoCache.bugs.cached, 1)
	require.Equal(t, 1, repoCache.bugs.lru.Len())

	bug2, _, err := repoCache.Bugs().New("title", "message")
	require.NoError(t, err)

	checkBugPresence(t, repoCache, bug1, true)
	checkBugPresence(t, repoCache, bug2, true)
	require.Len(t, repoCache.bugs.cached, 2)
	require.Equal(t, 2, repoCache.bugs.lru.Len())

	// Number of bugs should not exceed max size of lruCache, oldest one should be evicted
	bug3, _, err := repoCache.Bugs().New("title", "message")
	require.NoError(t, err)

	require.Len(t, repoCache.bugs.cached, 2)
	require.Equal(t, 2, repoCache.bugs.lru.Len())
	checkBugPresence(t, repoCache, bug1, false)
	checkBugPresence(t, repoCache, bug2, true)
	checkBugPresence(t, repoCache, bug3, true)

	// Accessing bug should update position in lruCache, and therefore it should not be evicted
	repoCache.bugs.lru.Get(bug2.Id())
	oldestId, _ := repoCache.bugs.lru.GetOldest()
	require.Equal(t, bug3.Id(), oldestId)

	checkBugPresence(t, repoCache, bug1, false)
	checkBugPresence(t, repoCache, bug2, true)
	checkBugPresence(t, repoCache, bug3, true)
	require.Len(t, repoCache.bugs.cached, 2)
	require.Equal(t, 2, repoCache.bugs.lru.Len())
}

func TestLongDescription(t *testing.T) {
	// See https://github.com/git-bug/git-bug/issues/606

	text := strings.Repeat("x", 65536)

	repo := repository.CreateGoGitTestRepo(t, false)

	backend := createTestRepoCacheNoEvents(t, repo)

	i, err := backend.Identities().New("René Descartes", "rene@descartes.fr")
	require.NoError(t, err)

	_, _, err = backend.Bugs().NewRaw(i, time.Now().Unix(), text, text, nil, nil)
	require.NoError(t, err)
}

func checkBugPresence(t *testing.T, cache *RepoCache, bug *BugCache, presence bool) {
	t.Helper()

	id := bug.Id()
	require.Equal(t, presence, cache.bugs.lru.Contains(id))
	b, ok := cache.bugs.cached[id]
	require.Equal(t, presence, ok)
	if ok {
		require.Equal(t, bug, b)
	}
}

func createTestRepoCacheNoEvents(t *testing.T, repo repository.TestedRepo) *RepoCache {
	t.Helper()

	cache, err := NewRepoCacheNoEvents(repo)
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, cache.Close())
	})

	return cache
}
