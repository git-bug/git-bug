package cache

import (
	"strings"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/entity"
)

// resolver has the resolving functions needed by filters.
// This exist mainly to go through the functions of the cache with proper locking.
type resolver interface {
	ResolveIdentityExcerpt(id entity.Id) (*IdentityExcerpt, error)
}

// Filter is a predicate that match a subset of bugs
type Filter func(excerpt *BugExcerpt, resolver resolver) bool

// StatusFilter return a Filter that match a bug status
func StatusFilter(query string) (Filter, error) {
	status, err := bug.StatusFromString(query)
	if err != nil {
		return nil, err
	}

	return func(excerpt *BugExcerpt, resolver resolver) bool {
		return excerpt.Status == status
	}, nil
}

// AuthorFilter return a Filter that match a bug author
func AuthorFilter(query string) Filter {
	return func(excerpt *BugExcerpt, resolver resolver) bool {
		query = strings.ToLower(query)

		// Normal identity
		if excerpt.AuthorId != "" {
			author, err := resolver.ResolveIdentityExcerpt(excerpt.AuthorId)
			if err != nil {
				panic(err)
			}

			return author.Match(query)
		}

		// Legacy identity support
		return strings.Contains(strings.ToLower(excerpt.LegacyAuthor.Name), query) ||
			strings.Contains(strings.ToLower(excerpt.LegacyAuthor.Login), query)
	}
}

// LabelFilter return a Filter that match a label
func LabelFilter(label string) Filter {
	return func(excerpt *BugExcerpt, resolver resolver) bool {
		for _, l := range excerpt.Labels {
			if string(l) == label {
				return true
			}
		}
		return false
	}
}

// ActorFilter return a Filter that match a bug actor
func ActorFilter(query string) Filter {
	return func(excerpt *BugExcerpt, resolver resolver) bool {
		query = strings.ToLower(query)

		for _, id := range excerpt.Actors {
			identityExcerpt, err := resolver.ResolveIdentityExcerpt(id)
			if err != nil {
				panic(err)
			}

			if identityExcerpt.Match(query) {
				return true
			}
		}
		return false
	}
}

// ParticipantFilter return a Filter that match a bug participant
func ParticipantFilter(query string) Filter {
	return func(excerpt *BugExcerpt, resolver resolver) bool {
		query = strings.ToLower(query)

		for _, id := range excerpt.Participants {
			identityExcerpt, err := resolver.ResolveIdentityExcerpt(id)
			if err != nil {
				panic(err)
			}

			if identityExcerpt.Match(query) {
				return true
			}
		}
		return false
	}
}

// TitleFilter return a Filter that match if the title contains the given query
func TitleFilter(query string) Filter {
	return func(excerpt *BugExcerpt, resolver resolver) bool {
		return strings.Contains(
			strings.ToLower(excerpt.Title),
			strings.ToLower(query),
		)
	}
}

// NoLabelFilter return a Filter that match the absence of labels
func NoLabelFilter() Filter {
	return func(excerpt *BugExcerpt, resolver resolver) bool {
		return len(excerpt.Labels) == 0
	}
}

// Filters is a collection of Filter that implement a complex filter
type Filters struct {
	Status      []Filter
	Author      []Filter
	Actor       []Filter
	Participant []Filter
	Label       []Filter
	Title       []Filter
	NoFilters   []Filter
}

// Match check if a bug match the set of filters
func (f *Filters) Match(excerpt *BugExcerpt, resolver resolver) bool {
	if match := f.orMatch(f.Status, excerpt, resolver); !match {
		return false
	}

	if match := f.orMatch(f.Author, excerpt, resolver); !match {
		return false
	}

	if match := f.orMatch(f.Participant, excerpt, resolver); !match {
		return false
	}

	if match := f.orMatch(f.Actor, excerpt, resolver); !match {
		return false
	}

	if match := f.andMatch(f.Label, excerpt, resolver); !match {
		return false
	}

	if match := f.andMatch(f.NoFilters, excerpt, resolver); !match {
		return false
	}

	if match := f.andMatch(f.Title, excerpt, resolver); !match {
		return false
	}

	return true
}

// Check if any of the filters provided match the bug
func (*Filters) orMatch(filters []Filter, excerpt *BugExcerpt, resolver resolver) bool {
	if len(filters) == 0 {
		return true
	}

	match := false
	for _, f := range filters {
		match = match || f(excerpt, resolver)
	}

	return match
}

// Check if all of the filters provided match the bug
func (*Filters) andMatch(filters []Filter, excerpt *BugExcerpt, resolver resolver) bool {
	if len(filters) == 0 {
		return true
	}

	match := true
	for _, f := range filters {
		match = match && f(excerpt, resolver)
	}

	return match
}
