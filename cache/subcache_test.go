package cache

import (
	"encoding/gob"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/entities/bug"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/misc/random_bugs"
	"github.com/git-bug/git-bug/query"
	"github.com/git-bug/git-bug/repository"
)

// requireDerivedBuiltFromRefs checks that the cache recorded, for every entity,
// the commit its reference points to, and nothing else.
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
	}

	c.bugs.muMaps.RLock()
	check(bug.Namespace, c.bugs.builtFrom, len(c.bugs.excerpts))
	c.bugs.muMaps.RUnlock()

	c.identities.muMaps.RLock()
	check(identity.Namespace, c.identities.builtFrom, len(c.identities.excerpts))
	c.identities.muMaps.RUnlock()
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

	t.Run("rebuilt from git while a loaded copy has pending changes", func(t *testing.T) {
		repo := repository.CreateGoGitTestRepo(t, false)
		c, _ := newTestCacheWithUser(t, repo)
		b, _, err := c.Bugs().New("title", "message")
		require.NoError(t, err)
		_, _, err = b.AddComment("markerpending")
		require.NoError(t, err)

		for event := range c.bugs.Build() {
			require.NoError(t, event.Err)
		}
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
		index, err := repo.GetIndex(bug.Namespace)
		require.NoError(t, err)
		count, err := index.DocCount()
		require.NoError(t, err)
		require.Zero(t, count)
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

		require.NoError(t, bug.Remove(repo, b.Id()))
		event, err := c.bugs.refresh(b.Id())
		require.NoError(t, err)
		require.Equal(t, EntityEventRemoved, event)

		requireDerivedBuiltFromRefs(t, repo, c)
		require.NotContains(t, c.bugs.cached, b.Id())
		index, err := repo.GetIndex(bug.Namespace)
		require.NoError(t, err)
		count, err := index.DocCount()
		require.NoError(t, err)
		require.Zero(t, count)
	})

	t.Run("a search hit without excerpt is skipped", func(t *testing.T) {
		repo := repository.CreateGoGitTestRepo(t, false)
		c, _ := newTestCacheWithUser(t, repo)

		// the index is written before the excerpt is published
		index, err := repo.GetIndex(bug.Namespace)
		require.NoError(t, err)
		require.NoError(t, index.IndexOne("notyetpublished", []string{"markerahead"}))

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
		read, err := bug.Read(repo, b.Id())
		require.NoError(t, err)

		require.NoError(t, c.Close())

		event, err := c.bugs.publishDerived(c.bugs.newCached(read))
		require.NoError(t, err)
		require.Zero(t, event)
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
