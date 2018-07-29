package resolvers

import (
	"context"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/graphql/connections"
	"github.com/MichaelMure/git-bug/graphql/models"
)

type repoResolver struct {
	cache cache.Cacher
	repo  cache.RepoCacher
}

func (repoResolver) AllBugs(ctx context.Context, obj *repoResolver, input models.ConnectionInput) (models.BugConnection, error) {

	// Simply pass a []string with the ids to the pagination algorithm
	source, err := obj.repo.AllBugIds()

	if err != nil {
		return models.BugConnection{}, err
	}

	// The edger create a custom edge holding just the id
	edger := func(id string, offset int) connections.Edge {
		return connections.LazyBugEdge{
			Id:     id,
			Cursor: connections.OffsetToCursor(offset),
		}
	}

	// The conMaker will finally load and compile bugs from git to replace the selected edges
	conMaker := func(lazyBugEdges []connections.LazyBugEdge, info models.PageInfo, totalCount int) (models.BugConnection, error) {
		edges := make([]models.BugEdge, len(lazyBugEdges))

		for i, lazyBugEdge := range lazyBugEdges {
			b, err := obj.repo.ResolveBug(lazyBugEdge.Id)

			if err != nil {
				return models.BugConnection{}, err
			}

			snap := b.Snapshot()

			edges[i] = models.BugEdge{
				Cursor: lazyBugEdge.Cursor,
				Node:   *snap,
			}
		}

		return models.BugConnection{
			Edges:      edges,
			PageInfo:   info,
			TotalCount: totalCount,
		}, nil
	}

	return connections.StringCon(source, edger, conMaker, input)
}

func (repoResolver) Bug(ctx context.Context, obj *repoResolver, prefix string) (*bug.Snapshot, error) {
	b, err := obj.repo.ResolveBugPrefix(prefix)

	if err != nil {
		return nil, err
	}

	return b.Snapshot(), nil
}
