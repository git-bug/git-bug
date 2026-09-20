package cache

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/entities/bug"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/misc/random_bugs"
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
	bug1A, _, err := cacheA.Bugs().New("bug1", "message")
	require.NoError(t, err)

	// A --> remote --> B
	_, err = cacheA.Push("origin")
	require.NoError(t, err)

	err = cacheB.Pull("origin")
	require.NoError(t, err)

	require.Len(t, cacheB.Bugs().AllIds(), 1)

	// a merge doesn't load the bug in memory, as a merge is not a use of it
	require.Empty(t, cacheB.bugs.cached)

	// retrieve and set identity
	reneB, err := cacheB.Identities().Resolve(reneA.Id())
	require.NoError(t, err)

	err = cacheB.SetUserIdentity(reneB)
	require.NoError(t, err)

	// B --> remote --> A
	bug2B, _, err := cacheB.Bugs().New("bug2", "message")
	require.NoError(t, err)

	_, err = cacheB.Push("origin")
	require.NoError(t, err)

	err = cacheA.Pull("origin")
	require.NoError(t, err)

	require.Len(t, cacheA.Bugs().AllIds(), 2)

	// in A, bug1 is loaded as it was created there, bug2 is not
	require.Len(t, cacheA.bugs.cached, 1)

	// B updates both bugs
	for _, id := range []entity.Id{bug1A.Id(), bug2B.Id()} {
		b, err := cacheB.Bugs().Resolve(id)
		require.NoError(t, err)
		_, _, err = b.AddComment("comment")
		require.NoError(t, err)
		require.NoError(t, b.Commit())
	}

	_, err = cacheB.Push("origin")
	require.NoError(t, err)

	err = cacheA.Pull("origin")
	require.NoError(t, err)

	// the loaded bug1 is replaced with the merged version, so that it's not
	// outdated, and bug2 is still not loaded
	require.Len(t, cacheA.bugs.cached, 1)
	require.Equal(t, 1, cacheA.bugs.lru.Len())

	bug1A, err = cacheA.Bugs().Resolve(bug1A.Id())
	require.NoError(t, err)
	require.Len(t, bug1A.Snapshot().Comments, 2)
}

// searchObserver records the events it receives, and whether the entity could
// already be found by a full-text search for term when the event was received.
type searchObserver struct {
	cache  *RepoCache
	term   string
	events []searchEvent
}

type searchEvent struct {
	event EntityEventType
	id    entity.Id
	found bool
}

func (o *searchObserver) EntityEvent(event EntityEventType, _ string, _ string, id entity.Id) {
	// no require here, as this doesn't run in the test goroutine
	found := false
	q, err := query.Parse(o.term)
	if err == nil {
		res, err := o.cache.Bugs().Query(q)
		found = err == nil && slices.Contains(res, id)
	}
	o.events = append(o.events, searchEvent{event: event, id: id, found: found})
}

// watch resets the recorded events, and sets the term to search for.
func (o *searchObserver) watch(term string) {
	o.term = term
	o.events = nil
}

