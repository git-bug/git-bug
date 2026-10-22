package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/99designs/keyring"
	"github.com/git-bug/gitconfig"
	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/entities/bug"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/misc/random_bugs"
	"github.com/git-bug/git-bug/query"
	"github.com/git-bug/git-bug/repository"
	"github.com/git-bug/git-bug/repository/repotest"
)

// requireDerivedBuiltFromRefs checks that the cache and the index recorded, for
// every entity, the commit its reference points to, and nothing else.
func requireDerivedBuiltFromRefs(t *testing.T, repo repository.Repo, c *RepoCache) {
	t.Helper()

	check := func(namespace string, builtFrom map[entity.Id]repository.Hash, excerpts int) {
		t.Helper()
		refs, err := repo.ListRefs(namespace)
		require.NoError(t, err)
		expected := make(map[entity.Id]repository.Hash, len(refs))
		for key, hash := range refs {
			expected[entity.Id(key)] = hash
		}
		require.Equal(t, expected, builtFrom)
		require.Equal(t, len(expected), excerpts)

		index, err := repo.GetIndex(namespace, formatVersion)
		require.NoError(t, err)
		indexBuiltFrom, err := index.AllBuiltFrom(t.Context())
		require.NoError(t, err)
		require.Equal(t, refs, indexBuiltFrom)
	}

	// copied under the lock and checked after, as a failed check stops the test
	// right away, and a lock still held would hang its cleanup
	c.bugs.muMaps.RLock()
	bugsBuiltFrom, bugsExcerpts := maps.Clone(c.bugs.builtFrom), len(c.bugs.excerpts)
	c.bugs.muMaps.RUnlock()
	check(bug.Namespace, bugsBuiltFrom, bugsExcerpts)

	c.identities.muMaps.RLock()
	identitiesBuiltFrom, identitiesExcerpts := maps.Clone(c.identities.builtFrom), len(c.identities.excerpts)
	c.identities.muMaps.RUnlock()
	check(identity.Namespace, identitiesBuiltFrom, identitiesExcerpts)
}

// clearBugIndex removes every bug from the index, as a lost index would look
// to the cache.
func clearBugIndex(t *testing.T, repo repository.Repo) {
	t.Helper()
	index, err := repo.GetIndex(bug.Namespace, formatVersion)
	require.NoError(t, err)
	builtFrom, err := index.AllBuiltFrom(t.Context())
	require.NoError(t, err)
	batch := index.NewBatch()
	for id := range builtFrom {
		batch.Remove(id)
	}
	require.NoError(t, batch.Apply(t.Context()))
}

func searchBugs(t *testing.T, c *RepoCache, term string) []entity.Id {
	t.Helper()
	q, err := query.Parse(term)
	require.NoError(t, err)
	res, err := c.Bugs().Query(t.Context(), q)
	require.NoError(t, err)
	return res
}

func lenComments(t *testing.T, c *RepoCache, id entity.Id) int {
	t.Helper()
	excerpt, err := c.Bugs().ResolveExcerpt(id)
	require.NoError(t, err)
	return excerpt.LenComments
}

// newTestCacheWithUser opens a cache on repo, with a user identity set.
func newTestCacheWithUser(t testing.TB, repo repository.Repo) (*RepoCache, *IdentityCache) {
	t.Helper()
	c := createTestRepoCacheNoEvents(t, repo)
	rene, err := c.Identities().New("René Descartes", "rene@descartes.fr")
	require.NoError(t, err)
	require.NoError(t, c.SetUserIdentity(t.Context(), rene))
	return c, rene
}

// countingRepo counts the index batches applied through it.
type countingRepo struct {
	repository.Repo
	batches int
}

func (r *countingRepo) GetIndex(namespace string, version uint) (repository.Index, error) {
	index, err := r.Repo.GetIndex(namespace, version)
	if err != nil {
		return nil, err
	}
	return countingIndex{Index: index, repo: r}, nil
}

type countingIndex struct {
	repository.Index
	repo *countingRepo
}

func (i countingIndex) NewBatch() repository.IndexBatch {
	return countingBatch{IndexBatch: i.Index.NewBatch(), repo: i.repo}
}

type countingBatch struct {
	repository.IndexBatch
	repo *countingRepo
}

func (b countingBatch) Apply(ctx context.Context) error {
	b.repo.batches++
	return b.IndexBatch.Apply(ctx)
}

