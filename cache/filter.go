package cache

import (
	"strings"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/query"
)

// resolver has the resolving functions needed by filters.
// This exist mainly to go through the functions of the cache with proper locking.
type resolver interface {
	ResolveIdentityExcerpt(id entity.Id) (*IdentityExcerpt, error)
}

// Filter is a predicate that match a subset of bugs
type Filter func(excerpt *BugExcerpt, resolver resolver) bool

// StatusFilter return a Filter that match a bug status
func StatusFilter(status bug.Status) Filter {
	return func(excerpt *BugExcerpt, resolver resolver) bool {
		return excerpt.Status == status
	}
}

// AuthorFilter return a Filter that match a bug author
func AuthorFilter(query string) Filter {
	return func(excerpt *BugExcerpt, resolver resolver) bool {
		query = strings.ToLower(query)

		author, err := resolver.ResolveIdentityExcerpt(excerpt.AuthorId)
		if err != nil {
			panic(err)
		}

		return author.Match(query)
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

// Matcher is a collection of Filter that implement a complex filter
type Matcher struct {
	Status      []Filter
	Author      []Filter
	Actor       []Filter
	Participant []Filter
	Label       []Filter
	Title       []Filter
	NoFilters   []Filter
}

// compileMatcher transform a query.Filters into a specialized matcher
// for the cache.
func compileMatcher(filters query.Filters) *Matcher {
	result := &Matcher{}

	for _, value := range filters.Status {
		result.Status = append(result.Status, StatusFilter(value))
	}
	for _, value := range filters.Author {
		result.Author = append(result.Author, AuthorFilter(value))
	}
	for _, value := range filters.Actor {
		result.Actor = append(result.Actor, ActorFilter(value))
	}
	for _, value := range filters.Participant {
		result.Participant = append(result.Participant, ParticipantFilter(value))
	}
	for _, value := range filters.Label {
		result.Label = append(result.Label, LabelFilter(value))
	}
	for _, value := range filters.Title {
		result.Title = append(result.Title, TitleFilter(value))
	}

	return result
}

// Match check if a bug match the set of filters
func (f *Matcher) Match(excerpt *BugExcerpt, resolver resolver) bool {
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
func (*Matcher) orMatch(filters []Filter, excerpt *BugExcerpt, resolver resolver) bool {
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
func (*Matcher) andMatch(filters []Filter, excerpt *BugExcerpt, resolver resolver) bool {
	if len(filters) == 0 {
		return true
	}

	match := true
	for _, f := range filters {
		match = match && f(excerpt, resolver)
	}

	return match
}
