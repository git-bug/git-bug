// Package broker fans out the entity events of the cache to many subscribers,
// each interested in a few topics.
package broker

import (
	"sync"

	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/entity"
)

// Topic is what a subscriber is interested in: a single entity of a type in a
// repository, or every entity of that type when Id is empty.
type Topic struct {
	Repo     string
	Typename string
	Id       entity.Id
}

// EntityTopic is the topic of a single entity.
func EntityTopic(repo string, typename string, id entity.Id) Topic {
	return Topic{Repo: repo, Typename: typename, Id: id}
}

// TypeTopic is the topic of every entity of a type.
func TypeTopic(repo string, typename string) Topic {
	return Topic{Repo: repo, Typename: typename}
}

var _ cache.Observer = &Broker{}

// Broker is an observer of the cache passing each event to the subscribers of
// its topics. Passing an event never waits on a subscriber: the event is
// recorded in the subscriber's mailbox, merged with those it hasn't taken yet,
// and the subscriber takes them at its own pace.
type Broker struct {
	mu     sync.Mutex
	topics map[Topic]map[*Subscriber]struct{}
}

func New() *Broker {
	return &Broker{topics: make(map[Topic]map[*Subscriber]struct{})}
}

// EntityEvent implements cache.Observer.
func (b *Broker) EntityEvent(event cache.EntityEventType, repoName string, typename string, id entity.Id) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for sub := range b.topics[EntityTopic(repoName, typename, id)] {
		sub.post(id, event)
	}
	for sub := range b.topics[TypeTopic(repoName, typename)] {
		sub.post(id, event)
	}
}

// Subscribe returns a subscriber to the given topics, to Close when done.
func (b *Broker) Subscribe(topics ...Topic) *Subscriber {
	sub := &Subscriber{
		broker:  b,
		topics:  topics,
		ready:   make(chan struct{}, 1),
		pending: make(map[entity.Id]cache.EntityEventType),
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	for _, topic := range topics {
		subs, ok := b.topics[topic]
		if !ok {
			subs = make(map[*Subscriber]struct{})
			b.topics[topic] = subs
		}
		subs[sub] = struct{}{}
	}
	return sub
}

func (b *Broker) unsubscribe(sub *Subscriber) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, topic := range sub.topics {
		delete(b.topics[topic], sub)
		if len(b.topics[topic]) == 0 {
			delete(b.topics, topic)
		}
	}
}

// Subscriber is the mailbox of the changes of its topics. Its memory is bounded
// by the entities of its topics, however slowly it takes the changes.
type Subscriber struct {
	broker *Broker
	topics []Topic

	// ready holds a token while there are pending changes
	ready chan struct{}

	mu      sync.Mutex
	pending map[entity.Id]cache.EntityEventType
}

func (s *Subscriber) post(id entity.Id, event cache.EntityEventType) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if prev, ok := s.pending[id]; ok {
		event = merge(prev, event)
	}
	s.pending[id] = event

	select {
	case s.ready <- struct{}{}:
	default:
	}
}

// merge folds two events of the same entity into the one telling what happened
// between the state the subscriber last knew and the current one.  The client
// also reads entities through queries, so it may know of one the subscriber
// wasn't told about yet.
func merge(prev, next cache.EntityEventType) cache.EntityEventType {
	switch {
	case prev == cache.EntityEventCreated && next == cache.EntityEventUpdated:
		// still new to the subscriber
		return prev
	case prev == cache.EntityEventRemoved && next == cache.EntityEventCreated:
		// the subscriber may know of a previous state
		return cache.EntityEventUpdated
	default:
		return next
	}
}

// Ready signals that there are changes to Take.
func (s *Subscriber) Ready() <-chan struct{} {
	return s.ready
}

// Take returns the pending events by entity, and clears them.
func (s *Subscriber) Take() map[entity.Id]cache.EntityEventType {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.pending) == 0 {
		return nil
	}
	taken := s.pending
	s.pending = make(map[entity.Id]cache.EntityEventType)
	return taken
}

// Close stops the delivery of the changes to the subscriber.
func (s *Subscriber) Close() {
	s.broker.unsubscribe(s)
}
