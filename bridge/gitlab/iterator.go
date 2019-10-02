package gitlab

import (
	"context"
	"sort"
	"time"

	"github.com/xanzy/go-gitlab"
)

type issueIterator struct {
	page  int
	index int
	cache []*gitlab.Issue
}

type noteIterator struct {
	page  int
	index int
	cache []*gitlab.Note
}

type labelEventIterator struct {
	page  int
	index int
	cache []*gitlab.LabelEvent
}

func (l *labelEventIterator) Len() int {
	return len(l.cache)
}

func (l *labelEventIterator) Swap(i, j int) {
	element := l.cache[i]
	l.cache[i] = l.cache[j]
	l.cache[j] = element
}

func (l *labelEventIterator) Less(i, j int) bool {
	return l.cache[i].ID < l.cache[j].ID
}

type iterator struct {
	// gitlab api v4 client
	gc *gitlab.Client

	// if since is given the iterator will query only the issues
	// updated after this date
	since time.Time

	// project id
	project string

	// number of issues and notes to query at once
	capacity int

	// shared context
	ctx context.Context

	// sticky error
	err error

	// issues iterator
	issue *issueIterator

	// notes iterator
	note *noteIterator

	// labelEvent iterator
	labelEvent *labelEventIterator
}

// NewIterator create a new iterator
func NewIterator(ctx context.Context, capacity int, projectID, token string, since time.Time) *iterator {
	return &iterator{
		gc:       buildClient(token),
		project:  projectID,
		since:    since,
		capacity: capacity,
		ctx:      ctx,
		issue: &issueIterator{
			index: -1,
			page:  1,
		},
		note: &noteIterator{
			index: -1,
			page:  1,
		},
		labelEvent: &labelEventIterator{
			index: -1,
			page:  1,
		},
	}
}

// Error return last encountered error
func (i *iterator) Error() error {
	return i.err
}

func (i *iterator) getNextIssues() bool {
	ctx, cancel := context.WithTimeout(i.ctx, defaultTimeout)
	defer cancel()

	issues, _, err := i.gc.Issues.ListProjectIssues(
		i.project,
		&gitlab.ListProjectIssuesOptions{
			ListOptions: gitlab.ListOptions{
				Page:    i.issue.page,
				PerPage: i.capacity,
			},
			Scope:        gitlab.String("all"),
			UpdatedAfter: &i.since,
			Sort:         gitlab.String("asc"),
		},
		gitlab.WithContext(ctx),
	)

	if err != nil {
		i.err = err
		return false
	}

	// if repository doesn't have any issues
	if len(issues) == 0 {
		return false
	}

	i.issue.cache = issues
	i.issue.index = 0
	i.issue.page++
	i.note.index = -1
	i.note.cache = nil

	return true
}

func (i *iterator) NextIssue() bool {
	if i.err != nil {
		return false
	}

	if i.ctx.Err() != nil {
		return false
	}

	// first query
	if i.issue.cache == nil {
		return i.getNextIssues()
	}

	// move cursor index
	if i.issue.index < len(i.issue.cache)-1 {
		i.issue.index++
		return true
	}

	return i.getNextIssues()
}

func (i *iterator) IssueValue() *gitlab.Issue {
	return i.issue.cache[i.issue.index]
}

func (i *iterator) getNextNotes() bool {
	ctx, cancel := context.WithTimeout(i.ctx, defaultTimeout)
	defer cancel()

	notes, _, err := i.gc.Notes.ListIssueNotes(
		i.project,
		i.IssueValue().IID,
		&gitlab.ListIssueNotesOptions{
			ListOptions: gitlab.ListOptions{
				Page:    i.note.page,
				PerPage: i.capacity,
			},
			Sort:    gitlab.String("asc"),
			OrderBy: gitlab.String("created_at"),
		},
		gitlab.WithContext(ctx),
	)

	if err != nil {
		i.err = err
		return false
	}

	if len(notes) == 0 {
		i.note.index = -1
		i.note.page = 1
		i.note.cache = nil
		return false
	}

	i.note.cache = notes
	i.note.page++
	i.note.index = 0
	return true
}

func (i *iterator) NextNote() bool {
	if i.err != nil {
		return false
	}

	if i.ctx.Err() != nil {
		return false
	}

	if len(i.note.cache) == 0 {
		return i.getNextNotes()
	}

	// move cursor index
	if i.note.index < len(i.note.cache)-1 {
		i.note.index++
		return true
	}

	return i.getNextNotes()
}

func (i *iterator) NoteValue() *gitlab.Note {
	return i.note.cache[i.note.index]
}

func (i *iterator) getLabelEvents() bool {
	ctx, cancel := context.WithTimeout(i.ctx, defaultTimeout)
	defer cancel()

	hasNextPage := true
	for hasNextPage {
		labelEvents, _, err := i.gc.ResourceLabelEvents.ListIssueLabelEvents(
			i.project,
			i.IssueValue().IID,
			&gitlab.ListLabelEventsOptions{
				ListOptions: gitlab.ListOptions{
					Page:    i.labelEvent.page,
					PerPage: i.capacity,
				},
			},
			gitlab.WithContext(ctx),
		)
		if err != nil {
			i.err = err
			return false
		}

		i.labelEvent.page++
		hasNextPage = len(labelEvents) != 0
		i.labelEvent.cache = append(i.labelEvent.cache, labelEvents...)
	}

	i.labelEvent.page = 1
	i.labelEvent.index = 0
	sort.Sort(i.labelEvent)

	// if the label events list is empty return false
	return len(i.labelEvent.cache) != 0
}

// because Gitlab
func (i *iterator) NextLabelEvent() bool {
	if i.err != nil {
		return false
	}

	if i.ctx.Err() != nil {
		return false
	}

	if len(i.labelEvent.cache) == 0 {
		return i.getLabelEvents()
	}

	// move cursor index
	if i.labelEvent.index < len(i.labelEvent.cache)-1 {
		i.labelEvent.index++
		return true
	}

	return false
}

func (i *iterator) LabelEventValue() *gitlab.LabelEvent {
	return i.labelEvent.cache[i.labelEvent.index]
}
