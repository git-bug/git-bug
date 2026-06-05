package iterator

import (
	"context"
	"errors"
	"strconv"
	"time"

	gitea "gitea.dev/sdk"
)

// Iterates all issues, along with their comments and labels.
// Not thread-safe.
type Iterator struct {
	// shared context
	ctx context.Context

	// to pass to sub-iterators
	conf config

	// sticky error
	err error

	issue *pageIterator[gitea.Issue]
	timeline *pageIterator[TimelineEvent]
}

type config struct {
	// gitea api v1 client
	gc *gitea.Client

	timeout time.Duration

	// if since is given the iterator will query only the issues
	// updated after this date
	since time.Time

	// name of the repository owner on Gitea
	owner string

	// name of the Gitea repository
	project string

	// number of issues and notes to query at once
	capacity int
}

// `more` is whether there are any remaining pages
type fetchPage[T any] = func(ctx context.Context, conf config, issue *gitea.Issue, page int) (items []*T, more bool, err error)

type pageIterator[T any] struct {
	page     int
	lastPage bool
	index    int
	cache    []*T

	fetch fetchPage[T]
}

// Currently not used: milestone, assignees, ref_issue
type TimelineEvent interface{ sealed() }

type CommentEvent struct {
	ID int64;
	Body string;
	Poster *gitea.User;
	Created, Updated time.Time;
}
func (*CommentEvent) sealed () {}

type RenameEvent struct {
}
func (*RenameEvent) sealed () {}

type ReopenEvent struct {
}
func (*ReopenEvent) sealed () {}

type CloseEvent struct {
}
func (*CloseEvent) sealed () {}

type LabelEvent struct {
	Label *gitea.Label;
	Poster *gitea.User;
	UpdatedAt time.Time;
	ID int;
	Kind LabelEventKind;
}
func (*LabelEvent) sealed () {}

type LabelEventKind int

const (
    LabelAdded LabelEventKind = iota
    LabelRemoved
)

func NewIterator(ctx context.Context, client *gitea.Client, capacity int, owner, project string, timeout time.Duration, since time.Time) *Iterator {
	return &Iterator{
		ctx: ctx,
		conf: config{
			gc:       client,
			timeout:  timeout,
			since:    since,
			owner:    owner,
			project:  project,
			capacity: capacity,
		},
		issue: newPageIterator[gitea.Issue](fetchIssues),
		timeline: newPageIterator[TimelineEvent](fetchTimeline),
	}
}

// Return last encountered error
func (i *Iterator) Error() error {
	return i.err
}

func (i *Iterator) NextIssue() bool {
	// Gracefully handle the case where we haven't started iterating.
	var currentIssue *gitea.Issue
	if i.issue.cache != nil {
		currentIssue = i.IssueValue()
	}
	more := i.advance(i.issue, currentIssue)

	if i.err != nil {
		return false
	}

	if more {
		i.timeline.Reset()
	}

	return more
}

// Panics if you haven't called NextIssue at least once.
func (i *Iterator) IssueValue() *gitea.Issue {
	return i.issue.Value()
}

// Returns `nil` if there are no more labels on the current issue.
// Panics if you haven't called NextIssue at least once.
// Call `Iterator.Error()` to determine if there was an error or just no more comments.
func (i *Iterator) NextEvent() bool {
	return i.advance(i.timeline, i.IssueValue())
}

// Panics if you haven't called NextIssue at least once.
func (i *Iterator) EventValue() TimelineEvent {
	return *i.timeline.Value()
}

type subIterator interface {
	Next(ctx context.Context, conf config, issue *gitea.Issue) (bool, error)
}

func (i *Iterator) advance(listing subIterator, currentIssue *gitea.Issue) bool {
	if i.err != nil {
		return false
	}

	if i.ctx.Err() != nil {
		return false
	}

	more, err := listing.Next(i.ctx, i.conf, currentIssue)
	if err != nil {
		i.err = err
		return false
	}

	return more
}

func newPageIterator[T any](fetch fetchPage[T]) *pageIterator[T] {
	ii := &pageIterator[T]{fetch: fetch}
	ii.Reset()
	return ii
}

