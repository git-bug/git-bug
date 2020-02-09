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

func (r rootQueryResolver) DefaultRepository(_ context.Context) (*models.Repository, error) {
	repo, err := r.cache.DefaultRepo()

	if err != nil {
		return nil, err
	}

	return &models.Repository{
		Cache: r.cache,
		Repo:  repo,
	}, nil
}

func (r rootQueryResolver) Repository(_ context.Context, ref string) (*models.Repository, error) {
	repo, err := r.cache.ResolveRepo(ref)

	if err != nil {
		return nil, err
	}

	return &models.Repository{
		Cache: r.cache,
		Repo:  repo,
	}, nil
}
