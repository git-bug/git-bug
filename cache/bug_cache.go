package cache

import (
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/bug/operations"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util"
)

type BugCacher interface {
	Snapshot() *bug.Snapshot
	ClearSnapshot()

	// Mutations
	AddComment(message string) error
	AddCommentWithFiles(message string, files []util.Hash) error
	ChangeLabels(added []string, removed []string) error
	Open() error
	Close() error
	SetTitle(title string) error

	Commit() error
	CommitAsNeeded() error
}

type BugCache struct {
	repo repository.Repo
	bug  *bug.Bug
	snap *bug.Snapshot
}

func NewBugCache(repo repository.Repo, b *bug.Bug) BugCacher {
	return &BugCache{
		repo: repo,
		bug:  b,
	}
}

func (c *BugCache) Snapshot() *bug.Snapshot {
	if c.snap == nil {
		snap := c.bug.Compile()
		c.snap = &snap
	}
	return c.snap
}

func (c *BugCache) ClearSnapshot() {
	c.snap = nil
}

func (c *BugCache) AddComment(message string) error {
	return c.AddCommentWithFiles(message, nil)
}

func (c *BugCache) AddCommentWithFiles(message string, files []util.Hash) error {
	author, err := bug.GetUser(c.repo)
	if err != nil {
		return err
	}

	operations.CommentWithFiles(c.bug, author, message, files)

	// TODO: perf --> the snapshot could simply be updated with the new op
	c.ClearSnapshot()

	return nil
}

func (c *BugCache) ChangeLabels(added []string, removed []string) error {
	author, err := bug.GetUser(c.repo)
	if err != nil {
		return err
	}

	err = operations.ChangeLabels(nil, c.bug, author, added, removed)
	if err != nil {
		return err
	}

	// TODO: perf --> the snapshot could simply be updated with the new op
	c.ClearSnapshot()

	return nil
}

func (c *BugCache) Open() error {
	author, err := bug.GetUser(c.repo)
	if err != nil {
		return err
	}

	operations.Open(c.bug, author)

	// TODO: perf --> the snapshot could simply be updated with the new op
	c.ClearSnapshot()

	return nil
}

func (c *BugCache) Close() error {
	author, err := bug.GetUser(c.repo)
	if err != nil {
		return err
	}

	operations.Close(c.bug, author)

	// TODO: perf --> the snapshot could simply be updated with the new op
	c.ClearSnapshot()

	return nil
}

func (c *BugCache) SetTitle(title string) error {
	author, err := bug.GetUser(c.repo)
	if err != nil {
		return err
	}

	operations.SetTitle(c.bug, author, title)

	// TODO: perf --> the snapshot could simply be updated with the new op
	c.ClearSnapshot()

	return nil
}

func (c *BugCache) Commit() error {
	return c.bug.Commit(c.repo)
}

func (c *BugCache) CommitAsNeeded() error {
	if c.bug.HasPendingOp() {
		return c.bug.Commit(c.repo)
	}
	return nil
}
