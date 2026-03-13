package models

import (
	"sync"
	"time"

	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/entities/board"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
)

// BoardWrapper is an interface used by the GraphQL resolvers to handle a board.
// Depending on the situation, a Board can already be fully loaded in memory or not.
// This interface is used to wrap either a lazyBoard or a loadedBoard depending on the situation.
type BoardWrapper interface {
	Id() entity.Id
	LastEdit() time.Time
	CreatedAt() time.Time

	Title() string
	Description() string
	Columns() ([]*board.Column, error)

	Actors() ([]IdentityWrapper, error)
	Operations() ([]dag.Operation, error)
}

var _ BoardWrapper = &lazyBoard{}

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

func (lb *lazyBoard) Columns() ([]*board.Column, error) {
	err := lb.load()
	if err != nil {
		return nil, err
	}
	return lb.snap.Columns, nil
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

var _ BoardWrapper = &loadedBoard{}

type loadedBoard struct {
	*board.Snapshot
}

func NewLoadedBoard(snap *board.Snapshot) *loadedBoard {
	return &loadedBoard{Snapshot: snap}
}

func (l *loadedBoard) LastEdit() time.Time {
	return l.Snapshot.EditTime()
}

func (l *loadedBoard) CreatedAt() time.Time {
	return l.Snapshot.CreateTime
}

func (l *loadedBoard) Title() string {
	return l.Snapshot.Title
}

func (l *loadedBoard) Description() string {
	return l.Snapshot.Description
}

func (l *loadedBoard) Columns() ([]*board.Column, error) {
	return l.Snapshot.Columns, nil
}

func (l *loadedBoard) Actors() ([]IdentityWrapper, error) {
	res := make([]IdentityWrapper, len(l.Snapshot.Actors))
	for i, actor := range l.Snapshot.Actors {
		res[i] = NewLoadedIdentity(actor)
	}
	return res, nil
}

func (l *loadedBoard) Operations() ([]dag.Operation, error) {
	return l.Snapshot.Operations, nil
}
