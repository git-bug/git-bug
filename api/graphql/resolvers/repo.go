package resolvers

import (
	"context"

	"github.com/git-bug/git-bug/api/auth"
	"github.com/git-bug/git-bug/api/graphql/connections"
	"github.com/git-bug/git-bug/api/graphql/graph"
	"github.com/git-bug/git-bug/api/graphql/models"
	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/query"
)

var _ graph.RepositoryResolver = &repoResolver{}

type repoResolver struct{}

func (repoResolver) Name(_ context.Context, obj *models.Repository) (*string, error) {
	name := obj.Repo.Name()
	return &name, nil
}

func (r repoResolver) AllBoards(ctx context.Context, obj *models.Repository, after *string, before *string, first *int, last *int, query *string) (*models.BoardConnection, error) {
	input := models.ConnectionInput{
		Before: before,
		After:  after,
		First:  first,
		Last:   last,
	}

	// Simply pass a []string with the ids to the pagination algorithm
	source := obj.Repo.Boards().AllIds()

	// The edger create a custom edge holding just the id
	edger := func(id entity.Id, offset int) connections.Edge {
		return connections.LazyBoardEdge{
			Id:     id,
			Cursor: connections.OffsetToCursor(offset),
		}
	}

	// The conMaker will finally load and compile boards from git to replace the selected edges
	conMaker := func(lazyBoardEdges []*connections.LazyBoardEdge, lazyNode []entity.Id, info *models.PageInfo, totalCount int) (*models.BoardConnection, error) {
		edges := make([]*models.BoardEdge, len(lazyBoardEdges))
		nodes := make([]models.BoardWrapper, len(lazyBoardEdges))

		for k, lazyBoardEdge := range lazyBoardEdges {
			excerpt, err := obj.Repo.Boards().ResolveExcerpt(lazyBoardEdge.Id)
			if err != nil {
				return nil, err
			}

			i := models.NewLazyBoard(obj.Repo, excerpt)

			edges[k] = &models.BoardEdge{
				Cursor: lazyBoardEdge.Cursor,
				Node:   i,
			}
			nodes[k] = i
		}

		return &models.BoardConnection{
			Edges:      edges,
			Nodes:      nodes,
			PageInfo:   info,
			TotalCount: totalCount,
		}, nil
	}

	return connections.Connection(source, edger, conMaker, input)
}

func (r repoResolver) Board(ctx context.Context, obj *models.Repository, prefix string) (models.BoardWrapper, error) {
	excerpt, err := obj.Repo.Boards().ResolveExcerptPrefix(prefix)
	if err != nil {
		return nil, err
	}

	return models.NewLazyBoard(obj.Repo, excerpt), nil
}

func (repoResolver) AllBugs(_ context.Context, obj *models.Repository, after *string, before *string, first *int, last *int, queryStr *string) (*models.BugConnection, error) {
	input := models.ConnectionInput{
		Before: before,
		After:  after,
		First:  first,
		Last:   last,
	}

	var q *query.Query
	if queryStr != nil {
		query2, err := query.Parse(*queryStr)
		if err != nil {
			return nil, err
		}
		q = query2
	} else {
		q = query.NewQuery()
	}

	// Simply pass a []string with the ids to the pagination algorithm
	source, err := obj.Repo.Bugs().Query(q)
	if err != nil {
		return nil, err
	}

	// The edger create a custom edge holding just the id
	edger := func(id entity.Id, offset int) connections.Edge {
		return connections.LazyBugEdge{
			Id:     id,
			Cursor: connections.OffsetToCursor(offset),
		}
	}

	// The conMaker will finally load and compile bugs from git to replace the selected edges
	conMaker := func(lazyBugEdges []*connections.LazyBugEdge, lazyNode []entity.Id, info *models.PageInfo, totalCount int) (*models.BugConnection, error) {
		edges := make([]*models.BugEdge, len(lazyBugEdges))
		nodes := make([]models.BugWrapper, len(lazyBugEdges))

		for i, lazyBugEdge := range lazyBugEdges {
			excerpt, err := obj.Repo.Bugs().ResolveExcerpt(lazyBugEdge.Id)
			if err != nil {
				return nil, err
			}

			b := models.NewLazyBug(obj.Repo, excerpt)

			edges[i] = &models.BugEdge{
				Cursor: lazyBugEdge.Cursor,
				Node:   b,
			}
			nodes[i] = b
		}

		return &models.BugConnection{
			Edges:      edges,
			Nodes:      nodes,
			PageInfo:   info,
			TotalCount: totalCount,
		}, nil
	}

	return connections.Connection(source, edger, conMaker, input)
}

