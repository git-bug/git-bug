package resolvers

import (
	"context"
	"fmt"

	"github.com/git-bug/git-bug/api/graphql/graph"
	"github.com/git-bug/git-bug/api/graphql/models"
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/entities/bug"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
)

var _ graph.SubscriptionResolver = &subscriptionResolver{}

type subscriptionResolver struct {
	cache *cache.MultiRepoCache
}

func (s subscriptionResolver) AllEvents(ctx context.Context, repoRef *string, typename *string) (<-chan *models.EntityEvent, error) {
	out := make(chan *models.EntityEvent)
	sub := &subscription[models.EntityEvent]{
		cache: s.cache,
		out:   out,
		makeEvent: func(repo *cache.RepoCache, excerpt cache.Excerpt, eventType cache.EntityEventType) *models.EntityEvent {
			switch excerpt := excerpt.(type) {
			case *cache.BugExcerpt:
				return &models.EntityEvent{Type: eventType, Entity: models.NewLazyBug(repo, excerpt)}
			case *cache.IdentityExcerpt:
				return &models.EntityEvent{Type: eventType, Entity: models.NewLazyIdentity(repo, excerpt)}
			default:
				panic(fmt.Sprintf("unknown excerpt type: %T", excerpt))
			}
		},
	}

	var repoRefStr string
	if repoRef != nil {
		repoRefStr = *repoRef
	}

	var typenameStr string
	if typename != nil {
		typenameStr = *typename
	}

	err := s.cache.RegisterObserver(sub, repoRefStr, typenameStr)
	if err != nil {
		return nil, err
	}

	go func() {
		<-ctx.Done()
		s.cache.UnregisterObserver(sub)
	}()

	return out, nil
}

func (s subscriptionResolver) BugEvents(ctx context.Context, repoRef *string) (<-chan *models.BugEvent, error) {
	out := make(chan *models.BugEvent)
	sub := &subscription[models.BugEvent]{
		cache: s.cache,
		out:   out,
		makeEvent: func(repo *cache.RepoCache, excerpt cache.Excerpt, event cache.EntityEventType) *models.BugEvent {
			return &models.BugEvent{Type: event, Bug: models.NewLazyBug(repo, excerpt.(*cache.BugExcerpt))}
		},
	}

	var repoRefStr string
	if repoRef != nil {
		repoRefStr = *repoRef
	}

	err := s.cache.RegisterObserver(sub, repoRefStr, bug.Typename)
	if err != nil {
		return nil, err
	}

	go func() {
		<-ctx.Done()
		s.cache.UnregisterObserver(sub)
	}()

	return out, nil
}

func (s subscriptionResolver) IdentityEvents(ctx context.Context, repoRef *string) (<-chan *models.IdentityEvent, error) {
	out := make(chan *models.IdentityEvent)
	sub := &subscription[models.IdentityEvent]{
		cache: s.cache,
		out:   out,
		makeEvent: func(repo *cache.RepoCache, excerpt cache.Excerpt, event cache.EntityEventType) *models.IdentityEvent {
			return &models.IdentityEvent{Type: event, Identity: models.NewLazyIdentity(repo, excerpt.(*cache.IdentityExcerpt))}
		},
	}

	var repoRefStr string
	if repoRef != nil {
		repoRefStr = *repoRef
	}

	err := s.cache.RegisterObserver(sub, repoRefStr, identity.Typename)
	if err != nil {
		return nil, err
	}

	go func() {
		<-ctx.Done()
		s.cache.UnregisterObserver(sub)
	}()

	return out, nil
}

var _ cache.Observer = &subscription[any]{}

type subscription[eventT any] struct {
	cache     *cache.MultiRepoCache
	out       chan *eventT
	filter    func(cache.Excerpt) bool
	makeEvent func(repo *cache.RepoCache, excerpt cache.Excerpt, event cache.EntityEventType) *eventT
}

func (s subscription[eventT]) EntityEvent(event cache.EntityEventType, repoName string, typename string, id entity.Id) {
	repo, err := s.cache.ResolveRepo(repoName)
	if err != nil {
		// something terrible happened
		return
	}
	var excerpt cache.Excerpt
	switch typename {
	case bug.Typename:
		excerpt, err = repo.Bugs().ResolveExcerpt(id)
	case identity.Typename:
		excerpt, err = repo.Identities().ResolveExcerpt(id)
	default:
		panic(fmt.Sprintf("unknown typename: %s", typename))
	}
	if s.filter != nil && !s.filter(excerpt) {
		return
	}
	s.out <- s.makeEvent(repo, excerpt, event)
}
