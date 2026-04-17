package resolvers

import (
	"context"

	"github.com/git-bug/git-bug/api/graphql/connections"
	"github.com/git-bug/git-bug/api/graphql/graph"
	"github.com/git-bug/git-bug/api/graphql/models"
	"github.com/git-bug/git-bug/api/repoctx"
	"github.com/git-bug/git-bug/cache"
)

var _ graph.QueryResolver = &rootQueryResolver{}

type rootQueryResolver struct {
	cache *cache.MultiRepoCache
}

// Repository resolves a repository by explicit ref, or (in multi-repo mode)
// falls back to the name carried in the request context (set from the
// X-Repo-Name HTTP header by the webui middleware). Unchanged for single-
// repo mode: ref==nil and only one repo registered still returns it.
func (r rootQueryResolver) Repository(ctx context.Context, ref *string) (*models.Repository, error) {
	var repo *cache.RepoCache
	var err error

	switch {
	case ref != nil:
		repo, err = r.cache.ResolveRepo(*ref)
	default:
		if name := repoctx.Name(ctx); name != "" {
			repo, err = r.cache.ResolveRepo(name)
		} else {
			repo, err = r.cache.DefaultRepo()
		}
	}

	if err != nil {
		return nil, nil
	}

	return &models.Repository{
		Repo: repo,
	}, nil
}

// Repositories returns all registered repositories as a relay connection.
func (r rootQueryResolver) Repositories(_ context.Context, after *string, before *string, first *int, last *int) (*models.RepositoryConnection, error) {
	input := models.ConnectionInput{
		After:  after,
		Before: before,
		First:  first,
		Last:   last,
	}

	source := r.cache.AllRepos()

	edger := func(repo *cache.RepoCache, offset int) connections.Edge {
		return models.RepositoryEdge{
			Node:   &models.Repository{Repo: repo},
			Cursor: connections.OffsetToCursor(offset),
		}
	}

	// NodeType is *cache.RepoCache (the source slice element), but the connection
	// nodes field wants []*models.Repository. Extract them from the edges, which
	// already hold the wrapped Repository built by the edger above.
	conMaker := func(edges []*models.RepositoryEdge, _ []*cache.RepoCache, info *models.PageInfo, totalCount int) (*models.RepositoryConnection, error) {
		nodes := make([]*models.Repository, len(edges))
		for i, e := range edges {
			nodes[i] = e.Node
		}
		return &models.RepositoryConnection{
			Edges:      edges,
			Nodes:      nodes,
			PageInfo:   info,
			TotalCount: totalCount,
		}, nil
	}

	return connections.Connection(source, edger, conMaker, input)
}
