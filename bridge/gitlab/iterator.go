package gitlab

import (
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

type iterator struct {
	// gitlab api v4 client
	gc *gitlab.Client

	// if since is given the iterator will query only the updated
	// issues after this date
	since time.Time

	// project id
	project string

	// number of issues and notes to query at once
	capacity int

	// sticky error
	err error

	// issues iterator
	issue *issueIterator

	// notes iterator
	note *noteIterator

	labelEvent *labelEventIterator
}

// NewIterator create a new iterator
func NewIterator(projectID, token string, since time.Time) *iterator {
	return &iterator{
		gc:       buildClient(token),
		project:  projectID,
		since:    since,
		capacity: 20,
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
	sort := "asc"
	scope := "all"
	issues, _, err := i.gc.Issues.ListProjectIssues(
		i.project,
		&gitlab.ListProjectIssuesOptions{
			ListOptions: gitlab.ListOptions{
				Page:    i.issue.page,
				PerPage: i.capacity,
			},
			Scope:        &scope,
			UpdatedAfter: &i.since,
			Sort:         &sort,
		},
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
	// first query
	if i.issue.cache == nil {
		return i.getNextIssues()
	}

	if i.err != nil {
		return false
	}

	// move cursor index
	if i.issue.index < min(i.capacity, len(i.issue.cache))-1 {
		i.issue.index++
		return true
	}

	return i.getNextIssues()
}

func (i *iterator) IssueValue() *gitlab.Issue {
	return i.issue.cache[i.issue.index]
}

func (i *iterator) getNextNotes() bool {
	sort := "asc"
	order := "created_at"
	notes, _, err := i.gc.Notes.ListIssueNotes(
		i.project,
		i.IssueValue().IID,
		&gitlab.ListIssueNotesOptions{
			ListOptions: gitlab.ListOptions{
				Page:    i.note.page,
				PerPage: i.capacity,
			},
			Sort:    &sort,
			OrderBy: &order,
		},
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

	if len(i.note.cache) == 0 {
		return i.getNextNotes()
	}

	// move cursor index
	if i.note.index < min(i.capacity, len(i.note.cache))-1 {
		i.note.index++
		return true
	}

	return i.getNextNotes()
}

func (i *iterator) NoteValue() *gitlab.Note {
	return i.note.cache[i.note.index]
}

func (i *iterator) getNextLabelEvents() bool {
	labelEvents, _, err := i.gc.ResourceLabelEvents.ListIssueLabelEvents(
		i.project,
		i.IssueValue().IID,
		&gitlab.ListLabelEventsOptions{
			ListOptions: gitlab.ListOptions{
				Page:    i.labelEvent.page,
				PerPage: i.capacity,
			},
		},
	)

	if err != nil {
		i.err = err
		return false
	}

	if len(labelEvents) == 0 {
		i.labelEvent.page = 1
		i.labelEvent.index = -1
		i.labelEvent.cache = nil
		return false
	}

	i.labelEvent.cache = labelEvents
	i.labelEvent.page++
	i.labelEvent.index = 0
	return true
}

func (i *iterator) NextLabelEvent() bool {
	if i.err != nil {
		return false
	}

	if len(i.labelEvent.cache) == 0 {
		return i.getNextLabelEvents()
	}

	// move cursor index
	if i.labelEvent.index < min(i.capacity, len(i.labelEvent.cache))-1 {
		i.labelEvent.index++
		return true
	}

	return i.getNextLabelEvents()
}

func (i *iterator) LabelEventValue() *gitlab.LabelEvent {
	return i.labelEvent.cache[i.labelEvent.index]
}

func min(a, b int) int {
	if a > b {
		return b
	}

	return a
}
