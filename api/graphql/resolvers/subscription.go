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

func (s subscriptionResolver) AllEvents(ctx context.Context, repoFilter *string) (<-chan *models.EntityEvent, error) {
	// TODO implement me
	panic("implement me")
}

var _ cache.Observer = &subscription[any]{}

type subscription[T any] struct {
	out             chan *T
	filter          func(T) bool
	excerptResolver func(id entity.Id) cache.Excerpt
	repo            *cache.RepoCache
}

func (s subscription[T]) EntityEvent(event cache.EntityEventType, typename string, id entity.Id) {

}

func (s subscriptionResolver) IdentityEvents(ctx context.Context, repoFilter *string) (<-chan *models.IdentityEvent, error) {
	out := make(chan *models.IdentityEvent)
	sub := &subscription[models.IdentityEvent]{
		out: out,
		filter: func(e models.IdentityEvent) bool { return true},
		excerptResolver: s.cache.
	}
}

func (s subscriptionResolver) BugEvents(ctx context.Context, repoFilter *string, query *string) (<-chan *models.BugEvent, error) {
	// TODO implement me
	panic("implement me")
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