// Entities created or updated by a merge must be up to date in the excerpts and
// the full-text index by the time observers are notified, the same way as
// entities created or changed locally.
func TestCacheMerge(t *testing.T) {
	repoA, repoB, _ := repository.SetupGoGitReposAndRemote(t)

	cacheA := createTestRepoCacheNoEvents(t, repoA)
	cacheB := createTestRepoCacheNoEvents(t, repoB)

	reneA, err := cacheA.Identities().New("René Descartes", "rene@descartes.fr")
	require.NoError(t, err)
	err = cacheA.SetUserIdentity(reneA)
	require.NoError(t, err)

	search := func(t *testing.T, cache *RepoCache, term string) []entity.Id {
		t.Helper()
		q, err := query.Parse(term)
		require.NoError(t, err)
		res, err := cache.Bugs().Query(q)
		require.NoError(t, err)
		return res
	}

	obs := &searchObserver{cache: cacheB}
	require.NoError(t, cacheB.registerObserver("repotest", bug.Typename, obs))

	// A creates a bug holding a unique marker
	bugA, _, err := cacheA.Bugs().New("title", "markercreate")
	require.NoError(t, err)

	_, err = cacheA.Push("origin")
	require.NoError(t, err)

	// a bug merged as new is searchable in B, already when observers are notified
	obs.watch("markercreate")
	err = cacheB.Pull("origin")
	require.NoError(t, err)
	require.Equal(t, []entity.Id{bugA.Id()}, search(t, cacheB, "markercreate"))
	require.Equal(t, []searchEvent{{EntityEventCreated, bugA.Id(), true}}, obs.events)

	// A adds a comment holding a second marker
	_, _, err = bugA.AddComment("markerupdate")
	require.NoError(t, err)
	err = bugA.Commit()
	require.NoError(t, err)

	_, err = cacheA.Push("origin")
	require.NoError(t, err)

	// the new text of a bug merged as updated is searchable in B, already when
	// observers are notified
	obs.watch("markerupdate")
	err = cacheB.Pull("origin")
	require.NoError(t, err)
	require.Equal(t, []entity.Id{bugA.Id()}, search(t, cacheB, "markerupdate"))
	require.Equal(t, []searchEvent{{EntityEventUpdated, bugA.Id(), true}}, obs.events)

	// a merge commit requires a user identity in B
	reneB, err := cacheB.Identities().Resolve(reneA.Id())
	require.NoError(t, err)
	err = cacheB.SetUserIdentity(reneB)
	require.NoError(t, err)

	// A comments and closes the bug while B comments, so B needs a merge commit
	_, _, err = bugA.AddComment("markerremote")
	require.NoError(t, err)
	_, err = bugA.Close()
	require.NoError(t, err)
	err = bugA.Commit()
	require.NoError(t, err)

	_, err = cacheA.Push("origin")
	require.NoError(t, err)

	bugB, err := cacheB.Bugs().Resolve(bugA.Id())
	require.NoError(t, err)
	_, _, err = bugB.AddComment("markerlocal")
	require.NoError(t, err)
	err = bugB.Commit()
	require.NoError(t, err)

	// the text of both sides of a merge commit is searchable in B, already when
	// observers are notified
	obs.watch("markerremote")
	err = cacheB.Pull("origin")
	require.NoError(t, err)
	require.Equal(t, []entity.Id{bugA.Id()}, search(t, cacheB, "markerremote"))
	require.Equal(t, []entity.Id{bugA.Id()}, search(t, cacheB, "markerlocal"))
	require.Equal(t, []searchEvent{{EntityEventUpdated, bugA.Id(), true}}, obs.events)

	// the excerpt holds the merged state as well
	require.Equal(t, []entity.Id{bugA.Id()}, search(t, cacheB, "status:closed"))

	// the merged bug can still be changed in B
	bugB, err = cacheB.Bugs().Resolve(bugA.Id())
	require.NoError(t, err)
	_, _, err = bugB.AddComment("markeraftermerge")
	require.NoError(t, err)
	err = bugB.Commit()
	require.NoError(t, err)

	// the index count matches the excerpts, so the next load won't detect a
	// mismatch and rebuild the cache
	indexCount := func(t *testing.T, name string) uint64 {
		t.Helper()
		idx, err := repoB.GetIndex(name)
		require.NoError(t, err)
		count, err := idx.DocCount()
		require.NoError(t, err)
		return count
	}
	require.Equal(t, uint64(len(cacheB.Bugs().AllIds())), indexCount(t, bug.Namespace))
	require.Equal(t, uint64(len(cacheB.Identities().AllIds())), indexCount(t, identity.Namespace))
}

// A merge failure is reported by Pull, but doesn't stop the other entities from
// being merged.
func TestCachePullAfterFailure(t *testing.T) {
	repoA, repoB, _ := repository.SetupGoGitReposAndRemote(t)

	cacheA := createTestRepoCacheNoEvents(t, repoA)
	cacheB := createTestRepoCacheNoEvents(t, repoB)

	reneA, err := cacheA.Identities().New("René Descartes", "rene@descartes.fr")
	require.NoError(t, err)
	err = cacheA.SetUserIdentity(reneA)
	require.NoError(t, err)

	_, err = cacheA.Push("origin")
	require.NoError(t, err)
	err = cacheB.Pull("origin")
	require.NoError(t, err)

	// the identity diverges, which can't be merged. Identities are merged before
	// bugs, so this failure is the first merge result.
	reneB, err := cacheB.Identities().Resolve(reneA.Id())
	require.NoError(t, err)
	err = reneB.Mutate(repoB, func(m *identity.Mutator) { m.Name = "René B" })
	require.NoError(t, err)
	err = reneB.Commit()
	require.NoError(t, err)

	err = reneA.Mutate(repoA, func(m *identity.Mutator) { m.Name = "René A" })
	require.NoError(t, err)
	err = reneA.Commit()
	require.NoError(t, err)

	// more than one bug, as the merge used to stop right after the first one
	bug1A, _, err := cacheA.Bugs().New("bug1", "message")
	require.NoError(t, err)
	bug2A, _, err := cacheA.Bugs().New("bug2", "message")
	require.NoError(t, err)

	_, err = cacheA.Push("origin")
	require.NoError(t, err)

	err = cacheB.Pull("origin")
	require.ErrorContains(t, err, "merge failure")
	require.ElementsMatch(t, []entity.Id{bug1A.Id(), bug2A.Id()}, cacheB.Bugs().AllIds())
}

