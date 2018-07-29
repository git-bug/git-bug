package resolvers

import (
	"context"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
)

type mutationResolver struct {
	cache cache.Cacher
}

func (r mutationResolver) getRepo(repoRef *string) (cache.RepoCacher, error) {
	if repoRef != nil {
		return r.cache.ResolveRepo(*repoRef)
	}

	return r.cache.DefaultRepo()
}

func (r mutationResolver) NewBug(ctx context.Context, repoRef *string, title string, message string) (bug.Snapshot, error) {
	repo, err := r.getRepo(repoRef)
	if err != nil {
		return bug.Snapshot{}, err
	}

	b, err := repo.NewBug(title, message)
	if err != nil {
		return bug.Snapshot{}, err
	}

	snap := b.Snapshot()

	return *snap, nil
}
