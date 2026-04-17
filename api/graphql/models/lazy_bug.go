package models

import (
	"sync"
	"time"

	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/entities/bug"
	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
)

// BugWrapper is an interface used by the GraphQL resolvers to handle a bug.
// Depending on the situation, a Bug can already be fully loaded in memory or not.
// This interface is used to wrap either a lazyBug or a loadedBug depending on the situation.
type BugWrapper interface {
	Id() entity.Id
	LastEdit() time.Time
	Kind() common.Kind
	Status() common.Status
	Title() string
	Comments() ([]bug.Comment, error)
	Labels() []common.Label
	Author() (IdentityWrapper, error)
	Actors() ([]IdentityWrapper, error)
	Participants() ([]IdentityWrapper, error)
	CreatedAt() time.Time
	Timeline() ([]bug.TimelineItem, error)
	Operations() ([]dag.Operation, error)

	// BaseRef returns the PR's base branch, or nil for issues.
	BaseRef() *string
	// HeadRef returns the PR's head branch, or nil for issues.
	HeadRef() *string
	// HeadCommit returns the PR's current head commit hash, or nil for issues.
	HeadCommit() *string
	// MergeCommit returns the merge commit hash for merged PRs, else nil.
	MergeCommit() *string
	// Reviews returns the list of PR reviews. Empty for issues.
	Reviews() ([]bug.Review, error)

	// IsAuthored is a sign-post method for gqlgen, to mark compliance to an interface.
	IsAuthored()
	// IsEntity is a sign post-method for gqlgen, to mark compliance to an interface.
	IsEntity()
}

var _ BugWrapper = &lazyBug{}

// lazyBug is a lazy-loading wrapper that fetches data from the cache (BugExcerpt) in priority,
// and load the complete bug and snapshot only when necessary.
type lazyBug struct {
	cache   *cache.RepoCache
	excerpt *cache.BugExcerpt

	mu   sync.Mutex
	snap *bug.Snapshot
}

func NewLazyBug(cache *cache.RepoCache, excerpt *cache.BugExcerpt) *lazyBug {
	return &lazyBug{
		cache:   cache,
		excerpt: excerpt,
	}
}

func (lb *lazyBug) load() error {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if lb.snap != nil {
		return nil
	}

	b, err := lb.cache.Bugs().Resolve(lb.excerpt.Id())
	if err != nil {
		return err
	}

	lb.snap = b.Snapshot()
	return nil
}

func (lb *lazyBug) identity(id entity.Id) (IdentityWrapper, error) {
	i, err := lb.cache.Identities().ResolveExcerpt(id)
	if err != nil {
		return nil, err
	}
	return &lazyIdentity{cache: lb.cache, excerpt: i}, nil
}

func (lb *lazyBug) Id() entity.Id {
	return lb.excerpt.Id()
}

func (lb *lazyBug) LastEdit() time.Time {
	return lb.excerpt.EditTime()
}

func (lb *lazyBug) Kind() common.Kind {
	return lb.excerpt.Kind
}

func (lb *lazyBug) Status() common.Status {
	return lb.excerpt.Status
}

