package models

import (
	"sync"
	"time"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/entity"
)

type BugWrapper interface {
	Id() entity.Id
	LastEdit() time.Time
	Status() bug.Status
	Title() string
	Comments() ([]bug.Comment, error)
	Labels() []bug.Label
	Author() (IdentityWrapper, error)
	Actors() ([]IdentityWrapper, error)
	Participants() ([]IdentityWrapper, error)
	CreatedAt() time.Time
	Timeline() ([]bug.TimelineItem, error)
	Operations() ([]bug.Operation, error)

	IsAuthored()
}

var _ BugWrapper = &LazyBug{}

type LazyBug struct {
	cache   *cache.RepoCache
	excerpt *cache.BugExcerpt

	mu   sync.Mutex
	snap *bug.Snapshot
}

func NewLazyBug(cache *cache.RepoCache, excerpt *cache.BugExcerpt) *LazyBug {
	return &LazyBug{
		cache:   cache,
		excerpt: excerpt,
	}
}

func (lb *LazyBug) load() error {
	if lb.snap != nil {
		return nil
	}

	lb.mu.Lock()
	defer lb.mu.Unlock()

	b, err := lb.cache.ResolveBug(lb.excerpt.Id)
	if err != nil {
		return err
	}

	lb.snap = b.Snapshot()
	return nil
}

func (lb *LazyBug) identity(id entity.Id) (IdentityWrapper, error) {
	i, err := lb.cache.ResolveIdentityExcerpt(id)
	if err != nil {
		return nil, err
	}
	return &LazyIdentity{cache: lb.cache, excerpt: i}, nil
}

// Sign post method for gqlgen
func (lb *LazyBug) IsAuthored() {}

func (lb *LazyBug) Id() entity.Id {
	return lb.excerpt.Id
}

func (lb *LazyBug) LastEdit() time.Time {
	return time.Unix(lb.excerpt.EditUnixTime, 0)
}

func (lb *LazyBug) Status() bug.Status {
	return lb.excerpt.Status
}

func (lb *LazyBug) Title() string {
	return lb.excerpt.Title
}

func (lb *LazyBug) Comments() ([]bug.Comment, error) {
	err := lb.load()
	if err != nil {
		return nil, err
	}
	return lb.snap.Comments, nil
}

func (lb *LazyBug) Labels() []bug.Label {
	return lb.excerpt.Labels
}

func (lb *LazyBug) Author() (IdentityWrapper, error) {
	return lb.identity(lb.excerpt.AuthorId)
}

func (lb *LazyBug) Actors() ([]IdentityWrapper, error) {
	result := make([]IdentityWrapper, len(lb.excerpt.Actors))
	for i, actorId := range lb.excerpt.Actors {
		actor, err := lb.identity(actorId)
		if err != nil {
			return nil, err
		}
		result[i] = actor
	}
	return result, nil
}

func (lb *LazyBug) Participants() ([]IdentityWrapper, error) {
	result := make([]IdentityWrapper, len(lb.excerpt.Participants))
	for i, participantId := range lb.excerpt.Participants {
		participant, err := lb.identity(participantId)
		if err != nil {
			return nil, err
		}
		result[i] = participant
	}
	return result, nil
}

func (lb *LazyBug) CreatedAt() time.Time {
	return time.Unix(lb.excerpt.CreateUnixTime, 0)
}

func (lb *LazyBug) Timeline() ([]bug.TimelineItem, error) {
	err := lb.load()
	if err != nil {
		return nil, err
	}
	return lb.snap.Timeline, nil
}

func (lb *LazyBug) Operations() ([]bug.Operation, error) {
	err := lb.load()
	if err != nil {
		return nil, err
	}
	result := make([]bug.Operation, len(lb.snap.Operations))
	for i, operation := range lb.snap.Operations {
		result[i] = operation
	}
	return result, nil
}

var _ BugWrapper = &LoadedBug{}

type LoadedBug struct {
	*bug.Snapshot
}

func NewLoadedBug(snap *bug.Snapshot) *LoadedBug {
	return &LoadedBug{Snapshot: snap}
}

func (l *LoadedBug) LastEdit() time.Time {
	return l.Snapshot.LastEditTime()
}

func (l *LoadedBug) Status() bug.Status {
	return l.Snapshot.Status
}

func (l *LoadedBug) Title() string {
	return l.Snapshot.Title
}

func (l *LoadedBug) Comments() ([]bug.Comment, error) {
	return l.Snapshot.Comments, nil
}

func (l *LoadedBug) Labels() []bug.Label {
	return l.Snapshot.Labels
}

func (l *LoadedBug) Author() (IdentityWrapper, error) {
	return NewLoadedIdentity(l.Snapshot.Author), nil
}

func (l *LoadedBug) Actors() ([]IdentityWrapper, error) {
	res := make([]IdentityWrapper, len(l.Snapshot.Actors))
	for i, actor := range l.Snapshot.Actors {
		res[i] = NewLoadedIdentity(actor)
	}
	return res, nil
}

func (l *LoadedBug) Participants() ([]IdentityWrapper, error) {
	res := make([]IdentityWrapper, len(l.Snapshot.Participants))
	for i, participant := range l.Snapshot.Participants {
		res[i] = NewLoadedIdentity(participant)
	}
	return res, nil
}

func (l *LoadedBug) CreatedAt() time.Time {
	return l.Snapshot.CreatedAt
}

func (l *LoadedBug) Timeline() ([]bug.TimelineItem, error) {
	return l.Snapshot.Timeline, nil
}

func (l *LoadedBug) Operations() ([]bug.Operation, error) {
	return l.Snapshot.Operations, nil
}
