package resolvers

import (
	"context"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/graphql/graph"
	"github.com/MichaelMure/git-bug/graphql/models"
)

var _ graph.QueryResolver = &rootQueryResolver{}

type rootQueryResolver struct {
	cache *cache.MultiRepoCache
}

func (r rootQueryResolver) DefaultRepository(ctx context.Context) (*models.Repository, error) {
	repo, err := r.cache.DefaultRepo()

	if err != nil {
		return nil, err
	}

	return &models.Repository{
		Cache: r.cache,
		Repo:  repo,
	}, nil
}

func (r rootQueryResolver) Repository(ctx context.Context, id string) (*models.Repository, error) {
	repo, err := r.cache.ResolveRepo(id)

	if err != nil {
		return nil, err
	}

	return &models.Repository{
		Cache: r.cache,
		Repo:  repo,
	}, nil
}
