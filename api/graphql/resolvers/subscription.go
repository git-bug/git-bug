package resolvers

import (
	"context"
	"fmt"

	"github.com/git-bug/git-bug/api/graphql/graph"
	"github.com/git-bug/git-bug/api/graphql/models"
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/entities/bug"
	"github.com/git-bug/git-bug/entity"
)

var _ graph.SubscriptionResolver = &subscriptionResolver{}

type subscriptionResolver struct {
	cache *cache.MultiRepoCache
}

func (s subscriptionResolver) BugChanged(ctx context.Context, repoRef *string, query *string) (<-chan *models.BugChange, error) {
	var repo *cache.RepoCache
	var err error

	if repoRef == nil {
		repo, err = s.cache.DefaultRepo()
	} else {
		repo, err = s.cache.ResolveRepo(*repoRef)
	}

	if err != nil {
		return nil, err
	}

	out := make(chan *models.BugChange)
	sub := bugSubscription{out: out, repo: repo}
	repo.RegisterObserver(bug.Typename, sub)

	go func() {
		<-ctx.Done()
		repo.RegisterObserver(bug.Typename, sub)
	}()

	return out, nil
}

type bugSubscription struct {
	out  chan *models.BugChange
	repo *cache.RepoCache
}

func (bs bugSubscription) EntityCreated(_ string, id entity.Id) {
	excerpt, err := bs.repo.Bugs().ResolveExcerpt(id)
	if err != nil {
		// Should never happen
		fmt.Printf("bug in the cache: could not resolve excerpt for %s: %s\n", id, err)
		return
	}
	bs.out <- &models.BugChange{
		Type: models.ChangeTypeCreated,
		Bug:  models.NewLazyBug(bs.repo, excerpt),
	}
}

func (bs bugSubscription) EntityUpdated(_ string, id entity.Id) {
	excerpt, err := bs.repo.Bugs().ResolveExcerpt(id)
	if err != nil {
		// Should never happen
		fmt.Printf("bug in the cache: could not resolve excerpt for %s: %s\n", id, err)
		return
	}
	bs.out <- &models.BugChange{
		Type: models.ChangeTypeUpdated,
		Bug:  models.NewLazyBug(bs.repo, excerpt),
	}
}
