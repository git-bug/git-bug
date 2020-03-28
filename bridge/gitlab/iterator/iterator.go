package iterator

import (
	"context"
	"time"

	"github.com/xanzy/go-gitlab"
)

type Iterator struct {
	// shared context
	ctx context.Context

	// to pass to sub-iterators
	conf config

	// sticky error
	err error

	// issues iterator
	issue *issueIterator

	// notes iterator
	note *noteIterator

	// labelEvent iterator
	labelEvent *labelEventIterator
}

type config struct {
	// gitlab api v4 client
	gc *gitlab.Client

	timeout time.Duration

	// if since is given the iterator will query only the issues
	// updated after this date
	since time.Time

	// project id
	project string

	// number of issues and notes to query at once
	capacity int
}

// NewIterator create a new iterator
func NewIterator(ctx context.Context, client *gitlab.Client, capacity int, projectID string, since time.Time) *Iterator {
	return &Iterator{
		ctx: ctx,
		conf: config{
			gc:       client,
			timeout:  60 * time.Second,
			since:    since,
			project:  projectID,
			capacity: capacity,
		},
		issue:      newIssueIterator(),
		note:       newNoteIterator(),
		labelEvent: newLabelEventIterator(),
	}
}

// Error return last encountered error
func (i *Iterator) Error() error {
	return i.err
}

func (i *Iterator) NextIssue() bool {
	if i.err != nil {
		return false
	}

	if i.ctx.Err() != nil {
		return false
	}

	more, err := i.issue.Next(i.ctx, i.conf)
	if err != nil {
		i.err = err
		return false
	}

	// Also reset the other sub iterators as they would
	// no longer be valid
	i.note.Reset(i.issue.Value().IID)
	i.labelEvent.Reset(i.issue.Value().IID)

	return more
}

func (i *Iterator) IssueValue() *gitlab.Issue {
	return i.issue.Value()
}

func (i *Iterator) NextNote() bool {
	if i.err != nil {
		return false
	}

	if i.ctx.Err() != nil {
		return false
	}

	more, err := i.note.Next(i.ctx, i.conf)
	if err != nil {
		i.err = err
		return false
	}

	return more
}

func (i *Iterator) NoteValue() *gitlab.Note {
	return i.note.Value()
}

func (i *Iterator) NextLabelEvent() bool {
	if i.err != nil {
		return false
	}

	if i.ctx.Err() != nil {
		return false
	}

	more, err := i.labelEvent.Next(i.ctx, i.conf)
	if err != nil {
		i.err = err
		return false
	}

	return more
}

func (i *Iterator) LabelEventValue() *gitlab.LabelEvent {
	return i.labelEvent.Value()
}