func (repoResolver) Bug(_ context.Context, obj *models.Repository, prefix string) (models.BugWrapper, error) {
	excerpt, err := obj.Repo.Bugs().ResolveExcerptPrefix(prefix)
	if err != nil {
		return nil, err
	}

	return models.NewLazyBug(obj.Repo, excerpt), nil
}

func (repoResolver) AllIdentities(_ context.Context, obj *models.Repository, after *string, before *string, first *int, last *int) (*models.IdentityConnection, error) {
	input := models.ConnectionInput{
		Before: before,
		After:  after,
		First:  first,
		Last:   last,
	}

	// Simply pass a []string with the ids to the pagination algorithm
	source := obj.Repo.Identities().AllIds()

	// The edger create a custom edge holding just the id
	edger := func(id entity.Id, offset int) connections.Edge {
		return connections.LazyIdentityEdge{
			Id:     id,
			Cursor: connections.OffsetToCursor(offset),
		}
	}

	// The conMaker will finally load and compile identities from git to replace the selected edges
	conMaker := func(lazyIdentityEdges []*connections.LazyIdentityEdge, lazyNode []entity.Id, info *models.PageInfo, totalCount int) (*models.IdentityConnection, error) {
		edges := make([]*models.IdentityEdge, len(lazyIdentityEdges))
		nodes := make([]models.IdentityWrapper, len(lazyIdentityEdges))

		for k, lazyIdentityEdge := range lazyIdentityEdges {
			excerpt, err := obj.Repo.Identities().ResolveExcerpt(lazyIdentityEdge.Id)
			if err != nil {
				return nil, err
			}

			i := models.NewLazyIdentity(obj.Repo, excerpt)

			edges[k] = &models.IdentityEdge{
				Cursor: lazyIdentityEdge.Cursor,
				Node:   i,
			}
			nodes[k] = i
		}

		return &models.IdentityConnection{
			Edges:      edges,
			Nodes:      nodes,
			PageInfo:   info,
			TotalCount: totalCount,
		}, nil
	}

	return connections.Connection(source, edger, conMaker, input)
}

func (repoResolver) Identity(_ context.Context, obj *models.Repository, prefix string) (models.IdentityWrapper, error) {
	excerpt, err := obj.Repo.Identities().ResolveExcerptPrefix(prefix)
	if err != nil {
		return nil, err
	}

	return models.NewLazyIdentity(obj.Repo, excerpt), nil
}

func (repoResolver) UserIdentity(ctx context.Context, obj *models.Repository) (models.IdentityWrapper, error) {
	id, err := auth.UserFromCtx(ctx, obj.Repo)
	if err == auth.ErrNotAuthenticated {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return models.NewLoadedIdentity(id.Identity), nil
}

func (repoResolver) ValidLabels(_ context.Context, obj *models.Repository, after *string, before *string, first *int, last *int) (*models.LabelConnection, error) {
	input := models.ConnectionInput{
		Before: before,
		After:  after,
		First:  first,
		Last:   last,
	}

	edger := func(label common.Label, offset int) connections.Edge {
		return models.LabelEdge{
			Node:   label,
			Cursor: connections.OffsetToCursor(offset),
		}
	}

	conMaker := func(edges []*models.LabelEdge, nodes []common.Label, info *models.PageInfo, totalCount int) (*models.LabelConnection, error) {
		return &models.LabelConnection{
			Edges:      edges,
			Nodes:      nodes,
			PageInfo:   info,
			TotalCount: totalCount,
		}, nil
	}

	return connections.Connection(obj.Repo.Bugs().ValidLabels(), edger, conMaker, input)
}
