package resolvers

import (
	"context"
	"github.com/MichaelMure/git-bug/cache"
)

type rootQueryResolver struct {
	cache cache.Cacher
}

func (r rootQueryResolver) DefaultRepository(ctx context.Context) (*repoResolver, error) {
	repo, err := r.cache.DefaultRepo()

	if err != nil {
		return nil, err
	}

	return &repoResolver{
		cache: r.cache,
		repo:  repo,
	}, nil
}

func (r rootQueryResolver) Repository(ctx context.Context, id string) (*repoResolver, error) {
	repo, err := r.cache.ResolveRepo(id)

	if err != nil {
		return nil, err
	}

	return &repoResolver{
		cache: r.cache,
		repo:  repo,
	}, nil
}