func (iter *pageIterator[T]) Value() *T {
	return iter.cache[iter.index]
}

func (iter *pageIterator[T]) Next(ctx context.Context, conf config, issue *gitea.Issue) (bool, error) {
	// move cursor index. this also handles the case of an empty cache.
	if iter.index < len(iter.cache)-1 {
		iter.index++
		return true, nil
	}

	return iter.getNext(ctx, conf, issue)
}

func (iter *pageIterator[T]) getNext(ctx context.Context, conf config, issue *gitea.Issue) (bool, error) {
	if iter.lastPage {
		return false, nil
	}

	ctx, cancel := context.WithTimeout(ctx, conf.timeout)
	defer cancel()

	// Fetchers can have an internal filter, such as for labels.
	// Allow that, and trust `more`, but run the loop until we get at least one item.
	more := true
	for more {
		var items []*T
		var err error
		items, more, err = iter.fetch(ctx, conf, issue, iter.page)

		if err != nil {
			iter.Reset()
			return false, err
		}

		iter.cache = items
		iter.index = 0
		iter.page++

		iter.lastPage = !more

		if len(iter.cache) != 0 {
			break
		}
	}

	return len(iter.cache) != 0, nil
}

func (iter *pageIterator[T]) Reset() {
	iter.index = -1
	iter.page = 1
	iter.lastPage = false
	iter.cache = nil
}

func fetchIssues(ctx context.Context, conf config, _ *gitea.Issue, page int) ([]*gitea.Issue, bool, error) {
	issues, resp, err := conf.gc.Issues.ListRepoIssues(
		ctx,
		conf.owner,
		conf.project,
		gitea.ListIssueOption{
			ListOptions: gitea.ListOptions{
				Page:     page,
				PageSize: conf.capacity,
			},
			State: gitea.StateAll,
			Type:  gitea.IssueTypeIssue,
			Since: conf.since,
		},
	)
	if err != nil {
		return nil, true, err
	}
	lastPage, err := reachedTotalCount(resp, conf, page, len(issues))
	return issues, !lastPage, err
}

func fetchTimeline(ctx context.Context, conf config, issue *gitea.Issue, page int) ([]*TimelineEvent, bool, error) {
	rawEvents, resp, err := conf.gc.Issues.ListIssueTimeline(
		ctx,
		conf.owner,
		conf.project,
		issue.Index,
		gitea.ListIssueCommentOptions{
			ListOptions: gitea.ListOptions{
				Page:     page,
				PageSize: conf.capacity,
			},
			Since: conf.since,
		},
	)
	if err != nil {
		return nil, true, err
	}
	lastPage, err := reachedTotalCount(resp, conf, page, len(rawEvents))
	if err != nil {
		return nil, true, err
	}

	events := make([]*TimelineEvent, 0, len(rawEvents))
	for _, rawEvent := range rawEvents {
		var event TimelineEvent
		switch rawEvent.Type {
			case "comment":
				event = &CommentEvent{
					Body: rawEvent.Body,
					// FIXME: Forgejo's API doesn't expose whether someone besides the author
					// edited this comment.
					Poster: rawEvent.Poster,
					// We need both of those to be able to distinguish new comments from
					// edits. We also have a policy decision to make: what to do if we see
					// an edit but never the original. We leave that up to the import
					// module.
					Created: rawEvent.Created,
					Updated: rawEvent.Updated,
				}
			case "label":
				var kind LabelEventKind
				if rawEvent.Body == "1" {
					kind = LabelAdded
				} else {
					kind = LabelRemoved
				}
				event = &LabelEvent{Kind: kind, Label: rawEvent.Label, Poster: rawEvent.Poster}
			case "close":
			case "reopen":
			case "rename":
			default:
				continue
		}
		events = append(events, &event)
	}
	return events, !lastPage, err
}

// Returns whether this is the last page.
func reachedTotalCount(resp *gitea.Response, conf config, page, items_len int) (bool, error) {
	header := resp.Header.Get("X-Total-Count")
	if header == "" {
		return false, errors.New("Missing X-Total-Count header")
	}
	total, err := strconv.Atoi(header)
	if err != nil {
		return false, err
	}
	return total <= page*conf.capacity, nil
}
