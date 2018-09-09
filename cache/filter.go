package cache

import (
	"strings"

	"github.com/MichaelMure/git-bug/bug"
)

type Filter func(excerpt *BugExcerpt) bool

func StatusFilter(query string) (Filter, error) {
	status, err := bug.StatusFromString(query)
	if err != nil {
		return nil, err
	}

	return func(excerpt *BugExcerpt) bool {
		return excerpt.Status == status
	}, nil
}

func AuthorFilter(query string) Filter {
	cleaned := strings.TrimFunc(query, func(r rune) bool {
		return r == '"' || r == '\''
	})

	return func(excerpt *BugExcerpt) bool {
		return excerpt.Author.Match(cleaned)
	}
}

func LabelFilter(label string) Filter {
	return func(excerpt *BugExcerpt) bool {
		for _, l := range excerpt.Labels {
			if string(l) == label {
				return true
			}
		}
		return false
	}
}

func NoLabelFilter() Filter {
	return func(excerpt *BugExcerpt) bool {
		return len(excerpt.Labels) == 0
	}
}

type Filters struct {
	Status    []Filter
	Author    []Filter
	Label     []Filter
	NoFilters []Filter
}

// Match check if a bug match the set of filters
func (f *Filters) Match(excerpt *BugExcerpt) bool {
	if match := f.orMatch(f.Status, excerpt); !match {
		return false
	}

	if match := f.orMatch(f.Author, excerpt); !match {
		return false
	}

	if match := f.orMatch(f.Label, excerpt); !match {
		return false
	}

	if match := f.andMatch(f.NoFilters, excerpt); !match {
		return false
	}

	return true
}

// Check if any of the filters provided match the bug
func (*Filters) orMatch(filters []Filter, excerpt *BugExcerpt) bool {
	if len(filters) == 0 {
		return true
	}

	match := false
	for _, f := range filters {
		match = match || f(excerpt)
	}

	return match
}

// Check if all of the filters provided match the bug
func (*Filters) andMatch(filters []Filter, excerpt *BugExcerpt) bool {
	if len(filters) == 0 {
		return true
	}

	match := true
	for _, f := range filters {
		match = match && f(excerpt)
	}

	return match
}