func TestSubCacheDerived(t *testing.T) {
	t.Run("excerpts are the same once stored in the index", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)
		c, rene := newTestCacheWithUser(t, repo)

		// every field set
		other, err := c.Identities().NewRaw("Blaise Pascal", "blaise@pascal.fr", "bpascal", "", nil,
			map[string]string{"origin": "test"})
		require.NoError(t, err)
		b, _, err := c.Bugs().NewRaw(rene, time.Now().Unix(), "title", "message", nil,
			map[string]string{"origin": "test"})
		require.NoError(t, err)
		require.NoError(t, b.Commit(t.Context()))
		_, _, err = b.AddCommentRaw(other, time.Now().Unix(), "comment", nil, nil)
		require.NoError(t, err)
		_, _, err = b.ChangeLabels(t.Context(), []string{"bug", "ui"}, nil)
		require.NoError(t, err)
		_, err = b.Close(t.Context())
		require.NoError(t, err)
		require.NoError(t, b.Commit(t.Context()))

		bugExcerpt, err := c.Bugs().ResolveExcerpt(b.Id())
		require.NoError(t, err)
		raw, err := json.Marshal(bugExcerpt)
		require.NoError(t, err)
		decodedBug, ok := c.bugs.decodeExcerpt(b.Id(), raw)
		require.True(t, ok)
		require.Equal(t, bugExcerpt, decodedBug)

		identityExcerpt, err := c.Identities().ResolveExcerpt(other.Id())
		require.NoError(t, err)
		raw, err = json.Marshal(identityExcerpt)
		require.NoError(t, err)
		decodedIdentity, ok := c.identities.decodeExcerpt(other.Id(), raw)
		require.True(t, ok)
		require.Equal(t, identityExcerpt, decodedIdentity)
	})

	t.Run("built from git", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)
		random_bugs.FillRepoWithSeed(repo, 5, 42)

		c := createTestRepoCacheNoEvents(t, repo)
		requireDerivedBuiltFromRefs(t, repo, c)

		// only the derived state is built, entities are not kept in memory, except
		// the identities resolved as the authors of bugs, which can be evicted
		require.Empty(t, c.bugs.cached)
		require.Equal(t, c.identities.lru.Len(), len(c.identities.cached))
	})

	t.Run("built from git, over several index batches", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)
		// more than twice the batch size, and not a multiple of it
		random_bugs.FillRepoWithSeed(repo, 200, 42)

		c := createTestRepoCacheNoEvents(t, repo)
		requireDerivedBuiltFromRefs(t, repo, c)
	})

	t.Run("created", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)
		c, _ := newTestCacheWithUser(t, repo)
		obs := &observer{}
		require.NoError(t, c.registerObserver("repotest", bug.Typename, obs))

		created, _, err := c.Bugs().New(t.Context(), "title", "message")
		require.NoError(t, err)
		_, _, err = created.AddComment(t.Context(), "comment")
		require.NoError(t, err)
		require.Empty(t, obs.created)

		// the staged creation and what follows it are announced once
		require.NoError(t, created.Commit(t.Context()))
		require.Equal(t, []observerEvent{{bug.Typename, created.Id()}}, obs.created)
		require.Empty(t, obs.updated)
		requireDerivedBuiltFromRefs(t, repo, c)

		_, _, err = created.AddComment(t.Context(), "another")
		require.NoError(t, err)
		require.NoError(t, created.Commit(t.Context()))
		require.Equal(t, []observerEvent{{bug.Typename, created.Id()}}, obs.created)
		require.Equal(t, []observerEvent{{bug.Typename, created.Id()}}, obs.updated)
	})

	t.Run("pending changes are visible only once committed", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)
		c, _ := newTestCacheWithUser(t, repo)
		b, _, err := c.Bugs().New(t.Context(), "title", "message")
		require.NoError(t, err)
		require.NoError(t, b.Commit(t.Context()))

		obs := &observer{}
		require.NoError(t, c.registerObserver("repotest", bug.Typename, obs))

		_, _, err = b.AddComment(t.Context(), "markerpending")
		require.NoError(t, err)
		require.Equal(t, 1, lenComments(t, c, b.Id()))
		require.Empty(t, searchBugs(t, c, "markerpending"))
		require.Empty(t, obs.updated)
		requireDerivedBuiltFromRefs(t, repo, c)

		require.NoError(t, b.Commit(t.Context()))
		require.Equal(t, 2, lenComments(t, c, b.Id()))
		require.Equal(t, []entity.Id{b.Id()}, searchBugs(t, c, "markerpending"))
		require.Len(t, obs.updated, 1)
		requireDerivedBuiltFromRefs(t, repo, c)

		// nothing to commit, nothing changes
		require.NoError(t, b.CommitAsNeeded(t.Context()))
		require.Len(t, obs.updated, 1)
	})

	t.Run("identity changes are visible only once committed", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)
		c, rene := newTestCacheWithUser(t, repo)

		require.NoError(t, rene.Mutate(repo, func(m *identity.Mutator) {
			m.Name = "René"
		}))
		excerpt, err := c.Identities().ResolveExcerpt(rene.Id())
		require.NoError(t, err)
		require.Equal(t, "René Descartes", excerpt.Name)
		requireDerivedBuiltFromRefs(t, repo, c)

		require.NoError(t, rene.CommitAsNeeded())
		excerpt, err = c.Identities().ResolveExcerpt(rene.Id())
		require.NoError(t, err)
		require.Equal(t, "René", excerpt.Name)
		requireDerivedBuiltFromRefs(t, repo, c)
	})

	t.Run("persisted, and loaded rather than rebuilt", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)

		c := openTestRepoCache(t, repo)
		rene, err := c.Identities().New("René Descartes", "rene@descartes.fr")
		require.NoError(t, err)
		require.NoError(t, c.SetUserIdentity(t.Context(), rene))
		b, _, err := c.Bugs().New(t.Context(), "title", "message")
		require.NoError(t, err)
		require.NoError(t, b.Commit(t.Context()))
		_, _, err = b.AddComment(t.Context(), "comment")
		require.NoError(t, err)
		require.NoError(t, b.Commit(t.Context()))
		require.NoError(t, c.Close())

		c = createTestRepoCacheNoEvents(t, repo)
		require.Empty(t, c.bugs.cached)
		require.Empty(t, c.identities.cached)
		requireDerivedBuiltFromRefs(t, repo, c)
	})

	t.Run("an index without record is repaired on load", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)

		c := openTestRepoCache(t, repo)
		rene, err := c.Identities().New("René Descartes", "rene@descartes.fr")
		require.NoError(t, err)
		require.NoError(t, c.SetUserIdentity(t.Context(), rene))
		created, _, err := c.Bugs().New(t.Context(), "title", "message")
		require.NoError(t, err)
		require.NoError(t, created.Commit(t.Context()))
		require.NoError(t, c.Close())

		// what an index written before it recorded anything, or a lost one,
		// looks like to the cache
		clearBugIndex(t, repo)

		c = createTestRepoCacheNoEvents(t, repo)
		requireDerivedBuiltFromRefs(t, repo, c)
	})

	// counted has a subcache count the entities it reads from git, the loaded
	// copies it refreshes, and the index batches it applies
	counted := func(sc *RepoCacheBug) (reads *int, refreshes *int, batches *int) {
		reads, refreshes = new(int), new(int)
		read := sc.actions.ReadWithResolver
		sc.actions.ReadWithResolver = func(ctx context.Context, repo repository.Repo, resolvers entity.Resolvers, id entity.Id) (*sharedBug, error) {
			*reads++
			return read(ctx, repo, resolvers, id)
		}
		refresh := sc.actions.Refresh
		sc.actions.Refresh = func(ctx context.Context, repo repository.Repo, resolvers entity.Resolvers, shared *sharedBug) (*sharedBug, error) {
			*refreshes++
			return refresh(ctx, repo, resolvers, shared)
		}
		repo := &countingRepo{Repo: sc.repo}
		sc.repo = repo
		return reads, refreshes, &repo.batches
	}

	t.Run("a change from outside is synced, refreshing only that entity", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)
		c, rene := newTestCacheWithUser(t, repo)
		var views []*BugCache
		for i := 0; i < 3; i++ {
			b, _, err := c.Bugs().New(t.Context(), "title", "message")
			require.NoError(t, err)
			require.NoError(t, b.Commit(t.Context()))
			views = append(views, b)
		}
		id := views[1].Id()

		obs := &observer{}
		require.NoError(t, c.registerObserver("repotest", bug.Typename, obs))
		reads, refreshes, batches := counted(c.bugs)

		require.NoError(t, c.bugs.syncAll(nil))
		require.Zero(t, *reads)
		require.Zero(t, *refreshes)
		require.Zero(t, *batches)

		changeOutside(t, repo, rene, id, "markeroutside")
		require.NoError(t, c.bugs.syncAll(nil))
		// the loaded copy is brought up to date, nothing is read from scratch
		require.Zero(t, *reads)
		require.Equal(t, 1, *refreshes)
		require.Equal(t, 1, *batches)
		require.Equal(t, []observerEvent{{bug.Typename, id}}, obs.updated)
		require.Equal(t, 2, lenComments(t, c, id))
		require.Equal(t, []entity.Id{id}, searchBugs(t, c, "markeroutside"))
		require.Len(t, views[1].Snapshot().Comments, 2)
		requireDerivedBuiltFromRefs(t, repo, c)
	})

	t.Run("a change from outside to an entity not loaded is synced, reading only that entity", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)
		c, rene := newTestCacheWithUser(t, repo)
		var ids []entity.Id
		for i := 0; i < 3; i++ {
			b, _, err := c.Bugs().New(t.Context(), "title", "message")
			require.NoError(t, err)
			require.NoError(t, b.Commit(t.Context()))
			ids = append(ids, b.Id())
		}
		c.setCacheSize(0)
		require.Empty(t, c.bugs.cached)

		obs := &observer{}
		require.NoError(t, c.registerObserver("repotest", bug.Typename, obs))
		reads, refreshes, batches := counted(c.bugs)

		changeOutside(t, repo, rene, ids[1], "markeroutside")
		require.NoError(t, c.bugs.syncAll(nil))
		require.Equal(t, 1, *reads)
		require.Zero(t, *refreshes)
		require.Equal(t, 1, *batches)
		require.Equal(t, []observerEvent{{bug.Typename, ids[1]}}, obs.updated)
		require.Empty(t, obs.created)
		require.Empty(t, obs.removed)
		require.Equal(t, 2, lenComments(t, c, ids[1]))
		require.Empty(t, c.bugs.cached)
		requireDerivedBuiltFromRefs(t, repo, c)
	})

	t.Run("many changes from outside are applied in a few batches", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)
		c, rene := newTestCacheWithUser(t, repo)

		// more than twice the batch size, and not a multiple of it
		count := 2*syncBatchSize + 1
		for i := 0; i < count; i++ {
			b, _, err := bug.Create(rene, time.Now().Unix(), "title", "message", nil, nil)
			require.NoError(t, err)
			require.NoError(t, b.Commit(repo))
		}

		obs := &observer{}
		require.NoError(t, c.registerObserver("repotest", bug.Typename, obs))
		reads, _, batches := counted(c.bugs)

		require.NoError(t, c.bugs.syncAll(nil))
		require.Equal(t, count, *reads)
		require.Equal(t, 3, *batches)
		require.Len(t, obs.created, count)
		requireDerivedBuiltFromRefs(t, repo, c)
	})

	t.Run("an index behind is repaired, the excerpts left as they are", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)
		c, _ := newTestCacheWithUser(t, repo)
		b, _, err := c.Bugs().New(t.Context(), "title", "markerindexed")
		require.NoError(t, err)
		require.NoError(t, b.Commit(t.Context()))

		clearBugIndex(t, repo)

		obs := &observer{}
		require.NoError(t, c.registerObserver("repotest", bug.Typename, obs))
		_, _, batches := counted(c.bugs)

		require.NoError(t, c.bugs.syncAll(nil))
		require.Equal(t, 1, *batches)
		require.Equal(t, []entity.Id{b.Id()}, searchBugs(t, c, "markerindexed"))
		requireDerivedBuiltFromRefs(t, repo, c)

		// the excerpts didn't change
		require.Equal(t, &observer{}, obs)
	})

	t.Run("a change synced by another process is read from the index", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)
		c, rene := newTestCacheWithUser(t, repo)
		b, _, err := c.Bugs().New(t.Context(), "title", "message")
		require.NoError(t, err)
		require.NoError(t, b.Commit(t.Context()))
		id := b.Id()
		// no loaded copy to bring up to date
		c.setCacheSize(0)

		// another process sees the change first, and updates the index
		changeOutside(t, repo, rene, id, "markeroutside")
		other := NewRepoCacheBug(repo, c.getResolvers, c.GetUserIdentity)
		require.NoError(t, other.syncAll(nil))

		obs := &observer{}
		require.NoError(t, c.registerObserver("repotest", bug.Typename, obs))
		reads, refreshes, batches := counted(c.bugs)

		require.NoError(t, c.bugs.syncAll(nil))
		require.Zero(t, *reads)
		require.Zero(t, *refreshes)
		require.Zero(t, *batches)
		require.Equal(t, []observerEvent{{bug.Typename, id}}, obs.updated)
		require.Equal(t, 2, lenComments(t, c, id))
		require.Equal(t, []entity.Id{id}, searchBugs(t, c, "markeroutside"))
		requireDerivedBuiltFromRefs(t, repo, c)
	})

	t.Run("excerpts behind are repaired, the index left as it is", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)
		c, _ := newTestCacheWithUser(t, repo)
		b, _, err := c.Bugs().New(t.Context(), "title", "message")
		require.NoError(t, err)
		require.NoError(t, b.Commit(t.Context()))

		older := c.bugs.excerpts[b.Id()]
		olderCommit := c.bugs.builtFrom[b.Id()]
		_, _, err = b.AddComment(t.Context(), "comment")
		require.NoError(t, err)
		require.NoError(t, b.Commit(t.Context()))

		// put back the excerpt from before the commit
		c.bugs.muMaps.Lock()
		c.bugs.excerpts[b.Id()] = older
		c.bugs.builtFrom[b.Id()] = olderCommit
		c.bugs.muMaps.Unlock()

		obs := &observer{}
		require.NoError(t, c.registerObserver("repotest", bug.Typename, obs))
		reads, refreshes, batches := counted(c.bugs)

		// the loaded copy is at the reference already: nothing is read
		require.NoError(t, c.bugs.syncAll(nil))
		require.Zero(t, *reads)
		require.Zero(t, *refreshes)
		require.Zero(t, *batches)
		require.Equal(t, []observerEvent{{bug.Typename, b.Id()}}, obs.updated)
		require.Equal(t, 2, lenComments(t, c, b.Id()))
		requireDerivedBuiltFromRefs(t, repo, c)
	})

	t.Run("changes fetched outside git-bug are picked up on load", func(t *testing.T) {
		repoA, repoB, _ := repotest.SetupGoGitReposAndRemote(t)
		cacheA, _ := newTestCacheWithUser(t, repoA)
		b, _, err := cacheA.Bugs().New(t.Context(), "title", "message")
		require.NoError(t, err)
		require.NoError(t, b.Commit(t.Context()))
		_, err = cacheA.Push(t.Context(), "origin")
		require.NoError(t, err)

		cacheB := openTestRepoCache(t, repoB)
		require.NoError(t, cacheB.Pull(t.Context(), "origin"))
		require.Equal(t, 1, lenComments(t, cacheB, b.Id()))
		require.NoError(t, cacheB.Close())

		// A changes the bug, and B gets it with no git-bug command: what a fetch
		// into the local refs, or a push into B, does
		bA, err := cacheA.Bugs().Resolve(t.Context(), b.Id())
		require.NoError(t, err)
		_, _, err = bA.AddComment(t.Context(), "markeroutside")
		require.NoError(t, err)
		require.NoError(t, bA.Commit(t.Context()))
		_, err = cacheA.Push(t.Context(), "origin")
		require.NoError(t, err)

		for _, namespace := range []string{identity.Namespace, bug.Namespace} {
			_, err = repoB.FetchRefs(t.Context(), "origin", namespace)
			require.NoError(t, err)
			tracking, err := repoB.ListTrackingRefs("origin", namespace)
			require.NoError(t, err)
			for key, commit := range tracking {
				local, err := repoB.ResolveRef(namespace, key)
				if errors.Is(err, repository.ErrNotFound) {
					local = ""
				} else {
					require.NoError(t, err)
				}
				require.NoError(t, repoB.UpdateRef(namespace, key, local, commit))
			}
		}

		cacheB = createTestRepoCacheNoEvents(t, repoB)
		require.Equal(t, 2, lenComments(t, cacheB, b.Id()))
		require.Equal(t, []entity.Id{b.Id()}, searchBugs(t, cacheB, "markeroutside"))
		requireDerivedBuiltFromRefs(t, repoB, cacheB)
	})

	t.Run("merged", func(t *testing.T) {
		repoA, repoB, _ := repotest.SetupGoGitReposAndRemote(t)
		cacheA, _ := newTestCacheWithUser(t, repoA)
		cacheB := createTestRepoCacheNoEvents(t, repoB)

		// as new
		b, _, err := cacheA.Bugs().New(t.Context(), "title", "message")
		require.NoError(t, err)
		require.NoError(t, b.Commit(t.Context()))
		_, err = cacheA.Push(t.Context(), "origin")
		require.NoError(t, err)
		require.NoError(t, cacheB.Pull(t.Context(), "origin"))
		requireDerivedBuiltFromRefs(t, repoB, cacheB)

		// as updated
		_, _, err = b.AddComment(t.Context(), "markerpulled")
		require.NoError(t, err)
		require.NoError(t, b.Commit(t.Context()))
		_, err = cacheA.Push(t.Context(), "origin")
		require.NoError(t, err)
		require.NoError(t, cacheB.Pull(t.Context(), "origin"))
		require.Equal(t, []entity.Id{b.Id()}, searchBugs(t, cacheB, "markerpulled"))
		requireDerivedBuiltFromRefs(t, repoB, cacheB)
	})

	t.Run("removed", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)
		c, _ := newTestCacheWithUser(t, repo)
		b1, _, err := c.Bugs().New(t.Context(), "title", "message")
		require.NoError(t, err)
		require.NoError(t, b1.Commit(t.Context()))
		created, _, err := c.Bugs().New(t.Context(), "title", "message")
		require.NoError(t, err)
		require.NoError(t, created.Commit(t.Context()))

		require.NoError(t, c.Bugs().Remove(b1.Id().String()))
		requireDerivedBuiltFromRefs(t, repo, c)

		require.NoError(t, c.RemoveAll())
		requireDerivedBuiltFromRefs(t, repo, c)
	})

	t.Run("a loaded copy without derived state is dropped once removed", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)
		c, rene := newTestCacheWithUser(t, repo)

		// created outside, and loaded before any sync
		b, _, err := bug.Create(rene, time.Now().Unix(), "title", "message", nil, nil)
		require.NoError(t, err)
		require.NoError(t, b.Commit(repo))
		_, err = c.Bugs().Resolve(t.Context(), b.Id())
		require.NoError(t, err)

		require.NoError(t, c.RemoveAll())
		require.NotContains(t, c.bugs.cached, b.Id())
		_, err = c.Bugs().Resolve(t.Context(), b.Id())
		require.True(t, entity.IsErrNotFound(err))
		requireDerivedBuiltFromRefs(t, repo, c)
	})

	t.Run("a missing author is not a removal", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)
		c, _ := newTestCacheWithUser(t, repo)

		author, err := identity.NewIdentity(repo, "Blaise Pascal", "blaise@pascal.fr")
		require.NoError(t, err)
		require.NoError(t, author.Commit(repo))
		b, _, err := bug.Create(author, time.Now().Unix(), "title", "message", nil, nil)
		require.NoError(t, err)
		require.NoError(t, b.Commit(repo))
		require.NoError(t, c.identities.syncAll(nil))
		require.NoError(t, c.bugs.syncAll(nil))

		// the author goes away, and the bug needs reading again
		require.NoError(t, repo.RemoveRef(identity.Namespace, author.Id().String()))
		require.NoError(t, c.identities.syncAll(nil))
		clearBugIndex(t, repo)

		obs := &observer{}
		require.NoError(t, c.registerObserver("repotest", bug.Typename, obs))

		require.Error(t, c.bugs.syncAll(nil))
		require.Empty(t, obs.removed)
		_, err = c.Bugs().ResolveExcerpt(b.Id())
		require.NoError(t, err)
	})

	t.Run("concurrent resolves of an entity not loaded read it once", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			repo := repotest.NewGoGitRepo(t, false)
			c, _ := newTestCacheWithUser(t, repo)
			c.setCacheSize(1)

			evicted, _, err := c.Bugs().New(t.Context(), "title", "message")
			require.NoError(t, err)
			require.NoError(t, evicted.Commit(t.Context()))
			created, _, err := c.Bugs().New(t.Context(), "title", "message")
			require.NoError(t, err)
			require.NoError(t, created.Commit(t.Context()))
			require.NotContains(t, c.bugs.cached, evicted.Id())

			var reads atomic.Int32
			release := make(chan struct{})
			read := c.bugs.actions.ReadWithResolver
			c.bugs.actions.ReadWithResolver = func(ctx context.Context, repo repository.Repo, resolvers entity.Resolvers, id entity.Id) (*sharedBug, error) {
				reads.Add(1)
				<-release
				return read(ctx, repo, resolvers, id)
			}

			const callers = 5
			views := make([]*BugCache, callers)
			var wg sync.WaitGroup
			for i := range callers {
				wg.Go(func() {
					b, err := c.Bugs().Resolve(t.Context(), evicted.Id())
					require.NoError(t, err)
					views[i] = b
				})
			}
			synctest.Wait()
			require.Equal(t, int32(1), reads.Load())

			close(release)
			wg.Wait()
			for _, b := range views {
				require.Same(t, views[0].shared, b.shared)
			}
		})
	})

	t.Run("an evicted copy keeps working, and the loaded one follows its commits", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)
		c, _ := newTestCacheWithUser(t, repo)
		c.setCacheSize(1)

		evicted, _, err := c.Bugs().New(t.Context(), "title", "message")
		require.NoError(t, err)
		require.NoError(t, evicted.Commit(t.Context()))
		created, _, err := c.Bugs().New(t.Context(), "title", "message")
		require.NoError(t, err)
		require.NoError(t, created.Commit(t.Context()))
		require.NotContains(t, c.bugs.cached, evicted.Id())

		// a second copy gets loaded, as the evicted one is still in use
		loaded, err := c.Bugs().Resolve(t.Context(), evicted.Id())
		require.NoError(t, err)
		require.NotSame(t, evicted.shared, loaded.shared)

		_, _, err = evicted.AddComment(t.Context(), "comment")
		require.NoError(t, err)
		require.NoError(t, evicted.Commit(t.Context()))
		require.Equal(t, 2, lenComments(t, c, evicted.Id()))
		requireDerivedBuiltFromRefs(t, repo, c)

		// the loaded copy was brought up to date by that commit's sync
		require.Len(t, loaded.Snapshot().Comments, 2)
		require.NoError(t, loaded.CommitAsNeeded(t.Context()))
		require.Equal(t, 2, lenComments(t, c, evicted.Id()))
		requireDerivedBuiltFromRefs(t, repo, c)

		// its own commits land on top
		_, _, err = loaded.AddComment(t.Context(), "another")
		require.NoError(t, err)
		require.NoError(t, loaded.Commit(t.Context()))
		require.Equal(t, 3, lenComments(t, c, evicted.Id()))
		requireDerivedBuiltFromRefs(t, repo, c)

		// the evicted copy is now behind, which its next commit catches up on
		require.Len(t, evicted.Snapshot().Comments, 2)
		_, _, err = evicted.AddComment(t.Context(), "last")
		require.NoError(t, err)
		require.NoError(t, evicted.Commit(t.Context()))
		require.Equal(t, []string{"message", "comment", "another", "last"}, commentMessages(evicted.Snapshot()))
		require.Equal(t, 4, lenComments(t, c, evicted.Id()))
		requireDerivedBuiltFromRefs(t, repo, c)
	})

	t.Run("an evicted copy committing with none loaded becomes the loaded one", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)
		c, _ := newTestCacheWithUser(t, repo)
		c.setCacheSize(1)

		evicted, _, err := c.Bugs().New(t.Context(), "title", "message")
		require.NoError(t, err)
		require.NoError(t, evicted.Commit(t.Context()))
		other, _, err := c.Bugs().New(t.Context(), "title", "message")
		require.NoError(t, err)
		require.NoError(t, other.Commit(t.Context()))
		require.NotContains(t, c.bugs.cached, evicted.Id())

		_, _, err = evicted.AddComment(t.Context(), "comment")
		require.NoError(t, err)
		require.NoError(t, evicted.Commit(t.Context()))
		require.Same(t, evicted.shared, c.bugs.cached[evicted.Id()])
		// still within the size, the other one going
		require.NotContains(t, c.bugs.cached, other.Id())
		requireDerivedBuiltFromRefs(t, repo, c)

		resolved, err := c.Bugs().Resolve(t.Context(), evicted.Id())
		require.NoError(t, err)
		require.Same(t, evicted.shared, resolved.shared)
	})

	t.Run("a new entity evicted right away is still usable", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)
		c, _ := newTestCacheWithUser(t, repo)
		c.setCacheSize(0)

		b, _, err := c.Bugs().New(t.Context(), "title", "message")
		require.NoError(t, err)
		require.NoError(t, b.Commit(t.Context()))
		require.NotContains(t, c.bugs.cached, b.Id())
		requireDerivedBuiltFromRefs(t, repo, c)

		_, _, err = b.AddComment(t.Context(), "comment")
		require.NoError(t, err)
		require.NoError(t, b.Commit(t.Context()))
		require.Equal(t, 2, lenComments(t, c, b.Id()))
		requireDerivedBuiltFromRefs(t, repo, c)
	})

	t.Run("synced from git while a loaded copy has pending changes", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)
		c, _ := newTestCacheWithUser(t, repo)
		b, _, err := c.Bugs().New(t.Context(), "title", "message")
		require.NoError(t, err)
		require.NoError(t, b.Commit(t.Context()))
		_, _, err = b.AddComment(t.Context(), "markerpending")
		require.NoError(t, err)

		clearBugIndex(t, repo)
		require.NoError(t, c.bugs.syncAll(nil))
		require.Equal(t, 1, lenComments(t, c, b.Id()))
		require.Empty(t, searchBugs(t, c, "markerpending"))
		requireDerivedBuiltFromRefs(t, repo, c)

		// the loaded copy is untouched, and its commit is followed
		require.True(t, b.NeedCommit())
		require.NoError(t, b.Commit(t.Context()))
		require.Equal(t, []entity.Id{b.Id()}, searchBugs(t, c, "markerpending"))
		requireDerivedBuiltFromRefs(t, repo, c)
	})

	t.Run("pending changes on a loaded copy don't hold back a commit", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)
		c, rene := newTestCacheWithUser(t, repo)
		b, _, err := c.Bugs().New(t.Context(), "title", "message")
		require.NoError(t, err)
		require.NoError(t, b.Commit(t.Context()))
		_, _, err = b.AddComment(t.Context(), "markerpending")
		require.NoError(t, err)

		// another copy commits, and the callback comes while the loaded copy
		// has pending changes
		other, err := bug.Read(t.Context(), repo, b.Id())
		require.NoError(t, err)
		_, _, err = bug.AddComment(other, rene, time.Now().Unix(), "markercommitted", nil, nil)
		require.NoError(t, err)
		require.NoError(t, other.Commit(repo))
		require.NoError(t, c.bugs.onCommit(b.shared))

		require.Equal(t, []entity.Id{b.Id()}, searchBugs(t, c, "markercommitted"))
		require.Empty(t, searchBugs(t, c, "markerpending"))
		require.True(t, b.NeedCommit())
		requireDerivedBuiltFromRefs(t, repo, c)

		// the loaded copy was brought up to date: the pending changes come after
		// the committed ones, and commit on top of them
		require.Equal(t, []string{"message", "markercommitted", "markerpending"}, commentMessages(b.Snapshot()))
		require.NoError(t, b.Commit(t.Context()))
		require.Equal(t, []string{"message", "markercommitted", "markerpending"}, commentsInGit(t, repo, b.Id()))
		require.Equal(t, []entity.Id{b.Id()}, searchBugs(t, c, "markerpending"))
		requireDerivedBuiltFromRefs(t, repo, c)
	})

	t.Run("a repeated or late callback changes nothing", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)
		c, _ := newTestCacheWithUser(t, repo)
		b, _, err := c.Bugs().New(t.Context(), "title", "message")
		require.NoError(t, err)
		require.NoError(t, b.Commit(t.Context()))
		_, _, err = b.AddComment(t.Context(), "comment")
		require.NoError(t, err)
		require.NoError(t, b.Commit(t.Context()))

		obs := &observer{}
		require.NoError(t, c.registerObserver("repotest", bug.Typename, obs))

		require.NoError(t, c.bugs.onCommit(b.shared))
		require.Empty(t, obs.updated)
		require.Equal(t, 2, lenComments(t, c, b.Id()))
		requireDerivedBuiltFromRefs(t, repo, c)
	})

	t.Run("a late callback doesn't resurrect a removed entity", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)
		c, _ := newTestCacheWithUser(t, repo)
		b, _, err := c.Bugs().New(t.Context(), "title", "message")
		require.NoError(t, err)
		require.NoError(t, b.Commit(t.Context()))

		require.NoError(t, c.Bugs().Remove(b.Id().String()))
		obs := &observer{}
		require.NoError(t, c.registerObserver("repotest", bug.Typename, obs))
		require.NoError(t, c.bugs.onCommit(b.shared))

		require.NotContains(t, c.bugs.cached, b.Id())
		require.Empty(t, obs.created)
		require.Empty(t, obs.updated)
		requireDerivedBuiltFromRefs(t, repo, c)
	})

	t.Run("an entity removed outside the cache is dropped", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)
		c, _ := newTestCacheWithUser(t, repo)
		b, _, err := c.Bugs().New(t.Context(), "title", "message")
		require.NoError(t, err)
		require.NoError(t, b.Commit(t.Context()))

		obs := &observer{}
		require.NoError(t, c.registerObserver("repotest", bug.Typename, obs))

		require.NoError(t, bug.Remove(repo, b.Id()))
		require.NoError(t, c.bugs.sync(b.Id()))
		require.Equal(t, []observerEvent{{bug.Typename, b.Id()}}, obs.removed)

		requireDerivedBuiltFromRefs(t, repo, c)
		require.NotContains(t, c.bugs.cached, b.Id())
	})

	t.Run("a search hit without excerpt is skipped", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)
		c, _ := newTestCacheWithUser(t, repo)

		// the index is written before the excerpt is published
		index, err := repo.GetIndex(bug.Namespace, formatVersion)
		require.NoError(t, err)
		batch := index.NewBatch()
		require.NoError(t, batch.Set("notyetpublished", "1111111111111111111111111111111111111111", []byte(`{}`), []string{"markerahead"}))
		require.NoError(t, batch.Apply(t.Context()))

		require.Empty(t, searchBugs(t, c, "markerahead"))
	})

	t.Run("nothing is published once closed", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)
		c := openTestRepoCache(t, repo)
		rene, err := c.Identities().New("René Descartes", "rene@descartes.fr")
		require.NoError(t, err)
		require.NoError(t, c.SetUserIdentity(t.Context(), rene))
		b, _, err := c.Bugs().New(t.Context(), "title", "message")
		require.NoError(t, err)
		require.NoError(t, b.Commit(t.Context()))
		// changed outside the cache, which only notices on a sync
		read, err := bug.Read(t.Context(), repo, b.Id())
		require.NoError(t, err)
		_, _, err = bug.AddComment(read, rene, time.Now().Unix(), "comment", nil, nil)
		require.NoError(t, err)
		require.NoError(t, read.Commit(repo))

		require.NoError(t, c.Close())

		require.NoError(t, c.bugs.sync(b.Id()))
		require.Nil(t, c.bugs.builtFrom)
	})

	t.Run("concurrent commits", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)
		c, _ := newTestCacheWithUser(t, repo)
		b, _, err := c.Bugs().New(t.Context(), "title", "message")
		require.NoError(t, err)
		require.NoError(t, b.Commit(t.Context()))

		const count = 10
		var wg sync.WaitGroup
		errs := make(chan error, count)
		for i := 0; i < count; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if _, _, err := b.AddComment(t.Context(), fmt.Sprintf("marker%d", i)); err != nil {
					errs <- err
					return
				}
				errs <- b.CommitAsNeeded(t.Context())
			}()
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			require.NoError(t, err)
		}

		// the derived state holds the last commit, and all of its content
		require.Equal(t, count+1, lenComments(t, c, b.Id()))
		for i := 0; i < count; i++ {
			require.Equal(t, []entity.Id{b.Id()}, searchBugs(t, c, fmt.Sprintf("marker%d", i)))
		}
		requireDerivedBuiltFromRefs(t, repo, c)
	})
}

