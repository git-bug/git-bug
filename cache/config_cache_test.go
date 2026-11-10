package cache

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/entities/config"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/repository"
	"github.com/git-bug/git-bug/repository/repotest"
)

// setUser creates an identity in the cache and makes it the user.
func setUser(t *testing.T, c *RepoCache, name string) *IdentityCache {
	t.Helper()
	i, err := c.Identities().New(t.Context(), name, name+"@example.com")
	require.NoError(t, err)
	require.NoError(t, c.SetUserIdentity(t.Context(), i))
	return i
}

func TestConfigCache(t *testing.T) {
	t.Run("created by the first change", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)
		c := createTestRepoCacheNoEvents(t, repo)
		setUser(t, c, "rene")
		obs := &observer{}
		c.ProjectConfig().RegisterObserver("repo", obs)

		// no config yet: every setting reads as its default
		require.Empty(t, c.ProjectConfig().Snapshot().Operations)
		_, err := config.Read(t.Context(), repo)
		require.True(t, entity.IsErrNotFound(err))

		view := c.ProjectConfig().View()
		_, err = view.RepoSetDescription(t.Context(), "a bug tracker")
		require.NoError(t, err)
		_, err = view.LabelsSet(t.Context(), "crash", config.LabelsDefinition{Color: config.SetField("#f44336")})
		require.NoError(t, err)

		// staged: the view sees it, nobody else does, and there is no config yet
		require.Equal(t, "a bug tracker", view.Snapshot().Repo.Description())
		require.Equal(t, entity.UnsetId, view.Id())
		require.Empty(t, c.ProjectConfig().Snapshot().Repo.Description())

		require.NoError(t, view.Commit(t.Context()))
		snap := c.ProjectConfig().Snapshot()
		require.Equal(t, "a bug tracker", snap.Repo.Description())
		require.Equal(t, []common.Label{"crash"}, snap.Labels.Names())
		require.Equal(t, []observerEvent{{config.Typename, view.Id()}}, obs.created)

		// a later change updates the same config
		view = c.ProjectConfig().View()
		_, err = view.LabelsAllow(t.Context(), "crash")
		require.NoError(t, err)
		require.NoError(t, view.Commit(t.Context()))
		require.Equal(t, []observerEvent{{config.Typename, view.Id()}}, obs.updated)

		read, err := config.Read(t.Context(), repo)
		require.NoError(t, err)
		require.Equal(t, view.Id(), read.Id())
		require.Len(t, read.Operations(), 4)
		require.Equal(t, []common.Label{"crash"}, read.Compile().Labels.AllowedNames())

		// read back when the cache is opened again
		require.NoError(t, c.Close())
		c = createTestRepoCacheNoEvents(t, repo)
		require.Equal(t, "a bug tracker", c.ProjectConfig().Snapshot().Repo.Description())
	})

	t.Run("a failed change stages nothing", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)
		c := createTestRepoCacheNoEvents(t, repo)
		setUser(t, c, "rene")

		view := c.ProjectConfig().View()
		_, err := view.LabelsRestrict(t.Context())
		require.Error(t, err)
		require.False(t, view.NeedCommit())

		_, err = view.LabelsSet(t.Context(), "", config.LabelsDefinition{})
		require.Error(t, err)
		require.False(t, view.NeedCommit())
	})

	t.Run("created concurrently", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)
		c := createTestRepoCacheNoEvents(t, repo)
		rene := setUser(t, c, "rene")
		obs := &observer{}
		c.ProjectConfig().RegisterObserver("repo", obs)

		// this view starts with no config
		view := c.ProjectConfig().View()

		// meanwhile, another process creates one
		other, _, err := config.Create(repo, rene, time.Now().Unix())
		require.NoError(t, err)
		_, err = config.LabelsSet(other, repo, rene, time.Now().Unix(), "other", config.LabelsDefinition{})
		require.NoError(t, err)
		require.NoError(t, other.Commit(t.Context(), repo))

		// the change is committed on top of that one
		_, err = view.LabelsSet(t.Context(), "mine", config.LabelsDefinition{})
		require.NoError(t, err)
		require.NoError(t, view.Commit(t.Context()))
		require.Equal(t, other.Id(), view.Id())

		snap := c.ProjectConfig().Snapshot()
		require.Equal(t, []common.Label{"mine", "other"}, snap.Labels.Names())
		read, err := config.Read(t.Context(), repo)
		require.NoError(t, err)
		require.Equal(t, other.Id(), read.Id())
		require.Len(t, read.Operations(), 3)
		require.Equal(t, []observerEvent{{config.Typename, other.Id()}}, obs.created)
	})

	t.Run("replaced by another config", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)
		c := createTestRepoCacheNoEvents(t, repo)
		rene := setUser(t, c, "rene")
		obs := &observer{}
		c.ProjectConfig().RegisterObserver("repo", obs)

		view := c.ProjectConfig().View()
		_, err := view.RepoSetDescription(t.Context(), "first")
		require.NoError(t, err)
		require.NoError(t, view.Commit(t.Context()))
		first := view.Id()

		// a change staged on the loaded config
		staged := c.ProjectConfig().View()
		_, err = staged.LabelsSet(t.Context(), "mine", config.LabelsDefinition{})
		require.NoError(t, err)

		// meanwhile, another process removes it and creates another one
		require.NoError(t, config.RemoveAll(t.Context(), repo))
		second, _, err := config.Create(repo, rene, time.Now().Unix())
		require.NoError(t, err)
		_, err = config.RepoSetDescription(second, repo, rene, time.Now().Unix(), "second")
		require.NoError(t, err)
		require.NoError(t, second.Commit(t.Context(), repo))
		require.NotEqual(t, first, second.Id())

		// the change lands on the config there is now
		require.NoError(t, staged.Commit(t.Context()))
		require.Equal(t, second.Id(), staged.Id())
		snap := c.ProjectConfig().Snapshot()
		require.Equal(t, "second", snap.Repo.Description())
		require.Equal(t, []common.Label{"mine"}, snap.Labels.Names())

		read, err := config.Read(t.Context(), repo)
		require.NoError(t, err)
		require.Equal(t, second.Id(), read.Id())
		require.Len(t, read.Operations(), 3)

		// one config removed, another created
		require.Equal(t, []observerEvent{{config.Typename, first}}, obs.removed)
		require.Equal(t, []observerEvent{{config.Typename, first}, {config.Typename, second.Id()}}, obs.created)
		require.Empty(t, obs.updated)
	})

	t.Run("removed outside while committing", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)
		c := createTestRepoCacheNoEvents(t, repo)
		setUser(t, c, "rene")
		obs := &observer{}
		c.ProjectConfig().RegisterObserver("repo", obs)

		view := c.ProjectConfig().View()
		_, err := view.RepoSetDescription(t.Context(), "first")
		require.NoError(t, err)
		require.NoError(t, view.Commit(t.Context()))
		first := view.Id()

		staged := c.ProjectConfig().View()
		_, err = staged.LabelsSet(t.Context(), "mine", config.LabelsDefinition{})
		require.NoError(t, err)

		// removed by another process, before the cache knows
		require.NoError(t, config.RemoveAll(t.Context(), repo))

		// the commit creates a new config, as after a removal through the cache
		require.NoError(t, staged.Commit(t.Context()))
		require.NotEqual(t, first, staged.Id())
		snap := c.ProjectConfig().Snapshot()
		require.Empty(t, snap.Repo.Description())
		require.Equal(t, []common.Label{"mine"}, snap.Labels.Names())
		require.Equal(t, []observerEvent{{config.Typename, first}}, obs.removed)
		require.Equal(t, []observerEvent{{config.Typename, first}, {config.Typename, staged.Id()}}, obs.created)
	})

	t.Run("an author missing", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)
		c := createTestRepoCacheNoEvents(t, repo)
		setUser(t, c, "rene")
		obs := &observer{}
		c.ProjectConfig().RegisterObserver("repo", obs)

		view := c.ProjectConfig().View()
		_, err := view.RepoSetDescription(t.Context(), "first")
		require.NoError(t, err)
		require.NoError(t, view.Commit(t.Context()))

		// another process changes the config, as an identity not in the repository
		stranger, err := identity.NewIdentity(repo, "stranger", "stranger@example.com")
		require.NoError(t, err)
		outside, err := config.Read(t.Context(), repo)
		require.NoError(t, err)
		_, err = config.RepoSetDescription(outside, repo, stranger, time.Now().Unix(), "stranger")
		require.NoError(t, err)
		require.NoError(t, outside.Commit(t.Context(), repo))

		// the config can't be read, but it's still there: the sync fails, for
		// the worker to retry, and the loaded config stays
		require.Error(t, c.projectConfig.sync())
		require.Equal(t, "first", c.ProjectConfig().Snapshot().Repo.Description())
		require.Empty(t, obs.removed)

		// same when loading: a warning, not a failure
		require.NoError(t, c.Close())
		c, events := NewRepoCache(repo, nil)
		t.Cleanup(func() { require.NoError(t, c.Close()) })
		var warnings []BuildEvent
		for event := range events {
			require.NoError(t, event.Err)
			if event.Event == BuildEventWarning {
				warnings = append(warnings, event)
			}
		}
		require.Len(t, warnings, 1)
		require.Equal(t, config.Typename, warnings[0].Typename)
		require.Empty(t, c.ProjectConfig().Snapshot().Repo.Description())

		// read once the identity is there
		require.NoError(t, stranger.Commit(t.Context(), repo))
		require.NoError(t, c.projectConfig.sync())
		require.Equal(t, "stranger", c.ProjectConfig().Snapshot().Repo.Description())
	})

	t.Run("removed", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)
		c := createTestRepoCacheNoEvents(t, repo)
		setUser(t, c, "rene")
		obs := &observer{}
		c.ProjectConfig().RegisterObserver("repo", obs)

		view := c.ProjectConfig().View()
		_, err := view.RepoSetDescription(t.Context(), "description")
		require.NoError(t, err)
		require.NoError(t, view.Commit(t.Context()))
		id := view.Id()

		// a change staged on the config, removed before it's committed
		staged := c.ProjectConfig().View()
		_, err = staged.RepoSetDescription(t.Context(), "new")
		require.NoError(t, err)

		require.NoError(t, c.ProjectConfig().RemoveAll(t.Context()))
		require.Empty(t, c.ProjectConfig().Snapshot().Repo.Description())
		require.Equal(t, []observerEvent{{config.Typename, id}}, obs.removed)

		// creates a new config when committed, as any change with no config
		require.NoError(t, staged.Commit(t.Context()))
		require.NotEqual(t, id, staged.Id())
		require.Equal(t, []observerEvent{{config.Typename, id}, {config.Typename, staged.Id()}}, obs.created)
		require.Equal(t, "new", c.ProjectConfig().Snapshot().Repo.Description())
	})
}

