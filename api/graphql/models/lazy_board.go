package models

import (
	"sync"
	"time"

	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/entities/board"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
)

// BoardWrapper is an interface used by the GraphQL resolvers to handle a board,
// loaded from the cache only when necessary, see lazyBoard.
type BoardWrapper interface {
	Id() entity.Id
	LastEdit() time.Time
	CreatedAt() time.Time

	Title() string
	Description() string
	Columns() ([]*BoardColumn, error)

	Actors() ([]IdentityWrapper, error)
	Operations() ([]dag.Operation, error)

	// IsEntity is a sign post-method for gqlgen, to mark compliance to an interface.
	IsEntity()
}

var _ BoardWrapper = &lazyBoard{}

// lazyBoard is a lazy-loading wrapper that fetches data from the cache (BoardExcerpt) in priority,
// and load the complete board and snapshot only when necessary.
type lazyBoard struct {
	cache   *cache.RepoCache
	excerpt *cache.BoardExcerpt

	mu   sync.Mutex
	snap *board.Snapshot
}

func NewLazyBoard(cache *cache.RepoCache, excerpt *cache.BoardExcerpt) *lazyBoard {
	return &lazyBoard{
		cache:   cache,
		excerpt: excerpt,
	}
}

func (lb *lazyBoard) load() error {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if lb.snap != nil {
		return nil
	}

	b, err := lb.cache.Boards().Resolve(lb.excerpt.Id())
	if err != nil {
		return err
	}

	lb.snap = b.Snapshot()
	return nil
}

func (lb *lazyBoard) identity(id entity.Id) (IdentityWrapper, error) {
	i, err := lb.cache.Identities().ResolveExcerpt(id)
	if err != nil {
		return nil, err
	}
	return &lazyIdentity{cache: lb.cache, excerpt: i}, nil
}

func (lb *lazyBoard) Id() entity.Id {
	return lb.excerpt.Id()
}

func (lb *lazyBoard) LastEdit() time.Time {
	return lb.excerpt.EditTime()
}

func (lb *lazyBoard) CreatedAt() time.Time {
	return lb.excerpt.CreateTime()
}

func (lb *lazyBoard) Title() string {
	return lb.excerpt.Title
}

func (lb *lazyBoard) Description() string {
	return lb.excerpt.Description
}

func (lb *lazyBoard) Columns() ([]*BoardColumn, error) {
	err := lb.load()
	if err != nil {
		return nil, err
	}
	res := make([]*BoardColumn, len(lb.snap.Columns))
	for i, column := range lb.snap.Columns {
		res[i] = &BoardColumn{Repo: lb.cache, Column: column}
	}
	return res, nil
}

func (lb *lazyBoard) Actors() ([]IdentityWrapper, error) {
	result := make([]IdentityWrapper, len(lb.excerpt.Actors))
	for i, actorId := range lb.excerpt.Actors {
		actor, err := lb.identity(actorId)
		if err != nil {
			return nil, err
		}
		result[i] = actor
	}
	return result, nil
}

func (lb *lazyBoard) Operations() ([]dag.Operation, error) {
	err := lb.load()
	if err != nil {
		return nil, err
	}
	return lb.snap.Operations, nil
}

// IsEntity is a sign post-method for gqlgen, to mark compliance to an interface.
func (lb *lazyBoard) IsEntity() {}

// BoardColumn is a wrapper around a board Column that includes the Repo, to
// keep the repo context in sub-resolvers.
type BoardColumn struct {
	Repo *cache.RepoCache
	*board.Column
}

// BoardItemBug is a wrapper around a board BugItem that includes the Repo, to
// resolve the bug it refers to.
type BoardItemBug struct {
	Repo *cache.RepoCache
	*board.BugItem
}
