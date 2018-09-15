package cache

import (
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/operations"
	"github.com/MichaelMure/git-bug/util/git"
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

func (c *BugCache) AddCommentWithFiles(message string, files []git.Hash) error {
	author, err := bug.GetUser(c.repoCache.repo)
	if err != nil {
		return err
	}

	err = operations.CommentWithFiles(c.bug, author, message, files)
	if err != nil {
		return err
	}

	return c.notifyUpdated()
}

func (c *BugCache) ChangeLabels(added []string, removed []string) ([]operations.LabelChangeResult, error) {
	author, err := bug.GetUser(c.repoCache.repo)
	if err != nil {
		return nil, err
	}

	changes, err := operations.ChangeLabels(c.bug, author, added, removed)
	if err != nil {
		return changes, err
	}

	err = c.notifyUpdated()
	if err != nil {
		return nil, err
	}

	return changes, nil
}

func (c *BugCache) Open() error {
	author, err := bug.GetUser(c.repoCache.repo)
	if err != nil {
		return err
	}

	err = operations.Open(c.bug, author)
	if err != nil {
		return err
	}

	return c.notifyUpdated()
}

func (c *BugCache) Close() error {
	author, err := bug.GetUser(c.repoCache.repo)
	if err != nil {
		return err
	}

	err = operations.Close(c.bug, author)
	if err != nil {
		return err
	}

	return c.notifyUpdated()
}

func (c *BugCache) SetTitle(title string) error {
	author, err := bug.GetUser(c.repoCache.repo)
	if err != nil {
		return err
	}

	err = operations.SetTitle(c.bug, author, title)
	if err != nil {
		return err
	}

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
