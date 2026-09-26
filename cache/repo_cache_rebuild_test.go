package cache

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/entities/bug"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/repository"
)

func rebuildNoEvents(t *testing.T, repo repository.ClockedRepo) (*RepoCache, error) {
	t.Helper()
	c, events := NewRepoCacheRebuild(repo)
	var firstErr error
	for event := range events {
		if event.Err != nil && firstErr == nil {
			firstErr = event.Err
		}
	}
	return c, firstErr
}

func gitBugRefs(t *testing.T, repo repository.ClockedRepo) map[string]repository.Hash {
	t.Helper()
	res := make(map[string]repository.Hash)
	for _, prefix := range []string{"refs/bugs/", "refs/identities/"} {
		refs, err := repo.ListRefs(prefix)
		require.NoError(t, err)
		for _, ref := range refs {
			h, err := repo.ResolveRef(ref)
			require.NoError(t, err)
			res[ref] = h
		}
	}
	return res
}

func TestCacheRebuild(t *testing.T) {
	repo := repository.CreateGoGitTestRepo(t, false)

	c, err := NewRepoCacheNoEvents(repo)
	require.NoError(t, err)
	iden, err := c.Identities().New("René Descartes", "rene@descartes.fr")
	require.NoError(t, err)
	require.NoError(t, c.SetUserIdentity(iden))

	removed, _, err := c.Bugs().New("removed", "message")
	require.NoError(t, err)
	moved, _, err := c.Bugs().New("moved", "message")
	require.NoError(t, err)
	removedId, movedId := removed.Id(), moved.Id()

	removedRef := "refs/" + bug.Namespace + "/" + removedId.String()
	movedRef := "refs/" + bug.Namespace + "/" + movedId.String()

	removedHash, err := repo.ResolveRef(removedRef)
	require.NoError(t, err)
	movedOldHash, err := repo.ResolveRef(movedRef)
	require.NoError(t, err)

	_, _, err = moved.AddComment("a comment")
	require.NoError(t, err)
	require.NoError(t, moved.Commit())
	require.NoError(t, c.Close())

	// changes made outside git-bug
	require.NoError(t, repo.RemoveRef(removedRef))
	movedNewHash, err := repo.ResolveRef(movedRef)
	require.NoError(t, err)
	require.NoError(t, repo.UpdateRef(movedRef, movedNewHash, movedOldHash))

	// C1: a normal open doesn't notice
	c, err = NewRepoCacheNoEvents(repo)
	require.NoError(t, err)
	require.Contains(t, c.Bugs().AllIds(), removedId)
	require.NoError(t, c.Close())

	refsBefore := gitBugRefs(t, repo)

	// C2, C4: rebuild reflects the refs
	c, err = rebuildNoEvents(t, repo)
	require.NoError(t, err)
	require.False(t, slices.Contains(c.Bugs().AllIds(), removedId))
	excerpt, err := c.Bugs().ResolveExcerpt(movedId)
	require.NoError(t, err)
	require.Equal(t, 1, excerpt.LenComments) // only the creation message
	require.NoError(t, c.Close())

	// C7: git data is untouched
	require.Equal(t, refsBefore, gitBugRefs(t, repo))

	// C3: a restored ref shows up again
	require.NoError(t, repo.UpdateRef(removedRef, "", removedHash))
	c, err = rebuildNoEvents(t, repo)
	require.NoError(t, err)
	require.Contains(t, c.Bugs().AllIds(), removedId)
	require.NoError(t, c.Close())
}

func TestCacheRebuildLocked(t *testing.T) {
	repo := repository.CreateGoGitTestRepo(t, false)

	c, err := NewRepoCacheNoEvents(repo)
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Close() })

	before := make(map[string][]byte)
	for _, namespace := range []string{bug.Namespace, identity.Namespace} {
		data, err := os.ReadFile(filepath.Join(repo.LocalStorage().Root(), cacheDir, namespace))
		require.NoError(t, err)
		before[namespace] = data
	}

	_, err = rebuildNoEvents(t, repo)
	require.ErrorContains(t, err, "already locked")

	for namespace, data := range before {
		after, err := os.ReadFile(filepath.Join(repo.LocalStorage().Root(), cacheDir, namespace))
		require.NoError(t, err)
		require.Equal(t, data, after)
	}
}

func TestCacheRebuildFailureLeavesNoExcerpts(t *testing.T) {
	repo := repository.CreateGoGitTestRepo(t, false)

	c, err := NewRepoCacheNoEvents(repo)
	require.NoError(t, err)
	require.NoError(t, c.Close())

	cachePath := filepath.Join(repo.LocalStorage().Root(), cacheDir)
	_, err = os.Stat(filepath.Join(cachePath, bug.Namespace))
	require.NoError(t, err)

	// a bug ref pointing to something that isn't a bug
	tree, err := repo.StoreTree(nil)
	require.NoError(t, err)
	commit, err := repo.StoreCommit(tree)
	require.NoError(t, err)
	badId := entity.DeriveId([]byte("not a bug"))
	badRef := "refs/" + bug.Namespace + "/" + badId.String()
	require.NoError(t, repo.UpdateRef(badRef, "", commit))

	c, err = rebuildNoEvents(t, repo)
	require.Error(t, err)
	require.NoError(t, c.Close())

	for _, namespace := range []string{bug.Namespace, identity.Namespace} {
		_, err = os.Stat(filepath.Join(cachePath, namespace))
		require.ErrorIs(t, err, os.ErrNotExist)
	}

	// the next normal open rebuilds
	require.NoError(t, repo.RemoveRef(badRef))
	c, events := NewRepoCache(repo)
	var rebuilt bool
	for event := range events {
		require.NoError(t, event.Err)
		rebuilt = rebuilt || event.Event == BuildEventCacheIsBuilt
	}
	require.True(t, rebuilt)
	require.NoError(t, c.Close())
}
