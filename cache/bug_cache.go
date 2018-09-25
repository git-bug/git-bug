package cache

import (
	"time"

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

func (c *BugCache) Id() string {
	return c.bug.Id()
}

func (c *BugCache) HumanId() string {
	return c.bug.HumanId()
}

func (c *BugCache) notifyUpdated() error {
	return c.repoCache.bugUpdated(c.bug.Id())
}

func (c *BugCache) AddComment(message string) error {
	return c.AddCommentWithFiles(message, nil)
}

func (c *BugCache) AddCommentWithFiles(message string, files []git.Hash) error {
	author, err := bug.GetUser(c.repoCache.repo)
	if err != nil {
		return err
	}

	return c.AddCommentRaw(author, time.Now().Unix(), message, files, nil)
}

func (c *BugCache) AddCommentRaw(author bug.Person, unixTime int64, message string, files []git.Hash, metadata map[string]string) error {
	err := operations.CommentWithFiles(c.bug, author, unixTime, message, files)
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

	return c.ChangeLabelsRaw(author, time.Now().Unix(), added, removed)
}

func (c *BugCache) ChangeLabelsRaw(author bug.Person, unixTime int64, added []string, removed []string) ([]operations.LabelChangeResult, error) {
	changes, err := operations.ChangeLabels(c.bug, author, unixTime, added, removed)
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

	return c.OpenRaw(author, time.Now().Unix())
}

func (c *BugCache) OpenRaw(author bug.Person, unixTime int64) error {
	err := operations.Open(c.bug, author, unixTime)
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

	return c.CloseRaw(author, time.Now().Unix())
}

func (c *BugCache) CloseRaw(author bug.Person, unixTime int64) error {
	err := operations.Close(c.bug, author, unixTime)
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

	return c.SetTitleRaw(author, time.Now().Unix(), title)
}

func (c *BugCache) SetTitleRaw(author bug.Person, unixTime int64, title string) error {
	err := operations.SetTitle(c.bug, author, unixTime, title)
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