func (lb *lazyBug) Title() string {
	return lb.excerpt.Title
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (lb *lazyBug) BaseRef() *string     { return nilIfEmpty(lb.excerpt.BaseRef) }
func (lb *lazyBug) HeadRef() *string     { return nilIfEmpty(lb.excerpt.HeadRef) }
func (lb *lazyBug) HeadCommit() *string  { return nilIfEmpty(lb.excerpt.HeadCommit) }
func (lb *lazyBug) MergeCommit() *string { return nilIfEmpty(lb.excerpt.MergeCommit) }

func (lb *lazyBug) Reviews() ([]bug.Review, error) {
	if err := lb.load(); err != nil {
		return nil, err
	}
	return lb.snap.Reviews, nil
}

func (lb *lazyBug) Comments() ([]bug.Comment, error) {
	err := lb.load()
	if err != nil {
		return nil, err
	}
	return lb.snap.Comments, nil
}

func (lb *lazyBug) Labels() []common.Label {
	return lb.excerpt.Labels
}

func (lb *lazyBug) Author() (IdentityWrapper, error) {
	return lb.identity(lb.excerpt.AuthorId)
}

func (lb *lazyBug) Actors() ([]IdentityWrapper, error) {
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

func (lb *lazyBug) Participants() ([]IdentityWrapper, error) {
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

func (lb *lazyBug) CreatedAt() time.Time {
	return lb.excerpt.CreateTime()
}

func (lb *lazyBug) Timeline() ([]bug.TimelineItem, error) {
	err := lb.load()
	if err != nil {
		return nil, err
	}
	return lb.snap.Timeline, nil
}

func (lb *lazyBug) Operations() ([]dag.Operation, error) {
	err := lb.load()
	if err != nil {
		return nil, err
	}
	return lb.snap.Operations, nil
}

// IsAuthored is a sign-post method for gqlgen, to mark compliance to an interface.
func (lb *lazyBug) IsAuthored() {}

// IsEntity is a sign post-method for gqlgen, to mark compliance to an interface.
func (lb *lazyBug) IsEntity() {}

var _ BugWrapper = &loadedBug{}

type loadedBug struct {
	*bug.Snapshot
}

func NewLoadedBug(snap *bug.Snapshot) *loadedBug {
	return &loadedBug{Snapshot: snap}
}

func (l *loadedBug) LastEdit() time.Time {
	return l.Snapshot.EditTime()
}

func (l *loadedBug) Kind() common.Kind {
	return l.Snapshot.Kind
}

func (l *loadedBug) Status() common.Status {
	return l.Snapshot.Status
}

func (l *loadedBug) Title() string {
	return l.Snapshot.Title
}

func (l *loadedBug) BaseRef() *string     { return nilIfEmpty(l.Snapshot.BaseRef) }
func (l *loadedBug) HeadRef() *string     { return nilIfEmpty(l.Snapshot.HeadRef) }
func (l *loadedBug) HeadCommit() *string  { return nilIfEmpty(l.Snapshot.HeadCommit) }
func (l *loadedBug) MergeCommit() *string { return nilIfEmpty(l.Snapshot.MergeCommit) }

func (l *loadedBug) Reviews() ([]bug.Review, error) {
	return l.Snapshot.Reviews, nil
}

func (l *loadedBug) Comments() ([]bug.Comment, error) {
	return l.Snapshot.Comments, nil
}

func (l *loadedBug) Labels() []common.Label {
	return l.Snapshot.Labels
}

func (l *loadedBug) Author() (IdentityWrapper, error) {
	return NewLoadedIdentity(l.Snapshot.Author), nil
}

func (l *loadedBug) Actors() ([]IdentityWrapper, error) {
	res := make([]IdentityWrapper, len(l.Snapshot.Actors))
	for i, actor := range l.Snapshot.Actors {
		res[i] = NewLoadedIdentity(actor)
	}
	return res, nil
}

func (l *loadedBug) Participants() ([]IdentityWrapper, error) {
	res := make([]IdentityWrapper, len(l.Snapshot.Participants))
	for i, participant := range l.Snapshot.Participants {
		res[i] = NewLoadedIdentity(participant)
	}
	return res, nil
}

func (l *loadedBug) CreatedAt() time.Time {
	return l.Snapshot.CreateTime
}

func (l *loadedBug) Timeline() ([]bug.TimelineItem, error) {
	return l.Snapshot.Timeline, nil
}

func (l *loadedBug) Operations() ([]dag.Operation, error) {
	return l.Snapshot.Operations, nil
}

// IsAuthored is a sign-post method for gqlgen, to mark compliance to an interface.
func (l *loadedBug) IsAuthored() {}

// IsEntity is a sign post-method for gqlgen, to mark compliance to an interface.
func (l *loadedBug) IsEntity() {}
