package cache

import (
	"encoding/gob"
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/util"
	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/entities/bug"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/misc/random_bugs"
	"github.com/git-bug/git-bug/query"
	"github.com/git-bug/git-bug/repository"
)

// requireDerivedBuiltFromRefs checks that the cache and the index recorded, for
// every entity, the commit its reference points to, and nothing else.
func requireDerivedBuiltFromRefs(t *testing.T, repo repository.ClockedRepo, c *RepoCache) {
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

		index, err := repo.GetIndex(namespace)
		require.NoError(t, err)
		indexBuiltFrom, err := index.BuiltFrom()
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

func searchBugs(t *testing.T, c *RepoCache, term string) []entity.Id {
	t.Helper()
	q, err := query.Parse(term)
	require.NoError(t, err)
	res, err := c.Bugs().Query(q)
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
func newTestCacheWithUser(t *testing.T, repo repository.TestedRepo) (*RepoCache, *IdentityCache) {
	t.Helper()
	c := createTestRepoCacheNoEvents(t, repo)
	rene, err := c.Identities().New("René Descartes", "rene@descartes.fr")
	require.NoError(t, err)
	require.NoError(t, c.SetUserIdentity(rene))
	return c, rene
}

// countingRepo counts the index batches applied through it.
type countingRepo struct {
	repository.ClockedRepo
	batches int
}

func (r *countingRepo) GetIndex(name string) (repository.Index, error) {
	index, err := r.ClockedRepo.GetIndex(name)
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

func (b countingBatch) Apply() error {
	b.repo.batches++
	return b.IndexBatch.Apply()
}

// unreadableRecordRepo has the index record fail to read, until the index is
// cleared.
type unreadableRecordRepo struct {
	repository.ClockedRepo
	unreadable bool
}

func (r *unreadableRecordRepo) GetIndex(name string) (repository.Index, error) {
	index, err := r.ClockedRepo.GetIndex(name)
	if err != nil {
		return nil, err
	}
	return unreadableRecordIndex{Index: index, repo: r}, nil
}

type unreadableRecordIndex struct {
	repository.Index
	repo *unreadableRecordRepo
}

func (i unreadableRecordIndex) BuiltFrom() (map[string]repository.Hash, error) {
	if i.repo.unreadable {
		return nil, fmt.Errorf("malformed index record")
	}
	return i.Index.BuiltFrom()
}

func (i unreadableRecordIndex) Clear() error {
	i.repo.unreadable = false
	return i.Index.Clear()
}

func TestSubCacheDerived(t *testing.T) {
	t.Run("built from git", func(t *testing.T) {
		repo := repository.CreateGoGitTestRepo(t, false)
		random_bugs.FillRepoWithSeed(repo, 5, 42)

		c := createTestRepoCacheNoEvents(t, repo)
		requireDerivedBuiltFromRefs(t, repo, c)

		// only the derived state is built, entities are not kept in memory, except
		// the identities resolved as the authors of bugs, which can be evicted
		require.Empty(t, c.bugs.cached)
		require.Equal(t, c.identities.lru.Len(), len(c.identities.cached))
	})

	t.Run("built from git, over several index batches", func(t *testing.T) {
		repo := repository.CreateGoGitTestRepo(t, false)
		// more than twice the batch size, and not a multiple of it
		random_bugs.FillRepoWithSeed(repo, 200, 42)

		c := createTestRepoCacheNoEvents(t, repo)
		requireDerivedBuiltFromRefs(t, repo, c)
	})

	t.Run("created", func(t *testing.T) {
		repo := repository.CreateGoGitTestRepo(t, false)
		c, _ := newTestCacheWithUser(t, repo)

		_, _, err := c.Bugs().New("title", "message")
		require.NoError(t, err)
		requireDerivedBuiltFromRefs(t, repo, c)
	})

	t.Run("pending changes are visible only once committed", func(t *testing.T) {
		repo := repository.CreateGoGitTestRepo(t, false)
		c, _ := newTestCacheWithUser(t, repo)
		b, _, err := c.Bugs().New("title", "message")
		require.NoError(t, err)

		obs := &observer{}
		require.NoError(t, c.registerObserver("repotest", bug.Typename, obs))

		_, _, err = b.AddComment("markerpending")
		require.NoError(t, err)
		require.Equal(t, 1, lenComments(t, c, b.Id()))
		require.Empty(t, searchBugs(t, c, "markerpending"))
		require.Empty(t, obs.updated)
		requireDerivedBuiltFromRefs(t, repo, c)

		require.NoError(t, b.Commit())
		require.Equal(t, 2, lenComments(t, c, b.Id()))
		require.Equal(t, []entity.Id{b.Id()}, searchBugs(t, c, "markerpending"))
		require.Len(t, obs.updated, 1)
		requireDerivedBuiltFromRefs(t, repo, c)

		// nothing to commit, nothing changes
		require.NoError(t, b.CommitAsNeeded())
		require.Len(t, obs.updated, 1)
	})

	t.Run("identity changes are visible only once committed", func(t *testing.T) {
		repo := repository.CreateGoGitTestRepo(t, false)
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
		repo := repository.CreateGoGitTestRepo(t, false)

		c, err := NewRepoCacheNoEvents(repo)
		require.NoError(t, err)
		rene, err := c.Identities().New("René Descartes", "rene@descartes.fr")
		require.NoError(t, err)
		require.NoError(t, c.SetUserIdentity(rene))
		b, _, err := c.Bugs().New("title", "message")
		require.NoError(t, err)
		_, _, err = b.AddComment("comment")
		require.NoError(t, err)
		require.NoError(t, b.Commit())
		require.NoError(t, c.Close())

		c = createTestRepoCacheNoEvents(t, repo)
		require.Empty(t, c.bugs.cached)
		require.Empty(t, c.identities.cached)
		requireDerivedBuiltFromRefs(t, repo, c)
	})

	t.Run("an index without record is repaired on load", func(t *testing.T) {
		repo := repository.CreateGoGitTestRepo(t, false)

		c, err := NewRepoCacheNoEvents(repo)
		require.NoError(t, err)
		rene, err := c.Identities().New("René Descartes", "rene@descartes.fr")
		require.NoError(t, err)
		require.NoError(t, c.SetUserIdentity(rene))
		_, _, err = c.Bugs().New("title", "message")
		require.NoError(t, err)
		require.NoError(t, c.Close())

		// what an index written before it recorded anything, or a lost one,
		// looks like to the cache
		index, err := repo.GetIndex(bug.Namespace)
		require.NoError(t, err)
		require.NoError(t, index.Clear())

		c = createTestRepoCacheNoEvents(t, repo)
		requireDerivedBuiltFromRefs(t, repo, c)
	})

	t.Run("a commit recorded without its excerpt is repaired on load", func(t *testing.T) {
		repo := repository.CreateGoGitTestRepo(t, false)

		c, err := NewRepoCacheNoEvents(repo)
		require.NoError(t, err)
		rene, err := c.Identities().New("René Descartes", "rene@descartes.fr")
		require.NoError(t, err)
		require.NoError(t, c.SetUserIdentity(rene))
		b, _, err := c.Bugs().New("title", "message")
		require.NoError(t, err)

		c.bugs.muMaps.Lock()
		delete(c.bugs.excerpts, b.Id())
		c.bugs.muMaps.Unlock()
		require.NoError(t, c.bugs.write())
		require.NoError(t, c.Close())

		c = createTestRepoCacheNoEvents(t, repo)
		require.Equal(t, 1, lenComments(t, c, b.Id()))
		requireDerivedBuiltFromRefs(t, repo, c)
	})

	t.Run("an index ahead of the excerpts is repaired on load", func(t *testing.T) {
		repo := repository.CreateGoGitTestRepo(t, false)

		c, err := NewRepoCacheNoEvents(repo)
		require.NoError(t, err)
		rene, err := c.Identities().New("René Descartes", "rene@descartes.fr")
		require.NoError(t, err)
		require.NoError(t, c.SetUserIdentity(rene))
		b, _, err := c.Bugs().New("title", "message")
		require.NoError(t, err)

		excerptFile := filepath.Join("cache", bug.Namespace)
		older, err := util.ReadFile(repo.LocalStorage(), excerptFile)
		require.NoError(t, err)

		_, _, err = b.AddComment("comment")
		require.NoError(t, err)
		require.NoError(t, b.Commit())
		require.NoError(t, c.Close())

		// what stopping after the index update, but before writing the excerpts,
		// leaves: as many entries on both sides, built from different commits
		require.NoError(t, util.WriteFile(repo.LocalStorage(), excerptFile, older, 0644))

		c = createTestRepoCacheNoEvents(t, repo)
		requireDerivedBuiltFromRefs(t, repo, c)
		require.Equal(t, 2, lenComments(t, c, b.Id()))
	})

	// counted has a subcache count the entities it reads, and the index batches it
	// applies
	counted := func(sc *RepoCacheBug) (reads *int, batches *int) {
		reads = new(int)
		read := sc.actions.ReadWithResolver
		sc.actions.ReadWithResolver = func(repo repository.ClockedRepo, resolvers entity.Resolvers, id entity.Id) (*bug.Bug, error) {
			*reads++
			return read(repo, resolvers, id)
		}
		repo := &countingRepo{ClockedRepo: sc.repo}
		sc.repo = repo
		return reads, &repo.batches
	}

	// changeOutside adds a comment to a bug without the cache knowing
	changeOutside := func(t *testing.T, repo repository.ClockedRepo, author identity.Interface, id entity.Id, message string) {
		t.Helper()
		b, err := bug.Read(repo, id)
		require.NoError(t, err)
		_, _, err = bug.AddComment(b, author, time.Now().Unix(), message, nil, nil)
		require.NoError(t, err)
		require.NoError(t, b.Commit(repo))
	}

	t.Run("a change from outside is synced, reading only that entity", func(t *testing.T) {
		repo := repository.CreateGoGitTestRepo(t, false)
		c, rene := newTestCacheWithUser(t, repo)
		var ids []entity.Id
		for i := 0; i < 3; i++ {
			b, _, err := c.Bugs().New("title", "message")
			require.NoError(t, err)
			ids = append(ids, b.Id())
		}

		obs := &observer{}
		require.NoError(t, c.registerObserver("repotest", bug.Typename, obs))
		reads, batches := counted(c.bugs)

		require.NoError(t, c.bugs.syncAll(nil))
		require.Zero(t, *reads)
		require.Zero(t, *batches)

		changeOutside(t, repo, rene, ids[1], "markeroutside")
		require.NoError(t, c.bugs.syncAll(nil))
		require.Equal(t, 1, *reads)
		require.Equal(t, 1, *batches)
		require.Equal(t, []observerEvent{{bug.Typename, ids[1]}}, obs.updated)
		require.Equal(t, 2, lenComments(t, c, ids[1]))
		require.Equal(t, []entity.Id{ids[1]}, searchBugs(t, c, "markeroutside"))
		requireDerivedBuiltFromRefs(t, repo, c)
	})

	t.Run("many changes from outside are applied in a few batches", func(t *testing.T) {
		repo := repository.CreateGoGitTestRepo(t, false)
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
		reads, batches := counted(c.bugs)

		require.NoError(t, c.bugs.syncAll(nil))
		require.Equal(t, count, *reads)
		require.Equal(t, 3, *batches)
		require.Len(t, obs.created, count)
		requireDerivedBuiltFromRefs(t, repo, c)
	})

	t.Run("an index behind is repaired, the excerpts left as they are", func(t *testing.T) {
		repo := repository.CreateGoGitTestRepo(t, false)
		c, _ := newTestCacheWithUser(t, repo)
		b, _, err := c.Bugs().New("title", "markerindexed")
		require.NoError(t, err)

		excerptFile := filepath.Join("cache", bug.Namespace)
		before, err := util.ReadFile(repo.LocalStorage(), excerptFile)
		require.NoError(t, err)

		index, err := repo.GetIndex(bug.Namespace)
		require.NoError(t, err)
		require.NoError(t, index.Clear())

		obs := &observer{}
		require.NoError(t, c.registerObserver("repotest", bug.Typename, obs))
		_, batches := counted(c.bugs)

		require.NoError(t, c.bugs.syncAll(nil))
		require.Equal(t, 1, *batches)
		require.Equal(t, []entity.Id{b.Id()}, searchBugs(t, c, "markerindexed"))
		requireDerivedBuiltFromRefs(t, repo, c)

		// the excerpts didn't change, nor did the file holding them
		require.Equal(t, &observer{}, obs)
		after, err := util.ReadFile(repo.LocalStorage(), excerptFile)
		require.NoError(t, err)
		require.Equal(t, before, after)
	})

	t.Run("excerpts behind are repaired, the index left as it is", func(t *testing.T) {
		repo := repository.CreateGoGitTestRepo(t, false)
		c, _ := newTestCacheWithUser(t, repo)
		b, _, err := c.Bugs().New("title", "message")
		require.NoError(t, err)

		older := c.bugs.excerpts[b.Id()]
		olderCommit := c.bugs.builtFrom[b.Id()]
		_, _, err = b.AddComment("comment")
		require.NoError(t, err)
		require.NoError(t, b.Commit())

		// put back the excerpt from before the commit
		c.bugs.muMaps.Lock()
		c.bugs.excerpts[b.Id()] = older
		c.bugs.builtFrom[b.Id()] = olderCommit
		c.bugs.muMaps.Unlock()

		obs := &observer{}
		require.NoError(t, c.registerObserver("repotest", bug.Typename, obs))
		reads, batches := counted(c.bugs)

		require.NoError(t, c.bugs.syncAll(nil))
		require.Equal(t, 1, *reads)
		require.Zero(t, *batches)
		require.Equal(t, []observerEvent{{bug.Typename, b.Id()}}, obs.updated)
		require.Equal(t, 2, lenComments(t, c, b.Id()))
		requireDerivedBuiltFromRefs(t, repo, c)
	})

	t.Run("changes fetched outside git-bug are picked up on load", func(t *testing.T) {
		repoA, repoB, _ := repository.SetupGoGitReposAndRemote(t)
		cacheA, _ := newTestCacheWithUser(t, repoA)
		b, _, err := cacheA.Bugs().New("title", "message")
		require.NoError(t, err)
		_, err = cacheA.Push("origin")
		require.NoError(t, err)

		cacheB, err := NewRepoCacheNoEvents(repoB)
		require.NoError(t, err)
		require.NoError(t, cacheB.Pull("origin"))
		require.Equal(t, 1, lenComments(t, cacheB, b.Id()))
		require.NoError(t, cacheB.Close())

		// A changes the bug, and B gets it with no git-bug command: what a fetch
		// into the local refs, or a push into B, does
		bA, err := cacheA.Bugs().Resolve(b.Id())
		require.NoError(t, err)
		_, _, err = bA.AddComment("markeroutside")
		require.NoError(t, err)
		require.NoError(t, bA.Commit())
		_, err = cacheA.Push("origin")
		require.NoError(t, err)

		for _, namespace := range []string{identity.Namespace, bug.Namespace} {
			_, err = repoB.FetchRefs("origin", namespace)
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
		repoA, repoB, _ := repository.SetupGoGitReposAndRemote(t)
		cacheA, _ := newTestCacheWithUser(t, repoA)
		cacheB := createTestRepoCacheNoEvents(t, repoB)

		// as new
		b, _, err := cacheA.Bugs().New("title", "message")
		require.NoError(t, err)
		_, err = cacheA.Push("origin")
		require.NoError(t, err)
		require.NoError(t, cacheB.Pull("origin"))
		requireDerivedBuiltFromRefs(t, repoB, cacheB)

		// as updated
		_, _, err = b.AddComment("markerpulled")
		require.NoError(t, err)
		require.NoError(t, b.Commit())
		_, err = cacheA.Push("origin")
		require.NoError(t, err)
		require.NoError(t, cacheB.Pull("origin"))
		require.Equal(t, []entity.Id{b.Id()}, searchBugs(t, cacheB, "markerpulled"))
		requireDerivedBuiltFromRefs(t, repoB, cacheB)
	})

	t.Run("removed", func(t *testing.T) {
		repo := repository.CreateGoGitTestRepo(t, false)
		c, _ := newTestCacheWithUser(t, repo)
		b1, _, err := c.Bugs().New("title", "message")
		require.NoError(t, err)
		_, _, err = c.Bugs().New("title", "message")
		require.NoError(t, err)

		require.NoError(t, c.Bugs().Remove(b1.Id().String()))
		requireDerivedBuiltFromRefs(t, repo, c)

		require.NoError(t, c.RemoveAll())
		requireDerivedBuiltFromRefs(t, repo, c)
	})

	t.Run("a loaded copy without derived state is dropped once removed", func(t *testing.T) {
		repo := repository.CreateGoGitTestRepo(t, false)
		c, rene := newTestCacheWithUser(t, repo)

		// created outside, and loaded before any sync
		b, _, err := bug.Create(rene, time.Now().Unix(), "title", "message", nil, nil)
		require.NoError(t, err)
		require.NoError(t, b.Commit(repo))
		_, err = c.Bugs().Resolve(b.Id())
		require.NoError(t, err)

		require.NoError(t, c.RemoveAll())
		require.NotContains(t, c.bugs.cached, b.Id())
		_, err = c.Bugs().Resolve(b.Id())
		require.True(t, entity.IsErrNotFound(err))
		requireDerivedBuiltFromRefs(t, repo, c)
	})

	t.Run("a missing author is not a removal", func(t *testing.T) {
		repo := repository.CreateGoGitTestRepo(t, false)
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
		index, err := repo.GetIndex(bug.Namespace)
		require.NoError(t, err)
		require.NoError(t, index.Clear())

		obs := &observer{}
		require.NoError(t, c.registerObserver("repotest", bug.Typename, obs))

		require.Error(t, c.bugs.syncAll(nil))
		require.Empty(t, obs.removed)
		_, err = c.Bugs().ResolveExcerpt(b.Id())
		require.NoError(t, err)
	})

	t.Run("an unreadable index record has a single sync repair every entity", func(t *testing.T) {
		repo := repository.CreateGoGitTestRepo(t, false)
		c, _ := newTestCacheWithUser(t, repo)
		b1, _, err := c.Bugs().New("title", "markerfirst")
		require.NoError(t, err)
		b2, _, err := c.Bugs().New("title", "markersecond")
		require.NoError(t, err)

		broken := &unreadableRecordRepo{ClockedRepo: c.bugs.repo, unreadable: true}
		c.bugs.repo = broken

		require.NoError(t, c.bugs.sync(b1.Id()))
		require.False(t, broken.unreadable)
		require.Equal(t, []entity.Id{b1.Id()}, searchBugs(t, c, "markerfirst"))
		require.Equal(t, []entity.Id{b2.Id()}, searchBugs(t, c, "markersecond"))
		requireDerivedBuiltFromRefs(t, repo, c)
	})

	t.Run("an evicted copy keeps working, and its commits are followed", func(t *testing.T) {
		repo := repository.CreateGoGitTestRepo(t, false)
		c, _ := newTestCacheWithUser(t, repo)
		c.setCacheSize(1)

		evicted, _, err := c.Bugs().New("title", "message")
		require.NoError(t, err)
		_, _, err = c.Bugs().New("title", "message")
		require.NoError(t, err)
		require.NotContains(t, c.bugs.cached, evicted.Id())

		// a new copy, loaded as the evicted one is still in use
		loaded, err := c.Bugs().Resolve(evicted.Id())
		require.NoError(t, err)
		require.NotSame(t, evicted, loaded)

		_, _, err = evicted.AddComment("comment")
		require.NoError(t, err)
		require.NoError(t, evicted.Commit())
		require.Equal(t, 2, lenComments(t, c, evicted.Id()))
		requireDerivedBuiltFromRefs(t, repo, c)

		// the stale copy committing nothing leaves the derived state alone
		require.NoError(t, loaded.CommitAsNeeded())
		require.Equal(t, 2, lenComments(t, c, evicted.Id()))
		requireDerivedBuiltFromRefs(t, repo, c)

		// and its commits are rejected
		_, _, err = loaded.AddComment("stale")
		require.NoError(t, err)
		require.ErrorIs(t, loaded.Commit(), repository.ErrRefChanged)
		requireDerivedBuiltFromRefs(t, repo, c)
	})

	t.Run("a new entity evicted right away is still usable", func(t *testing.T) {
		repo := repository.CreateGoGitTestRepo(t, false)
		c, _ := newTestCacheWithUser(t, repo)
		c.setCacheSize(1)

		// pending changes make the older entity impossible to evict
		pending, _, err := c.Bugs().New("title", "message")
		require.NoError(t, err)
		_, _, err = pending.AddComment("pending")
		require.NoError(t, err)

		b, _, err := c.Bugs().New("title", "message")
		require.NoError(t, err)
		require.NotContains(t, c.bugs.cached, b.Id())
		requireDerivedBuiltFromRefs(t, repo, c)

		_, _, err = b.AddComment("comment")
		require.NoError(t, err)
		require.NoError(t, b.Commit())
		require.Equal(t, 2, lenComments(t, c, b.Id()))
		requireDerivedBuiltFromRefs(t, repo, c)
	})

	t.Run("synced from git while a loaded copy has pending changes", func(t *testing.T) {
		repo := repository.CreateGoGitTestRepo(t, false)
		c, _ := newTestCacheWithUser(t, repo)
		b, _, err := c.Bugs().New("title", "message")
		require.NoError(t, err)
		_, _, err = b.AddComment("markerpending")
		require.NoError(t, err)

		index, err := repo.GetIndex(bug.Namespace)
		require.NoError(t, err)
		require.NoError(t, index.Clear())
		require.NoError(t, c.bugs.syncAll(nil))
		require.Equal(t, 1, lenComments(t, c, b.Id()))
		require.Empty(t, searchBugs(t, c, "markerpending"))
		requireDerivedBuiltFromRefs(t, repo, c)

		// the loaded copy is untouched, and its commit is followed
		require.True(t, b.NeedCommit())
		require.NoError(t, b.Commit())
		require.Equal(t, []entity.Id{b.Id()}, searchBugs(t, c, "markerpending"))
		requireDerivedBuiltFromRefs(t, repo, c)
	})

	t.Run("pending changes on a loaded copy don't hold back a commit", func(t *testing.T) {
		repo := repository.CreateGoGitTestRepo(t, false)
		c, rene := newTestCacheWithUser(t, repo)
		b, _, err := c.Bugs().New("title", "message")
		require.NoError(t, err)
		_, _, err = b.AddComment("markerpending")
		require.NoError(t, err)

		// another copy commits, and the callback comes while the loaded copy
		// has pending changes
		other, err := bug.Read(repo, b.Id())
		require.NoError(t, err)
		_, _, err = bug.AddComment(other, rene, time.Now().Unix(), "markercommitted", nil, nil)
		require.NoError(t, err)
		require.NoError(t, other.Commit(repo))
		require.NoError(t, c.bugs.onCommit(b.Id()))

		require.Equal(t, []entity.Id{b.Id()}, searchBugs(t, c, "markercommitted"))
		require.Empty(t, searchBugs(t, c, "markerpending"))
		require.True(t, b.NeedCommit())
		requireDerivedBuiltFromRefs(t, repo, c)
	})

	t.Run("a repeated or late callback changes nothing", func(t *testing.T) {
		repo := repository.CreateGoGitTestRepo(t, false)
		c, _ := newTestCacheWithUser(t, repo)
		b, _, err := c.Bugs().New("title", "message")
		require.NoError(t, err)
		_, _, err = b.AddComment("comment")
		require.NoError(t, err)
		require.NoError(t, b.Commit())

		obs := &observer{}
		require.NoError(t, c.registerObserver("repotest", bug.Typename, obs))

		require.NoError(t, c.bugs.onCommit(b.Id()))
		require.Empty(t, obs.updated)
		require.Equal(t, 2, lenComments(t, c, b.Id()))
		requireDerivedBuiltFromRefs(t, repo, c)
	})

	t.Run("a late callback doesn't resurrect a removed entity", func(t *testing.T) {
		repo := repository.CreateGoGitTestRepo(t, false)
		c, _ := newTestCacheWithUser(t, repo)
		b, _, err := c.Bugs().New("title", "message")
		require.NoError(t, err)

		require.NoError(t, c.Bugs().Remove(b.Id().String()))
		require.NoError(t, c.bugs.onCommit(b.Id()))

		requireDerivedBuiltFromRefs(t, repo, c)
	})

	t.Run("an entity removed while being added is not registered", func(t *testing.T) {
		repo := repository.CreateGoGitTestRepo(t, false)
		c, _ := newTestCacheWithUser(t, repo)
		b, _, err := c.Bugs().New("title", "message")
		require.NoError(t, err)
		read, err := bug.Read(repo, b.Id())
		require.NoError(t, err)
		require.NoError(t, c.Bugs().Remove(b.Id().String()))

		obs := &observer{}
		require.NoError(t, c.registerObserver("repotest", bug.Typename, obs))

		_, err = c.bugs.add(read)
		require.Error(t, err)
		require.NotContains(t, c.bugs.cached, b.Id())
		require.Empty(t, obs.created)
		requireDerivedBuiltFromRefs(t, repo, c)
	})

	t.Run("an entity removed outside the cache is dropped", func(t *testing.T) {
		repo := repository.CreateGoGitTestRepo(t, false)
		c, _ := newTestCacheWithUser(t, repo)
		b, _, err := c.Bugs().New("title", "message")
		require.NoError(t, err)

		obs := &observer{}
		require.NoError(t, c.registerObserver("repotest", bug.Typename, obs))

		require.NoError(t, bug.Remove(repo, b.Id()))
		require.NoError(t, c.bugs.sync(b.Id()))
		require.Equal(t, []observerEvent{{bug.Typename, b.Id()}}, obs.removed)

		requireDerivedBuiltFromRefs(t, repo, c)
		require.NotContains(t, c.bugs.cached, b.Id())
	})

	t.Run("a search hit without excerpt is skipped", func(t *testing.T) {
		repo := repository.CreateGoGitTestRepo(t, false)
		c, _ := newTestCacheWithUser(t, repo)

		// the index is written before the excerpt is published
		index, err := repo.GetIndex(bug.Namespace)
		require.NoError(t, err)
		batch := index.NewBatch()
		require.NoError(t, batch.Set("notyetpublished", []string{"markerahead"}, "1111111111111111111111111111111111111111"))
		require.NoError(t, batch.Apply())

		require.Empty(t, searchBugs(t, c, "markerahead"))
	})

	t.Run("nothing is published once closed", func(t *testing.T) {
		repo := repository.CreateGoGitTestRepo(t, false)
		c, err := NewRepoCacheNoEvents(repo)
		require.NoError(t, err)
		rene, err := c.Identities().New("René Descartes", "rene@descartes.fr")
		require.NoError(t, err)
		require.NoError(t, c.SetUserIdentity(rene))
		b, _, err := c.Bugs().New("title", "message")
		require.NoError(t, err)
		// changed outside the cache, which only notices on a sync
		read, err := bug.Read(repo, b.Id())
		require.NoError(t, err)
		_, _, err = bug.AddComment(read, rene, time.Now().Unix(), "comment", nil, nil)
		require.NoError(t, err)
		require.NoError(t, read.Commit(repo))

		require.NoError(t, c.Close())

		require.NoError(t, c.bugs.sync(b.Id()))
		require.Nil(t, c.bugs.builtFrom)
	})

	t.Run("concurrent commits", func(t *testing.T) {
		repo := repository.CreateGoGitTestRepo(t, false)
		c, _ := newTestCacheWithUser(t, repo)
		b, _, err := c.Bugs().New("title", "message")
		require.NoError(t, err)

		const count = 10
		var wg sync.WaitGroup
		errs := make(chan error, count)
		for i := 0; i < count; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if _, _, err := b.AddComment(fmt.Sprintf("marker%d", i)); err != nil {
					errs <- err
					return
				}
				errs <- b.CommitAsNeeded()
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

func TestSubCacheWriteNeverTorn(t *testing.T) {
	repo := repository.CreateGoGitTestRepo(t, false)
	random_bugs.FillRepoWithSeed(repo, 50, 42)
	c := createTestRepoCacheNoEvents(t, repo)

	// read the file the way another process would: outside of the cache locks
	readRaw := func() error {
		var f billy.File
		err := retryOnWindows(func() (err error) {
			f, err = repo.LocalStorage().Open(filepath.Join("cache", bug.Namespace))
			return err
		})
		if err != nil {
			return err
		}
		defer f.Close()
		aux := struct {
			Version   uint
			Excerpts  map[entity.Id]*BugExcerpt
			BuiltFrom map[entity.Id]repository.Hash
		}{}
		return gob.NewDecoder(f).Decode(&aux)
	}

	const writes = 200
	done := make(chan error)
	go func() {
		for range writes {
			if err := c.bugs.write(); err != nil {
				done <- err
				return
			}
		}
		done <- nil
	}()

	for {
		select {
		case err := <-done:
			require.NoError(t, err)

			// no temporary file is left behind
			files, err := repo.LocalStorage().ReadDir("cache")
			require.NoError(t, err)
			var names []string
			for _, fi := range files {
				names = append(names, fi.Name())
			}
			require.ElementsMatch(t, []string{bug.Namespace, identity.Namespace}, names)
			return
		default:
			require.NoError(t, readRaw())
		}
	}
}