// listHookRepo calls onList, if set, each time the bug refs are listed.
type listHookRepo struct {
	repository.Repo
	onList func() error
}

func (r *listHookRepo) ListRefs(namespace string) (map[string]repository.Hash, error) {
	refs, err := r.Repo.ListRefs(namespace)
	if err == nil && namespace == bug.Namespace && r.onList != nil {
		err = r.onList()
	}
	return refs, err
}

// chanSource is a change source reporting what the test sends on it.
type chanSource struct {
	changes chan repository.Change
	// ctx is the context of the latest subscription
	ctx atomic.Pointer[context.Context]
}

func newChanSource() *chanSource {
	return &chanSource{changes: make(chan repository.Change)}
}

func (s *chanSource) Subscribe(ctx context.Context, _ []string) (<-chan repository.Change, error) {
	s.ctx.Store(&ctx)
	out := make(chan repository.Change)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case change := <-s.changes:
				select {
				case <-ctx.Done():
					return
				case out <- change:
				}
			}
		}
	}()
	return out, nil
}

// failingSource is a change source that can't work.
type failingSource struct{}

func (failingSource) Subscribe(context.Context, []string) (<-chan repository.Change, error) {
	return nil, errors.New("no notification here")
}

func TestSubCacheWorker(t *testing.T) {
	// open opens a cache on repo following source, with a user identity set
	open := func(t *testing.T, repo repository.Repo, source repository.ChangeSource) (*RepoCache, *IdentityCache) {
		t.Helper()
		c, err := NewRepoCacheNoEvents(repo, source)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, c.Close()) })
		rene, err := c.Identities().New("René Descartes", "rene@descartes.fr")
		require.NoError(t, err)
		require.NoError(t, c.SetUserIdentity(t.Context(), rene))
		return c, rene
	}

	// setup opens a cache on a mock repo following a chanSource, and hooks the
	// listing of the bug refs. Run in a synctest bubble, time is fake, and
	// synctest.Wait returns once every goroutine of the bubble, the workers
	// included, is idle.
	setup := func(t *testing.T) (repository.Repo, *RepoCache, *IdentityCache, *listHookRepo, *chanSource) {
		t.Helper()
		repo := repository.NewMockRepo()
		hook := &listHookRepo{Repo: repo}
		source := newChanSource()
		c, rene := open(t, hook, source)
		synctest.Wait()
		return repo, c, rene, hook, source
	}
	createOutside := func(t *testing.T, repo repository.Repo, author identity.Interface) entity.Id {
		t.Helper()
		b, _, err := bug.Create(author, time.Now().Unix(), "title", "message", nil, nil)
		require.NoError(t, err)
		require.NoError(t, b.Commit(repo))
		return b.Id()
	}
	hasExcerpt := func(t *testing.T, c *RepoCache, id entity.Id) bool {
		t.Helper()
		_, err := c.Bugs().ResolveExcerpt(id)
		return err == nil
	}
	workerErr := func(c *RepoCache) error {
		c.bugs.muSyncWorker.Lock()
		defer c.bugs.muSyncWorker.Unlock()
		return c.bugs.syncWorkerErr
	}
	naming := func(ids ...entity.Id) repository.Change {
		keys := make([]string, len(ids))
		for i, id := range ids {
			keys[i] = id.String()
		}
		return repository.Change{Keys: map[string][]string{bug.Namespace: keys}}
	}

	t.Run("a change naming entities syncs only those", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			repo, c, rene, hook, source := setup(t)
			// atomic: the worker syncs on its own, which orders nothing with the test
			var lists atomic.Int32
			hook.onList = func() error { lists.Add(1); return nil }
			named := createOutside(t, repo, rene)
			other := createOutside(t, repo, rene)

			time.Sleep(syncInterval)
			source.changes <- naming(named)
			synctest.Wait()
			require.True(t, hasExcerpt(t, c, named))
			require.False(t, hasExcerpt(t, c, other))
			require.Zero(t, lists.Load())
		})
	})

	t.Run("an empty change syncs every entity", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			repo, c, rene, hook, source := setup(t)
			var lists atomic.Int32
			hook.onList = func() error { lists.Add(1); return nil }
			first := createOutside(t, repo, rene)
			second := createOutside(t, repo, rene)

			time.Sleep(syncInterval)
			source.changes <- repository.Change{}
			synctest.Wait()
			require.True(t, hasExcerpt(t, c, first))
			require.True(t, hasExcerpt(t, c, second))
			require.EqualValues(t, 1, lists.Load())
		})
	})

	t.Run("changes within the interval are synced together once it passed", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			repo, c, rene, hook, source := setup(t)
			var lists atomic.Int32
			hook.onList = func() error { lists.Add(1); return nil }
			first := createOutside(t, repo, rene)
			second := createOutside(t, repo, rene)
			third := createOutside(t, repo, rene)

			// the sync of Load just ended
			source.changes <- naming(first)
			source.changes <- naming(second)
			synctest.Wait()
			require.False(t, hasExcerpt(t, c, first))
			require.False(t, hasExcerpt(t, c, second))

			time.Sleep(syncInterval)
			synctest.Wait()
			require.True(t, hasExcerpt(t, c, first))
			require.True(t, hasExcerpt(t, c, second))
			require.False(t, hasExcerpt(t, c, third))
			require.Zero(t, lists.Load())

			// an empty change turns the next sync into a full one
			source.changes <- naming(first)
			source.changes <- repository.Change{}
			time.Sleep(syncInterval)
			synctest.Wait()
			require.True(t, hasExcerpt(t, c, third))
			require.EqualValues(t, 1, lists.Load())
		})
	})

	t.Run("keys of other namespaces or not entities are ignored", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			repo, c, rene, hook, source := setup(t)
			var lists atomic.Int32
			hook.onList = func() error { lists.Add(1); return nil }
			id := createOutside(t, repo, rene)

			time.Sleep(syncInterval)
			source.changes <- repository.Change{Keys: map[string][]string{
				bug.Namespace: {"not-an-id"},
				"elsewhere":   {id.String()},
			}}
			synctest.Wait()
			require.False(t, hasExcerpt(t, c, id))
			require.Zero(t, lists.Load())
		})
	})

	t.Run("reads start no sync", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			repo, c, rene, hook, _ := setup(t)
			var lists atomic.Int32
			hook.onList = func() error { lists.Add(1); return nil }
			id := createOutside(t, repo, rene)

			for range 3 {
				c.Bugs().AllIds()
				searchBugs(t, c, "title")
				time.Sleep(time.Hour)
			}
			synctest.Wait()
			require.Zero(t, lists.Load())
			require.False(t, hasExcerpt(t, c, id))
		})
	})

	t.Run("reads during a sync are not held", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			repo, c, rene, hook, source := setup(t)
			held, release := make(chan struct{}), make(chan struct{})
			hook.onList = func() error {
				close(held)
				<-release
				return nil
			}
			id := createOutside(t, repo, rene)

			time.Sleep(syncInterval)
			source.changes <- repository.Change{}
			<-held
			require.Empty(t, searchBugs(t, c, "title"))
			hook.onList = nil
			close(release)

			synctest.Wait()
			require.Equal(t, []entity.Id{id}, searchBugs(t, c, "title"))
		})
	})

	t.Run("a failed sync is kept, and retried at the next change", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			repo, c, rene, hook, source := setup(t)
			hook.onList = func() error { return fmt.Errorf("listing failed") }
			id := createOutside(t, repo, rene)

			time.Sleep(syncInterval)
			source.changes <- repository.Change{}
			synctest.Wait()
			require.ErrorContains(t, workerErr(c), "listing failed")

			hook.onList = nil
			time.Sleep(syncInterval)
			source.changes <- repository.Change{}
			synctest.Wait()
			require.NoError(t, workerErr(c))
			require.True(t, hasExcerpt(t, c, id))
		})
	})

	t.Run("entities of a failed sync are retried at the next change, whatever it names", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			repo, c, rene, _, source := setup(t)

			// a bug whose author isn't in the repository yet can't be read
			author, err := identity.NewIdentity(repo, "Blaise Pascal", "blaise@pascal.fr")
			require.NoError(t, err)
			failing := createOutside(t, repo, author)

			time.Sleep(syncInterval)
			source.changes <- naming(failing)
			synctest.Wait()
			require.Error(t, workerErr(c))
			require.False(t, hasExcerpt(t, c, failing))

			require.NoError(t, author.Commit(repo))
			other := createOutside(t, repo, rene)
			time.Sleep(syncInterval)
			source.changes <- naming(other)
			synctest.Wait()
			require.NoError(t, workerErr(c))
			require.True(t, hasExcerpt(t, c, failing))
			require.True(t, hasExcerpt(t, c, other))
		})
	})

	t.Run("observers are notified of changes made outside", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			repo, c, rene, _, source := setup(t)
			observer := &recordingObserver{}
			c.registerAllObservers("repo", observer)
			id := createOutside(t, repo, rene)

			time.Sleep(syncInterval)
			source.changes <- naming(id)
			synctest.Wait()
			require.Equal(t, []recordedEvent{{EntityEventCreated, id}}, observer.events())

			changeOutside(t, repo, rene, id, "markeroutside")
			time.Sleep(syncInterval)
			source.changes <- naming(id)
			synctest.Wait()
			require.Equal(t, []recordedEvent{{EntityEventCreated, id}, {EntityEventUpdated, id}}, observer.events())
		})
	})

	t.Run("Close ends the subscription", func(t *testing.T) {
		repo := repository.NewMockRepo()
		source := newChanSource()
		c, err := NewRepoCacheNoEvents(repo, source)
		require.NoError(t, err)
		ctx := *source.ctx.Load()
		require.NoError(t, ctx.Err())
		require.NoError(t, c.Close())
		require.Error(t, ctx.Err())
	})

	t.Run("a source that can't work is a warning", func(t *testing.T) {
		repo := repository.NewMockRepo()
		c, events := NewRepoCache(repo, failingSource{})
		var warnings []error
		for event := range events {
			require.NoError(t, event.Err)
			if event.Event == BuildEventWarning {
				warnings = append(warnings, event.Warning)
			}
		}
		t.Cleanup(func() { require.NoError(t, c.Close()) })
		require.Len(t, warnings, 1)
		require.ErrorContains(t, warnings[0], "no notification here")

		_, err := c.Identities().New("René Descartes", "rene@descartes.fr")
		require.NoError(t, err)
	})

	// with a real repository and its own sources, outside of a bubble as
	// database/sql and the sources run goroutines of their own: a bug modified
	// outside of the cache shows up with nothing reading
	for name, source := range map[string]func(gitDir string) repository.ChangeSource{
		"watch": repository.NewWatchSource,
		"poll": func(gitDir string) repository.ChangeSource {
			return repository.NewPollSource(gitDir, 50*time.Millisecond)
		},
	} {
		t.Run("a bug modified outside with GoGitRepo, "+name, func(t *testing.T) {
			repo := repotest.NewGoGitRepo(t, false)
			c, rene := open(t, repo, source(repo.GitDir))
			b, _, err := c.Bugs().New(t.Context(), "title", "message")
			require.NoError(t, err)
			require.NoError(t, b.Commit(t.Context()))

			other, err := bug.Read(t.Context(), repo, b.Id())
			require.NoError(t, err)
			_, _, err = bug.AddComment(other, rene, time.Now().Unix(), "markeroutside", nil, nil)
			require.NoError(t, err)
			require.NoError(t, other.Commit(repo))

			require.Eventually(t, func() bool {
				return lenComments(t, c, b.Id()) == 2
			}, 5*time.Second, 20*time.Millisecond)
			require.Equal(t, []entity.Id{b.Id()}, searchBugs(t, c, "markeroutside"))
			requireDerivedBuiltFromRefs(t, repo, c)
		})
	}

	// two caches on one repository, each on its own handle as two processes
	// would be: the edits of both survive and reach both, and closing one leaves
	// the other working
	t.Run("two caches on one repository with GoGitRepo", func(t *testing.T) {
		// reopen opens a handle on the repository of repo, sharing nothing in
		// memory with it
		reopen := func(t *testing.T, repo *repotest.GoGitRepo) repository.Repo {
			t.Helper()
			git, err := repository.OpenGoGitRepo(repo.GitDir)
			require.NoError(t, err)
			config := repository.NewGitConfig(gitconfig.Env{
				GitDir:       repo.GitDir,
				GlobalConfig: []string{filepath.Join(t.TempDir(), ".gitconfig")},
			})
			storage := repository.NewLocalStorage(filepath.Join(repo.GitDir, "git-bug"))
			handle := &repository.ComposedRepo{
				RepoConfig:  repository.NewRepoConfig(config),
				RepoKeyring: repository.NewRepoKeyring(keyring.NewArrayKeyring(nil)),
				RepoStorage: repository.NewRepoStorage(storage),
				RepoIndex:   repository.NewSqliteIndex(filepath.Join(storage.Root(), "index.db")),
				RepoData:    git,
				RepoBrowse:  git,
				RepoClock:   repository.NewPersistedClocks(storage),
			}
			t.Cleanup(func() {
				if err := handle.Close(); err != nil {
					t.Error(err)
				}
			})
			return handle
		}
		poll := func(gitDir string) repository.ChangeSource {
			return repository.NewPollSource(gitDir, 50*time.Millisecond)
		}
		hasComments := func(t *testing.T, c *RepoCache, id entity.Id, count int) {
			t.Helper()
			require.Eventually(t, func() bool {
				return lenComments(t, c, id) == count
			}, 5*time.Second, 20*time.Millisecond)
		}

		repo := repotest.NewGoGitRepo(t, false)
		first, _ := open(t, repo, poll(repo.GitDir))
		inFirst, _, err := first.Bugs().New(t.Context(), "title", "message")
		require.NoError(t, err)
		require.NoError(t, inFirst.Commit(t.Context()))
		id := inFirst.Id()

		otherRepo := reopen(t, repo)
		second, err := NewRepoCacheNoEvents(otherRepo, poll(repo.GitDir))
		require.NoError(t, err)
		// the user identity is in the configuration both handles read
		_, err = second.GetUserIdentity(t.Context())
		require.NoError(t, err)

		// each cache edits from its own loaded copy, the one of first possibly
		// not knowing yet of the edit of second
		inSecond, err := second.Bugs().Resolve(t.Context(), id)
		require.NoError(t, err)
		_, _, err = inSecond.AddComment(t.Context(), "markersecond")
		require.NoError(t, err)
		require.NoError(t, inSecond.Commit(t.Context()))
		_, _, err = inFirst.AddComment(t.Context(), "markerfirst")
		require.NoError(t, err)
		require.NoError(t, inFirst.Commit(t.Context()))

		for _, c := range []*RepoCache{first, second} {
			hasComments(t, c, id, 3)
			require.Equal(t, []entity.Id{id}, searchBugs(t, c, "markerfirst"))
			require.Equal(t, []entity.Id{id}, searchBugs(t, c, "markersecond"))
		}
		requireDerivedBuiltFromRefs(t, repo, first)
		requireDerivedBuiltFromRefs(t, otherRepo, second)

		require.NoError(t, second.Close())

		_, _, err = inFirst.AddComment(t.Context(), "markerafter")
		require.NoError(t, err)
		require.NoError(t, inFirst.Commit(t.Context()))
		hasComments(t, first, id, 4)
		require.Equal(t, []entity.Id{id}, searchBugs(t, first, "markerafter"))
		requireDerivedBuiltFromRefs(t, repo, first)
	})
}