// Pulling into a fresh repo must not require a user identity, otherwise it's
// impossible to adopt an identity that only exists on the remote.
// See https://github.com/git-bug/git-bug/issues/1003
func TestCachePullWithoutIdentity(t *testing.T) {
	repoA, repoB, _ := repository.SetupGoGitReposAndRemote(t)

	cacheA := createTestRepoCacheNoEvents(t, repoA)
	cacheB := createTestRepoCacheNoEvents(t, repoB)

	reneA, err := cacheA.Identities().New("René Descartes", "rene@descartes.fr")
	require.NoError(t, err)
	err = cacheA.SetUserIdentity(reneA)
	require.NoError(t, err)

	_, _, err = cacheA.Bugs().New("bug1", "message")
	require.NoError(t, err)

	_, err = cacheA.Push("origin")
	require.NoError(t, err)

	// B has no identity set
	_, err = cacheB.GetUserIdentity()
	require.ErrorIs(t, err, identity.ErrNoIdentitySet)

	err = cacheB.Pull("origin")
	require.NoError(t, err)

	require.Len(t, cacheB.Identities().AllIds(), 1)
	require.Len(t, cacheB.Bugs().AllIds(), 1)

	// adopt the pulled identity
	reneB, err := cacheB.Identities().Resolve(reneA.Id())
	require.NoError(t, err)
	err = cacheB.SetUserIdentity(reneB)
	require.NoError(t, err)

	userB, err := cacheB.GetUserIdentity()
	require.NoError(t, err)
	require.Equal(t, reneA.Id(), userB.Id())

	// concurrent edition of the bug on both sides
	bugA, err := cacheA.Bugs().ResolvePrefix("")
	require.NoError(t, err)
	_, _, err = bugA.AddComment("from A")
	require.NoError(t, err)
	require.NoError(t, bugA.Commit())
	_, err = cacheA.Push("origin")
	require.NoError(t, err)

	bugB, err := cacheB.Bugs().Resolve(bugA.Id())
	require.NoError(t, err)
	_, _, err = bugB.AddComment("from B")
	require.NoError(t, err)
	require.NoError(t, bugB.Commit())

	// a merge commit is required, which needs an identity
	err = cacheB.ClearUserIdentity()
	require.NoError(t, err)

	_, err = cacheB.Fetch("origin")
	require.NoError(t, err)
	var mergeErrs []error
	for result := range cacheB.MergeAll("origin") {
		if result.Err != nil {
			mergeErrs = append(mergeErrs, result.Err)
		}
	}
	require.Len(t, mergeErrs, 1)
	require.ErrorIs(t, mergeErrs[0], identity.ErrNoIdentitySet)
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

func TestBuildConcurrentResolve(t *testing.T) {
	// See https://github.com/git-bug/git-bug/issues/1226
	// Bugs and identities subcaches are built concurrently, and reading bugs
	// resolves their authors through the identities subcache.

	repo := repository.CreateGoGitTestRepo(t, false)
	random_bugs.FillRepoWithSeed(repo, 20, 42)

	repoCache := createTestRepoCacheNoEvents(t, repo)

	for _, id := range repoCache.Bugs().AllIds() {
		b, err := repoCache.Bugs().Resolve(id)
		require.NoError(t, err)

		author := b.Snapshot().Author
		cached, err := repoCache.Identities().Resolve(author.Id())
		require.NoError(t, err)

		// a single copy of each identity should exist in memory
		require.Same(t, cached, author)
	}
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

func TestResolveOperationWithMetadataFromSetMetadata(t *testing.T) {
	// See https://github.com/git-bug/git-bug/issues/1582
	// Bridges mark exported operations with a SetMetadata operation. That
	// metadata must be found on a freshly loaded entity, before anything
	// compiled its snapshot.

	repo := repository.CreateGoGitTestRepo(t, false)

	backend, err := NewRepoCacheNoEvents(repo)
	require.NoError(t, err)

	i, err := backend.Identities().New("René Descartes", "rene@descartes.fr")
	require.NoError(t, err)
	require.NoError(t, backend.SetUserIdentity(i))

	b, _, err := backend.Bugs().New("title", "message")
	require.NoError(t, err)
	_, commentOp, err := b.AddComment("comment")
	require.NoError(t, err)
	require.NoError(t, b.Commit())

	_, err = b.SetMetadata(commentOp.Id(), map[string]string{"key": "value"})
	require.NoError(t, err)
	require.NoError(t, b.Commit())

	require.NoError(t, backend.Close())

	// reopen the cache, as a separate process would
	backend = createTestRepoCacheNoEvents(t, repo)

	b, err = backend.Bugs().Resolve(b.Id())
	require.NoError(t, err)

	opId, err := b.ResolveOperationWithMetadata("key", "value")
	require.NoError(t, err)
	require.Equal(t, commentOp.Id(), opId)
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
