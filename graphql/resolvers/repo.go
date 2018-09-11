package resolvers

import (
	"context"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/graphql/connections"
	"github.com/MichaelMure/git-bug/graphql/models"
)

type repoResolver struct{}

func (repoResolver) AllBugs(ctx context.Context, obj *models.Repository, after *string, before *string, first *int, last *int, queryStr *string) (models.BugConnection, error) {
	input := models.ConnectionInput{
		Before: before,
		After:  after,
		First:  first,
		Last:   last,
	}

	var query *cache.Query
	if queryStr != nil {
		query2, err := cache.ParseQuery(*queryStr)
		if err != nil {
			return models.BugConnection{}, err
		}
		query = query2
	} else {
		query = cache.NewQuery()
	}

	// Simply pass a []string with the ids to the pagination algorithm
	source := obj.Repo.QueryBugs(query)

	// The edger create a custom edge holding just the id
	edger := func(id string, offset int) connections.Edge {
		return connections.LazyBugEdge{
			Id:     id,
			Cursor: connections.OffsetToCursor(offset),
		}
	}

	// The conMaker will finally load and compile bugs from git to replace the selected edges
	conMaker := func(lazyBugEdges []connections.LazyBugEdge, lazyNode []string, info models.PageInfo, totalCount int) (models.BugConnection, error) {
		edges := make([]models.BugEdge, len(lazyBugEdges))
		nodes := make([]bug.Snapshot, len(lazyBugEdges))

		for i, lazyBugEdge := range lazyBugEdges {
			b, err := obj.Repo.ResolveBug(lazyBugEdge.Id)

			if err != nil {
				return models.BugConnection{}, err
			}

			snap := b.Snapshot()

			edges[i] = models.BugEdge{
				Cursor: lazyBugEdge.Cursor,
				Node:   *snap,
			}
			nodes[i] = *snap
		}

		return models.BugConnection{
			Edges:      edges,
			Nodes:      nodes,
			PageInfo:   info,
			TotalCount: totalCount,
		}, nil
	}

	return connections.StringCon(source, edger, conMaker, input)
}

func (repoResolver) Bug(ctx context.Context, obj *models.Repository, prefix string) (*bug.Snapshot, error) {
	b, err := obj.Repo.ResolveBugPrefix(prefix)

	if err != nil {
		return nil, err
	}

	return b.Snapshot(), nil
}
