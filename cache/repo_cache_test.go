package cache

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/query"
	"github.com/MichaelMure/git-bug/repository"
)

func TestCache(t *testing.T) {
	repo := repository.CreateGoGitTestRepo(false)
	defer repository.CleanupTestRepos(repo)

	cache, err := NewRepoCache(repo)
	require.NoError(t, err)

	// Create, set and get user identity
	iden1, err := cache.NewIdentity("René Descartes", "rene@descartes.fr")
	require.NoError(t, err)
	err = cache.SetUserIdentity(iden1)
	require.NoError(t, err)
	userIden, err := cache.GetUserIdentity()
	require.NoError(t, err)
	require.Equal(t, iden1.Id(), userIden.Id())

	// it's possible to create two identical identities
	iden2, err := cache.NewIdentity("René Descartes", "rene@descartes.fr")
	require.NoError(t, err)

	// Two identical identities yield a different id
	require.NotEqual(t, iden1.Id(), iden2.Id())

	// There is now two identities in the cache
	require.Len(t, cache.AllIdentityIds(), 2)
	require.Len(t, cache.identitiesExcerpts, 2)
	require.Len(t, cache.identities, 2)

	// Create a bug
	bug1, _, err := cache.NewBug("title", "message")
	require.NoError(t, err)

	// It's possible to create two identical bugs
	bug2, _, err := cache.NewBug("title", "message")
	require.NoError(t, err)

	// two identical bugs yield a different id
	require.NotEqual(t, bug1.Id(), bug2.Id())

	// There is now two bugs in the cache
	require.Len(t, cache.AllBugsIds(), 2)
	require.Len(t, cache.bugExcerpts, 2)
	require.Len(t, cache.bugs, 2)

	// Resolving
	_, err = cache.ResolveIdentity(iden1.Id())
	require.NoError(t, err)
	_, err = cache.ResolveIdentityExcerpt(iden1.Id())
	require.NoError(t, err)
	_, err = cache.ResolveIdentityPrefix(iden1.Id().String()[:10])
	require.NoError(t, err)

	_, err = cache.ResolveBug(bug1.Id())
	require.NoError(t, err)
	_, err = cache.ResolveBugExcerpt(bug1.Id())
	require.NoError(t, err)
	_, err = cache.ResolveBugPrefix(bug1.Id().String()[:10])
	require.NoError(t, err)

	// Querying
	q, err := query.Parse("status:open author:descartes sort:edit-asc")
	require.NoError(t, err)
	require.Len(t, cache.QueryBugs(q), 2)

	// Close
	require.NoError(t, cache.Close())
	require.Empty(t, cache.bugs)
	require.Empty(t, cache.bugExcerpts)
	require.Empty(t, cache.identities)
	require.Empty(t, cache.identitiesExcerpts)

	// Reload, only excerpt are loaded
	cache, err = NewRepoCache(repo)
	require.NoError(t, err)
	require.Empty(t, cache.bugs)
	require.Empty(t, cache.identities)
	require.Len(t, cache.bugExcerpts, 2)
	require.Len(t, cache.identitiesExcerpts, 2)

	// Resolving load from the disk
	_, err = cache.ResolveIdentity(iden1.Id())
	require.NoError(t, err)
	_, err = cache.ResolveIdentityExcerpt(iden1.Id())
	require.NoError(t, err)
	_, err = cache.ResolveIdentityPrefix(iden1.Id().String()[:10])
	require.NoError(t, err)

	_, err = cache.ResolveBug(bug1.Id())
	require.NoError(t, err)
	_, err = cache.ResolveBugExcerpt(bug1.Id())
	require.NoError(t, err)
	_, err = cache.ResolveBugPrefix(bug1.Id().String()[:10])
	require.NoError(t, err)
}

func TestPushPull(t *testing.T) {
	repoA, repoB, remote := repository.SetupReposAndRemote()
	defer repository.CleanupTestRepos(repoA, repoB, remote)

	cacheA, err := NewRepoCache(repoA)
	require.NoError(t, err)

	cacheB, err := NewRepoCache(repoB)
	require.NoError(t, err)

	// Create, set and get user identity
	reneA, err := cacheA.NewIdentity("René Descartes", "rene@descartes.fr")
	require.NoError(t, err)
	err = cacheA.SetUserIdentity(reneA)
	require.NoError(t, err)

	// distribute the identity
	_, err = cacheA.Push("origin")
	require.NoError(t, err)
	err = cacheB.Pull("origin")
	require.NoError(t, err)

	// Create a bug in A
	_, _, err = cacheA.NewBug("bug1", "message")
	require.NoError(t, err)

	// A --> remote --> B
	_, err = cacheA.Push("origin")
	require.NoError(t, err)

	err = cacheB.Pull("origin")
	require.NoError(t, err)

	require.Len(t, cacheB.AllBugsIds(), 1)

	// retrieve and set identity
	reneB, err := cacheB.ResolveIdentity(reneA.Id())
	require.NoError(t, err)

	err = cacheB.SetUserIdentity(reneB)
	require.NoError(t, err)

	// B --> remote --> A
	_, _, err = cacheB.NewBug("bug2", "message")
	require.NoError(t, err)

	_, err = cacheB.Push("origin")
	require.NoError(t, err)

	err = cacheA.Pull("origin")
	require.NoError(t, err)

	require.Len(t, cacheA.AllBugsIds(), 2)
}

