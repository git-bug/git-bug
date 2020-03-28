package iterator

import (
	"context"

	"github.com/xanzy/go-gitlab"
)

type noteIterator struct {
	issue int
	page  int
	index int
	cache []*gitlab.Note
}

func newNoteIterator() *noteIterator {
	in := &noteIterator{}
	in.Reset(-1)
	return in
}

func (in *noteIterator) Next(ctx context.Context, conf config) (bool, error) {
	// first query
	if in.cache == nil {
		return in.getNext(ctx, conf)
	}

	// move cursor index
	if in.index < len(in.cache)-1 {
		in.index++
		return true, nil
	}

	return in.getNext(ctx, conf)
}

func (in *noteIterator) Value() *gitlab.Note {
	return in.cache[in.index]
}

func (in *noteIterator) getNext(ctx context.Context, conf config) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, conf.timeout)
	defer cancel()

	notes, _, err := conf.gc.Notes.ListIssueNotes(
		conf.project,
		in.issue,
		&gitlab.ListIssueNotesOptions{
			ListOptions: gitlab.ListOptions{
				Page:    in.page,
				PerPage: conf.capacity,
			},
			Sort:    gitlab.String("asc"),
			OrderBy: gitlab.String("created_at"),
		},
		gitlab.WithContext(ctx),
	)

	if err != nil {
		in.Reset(-1)
		return false, err
	}

	if len(notes) == 0 {
		return false, nil
	}

	in.cache = notes
	in.index = 0
	in.page++

	return true, nil
}

func (in *noteIterator) Reset(issue int) {
	in.issue = issue
	in.index = -1
	in.page = -1
	in.cache = nil
}
