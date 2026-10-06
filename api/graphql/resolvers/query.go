package resolvers

import (
	"context"
	"maps"
	"slices"

	"github.com/git-bug/git-bug/api/graphql/connections"
	"github.com/git-bug/git-bug/api/graphql/graph"
	"github.com/git-bug/git-bug/api/graphql/models"
	"github.com/git-bug/git-bug/cache"
)

var _ graph.QueryResolver = &rootQueryResolver{}

type rootQueryResolver struct {
	cache *cache.MultiRepoCache
}

func (r rootQueryResolver) Repository(_ context.Context, ref *string) (*models.Repository, error) {
	var name string
	var repo *cache.RepoCache
	var err error

	if ref == nil {
		name, repo, err = r.cache.DefaultRepo()
	} else {
		name = *ref
		repo, err = r.cache.ResolveRepo(name)
	}

	if err != nil {
		return nil, nil
	}

	return &models.Repository{
		Name: name,
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

	// sorted by name, for the cursors to be stable
	repos := r.cache.AllRepos()
	source := make([]*models.Repository, 0, len(repos))
	for _, name := range slices.Sorted(maps.Keys(repos)) {
		source = append(source, &models.Repository{Name: name, Repo: repos[name]})
	}

	edger := func(repo *models.Repository, offset int) connections.Edge {
		return models.RepositoryEdge{
			Node:   repo,
			Cursor: connections.OffsetToCursor(offset),
		}
	}

	conMaker := func(edges []*models.RepositoryEdge, _ []*models.Repository, info *models.PageInfo, totalCount int) (*models.RepositoryConnection, error) {
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
