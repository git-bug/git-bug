package cache

import (
	"fmt"
	"time"

	"github.com/git-bug/git-bug/entities/board"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/repository"
)

var _ board.Mutable = &BoardCache{}

// BoardCache is a wrapper around a Board. It provides multiple functions:
//
// 1. Provides a higher level API to use than the raw API from Board.
// 2. Maintain an up-to-date Snapshot available.
// 3. Deal with concurrency.
//
// It is a view of the board loaded in the cache, see CachedEntityBase: what it
// stages is its own until committed.
type BoardCache struct {
	CachedEntityBase[*board.Snapshot, board.Operation]
}

func newBoardCache(shared *sharedBoard, repo repository.Repo, resolvers func() entity.Resolvers, getUserIdentity getUserIdentityFunc, onCommit func() error) *BoardCache {
	return &BoardCache{
		CachedEntityBase: CachedEntityBase[*board.Snapshot, board.Operation]{
			repo:            repo,
			resolvers:       resolvers,
			onCommit:        onCommit,
			getUserIdentity: getUserIdentity,
			shared:          shared,
		},
	}
}

func (c *BoardCache) AddItemDraft(columnId entity.CombinedId, title string) (entity.CombinedId, *board.AddItemDraftOperation, error) {
	author, err := c.getUserIdentity()
	if err != nil {
		return entity.UnsetCombinedId, nil, err
	}

	return c.AddItemDraftRaw(author, time.Now().Unix(), columnId, title, nil)
}

func (c *BoardCache) AddItemDraftRaw(author identity.Interface, unixTime int64, columnId entity.CombinedId, title string, metadata map[string]string) (entity.CombinedId, *board.AddItemDraftOperation, error) {
	column, err := c.Snapshot().SearchColumn(columnId)
	if err != nil {
		return entity.UnsetCombinedId, nil, err
	}

	return board.AddItemDraft(c, author, unixTime, column.Id, title, metadata)
}

// AddItemEntity adds an item referring to the given entity (a bug ...) to a column.
func (c *BoardCache) AddItemEntity(columnId entity.CombinedId, e entity.Resolved) (entity.CombinedId, *board.AddItemEntityOperation, error) {
	author, err := c.getUserIdentity()
	if err != nil {
		return entity.UnsetCombinedId, nil, err
	}

	return c.AddItemEntityRaw(author, time.Now().Unix(), columnId, e, nil)
}

func (c *BoardCache) AddItemEntityRaw(author identity.Interface, unixTime int64, columnId entity.CombinedId, e entity.Resolved, metadata map[string]string) (entity.CombinedId, *board.AddItemEntityOperation, error) {
	column, err := c.Snapshot().SearchColumn(columnId)
	if err != nil {
		return entity.UnsetCombinedId, nil, err
	}

	var entityType board.ItemEntityType
	switch e.(type) {
	case *BugCache, *BugExcerpt:
		entityType = board.EntityTypeBug
	default:
		return entity.UnsetCombinedId, nil, fmt.Errorf("unsupported entity type %T", e)
	}

	return board.AddItemEntity(c, author, unixTime, column.Id, entityType, e.Id(), metadata)
}

func (c *BoardCache) SetDescription(description string) (*board.SetDescriptionOperation, error) {
	author, err := c.getUserIdentity()
	if err != nil {
		return nil, err
	}

	return c.SetDescriptionRaw(author, time.Now().Unix(), description, nil)
}

func (c *BoardCache) SetDescriptionRaw(author identity.Interface, unixTime int64, description string, metadata map[string]string) (*board.SetDescriptionOperation, error) {
	return board.SetDescription(c, author, unixTime, description, metadata)
}

func (c *BoardCache) SetTitle(title string) (*board.SetTitleOperation, error) {
	author, err := c.getUserIdentity()
	if err != nil {
		return nil, err
	}

	return c.SetTitleRaw(author, time.Now().Unix(), title, nil)
}

func (c *BoardCache) SetTitleRaw(author identity.Interface, unixTime int64, title string, metadata map[string]string) (*board.SetTitleOperation, error) {
	return board.SetTitle(c, author, unixTime, title, metadata)
}