func TestConfigCachePushPull(t *testing.T) {
	repoA, repoB, _ := repotest.SetupGoGitReposAndRemote(t)
	cacheA := createTestRepoCacheNoEvents(t, repoA)
	cacheB := createTestRepoCacheNoEvents(t, repoB)
	setUser(t, cacheA, "rene")
	obs := &observer{}
	cacheB.ProjectConfig().RegisterObserver("repo", obs)

	view := cacheA.ProjectConfig().View()
	_, err := view.RepoSetDescription(t.Context(), "from A")
	require.NoError(t, err)
	require.NoError(t, view.Commit(t.Context()))
	_, err = cacheA.Push(t.Context(), "origin")
	require.NoError(t, err)

	require.NoError(t, cacheB.Pull(t.Context(), "origin"))
	require.Equal(t, "from A", cacheB.ProjectConfig().Snapshot().Repo.Description())
	require.Equal(t, []observerEvent{{config.Typename, view.Id()}}, obs.created)

	// B changes it, A pulls the change
	reneB, err := cacheB.Identities().Resolve(t.Context(), view.Snapshot().Operations[0].Author().Id())
	require.NoError(t, err)
	require.NoError(t, cacheB.SetUserIdentity(t.Context(), reneB))
	viewB := cacheB.ProjectConfig().View()
	_, err = viewB.RepoSetDescription(t.Context(), "from B")
	require.NoError(t, err)
	require.NoError(t, viewB.Commit(t.Context()))
	_, err = cacheB.Push(t.Context(), "origin")
	require.NoError(t, err)

	require.NoError(t, cacheA.Pull(t.Context(), "origin"))
	require.Equal(t, "from B", cacheA.ProjectConfig().Snapshot().Repo.Description())
}

