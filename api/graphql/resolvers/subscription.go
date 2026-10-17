package resolvers

import (
	"context"
	"time"

	"github.com/git-bug/git-bug/api/graphql/broker"
	"github.com/git-bug/git-bug/api/graphql/graph"
	"github.com/git-bug/git-bug/api/graphql/models"
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/entities/bug"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
)

var _ graph.SubscriptionResolver = &subscriptionResolver{}

// entitiesChangedInterval is the least time between two batches of
// entitiesChanged, so that a burst of changes, like a pull, makes few refetches.
const entitiesChangedInterval = time.Second

// cacheTypenames are the cache typenames of the entity types.
var cacheTypenames = map[models.EntityType]string{
	models.EntityTypeBug:      bug.Typename,
	models.EntityTypeIdentity: identity.Typename,
}

type subscriptionResolver struct {
	cache  *cache.MultiRepoCache
	broker *broker.Broker
}

func (s subscriptionResolver) EntitiesChanged(ctx context.Context, repoRef *string, typeArg models.EntityType, ids []entity.Id) (<-chan []*models.EntityChange, error) {
	repoName, err := s.resolveRepo(repoRef)
	if err != nil {
		return nil, err
	}

	typename := cacheTypenames[typeArg]
	topics := []broker.Topic{broker.TypeTopic(repoName, typename)}
	if ids != nil {
		topics = make([]broker.Topic, len(ids))
		for i, id := range ids {
			topics[i] = broker.EntityTopic(repoName, typename, id)
		}
	}
	sub := s.broker.Subscribe(topics...)

	out := make(chan []*models.EntityChange)
	go func() {
		defer close(out)
		defer sub.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case <-sub.Ready():
			}
			taken := sub.Take()
			if len(taken) == 0 {
				continue
			}
			changes := make([]*models.EntityChange, 0, len(taken))
			for id, event := range taken {
				changes = append(changes, &models.EntityChange{Event: event, ID: id})
			}
			select {
			case <-ctx.Done():
				return
			case out <- changes:
			}
			// the changes arriving meanwhile wait in the mailbox for the next batch
			select {
			case <-ctx.Done():
				return
			case <-time.After(entitiesChangedInterval):
			}
		}
	}()
	return out, nil
}

func (s subscriptionResolver) resolveRepo(repoRef *string) (string, error) {
	if repoRef == nil {
		name, _, err := s.cache.DefaultRepo()
		return name, err
	}
	_, err := s.cache.ResolveRepo(*repoRef)
	return *repoRef, err
}
