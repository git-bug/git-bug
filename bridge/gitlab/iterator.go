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

type commentIterator struct {
	page  int
	index int
	cache []*gitlab.Note
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

	// comments iterator
	comment *commentIterator
}

// NewIterator create a new iterator
func NewIterator(projectID, token string, capacity int, since time.Time) *iterator {
	return &iterator{
		gc:       buildClient(token),
		project:  projectID,
		since:    since,
		capacity: capacity,
		issue: &issueIterator{
			index: -1,
			page:  1,
		},
		comment: &commentIterator{
			index: -1,
			page:  1,
		},
	}
}

// Error return last encountered error
func (i *iterator) Error() error {
	return i.err
}

func (i *iterator) getIssues() ([]*gitlab.Issue, error) {
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
		},
	)

	return issues, err
}

func (i *iterator) NextIssue() bool {
	// first query
	if i.issue.cache == nil {
		issues, err := i.getIssues()
		if err != nil {
			i.err = err
			return false
		}

		// if repository doesn't have any issues
		if len(issues) == 0 {
			return false
		}

		i.issue.cache = issues
		i.issue.index++
		return true
	}

	if i.err != nil {
		return false
	}

	// move cursor index
	if i.issue.index < min(i.capacity, len(i.issue.cache)) {
		i.issue.index++
		return true
	}

	// query next issues
	issues, err := i.getIssues()
	if err != nil {
		i.err = err
		return false
	}

	// no more issues to query
	if len(issues) == 0 {
		return false
	}

	i.issue.page++
	i.issue.index = 0
	i.comment.index = 0

	return true
}

func (i *iterator) IssueValue() *gitlab.Issue {
	return i.issue.cache[i.issue.index]
}

func (i *iterator) getComments() ([]*gitlab.Note, error) {
	notes, _, err := i.gc.Notes.ListIssueNotes(
		i.project,
		i.IssueValue().IID,
		&gitlab.ListIssueNotesOptions{
			ListOptions: gitlab.ListOptions{
				Page:    i.issue.page,
				PerPage: i.capacity,
			},
		},
	)

	return notes, err
}

func (i *iterator) NextComment() bool {
	if i.err != nil {
		return false
	}

	if len(i.comment.cache) == 0 {
		// query next issues
		comments, err := i.getComments()
		if err != nil {
			i.err = err
			return false
		}

		if len(comments) == 0 {
			i.comment.index = 0
			i.comment.page = 1
			return false
		}

		i.comment.page++
		i.comment.index = 0

		return true
	}

	// move cursor index
	if i.comment.index < min(i.capacity, len(i.comment.cache)) {
		i.comment.index++
		return true
	}

	// query next issues
	comments, err := i.getComments()
	if err != nil {
		i.err = err
		return false
	}

	if len(comments) == 0 {
		i.comment.index = 0
		i.comment.page = 1
		return false
	}

	i.comment.page++
	i.comment.index = 0

	return false
}

func (i *iterator) CommentValue() *gitlab.Note {
	return i.comment.cache[i.comment.index]
}

func min(a, b int) int {
	if a > b {
		return b
	}

	return a
}
