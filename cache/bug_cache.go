package cache

import (
	"fmt"
	"time"

	"github.com/git-bug/git-bug/entities/bug"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/repository"
)

var ErrNoMatchingOp = fmt.Errorf("no matching operation found")

var _ bug.Mutable = &BugCache{}

// BugCache is a wrapper around a Bug. It provides multiple functions:
//
// 1. Provide a higher level API to use than the raw API from Bug.
// 2. Maintain an up-to-date Snapshot available.
// 3. Deal with concurrency.
//
// It is a view of the bug loaded in the cache, see CachedEntityBase: what it
// stages is its own until committed.
type BugCache struct {
	CachedEntityBase[*bug.Snapshot, bug.Operation]
}

func newBugCache(shared *sharedBug, repo repository.ClockedRepo, resolvers func() entity.Resolvers, getUserIdentity getUserIdentityFunc, onCommit func() error) *BugCache {
	return &BugCache{
		CachedEntityBase: CachedEntityBase[*bug.Snapshot, bug.Operation]{
			repo:            repo,
			resolvers:       resolvers,
			onCommit:        onCommit,
			getUserIdentity: getUserIdentity,
			shared:          shared,
		},
	}
}

func (c *BugCache) AddComment(message string) (entity.CombinedId, *bug.AddCommentOperation, error) {
	return c.AddCommentWithFiles(message, nil)
}

func (c *BugCache) AddCommentWithFiles(message string, files []repository.Hash) (entity.CombinedId, *bug.AddCommentOperation, error) {
	author, err := c.getUserIdentity()
	if err != nil {
		return entity.UnsetCombinedId, nil, err
	}

	return c.AddCommentRaw(author, time.Now().Unix(), message, files, nil)
}

func (c *BugCache) AddCommentRaw(author identity.Interface, unixTime int64, message string, files []repository.Hash, metadata map[string]string) (entity.CombinedId, *bug.AddCommentOperation, error) {
	return bug.AddComment(c, author, unixTime, message, files, metadata)
}

func (c *BugCache) ChangeLabels(added []string, removed []string) ([]bug.LabelChangeResult, *bug.LabelChangeOperation, error) {
	author, err := c.getUserIdentity()
	if err != nil {
		return nil, nil, err
	}

	return c.ChangeLabelsRaw(author, time.Now().Unix(), added, removed, nil)
}

func (c *BugCache) ChangeLabelsRaw(author identity.Interface, unixTime int64, added []string, removed []string, metadata map[string]string) ([]bug.LabelChangeResult, *bug.LabelChangeOperation, error) {
	return bug.ChangeLabels(c, author, unixTime, added, removed, metadata)
}

func (c *BugCache) ForceChangeLabels(added []string, removed []string) (*bug.LabelChangeOperation, error) {
	author, err := c.getUserIdentity()
	if err != nil {
		return nil, err
	}

	return c.ForceChangeLabelsRaw(author, time.Now().Unix(), added, removed, nil)
}

func (c *BugCache) ForceChangeLabelsRaw(author identity.Interface, unixTime int64, added []string, removed []string, metadata map[string]string) (*bug.LabelChangeOperation, error) {
	return bug.ForceChangeLabels(c, author, unixTime, added, removed, metadata)
}

func (c *BugCache) Open() (*bug.SetStatusOperation, error) {
	author, err := c.getUserIdentity()
	if err != nil {
		return nil, err
	}

	return c.OpenRaw(author, time.Now().Unix(), nil)
}

func (c *BugCache) OpenRaw(author identity.Interface, unixTime int64, metadata map[string]string) (*bug.SetStatusOperation, error) {
	return bug.Open(c, author, unixTime, metadata)
}

func (c *BugCache) Close() (*bug.SetStatusOperation, error) {
	author, err := c.getUserIdentity()
	if err != nil {
		return nil, err
	}

	return c.CloseRaw(author, time.Now().Unix(), nil)
}

func (c *BugCache) CloseRaw(author identity.Interface, unixTime int64, metadata map[string]string) (*bug.SetStatusOperation, error) {
	return bug.Close(c, author, unixTime, metadata)
}

func (c *BugCache) SetTitle(title string) (*bug.SetTitleOperation, error) {
	author, err := c.getUserIdentity()
	if err != nil {
		return nil, err
	}

	return c.SetTitleRaw(author, time.Now().Unix(), title, nil)
}

func (c *BugCache) SetTitleRaw(author identity.Interface, unixTime int64, title string, metadata map[string]string) (*bug.SetTitleOperation, error) {
	return bug.SetTitle(c, author, unixTime, title, metadata)
}

// EditCreateComment is a convenience function to edit the body of a bug (the first comment)
func (c *BugCache) EditCreateComment(body string) (entity.CombinedId, *bug.EditCommentOperation, error) {
	author, err := c.getUserIdentity()
	if err != nil {
		return entity.UnsetCombinedId, nil, err
	}

	return c.EditCreateCommentRaw(author, time.Now().Unix(), body, nil)
}

// EditCreateCommentRaw is a convenience function to edit the body of a bug (the first comment)
func (c *BugCache) EditCreateCommentRaw(author identity.Interface, unixTime int64, body string, metadata map[string]string) (entity.CombinedId, *bug.EditCommentOperation, error) {
	return bug.EditCreateComment(c, author, unixTime, body, nil, metadata)
}

func (c *BugCache) EditComment(target entity.CombinedId, message string) (*bug.EditCommentOperation, error) {
	author, err := c.getUserIdentity()
	if err != nil {
		return nil, err
	}

	return c.EditCommentRaw(author, time.Now().Unix(), target, message, nil)
}

func (c *BugCache) EditCommentRaw(author identity.Interface, unixTime int64, target entity.CombinedId, message string, metadata map[string]string) (*bug.EditCommentOperation, error) {
	comment, err := c.Snapshot().SearchComment(target)
	if err != nil {
		return nil, err
	}

	commentId, op, err := bug.EditComment(c, author, unixTime, comment.TargetId(), message, nil, metadata)
	if err != nil {
		return nil, err
	}
	if commentId != target {
		panic("EditComment returned unexpected comment id")
	}
	return op, nil
}

func (c *BugCache) SetMetadata(target entity.Id, newMetadata map[string]string) (*dag.SetMetadataOperation[*bug.Snapshot], error) {
	author, err := c.getUserIdentity()
	if err != nil {
		return nil, err
	}

	return c.SetMetadataRaw(author, time.Now().Unix(), target, newMetadata)
}

func (c *BugCache) SetMetadataRaw(author identity.Interface, unixTime int64, target entity.Id, newMetadata map[string]string) (*dag.SetMetadataOperation[*bug.Snapshot], error) {
	return bug.SetMetadata(c, author, unixTime, target, newMetadata)
}
