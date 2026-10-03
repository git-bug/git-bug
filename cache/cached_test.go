package cache

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/entities/bug"
	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/repository"
)

// commentsInGit returns the comments of a bug as stored in the repository, read
// outside the cache.
func commentsInGit(t *testing.T, repo repository.ClockedRepo, id entity.Id) []string {
	t.Helper()
	b, err := bug.Read(repo, id)
	require.NoError(t, err)
	return commentMessages(b.Compile())
}

func commentMessages(snap *bug.Snapshot) []string {
	messages := make([]string, 0, len(snap.Comments))
	for _, comment := range snap.Comments {
		messages = append(messages, comment.Message)
	}
	return messages
}

// changeOutside adds a comment to a bug without the cache knowing
func changeOutside(t *testing.T, repo repository.ClockedRepo, author identity.Interface, id entity.Id, message string) {
	t.Helper()
	b, err := bug.Read(repo, id)
	require.NoError(t, err)
	_, _, err = bug.AddComment(b, author, time.Now().Unix(), message, nil, nil)
	require.NoError(t, err)
	require.NoError(t, b.Commit(repo))
}

func TestViews(t *testing.T) {
	t.Run("each caller gets its own view, sharing the committed state", func(t *testing.T) {
		repo := repository.CreateGoGitTestRepo(t, false)
		c, _ := newTestCacheWithUser(t, repo)
		created, _, err := c.Bugs().New("title", "message")
		require.NoError(t, err)

		a, err := c.Bugs().Resolve(created.Id())
		require.NoError(t, err)
		b, err := c.Bugs().Resolve(created.Id())
		require.NoError(t, err)
		require.NotSame(t, a, b)
		require.Same(t, a.shared, b.shared)

		// what a view stages is its own
		_, _, err = a.AddComment("from a")
		require.NoError(t, err)
		require.True(t, a.NeedCommit())
		require.False(t, b.NeedCommit())
		require.Equal(t, []string{"message", "from a"}, commentMessages(a.Snapshot()))
		require.Equal(t, []string{"message"}, commentMessages(b.Snapshot()))
		require.Equal(t, []string{"message"}, commentMessages(created.Snapshot()))
		require.Equal(t, 1, lenComments(t, c, created.Id()))

		_, _, err = b.AddComment("from b")
		require.NoError(t, err)
		require.Equal(t, []string{"message", "from b"}, commentMessages(b.Snapshot()))

		// a commit writes the operations of that view only, and every view sees it,
		// the ones with staged operations showing them after the committed state
		require.NoError(t, a.Commit())
		require.False(t, a.NeedCommit())
		require.Equal(t, []string{"message", "from a"}, commentsInGit(t, repo, created.Id()))
		require.Equal(t, 2, lenComments(t, c, created.Id()))
		require.Equal(t, []string{"message", "from a"}, commentMessages(created.Snapshot()))
		require.True(t, b.NeedCommit())
		require.Equal(t, []string{"message", "from a", "from b"}, commentMessages(b.Snapshot()))

		require.NoError(t, b.Commit())
		require.Equal(t, []string{"message", "from a", "from b"}, commentsInGit(t, repo, created.Id()))
		require.Equal(t, 3, lenComments(t, c, created.Id()))
		requireDerivedBuiltFromRefs(t, repo, c)
	})

	t.Run("staged operations are committed as one", func(t *testing.T) {
		repo := repository.CreateGoGitTestRepo(t, false)
		c, _ := newTestCacheWithUser(t, repo)
		b, _, err := c.Bugs().New("title", "message")
		require.NoError(t, err)
		before, err := repo.ListCommits(b.LastCommit())
		require.NoError(t, err)

		_, _, err = b.AddComment("comment")
		require.NoError(t, err)
		_, err = b.Close()
		require.NoError(t, err)
		require.NoError(t, b.Commit())

		after, err := repo.ListCommits(b.LastCommit())
		require.NoError(t, err)
		require.Len(t, after, len(before)+1)
		require.Equal(t, common.ClosedStatus, b.Snapshot().Status)
		require.Equal(t, []string{"message", "comment"}, commentsInGit(t, repo, b.Id()))
	})

	t.Run("a commit on a reference moved outside is redone on the new state", func(t *testing.T) {
		repo := repository.CreateGoGitTestRepo(t, false)
		c, rene := newTestCacheWithUser(t, repo)
		b, _, err := c.Bugs().New("title", "message")
		require.NoError(t, err)
		_, _, err = b.AddComment("markerview")
		require.NoError(t, err)

		// moved after the view staged its operation
		changeOutside(t, repo, rene, b.Id(), "markeroutside")
		ref, err := repo.ResolveRef(bug.Namespace, b.Id().String())
		require.NoError(t, err)
		require.NotEqual(t, ref, b.LastCommit())

		require.NoError(t, b.Commit())
		require.Equal(t, []string{"message", "markeroutside", "markerview"}, commentsInGit(t, repo, b.Id()))
		require.Equal(t, []string{"message", "markeroutside", "markerview"}, commentMessages(b.Snapshot()))
		ref, err = repo.ResolveRef(bug.Namespace, b.Id().String())
		require.NoError(t, err)
		require.Equal(t, ref, b.LastCommit())
		require.Equal(t, 3, lenComments(t, c, b.Id()))
		require.Equal(t, []entity.Id{b.Id()}, searchBugs(t, c, "markeroutside"))
		require.Equal(t, []entity.Id{b.Id()}, searchBugs(t, c, "markerview"))
		requireDerivedBuiltFromRefs(t, repo, c)
	})

	t.Run("a failed commit drops the staged operations", func(t *testing.T) {
		repo := repository.CreateGoGitTestRepo(t, false)
		c, _ := newTestCacheWithUser(t, repo)
		b, _, err := c.Bugs().New("title", "message")
		require.NoError(t, err)

		// an operation already committed can't be committed again
		b.Append(b.Snapshot().Operations[0].(bug.Operation))
		require.True(t, b.NeedCommit())
		require.Error(t, b.Commit())
		require.False(t, b.NeedCommit())
		require.Len(t, b.Snapshot().Operations, 1)

		// the view is usable afterwards, and nothing of the failure gets saved
		_, _, err = b.AddComment("after")
		require.NoError(t, err)
		require.NoError(t, b.Commit())
		require.Equal(t, []string{"message", "after"}, commentsInGit(t, repo, b.Id()))
		requireDerivedBuiltFromRefs(t, repo, c)
	})

	t.Run("a loaded copy follows a reference moved outside on sync", func(t *testing.T) {
		repo := repository.CreateGoGitTestRepo(t, false)
		c, rene := newTestCacheWithUser(t, repo)
		b, _, err := c.Bugs().New("title", "message")
		require.NoError(t, err)
		other, err := c.Bugs().Resolve(b.Id())
		require.NoError(t, err)

		changeOutside(t, repo, rene, b.Id(), "markeroutside")
		require.Equal(t, []string{"message"}, commentMessages(b.Snapshot()))

		// as Load does
		require.NoError(t, c.bugs.syncAll(nil))
		require.Equal(t, []string{"message", "markeroutside"}, commentMessages(b.Snapshot()))
		require.Equal(t, []string{"message", "markeroutside"}, commentMessages(other.Snapshot()))
		require.Equal(t, 2, lenComments(t, c, b.Id()))
		requireDerivedBuiltFromRefs(t, repo, c)
	})

	t.Run("concurrent views commit independently", func(t *testing.T) {
		repo := repository.CreateGoGitTestRepo(t, false)
		c, _ := newTestCacheWithUser(t, repo)
		created, _, err := c.Bugs().New("title", "message")
		require.NoError(t, err)

		const count = 10
		var wg sync.WaitGroup
		errs := make(chan error, count)
		for i := range count {
			wg.Add(1)
			go func() {
				defer wg.Done()
				v, err := c.Bugs().Resolve(created.Id())
				if err != nil {
					errs <- err
					return
				}
				if _, _, err := v.AddComment(fmt.Sprintf("marker%d", i)); err != nil {
					errs <- err
					return
				}
				errs <- v.Commit()
			}()
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			require.NoError(t, err)
		}

		require.Len(t, commentsInGit(t, repo, created.Id()), count+1)
		require.Len(t, created.Snapshot().Comments, count+1)
		require.Equal(t, count+1, lenComments(t, c, created.Id()))
		for i := range count {
			require.Equal(t, []entity.Id{created.Id()}, searchBugs(t, c, fmt.Sprintf("marker%d", i)))
		}
		requireDerivedBuiltFromRefs(t, repo, c)
	})

	t.Run("identity views stage their own versions", func(t *testing.T) {
		repo := repository.CreateGoGitTestRepo(t, false)
		c, rene := newTestCacheWithUser(t, repo)
		other, err := c.Identities().Resolve(rene.Id())
		require.NoError(t, err)

		require.NoError(t, rene.Mutate(repo, func(m *identity.Mutator) { m.Name = "René Descartes, renamed" }))
		require.Equal(t, "René Descartes, renamed", rene.Name())
		require.Equal(t, "René Descartes", other.Name())

		// a sync while the rename is pending derives from the committed state
		require.NoError(t, c.identities.syncAll(nil))
		excerpt, err := c.Identities().ResolveExcerpt(rene.Id())
		require.NoError(t, err)
		require.Equal(t, "René Descartes", excerpt.Name)

		require.NoError(t, rene.Commit())
		excerpt, err = c.Identities().ResolveExcerpt(rene.Id())
		require.NoError(t, err)
		require.Equal(t, "René Descartes, renamed", excerpt.Name)
		resolved, err := c.Identities().Resolve(rene.Id())
		require.NoError(t, err)
		require.Equal(t, "René Descartes, renamed", resolved.Name())
		// a view resolved before keeps its copy
		require.Equal(t, "René Descartes", other.Name())
	})

	t.Run("a refreshed identity replaces the loaded one", func(t *testing.T) {
		repo := repository.CreateGoGitTestRepo(t, false)
		c, rene := newTestCacheWithUser(t, repo)

		// changed outside the cache
		read, err := identity.Read(repo, rene.Id())
		require.NoError(t, err)
		require.NoError(t, read.Mutate(repo, func(m *identity.Mutator) { m.Name = "René Descartes, renamed" }))
		require.NoError(t, read.Commit(repo))

		require.NoError(t, c.identities.sync(rene.Id()))
		excerpt, err := c.Identities().ResolveExcerpt(rene.Id())
		require.NoError(t, err)
		require.Equal(t, "René Descartes, renamed", excerpt.Name)
		resolved, err := c.Identities().Resolve(rene.Id())
		require.NoError(t, err)
		require.Equal(t, "René Descartes, renamed", resolved.Name())
		// a view resolved before keeps the previous copy
		require.Equal(t, "René Descartes", rene.Name())
	})
}
