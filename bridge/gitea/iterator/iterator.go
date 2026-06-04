package iterator

import (
	"context"
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

	// issues iterator
	issue *pageIterator[gitea.Issue]

	// comments iterator
	comment *pageIterator[gitea.Comment]

	// labels iterator
	label *pageIterator[gitea.Label]
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
		comment: newPageIterator[gitea.Comment](fetchComments),
		issue: newPageIterator[gitea.Issue](fetchIssues),
		label: newPageIterator[gitea.Label](fetchLabels),
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
		i.comment.Reset()
		i.label.Reset()
	}

	return more
}

// Panics if you haven't called NextIssue at least once.
func (i *Iterator) IssueValue() *gitea.Issue {
	return i.issue.Value()
}

// Returns `nil` if there are no more comments on the current issue.
// Panics if you haven't called NextIssue at least once.
// Call `Iterator.Error()` to determine if there was an error or just no more comments.
func (i *Iterator) NextComment() bool {
	return i.advance(i.comment, i.IssueValue())
}

// Panics if you haven't called NextIssue at least once.
func (i *Iterator) CommentValue() *gitea.Comment {
	return i.comment.Value()
}

// Returns `nil` if there are no more labels on the current issue.
// Panics if you haven't called NextIssue at least once.
// Call `Iterator.Error()` to determine if there was an error or just no more comments.
func (i *Iterator) NextLabel() bool {
	return i.advance(i.label, i.IssueValue())
}

// Panics if you haven't called NextIssue at least once.
func (i *Iterator) LabelValue() *gitea.Label {
	return i.label.Value()
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

	items, more, err := iter.fetch(ctx, conf, issue, iter.page)

	if err != nil {
		iter.Reset()
		return false, err
	}

	iter.lastPage = !more

	if len(items) == 0 {
		return false, nil
	}

	iter.cache = items
	iter.index = 0
	iter.page++

	return true, nil
}

func (iter *pageIterator[T]) Reset() {
	iter.index = -1
	iter.page = 1
	iter.lastPage = false
	iter.cache = nil
}

func fetchIssues(ctx context.Context, conf config, _ *gitea.Issue, page int) (issues []*gitea.Issue, more bool, err error) {
	more = true
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
		return
	}

	total, err := strconv.Atoi(resp.Header.Get("X-Total-Count"))
	if err != nil && (len(issues) == 0 || total <= page*conf.capacity) {
		more = false
	}

	return
}

func fetchComments(ctx context.Context, conf config, issue *gitea.Issue, page int) ([]*gitea.Comment, bool, error) {
	comments, _, err := conf.gc.Issues.ListIssueComments(
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
	more := len(comments) == conf.capacity
	return comments, more, err
}

func fetchLabels(ctx context.Context, conf config, issue *gitea.Issue, page int) ([]*gitea.Label, bool, error) {
	labels, _, err := conf.gc.Issues.GetIssueLabels(
		ctx,
		conf.owner,
		conf.project,
		issue.Index,
		// Labels are always returned as a single giant body, with no pagination.
		gitea.ListLabelsOptions{},
	)
	return labels, false, err
}
