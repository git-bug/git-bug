package resolvers

import (
	"context"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
)

type repoResolver struct {
	cache cache.Cacher
	repo  cache.RepoCacher
}

func (repoResolver) AllBugs(ctx context.Context, obj *repoResolver, input ConnectionInput) (BugConnection, error) {
	panic("implement me")
}

func (repoResolver) Bug(ctx context.Context, obj *repoResolver, prefix string) (*bug.Snapshot, error) {
	b, err := obj.repo.ResolveBugPrefix(prefix)

	if err != nil {
		return nil, err
	}

	return b.Snapshot(), nil
}
