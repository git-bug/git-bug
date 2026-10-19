package cache

import (
	"errors"
	"time"

	"github.com/git-bug/git-bug/entities/board"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/repository"
)

// sharedBoard is the single loaded copy of a board, see sharedEntity.
type sharedBoard = sharedEntity[*board.Snapshot, board.Operation]

func newSharedBoard(b *board.Board) *sharedBoard {
	return &sharedBoard{Tracked: b}
}

type RepoCacheBoard struct {
	*SubCache[*sharedBoard, *BoardExcerpt, *BoardCache]
}

func NewRepoCacheBoard(repo repository.Repo,
	resolvers func() entity.Resolvers,
	getUserIdentity getUserIdentityFunc) *RepoCacheBoard {

	makeView := func(shared *sharedBoard, onCommit func() error) *BoardCache {
		return newBoardCache(shared, repo, resolvers, getUserIdentity, onCommit)
	}

	makeIndexData := func(b *sharedBoard) []string {
		snap := b.Compile()
		return []string{snap.Title, snap.Description}
	}

	actions := Actions[*sharedBoard]{
		ReadWithResolver: func(repo repository.Repo, resolvers entity.Resolvers, id entity.Id) (*sharedBoard, error) {
			b, err := board.ReadWithResolver(repo, resolvers, id)
			if err != nil {
				return nil, err
			}
			return newSharedBoard(b), nil
		},
		Refresh: func(repo repository.Repo, resolvers entity.Resolvers, shared *sharedBoard) (*sharedBoard, error) {
			return shared, shared.Repair(repo, resolvers)
		},
		Remove:       board.Remove,
		RemoveAll:    board.RemoveAll,
		MergeAll:     board.MergeAll,
		EnsureClocks: board.EnsureClocks,
	}

	sc := NewSubCache[*sharedBoard, *BoardExcerpt, *BoardCache](
		repo, resolvers, getUserIdentity,
		makeView, newBoardExcerpt, makeIndexData, actions,
		board.Typename, board.Namespace,
		formatVersion, defaultMaxLoadedBugs,
	)

	return &RepoCacheBoard{SubCache: sc}
}

// ResolveBoardCreateMetadata retrieve a board that has the exact given metadata on
// its Create operation, that is, the first operation. It fails if multiple boards
// match.
func (c *RepoCacheBoard) ResolveBoardCreateMetadata(key string, value string) (*BoardCache, error) {
	return c.ResolveMatcher(func(excerpt *BoardExcerpt) bool {
		return excerpt.CreateMetadata[key] == value
	})
}

// ResolveColumn finds the board and column id that matches the given prefix.
func (c *RepoCacheBoard) ResolveColumn(prefix string) (*BoardCache, entity.CombinedId, error) {
	boardPrefix, _ := entity.SeparateIds(prefix)
	boardCandidate := make([]entity.Id, 0, 5)

	// build a list of possible matching boards
	c.muMaps.RLock()
	for _, excerpt := range c.excerpts {
		if excerpt.Id().HasPrefix(boardPrefix) {
			boardCandidate = append(boardCandidate, excerpt.Id())
		}
	}
	c.muMaps.RUnlock()

	matchingBoardIds := make([]entity.Id, 0, 5)
	matchingColumnId := entity.UnsetCombinedId
	var matchingBoard *BoardCache

	// search for matching columns
	// searching every board candidate allow for some collision with the board prefix only,
	// before being refined with the full column prefix
	for _, boardId := range boardCandidate {
		b, err := c.Resolve(boardId)
		if err != nil {
			return nil, entity.UnsetCombinedId, err
		}

		for _, column := range b.Snapshot().Columns {
			if column.CombinedId.HasPrefix(prefix) {
				matchingBoardIds = append(matchingBoardIds, boardId)
				matchingBoard = b
				matchingColumnId = column.CombinedId
			}
		}
	}

	if len(matchingBoardIds) > 1 {
		return nil, entity.UnsetCombinedId, entity.NewErrMultipleMatch("board/column", matchingBoardIds)
	} else if len(matchingBoardIds) == 0 {
		return nil, entity.UnsetCombinedId, errors.New("column doesn't exist")
	}

	return matchingBoard, matchingColumnId, nil
}

// TODO: resolve item?

// New creates a new board.
// The new board is written in the repository (commit)
func (c *RepoCacheBoard) New(title, description string, columns []string) (*BoardCache, *board.CreateOperation, error) {
	author, err := c.getUserIdentity()
	if err != nil {
		return nil, nil, err
	}

	return c.NewRaw(author, time.Now().Unix(), title, description, columns, nil)
}

// NewDefaultColumns creates a new board with the default columns.
// The new board is written in the repository (commit)
func (c *RepoCacheBoard) NewDefaultColumns(title, description string) (*BoardCache, *board.CreateOperation, error) {
	return c.New(title, description, board.DefaultColumns)
}

// NewRaw create a new board with the given title, description, and columns.
// The new board is written in the repository (commit).
func (c *RepoCacheBoard) NewRaw(author identity.Interface, unixTime int64, title, description string, columns []string, metadata map[string]string) (*BoardCache, *board.CreateOperation, error) {
	b, op, err := board.Create(author, unixTime, title, description, columns, metadata)
	if err != nil {
		return nil, nil, err
	}

	err = b.Commit(c.repo)
	if err != nil {
		return nil, nil, err
	}

	cached, err := c.add(newSharedBoard(b))
	if err != nil {
		return nil, nil, err
	}

	return cached, op, nil
}
