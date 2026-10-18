package broker

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/entity"
)

func TestBroker(t *testing.T) {
	id1 := entity.DeriveId([]byte("1"))
	id2 := entity.DeriveId([]byte("2"))

	ready := func(sub *Subscriber) bool {
		select {
		case <-sub.Ready():
			return true
		default:
			return false
		}
	}

	t.Run("routes by topic", func(t *testing.T) {
		b := New()
		entitySub := b.Subscribe(EntityTopic("repo", "bug", id1))
		defer entitySub.Close()
		typeSub := b.Subscribe(TypeTopic("repo", "bug"))
		defer typeSub.Close()

		b.EntityEvent(cache.EntityEventUpdated, "repo", "bug", id1)
		b.EntityEvent(cache.EntityEventCreated, "repo", "bug", id2)
		b.EntityEvent(cache.EntityEventRemoved, "repo", "identity", id1)
		b.EntityEvent(cache.EntityEventRemoved, "other", "bug", id1)

		require.True(t, ready(entitySub))
		require.Equal(t, map[entity.Id]cache.EntityEventType{
			id1: cache.EntityEventUpdated,
		}, entitySub.Take())

		require.True(t, ready(typeSub))
		require.Equal(t, map[entity.Id]cache.EntityEventType{
			id1: cache.EntityEventUpdated,
			id2: cache.EntityEventCreated,
		}, typeSub.Take())

		require.False(t, ready(entitySub))
		require.Nil(t, entitySub.Take())
	})

	t.Run("merges the changes not taken", func(t *testing.T) {
		for _, tc := range []struct {
			name   string
			events []cache.EntityEventType
			want   cache.EntityEventType
		}{
			{"created then updated", []cache.EntityEventType{cache.EntityEventCreated, cache.EntityEventUpdated}, cache.EntityEventCreated},
			{"created then removed", []cache.EntityEventType{cache.EntityEventCreated, cache.EntityEventRemoved}, cache.EntityEventRemoved},
			{"updated then removed", []cache.EntityEventType{cache.EntityEventUpdated, cache.EntityEventRemoved}, cache.EntityEventRemoved},
			{"removed then created", []cache.EntityEventType{cache.EntityEventRemoved, cache.EntityEventCreated}, cache.EntityEventUpdated},
			{"updated twice", []cache.EntityEventType{cache.EntityEventUpdated, cache.EntityEventUpdated}, cache.EntityEventUpdated},
		} {
			t.Run(tc.name, func(t *testing.T) {
				b := New()
				sub := b.Subscribe(EntityTopic("repo", "bug", id1))
				defer sub.Close()
				for _, event := range tc.events {
					b.EntityEvent(event, "repo", "bug", id1)
				}
				require.Equal(t, map[entity.Id]cache.EntityEventType{id1: tc.want}, sub.Take())
			})
		}
	})

	t.Run("never waits on a subscriber", func(t *testing.T) {
		b := New()
		sub := b.Subscribe(TypeTopic("repo", "bug"))
		defer sub.Close()

		// nothing takes the changes meanwhile
		for range 1000 {
			b.EntityEvent(cache.EntityEventUpdated, "repo", "bug", id1)
			b.EntityEvent(cache.EntityEventUpdated, "repo", "bug", id2)
		}
		require.Len(t, sub.Take(), 2)
	})

	t.Run("stops delivering when closed", func(t *testing.T) {
		b := New()
		sub := b.Subscribe(EntityTopic("repo", "bug", id1), TypeTopic("repo", "bug"))
		sub.Close()

		b.EntityEvent(cache.EntityEventUpdated, "repo", "bug", id1)
		require.False(t, ready(sub))
		require.Empty(t, b.topics)
	})
}
