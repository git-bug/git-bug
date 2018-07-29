package resolvers

import (
	"context"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/graphql/models"
)

type rootQueryResolver struct {
	cache cache.Cacher
}

func (r rootQueryResolver) DefaultRepository(ctx context.Context) (*models.Repository, error) {
	repo, err := r.cache.DefaultRepo()

	if err != nil {
		return nil, err
	}

	return &repoResolver{
		cache: r.cache,
		repo:  repo,
	}, nil
}

func (r rootQueryResolver) Repository(ctx context.Context, id string) (*models.Repository, error) {
	repo, err := r.cache.ResolveRepo(id)

	if err != nil {
		return nil, err
	}

	return &repoResolver{
		cache: r.cache,
		repo:  repo,
	}, nil
}

