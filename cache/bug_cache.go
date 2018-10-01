package cache

import (
	"time"

	"github.com/MichaelMure/git-bug/bug"
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
	op, err := bug.AddCommentWithFiles(c.bug, author, unixTime, message, files)
	if err != nil {
		return err
	}

	for key, value := range metadata {
		op.SetMetadata(key, value)
	}

	return c.notifyUpdated()
}

func (c *BugCache) ChangeLabels(added []string, removed []string) ([]bug.LabelChangeResult, error) {
	author, err := bug.GetUser(c.repoCache.repo)
	if err != nil {
		return nil, err
	}

	return c.ChangeLabelsRaw(author, time.Now().Unix(), added, removed, nil)
}

func (c *BugCache) ChangeLabelsRaw(author bug.Person, unixTime int64, added []string, removed []string, metadata map[string]string) ([]bug.LabelChangeResult, error) {
	changes, op, err := bug.ChangeLabels(c.bug, author, unixTime, added, removed)
	if err != nil {
		return changes, err
	}

	for key, value := range metadata {
		op.SetMetadata(key, value)
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

	return c.OpenRaw(author, time.Now().Unix(), nil)
}

func (c *BugCache) OpenRaw(author bug.Person, unixTime int64, metadata map[string]string) error {
	op, err := bug.Open(c.bug, author, unixTime)
	if err != nil {
		return err
	}

	for key, value := range metadata {
		op.SetMetadata(key, value)
	}

	return c.notifyUpdated()
}

func (c *BugCache) Close() error {
	author, err := bug.GetUser(c.repoCache.repo)
	if err != nil {
		return err
	}

	return c.CloseRaw(author, time.Now().Unix(), nil)
}

func (c *BugCache) CloseRaw(author bug.Person, unixTime int64, metadata map[string]string) error {
	op, err := bug.Close(c.bug, author, unixTime)
	if err != nil {
		return err
	}

	for key, value := range metadata {
		op.SetMetadata(key, value)
	}

	return c.notifyUpdated()
}

func (c *BugCache) SetTitle(title string) error {
	author, err := bug.GetUser(c.repoCache.repo)
	if err != nil {
		return err
	}

	return c.SetTitleRaw(author, time.Now().Unix(), title, nil)
}

func (c *BugCache) SetTitleRaw(author bug.Person, unixTime int64, title string, metadata map[string]string) error {
	op, err := bug.SetTitle(c.bug, author, unixTime, title)
	if err != nil {
		return err
	}

	for key, value := range metadata {
		op.SetMetadata(key, value)
	}

	return c.notifyUpdated()
}

func (c *BugCache) EditComment(target git.Hash, message string) error {
	author, err := bug.GetUser(c.repoCache.repo)
	if err != nil {
		return err
	}

	return c.EditCommentRaw(author, time.Now().Unix(), target, message, nil)
}

func (c *BugCache) EditCommentRaw(author bug.Person, unixTime int64, target git.Hash, message string, metadata map[string]string) error {
	op, err := bug.EditComment(c.bug, author, unixTime, target, message)
	if err != nil {
		return err
	}

	for key, value := range metadata {
		op.SetMetadata(key, value)
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
