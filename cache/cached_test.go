package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/entities/bug"
	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/repository"
	"github.com/git-bug/git-bug/repository/repotest"
)

// commentsInGit returns the comments of a bug as stored in the repository, read
// outside the cache.
func commentsInGit(t *testing.T, repo repository.Repo, id entity.Id) []string {
	t.Helper()
	b, err := bug.Read(t.Context(), repo, id)
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
func changeOutside(t *testing.T, repo repository.Repo, author identity.Interface, id entity.Id, message string) {
	t.Helper()
	b, err := bug.Read(t.Context(), repo, id)
	require.NoError(t, err)
	_, _, err = bug.AddComment(b, author, time.Now().Unix(), message, nil, nil)
	require.NoError(t, err)
	require.NoError(t, b.Commit(repo))
}

func TestViews(t *testing.T) {
	t.Run("each caller gets its own view, sharing the committed state", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)
		c, _ := newTestCacheWithUser(t, repo)
		created, _, err := c.Bugs().New(t.Context(), "title", "message")
		require.NoError(t, err)
		require.NoError(t, created.Commit(t.Context()))

		a, err := c.Bugs().Resolve(t.Context(), created.Id())
		require.NoError(t, err)
		b, err := c.Bugs().Resolve(t.Context(), created.Id())
		require.NoError(t, err)
		require.NotSame(t, a, b)
		require.Same(t, a.shared, b.shared)

		// what a view stages is its own
		_, _, err = a.AddComment(t.Context(), "from a")
		require.NoError(t, err)
		require.True(t, a.NeedCommit())
		require.False(t, b.NeedCommit())
		require.Equal(t, []string{"message", "from a"}, commentMessages(a.Snapshot()))
		require.Equal(t, []string{"message"}, commentMessages(b.Snapshot()))
		require.Equal(t, []string{"message"}, commentMessages(created.Snapshot()))
		require.Equal(t, 1, lenComments(t, c, created.Id()))

		_, _, err = b.AddComment(t.Context(), "from b")
		require.NoError(t, err)
		require.Equal(t, []string{"message", "from b"}, commentMessages(b.Snapshot()))

		// a commit writes the operations of that view only, and every view sees it,
		// the ones with staged operations showing them after the committed state
		require.NoError(t, a.Commit(t.Context()))
		require.False(t, a.NeedCommit())
		require.Equal(t, []string{"message", "from a"}, commentsInGit(t, repo, created.Id()))
		require.Equal(t, 2, lenComments(t, c, created.Id()))
		require.Equal(t, []string{"message", "from a"}, commentMessages(created.Snapshot()))
		require.True(t, b.NeedCommit())
		require.Equal(t, []string{"message", "from a", "from b"}, commentMessages(b.Snapshot()))

		require.NoError(t, b.Commit(t.Context()))
		require.Equal(t, []string{"message", "from a", "from b"}, commentsInGit(t, repo, created.Id()))
		require.Equal(t, 3, lenComments(t, c, created.Id()))
		requireDerivedBuiltFromRefs(t, repo, c)
	})

	t.Run("staged metadata is the view's own", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)
		c, _ := newTestCacheWithUser(t, repo)
		created, _, err := c.Bugs().New(t.Context(), "title", "message")
		require.NoError(t, err)
		require.NoError(t, created.Commit(t.Context()))
		target := created.Snapshot().Operations[0].Id()

		a, err := c.Bugs().Resolve(t.Context(), created.Id())
		require.NoError(t, err)
		b, err := c.Bugs().Resolve(t.Context(), created.Id())
		require.NoError(t, err)

		_, err = a.SetMetadata(t.Context(), target, map[string]string{"key": "from a"})
		require.NoError(t, err)
		val, ok := a.Snapshot().GetCreateMetadata("key")
		require.True(t, ok)
		require.Equal(t, "from a", val)
		_, ok = b.Snapshot().GetCreateMetadata("key")
		require.False(t, ok)
		_, ok = created.Snapshot().GetCreateMetadata("key")
		require.False(t, ok)
		_, err = b.ResolveOperationWithMetadata("key", "from a")
		require.ErrorIs(t, err, ErrNoMatchingOp)

		// the first value committed wins, in memory as in git
		_, err = b.SetMetadata(t.Context(), target, map[string]string{"key": "from b"})
		require.NoError(t, err)
		require.NoError(t, b.Commit(t.Context()))
		require.NoError(t, a.Commit(t.Context()))
		for _, snap := range []*bug.Snapshot{a.Snapshot(), b.Snapshot(), created.Snapshot()} {
			val, _ = snap.GetCreateMetadata("key")
			require.Equal(t, "from b", val)
		}
		inGit, err := bug.Read(t.Context(), repo, created.Id())
		require.NoError(t, err)
		val, _ = inGit.Compile().GetCreateMetadata("key")
		require.Equal(t, "from b", val)
	})

	t.Run("concurrent views stage metadata independently", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)
		c, _ := newTestCacheWithUser(t, repo)
		created, _, err := c.Bugs().New(t.Context(), "title", "message")
		require.NoError(t, err)
		require.NoError(t, created.Commit(t.Context()))
		target := created.Snapshot().Operations[0].Id()

		const count = 10
		var wg sync.WaitGroup
		errs := make(chan error, count)
		for i := range count {
			wg.Add(1)
			go func() {
				defer wg.Done()
				v, err := c.Bugs().Resolve(t.Context(), created.Id())
				if err != nil {
					errs <- err
					return
				}
				key := fmt.Sprintf("key%d", i)
				if _, err := v.SetMetadata(t.Context(), target, map[string]string{key: "value"}); err != nil {
					errs <- err
					return
				}
				// reads the metadata of the shared operations, while the other
				// views apply theirs
				if _, err := v.ResolveOperationWithMetadata(key, "value"); err != nil {
					errs <- err
					return
				}
				errs <- nil
			}()
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			require.NoError(t, err)
		}
		require.Empty(t, created.Snapshot().Operations[0].AllMetadata())
	})

	t.Run("staged operations are committed as one", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)
		c, _ := newTestCacheWithUser(t, repo)
		b, _, err := c.Bugs().New(t.Context(), "title", "message")
		require.NoError(t, err)
		require.NoError(t, b.Commit(t.Context()))
		before, err := repo.ListCommits(t.Context(), b.LastCommit())
		require.NoError(t, err)

		_, _, err = b.AddComment(t.Context(), "comment")
		require.NoError(t, err)
		_, err = b.Close(t.Context())
		require.NoError(t, err)
		require.NoError(t, b.Commit(t.Context()))

		after, err := repo.ListCommits(t.Context(), b.LastCommit())
		require.NoError(t, err)
		require.Len(t, after, len(before)+1)
		require.Equal(t, common.ClosedStatus, b.Snapshot().Status)
		require.Equal(t, []string{"message", "comment"}, commentsInGit(t, repo, b.Id()))
	})

	t.Run("a creation is staged with what follows, until committed", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)
		c, _ := newTestCacheWithUser(t, repo)
		b, _, err := c.Bugs().New(t.Context(), "title", "message")
		require.NoError(t, err)
		id := b.Id()
		require.NotEqual(t, entity.UnsetId, id)

		_, _, err = b.AddComment(t.Context(), "comment")
		require.NoError(t, err)
		require.True(t, b.NeedCommit())
		require.Equal(t, []string{"message", "comment"}, commentMessages(b.Snapshot()))

		// only the view knows the bug
		_, err = c.Bugs().Resolve(t.Context(), id)
		require.True(t, entity.IsErrNotFound(err))
		require.Empty(t, c.Bugs().AllIds())
		_, err = repo.ResolveRef(bug.Namespace, id.String())
		require.ErrorIs(t, err, repository.ErrNotFound)

		require.NoError(t, b.Commit(t.Context()))
		require.Equal(t, id, b.Id())
		commits, err := repo.ListCommits(t.Context(), b.LastCommit())
		require.NoError(t, err)
		require.Len(t, commits, 1)
		require.Equal(t, []string{"message", "comment"}, commentsInGit(t, repo, id))
		requireDerivedBuiltFromRefs(t, repo, c)

		// the copy of the view is the loaded one
		resolved, err := c.Bugs().Resolve(t.Context(), id)
		require.NoError(t, err)
		require.Same(t, b.shared, resolved.shared)
	})

	t.Run("a commit on a reference moved outside is redone on the new state", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)
		c, rene := newTestCacheWithUser(t, repo)
		b, _, err := c.Bugs().New(t.Context(), "title", "message")
		require.NoError(t, err)
		require.NoError(t, b.Commit(t.Context()))
		_, _, err = b.AddComment(t.Context(), "markerview")
		require.NoError(t, err)

		// moved after the view staged its operation
		changeOutside(t, repo, rene, b.Id(), "markeroutside")
		ref, err := repo.ResolveRef(bug.Namespace, b.Id().String())
		require.NoError(t, err)
		require.NotEqual(t, ref, b.LastCommit())

		require.NoError(t, b.Commit(t.Context()))
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

	t.Run("a failed commit keeps the staged operations", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)
		c, _ := newTestCacheWithUser(t, repo)
		b, _, err := c.Bugs().New(t.Context(), "title", "message")
		require.NoError(t, err)
		require.NoError(t, b.Commit(t.Context()))
		_, committed, err := b.AddComment(t.Context(), "first")
		require.NoError(t, err)
		require.NoError(t, b.Commit(t.Context()))

		// an operation already committed can't be committed again
		_, _, err = b.AddComment(t.Context(), "staged")
		require.NoError(t, err)
		b.Append(committed)
		require.Error(t, b.Commit(t.Context()))
		require.True(t, b.NeedCommit())
		require.Len(t, b.Snapshot().Operations, 4)
		require.Equal(t, []string{"message", "first"}, commentsInGit(t, repo, b.Id()))
		requireDerivedBuiltFromRefs(t, repo, c)

		// committing again fails the same way, and nothing gets saved
		require.Error(t, b.CommitAsNeeded(t.Context()))
		require.True(t, b.NeedCommit())
		require.Equal(t, []string{"message", "first"}, commentsInGit(t, repo, b.Id()))

		// other views are not affected, and a new view starts without them
		other, err := c.Bugs().Resolve(t.Context(), b.Id())
		require.NoError(t, err)
		require.False(t, other.NeedCommit())
		_, _, err = other.AddComment(t.Context(), "other")
		require.NoError(t, err)
		require.NoError(t, other.Commit(t.Context()))
		require.Equal(t, []string{"message", "first", "other"}, commentsInGit(t, repo, b.Id()))
		requireDerivedBuiltFromRefs(t, repo, c)
	})

	t.Run("a commit failed on a transient error succeeds once retried", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)
		c, _ := newTestCacheWithUser(t, repo)
		b, _, err := c.Bugs().New(t.Context(), "title", "message")
		require.NoError(t, err)
		require.NoError(t, b.Commit(t.Context()))

		_, _, err = b.AddComment(t.Context(), "retried")
		require.NoError(t, err)

		// a lock held by someone else on the reference fails the commit
		lockPath := filepath.Join(repo.GitDir, "refs", bug.Namespace, b.Id().String()+".lock")
		require.NoError(t, os.WriteFile(lockPath, []byte("held"), 0666))
		require.Error(t, b.Commit(t.Context()))
		require.True(t, b.NeedCommit())
		require.Equal(t, []string{"message"}, commentsInGit(t, repo, b.Id()))

		require.NoError(t, os.Remove(lockPath))
		require.NoError(t, b.Commit(t.Context()))
		require.False(t, b.NeedCommit())
		require.Equal(t, []string{"message", "retried"}, commentsInGit(t, repo, b.Id()))
		require.Equal(t, []string{"message", "retried"}, commentMessages(b.Snapshot()))
		requireDerivedBuiltFromRefs(t, repo, c)
	})

	t.Run("a loaded copy follows a reference moved outside on sync", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)
		c, rene := newTestCacheWithUser(t, repo)
		b, _, err := c.Bugs().New(t.Context(), "title", "message")
		require.NoError(t, err)
		require.NoError(t, b.Commit(t.Context()))
		other, err := c.Bugs().Resolve(t.Context(), b.Id())
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
		repo := repotest.NewGoGitRepo(t, false)
		c, _ := newTestCacheWithUser(t, repo)
		created, _, err := c.Bugs().New(t.Context(), "title", "message")
		require.NoError(t, err)
		require.NoError(t, created.Commit(t.Context()))

		const count = 10
		var wg sync.WaitGroup
		errs := make(chan error, count)
		for i := range count {
			wg.Add(1)
			go func() {
				defer wg.Done()
				v, err := c.Bugs().Resolve(t.Context(), created.Id())
				if err != nil {
					errs <- err
					return
				}
				if _, _, err := v.AddComment(t.Context(), fmt.Sprintf("marker%d", i)); err != nil {
					errs <- err
					return
				}
				errs <- v.Commit(t.Context())
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
		repo := repotest.NewGoGitRepo(t, false)
		c, rene := newTestCacheWithUser(t, repo)
		other, err := c.Identities().Resolve(t.Context(), rene.Id())
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
		resolved, err := c.Identities().Resolve(t.Context(), rene.Id())
		require.NoError(t, err)
		require.Equal(t, "René Descartes, renamed", resolved.Name())
		// a view resolved before keeps its copy
		require.Equal(t, "René Descartes", other.Name())
	})

	t.Run("a refreshed identity replaces the loaded one", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)
		c, rene := newTestCacheWithUser(t, repo)

		// changed outside the cache
		read, err := identity.Read(t.Context(), repo, rene.Id())
		require.NoError(t, err)
		require.NoError(t, read.Mutate(repo, func(m *identity.Mutator) { m.Name = "René Descartes, renamed" }))
		require.NoError(t, read.Commit(repo))

		require.NoError(t, c.identities.sync(rene.Id()))
		excerpt, err := c.Identities().ResolveExcerpt(rene.Id())
		require.NoError(t, err)
		require.Equal(t, "René Descartes, renamed", excerpt.Name)
		resolved, err := c.Identities().Resolve(t.Context(), rene.Id())
		require.NoError(t, err)
		require.Equal(t, "René Descartes, renamed", resolved.Name())
		// a view resolved before keeps the previous copy
		require.Equal(t, "René Descartes", rene.Name())
	})

	t.Run("an identity committed after its loaded copy is evicted", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)
		c, rene := newTestCacheWithUser(t, repo)
		c.setCacheSize(1)
		_, err := c.Identities().New("Other", "other@example.com")
		require.NoError(t, err)
		require.NotContains(t, c.identities.cached, rene.Id())

		// the view commits a copy of its own, the evicted one stays behind
		require.NoError(t, rene.Mutate(repo, func(m *identity.Mutator) { m.Name = "René Descartes, renamed" }))
		require.NoError(t, rene.Commit())

		resolved, err := c.Identities().Resolve(t.Context(), rene.Id())
		require.NoError(t, err)
		require.Equal(t, "René Descartes, renamed", resolved.Name())
		requireDerivedBuiltFromRefs(t, repo, c)
	})

	t.Run("the id of a new bug holds while it's committed", func(t *testing.T) {
		repo := repotest.NewGoGitRepo(t, false)
		c, _ := newTestCacheWithUser(t, repo)
		b, _, err := c.Bugs().New(t.Context(), "title", "message")
		require.NoError(t, err)
		id := b.Id()

		// a commit in progress holds the view: the id is asked meanwhile, reading
		// the copy before the commit lands, and the staged operations after
		b.mu.Lock()
		got := make(chan entity.Id)
		go func() { got <- b.Id() }()
		// give it time to read the copy and wait on the view
		time.Sleep(50 * time.Millisecond)
		require.NoError(t, b.commitLocked(t.Context()))
		b.mu.Unlock()

		require.Equal(t, id, <-got)
	})
}