func TestConfigCacheConflict(t *testing.T) {
	repoA, repoB, _ := repotest.SetupGoGitReposAndRemote(t)
	cacheA := createTestRepoCacheNoEvents(t, repoA)
	cacheB := createTestRepoCacheNoEvents(t, repoB)
	setUser(t, cacheA, "rene")
	setUser(t, cacheB, "isaac")
	// distribute the identities first, the merge of the configs needs them
	_, err := cacheA.Push(t.Context(), "origin")
	require.NoError(t, err)
	require.NoError(t, cacheB.Pull(t.Context(), "origin"))

	for _, c := range []*RepoCache{cacheA, cacheB} {
		view := c.ProjectConfig().View()
		_, err := view.RepoSetDescription(t.Context(), "description")
		require.NoError(t, err)
		require.NoError(t, view.Commit(t.Context()))
	}
	_, err = cacheA.Push(t.Context(), "origin")
	require.NoError(t, err)

	before := cacheB.ProjectConfig().Snapshot().Operations[0].Id()
	_, err = cacheB.Fetch(t.Context(), "origin")
	require.NoError(t, err)
	var statuses []entity.MergeStatus
	for result := range cacheB.MergeAll(t.Context(), "origin") {
		require.NoError(t, result.Err)
		if result.Status != entity.MergeStatusNothing {
			statuses = append(statuses, result.Status)
		}
	}
	require.Contains(t, statuses, entity.MergeStatusConflict)
	require.Error(t, cacheB.Pull(t.Context(), "origin"))

	// B keeps its own config
	require.Equal(t, before, cacheB.ProjectConfig().Snapshot().Operations[0].Id())
}

