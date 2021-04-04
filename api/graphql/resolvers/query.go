package resolvers

import (
	"context"

	"github.com/MichaelMure/git-bug/api/graphql/graph"
	"github.com/MichaelMure/git-bug/api/graphql/models"
	"github.com/MichaelMure/git-bug/cache"
)

var _ graph.QueryResolver = &rootQueryResolver{}

type rootQueryResolver struct {
	cache *cache.MultiRepoCache
}

func (r rootQueryResolver) Repository(_ context.Context, ref *string) (*models.Repository, error) {
	var repo *cache.RepoCache
	var err error

	if ref == nil {
		repo, err = r.cache.DefaultRepo()
	} else {
		repo, err = r.cache.ResolveRepo(*ref)
	}

	if err != nil {
		return nil, err
	}

	return &models.Repository{
		Cache: r.cache,
		Repo:  repo,
	}, nil
}