func TestRemove(t *testing.T) {
	repo := repository.CreateGoGitTestRepo(false)
	remoteA := repository.CreateGoGitTestRepo(true)
	remoteB := repository.CreateGoGitTestRepo(true)
	defer repository.CleanupTestRepos(repo, remoteA, remoteB)

	err := repo.AddRemote("remoteA", remoteA.GetLocalRemote())
	require.NoError(t, err)

	err = repo.AddRemote("remoteB", remoteB.GetLocalRemote())
	require.NoError(t, err)

	repoCache, err := NewRepoCache(repo)
	require.NoError(t, err)

	rene, err := repoCache.NewIdentity("René Descartes", "rene@descartes.fr")
	require.NoError(t, err)

	err = repoCache.SetUserIdentity(rene)
	require.NoError(t, err)

	_, _, err = repoCache.NewBug("title", "message")
	require.NoError(t, err)

	// and one more for testing
	b1, _, err := repoCache.NewBug("title", "message")
	require.NoError(t, err)

	_, err = repoCache.Push("remoteA")
	require.NoError(t, err)

	_, err = repoCache.Push("remoteB")
	require.NoError(t, err)

	_, err = repoCache.Fetch("remoteA")
	require.NoError(t, err)

	_, err = repoCache.Fetch("remoteB")
	require.NoError(t, err)

	err = repoCache.RemoveBug(b1.Id().String())
	require.NoError(t, err)
	assert.Equal(t, 1, len(repoCache.bugs))
	assert.Equal(t, 1, len(repoCache.bugExcerpts))

	_, err = repoCache.ResolveBug(b1.Id())
	assert.Error(t, bug.ErrBugNotExist, err)
}

func TestCacheEviction(t *testing.T) {
	repo := repository.CreateGoGitTestRepo(false)
	repoCache, err := NewRepoCache(repo)
	require.NoError(t, err)
	repoCache.setCacheSize(2)

	require.Equal(t, 2, repoCache.maxLoadedBugs)
	require.Equal(t, 0, repoCache.loadedBugs.Len())
	require.Equal(t, 0, len(repoCache.bugs))

	// Generating some bugs
	rene, err := repoCache.NewIdentity("René Descartes", "rene@descartes.fr")
	require.NoError(t, err)
	err = repoCache.SetUserIdentity(rene)
	require.NoError(t, err)

	bug1, _, err := repoCache.NewBug("title", "message")
	require.NoError(t, err)

	checkBugPresence(t, repoCache, bug1, true)
	require.Equal(t, 1, repoCache.loadedBugs.Len())
	require.Equal(t, 1, len(repoCache.bugs))

	bug2, _, err := repoCache.NewBug("title", "message")
	require.NoError(t, err)

	checkBugPresence(t, repoCache, bug1, true)
	checkBugPresence(t, repoCache, bug2, true)
	require.Equal(t, 2, repoCache.loadedBugs.Len())
	require.Equal(t, 2, len(repoCache.bugs))

	// Number of bugs should not exceed max size of lruCache, oldest one should be evicted
	bug3, _, err := repoCache.NewBug("title", "message")
	require.NoError(t, err)

	require.Equal(t, 2, repoCache.loadedBugs.Len())
	require.Equal(t, 2, len(repoCache.bugs))
	checkBugPresence(t, repoCache, bug1, false)
	checkBugPresence(t, repoCache, bug2, true)
	checkBugPresence(t, repoCache, bug3, true)

	// Accessing bug should update position in lruCache and therefore it should not be evicted
	repoCache.loadedBugs.Get(bug2.Id())
	oldestId, _ := repoCache.loadedBugs.GetOldest()
	require.Equal(t, bug3.Id(), oldestId)

	checkBugPresence(t, repoCache, bug1, false)
	checkBugPresence(t, repoCache, bug2, true)
	checkBugPresence(t, repoCache, bug3, true)
	require.Equal(t, 2, repoCache.loadedBugs.Len())
	require.Equal(t, 2, len(repoCache.bugs))
}

func checkBugPresence(t *testing.T, cache *RepoCache, bug *BugCache, presence bool) {
	id := bug.Id()
	require.Equal(t, presence, cache.loadedBugs.Contains(id))
	b, ok := cache.bugs[id]
	require.Equal(t, presence, ok)
	if ok {
		require.Equal(t, bug, b)
	}
}