func TestConfigCacheFollowsChanges(t *testing.T) {
	// time is fake in the bubble, and synctest.Wait returns once every goroutine
	// of the bubble, the workers included, is idle
	synctest.Test(t, func(t *testing.T) {
		repo := repository.NewMockRepo()
		source := newChanSource()
		c, err := NewRepoCacheNoEvents(repo, source)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, c.Close()) })
		rene := setUser(t, c, "rene")
		obs := &observer{}
		c.ProjectConfig().RegisterObserver("repo", obs)
		synctest.Wait()

		changed := repository.Change{Keys: map[string][]string{config.Namespace: {"config"}}}

		// created outside: seen once reported
		outside, _, err := config.Create(repo, rene, time.Now().Unix())
		require.NoError(t, err)
		_, err = config.RepoSetDescription(outside, repo, rene, time.Now().Unix(), "created")
		require.NoError(t, err)
		require.NoError(t, outside.Commit(t.Context(), repo))
		require.Empty(t, c.ProjectConfig().Snapshot().Repo.Description())

		time.Sleep(syncInterval)
		source.changes <- changed
		synctest.Wait()
		require.Equal(t, "created", c.ProjectConfig().Snapshot().Repo.Description())
		require.Equal(t, []observerEvent{{config.Typename, outside.Id()}}, obs.created)

		// changed outside
		_, err = config.RepoSetDescription(outside, repo, rene, time.Now().Unix(), "changed")
		require.NoError(t, err)
		require.NoError(t, outside.Commit(t.Context(), repo))

		time.Sleep(syncInterval)
		source.changes <- changed
		synctest.Wait()
		require.Equal(t, "changed", c.ProjectConfig().Snapshot().Repo.Description())
		require.Equal(t, []observerEvent{{config.Typename, outside.Id()}}, obs.updated)

		// removed outside
		require.NoError(t, config.RemoveAll(t.Context(), repo))

		time.Sleep(syncInterval)
		source.changes <- changed
		synctest.Wait()
		require.Empty(t, c.ProjectConfig().Snapshot().Repo.Description())
		require.Equal(t, []observerEvent{{config.Typename, outside.Id()}}, obs.removed)
	})
}
