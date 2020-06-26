package cache

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MichaelMure/git-bug/query"
	"github.com/MichaelMure/git-bug/repository"
)

func TestCache(t *testing.T) {
	repo := repository.CreateTestRepo(false)
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
	require.NoError(t, cache.load())
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