// recordingObserver records the events it is notified of.
type recordingObserver struct {
	mu   sync.Mutex
	seen []recordedEvent
}

type recordedEvent struct {
	event EntityEventType
	id    entity.Id
}

func (o *recordingObserver) EntityEvent(event EntityEventType, _ string, _ string, id entity.Id) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.seen = append(o.seen, recordedEvent{event, id})
}

func (o *recordingObserver) events() []recordedEvent {
	o.mu.Lock()
	defer o.mu.Unlock()
	return slices.Clone(o.seen)
}

// BenchmarkSync measures a sync that finds nothing to do, of one entity and of
// every entity, and a commit, which syncs its entity, on a repository of 200
// random bugs.
func BenchmarkSync(b *testing.B) {
	repo := repotest.NewGoGitRepo(b, false)
	random_bugs.FillRepoWithSeed(repo, 200, 42)
	c, _ := newTestCacheWithUser(b, repo)

	ids := c.bugs.AllIds()
	require.NotEmpty(b, ids)
	id := ids[0]

	b.Run("sync one, nothing to do", func(b *testing.B) {
		for b.Loop() {
			require.NoError(b, c.bugs.sync(id))
		}
	})

	b.Run("sync all, nothing to do", func(b *testing.B) {
		for b.Loop() {
			require.NoError(b, c.bugs.syncAll(nil))
		}
	})

	b.Run("commit", func(b *testing.B) {
		view, err := c.bugs.Resolve(b.Context(), id)
		require.NoError(b, err)
		for b.Loop() {
			_, _, err := view.AddComment(b.Context(), "a comment")
			require.NoError(b, err)
			require.NoError(b, view.Commit(b.Context()))
		}
	})
}
