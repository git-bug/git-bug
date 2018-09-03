package cache

import (
	"io"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/bug/operations"
	"github.com/MichaelMure/git-bug/util"
)

type BugCache struct {
	repoCache *RepoCache
	bug       *bug.WithSnapshot
}

func NewBugCache(repoCache *RepoCache, b *bug.Bug) *BugCache {
	return &BugCache{
		repoCache: repoCache,
		bug:       &bug.WithSnapshot{Bug: b},
	}
}

func (c *BugCache) Snapshot() *bug.Snapshot {
	return c.bug.Snapshot()
}

func (c *BugCache) HumanId() string {
	return c.bug.HumanId()
}

func (c *BugCache) notifyUpdated() error {
	return c.repoCache.bugUpdated(c.bug.Id())
}

func (c *BugCache) AddComment(message string) error {
	if err := c.AddCommentWithFiles(message, nil); err != nil {
		return err
	}

	return c.notifyUpdated()
}

func (c *BugCache) AddCommentWithFiles(message string, files []util.Hash) error {
	author, err := bug.GetUser(c.repoCache.repo)
	if err != nil {
		return err
	}

	operations.CommentWithFiles(c.bug, author, message, files)

	return c.notifyUpdated()
}

func (c *BugCache) ChangeLabels(out io.Writer, added []string, removed []string) error {
	author, err := bug.GetUser(c.repoCache.repo)
	if err != nil {
		return err
	}

	err = operations.ChangeLabels(out, c.bug, author, added, removed)
	if err != nil {
		return err
	}

	return c.notifyUpdated()
}

func (c *BugCache) Open() error {
	author, err := bug.GetUser(c.repoCache.repo)
	if err != nil {
		return err
	}

	operations.Open(c.bug, author)

	return c.notifyUpdated()
}

func (c *BugCache) Close() error {
	author, err := bug.GetUser(c.repoCache.repo)
	if err != nil {
		return err
	}

	operations.Close(c.bug, author)

	return c.notifyUpdated()
}

func (c *BugCache) SetTitle(title string) error {
	author, err := bug.GetUser(c.repoCache.repo)
	if err != nil {
		return err
	}

	operations.SetTitle(c.bug, author, title)

	return c.notifyUpdated()
}

func (c *BugCache) Commit() error {
	return c.bug.Commit(c.repoCache.repo)
}

func (c *BugCache) CommitAsNeeded() error {
	if c.bug.HasPendingOp() {
		return c.bug.Commit(c.repoCache.repo)
	}
	return nil
}
