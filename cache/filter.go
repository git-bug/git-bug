package cache

import (
	"strings"

	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/query"
)

// Filter is a predicate that matches a subset of bugs
type Filter func(excerpt *BugExcerpt, resolvers entity.Resolvers) bool

// StatusFilter return a Filter that matches a bug status
func StatusFilter(status common.Status) Filter {
	return func(excerpt *BugExcerpt, resolvers entity.Resolvers) bool {
		return excerpt.Status == status
	}
}

// AuthorFilter return a Filter that matches a bug author
func AuthorFilter(query string) Filter {
	return func(excerpt *BugExcerpt, resolvers entity.Resolvers) bool {
		query = strings.ToLower(query)

		author, err := entity.Resolve[*IdentityExcerpt](resolvers, excerpt.AuthorId)
		if err != nil {
			panic(err)
		}

		return author.Match(query)
	}
}

// MetadataFilter return a Filter that matches a bug metadata at creation time
func MetadataFilter(pair query.StringPair) Filter {
	return func(excerpt *BugExcerpt, resolvers entity.Resolvers) bool {
		if value, ok := excerpt.CreateMetadata[pair.Key]; ok {
			return value == pair.Value
		}
		return false
	}
}

// LabelFilter return a Filter that matches a label
func LabelFilter(label string) Filter {
	return func(excerpt *BugExcerpt, resolvers entity.Resolvers) bool {
		for _, l := range excerpt.Labels {
			if string(l) == label {
				return true
			}
		}
		return false
	}
}

// ActorFilter return a Filter that matches a bug actor
func ActorFilter(query string) Filter {
	return func(excerpt *BugExcerpt, resolvers entity.Resolvers) bool {
		query = strings.ToLower(query)

		for _, id := range excerpt.Actors {
			identityExcerpt, err := entity.Resolve[*IdentityExcerpt](resolvers, id)
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

// ParticipantFilter return a Filter that matches a bug participant
func ParticipantFilter(query string) Filter {
	return func(excerpt *BugExcerpt, resolvers entity.Resolvers) bool {
		query = strings.ToLower(query)

		for _, id := range excerpt.Participants {
			identityExcerpt, err := entity.Resolve[*IdentityExcerpt](resolvers, id)
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

// TitleFilter return a Filter that matches if the title contains the given query
func TitleFilter(query string) Filter {
	return func(excerpt *BugExcerpt, resolvers entity.Resolvers) bool {
		return strings.Contains(
			strings.ToLower(excerpt.Title),
			strings.ToLower(query),
		)
	}
}

// NoLabelFilter return a Filter that matches the absence of labels
func NoLabelFilter() Filter {
	return func(excerpt *BugExcerpt, resolvers entity.Resolvers) bool {
		return len(excerpt.Labels) == 0
	}
}

// Matcher is a collection of Filter that implement a complex filter
type Matcher struct {
	Status      []Filter
	Author      []Filter
	Metadata    []Filter
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
	for _, value := range filters.Metadata {
		result.Metadata = append(result.Metadata, MetadataFilter(value))
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
	if filters.NoLabel {
		result.NoFilters = append(result.NoFilters, NoLabelFilter())
	}

	return result
}

// Match check if a bug matches the set of filters
func (f *Matcher) Match(excerpt *BugExcerpt, resolvers entity.Resolvers) bool {
	if match := f.orMatch(f.Status, excerpt, resolvers); !match {
		return false
	}

	if match := f.orMatch(f.Author, excerpt, resolvers); !match {
		return false
	}

	if match := f.orMatch(f.Metadata, excerpt, resolvers); !match {
		return false
	}

	if match := f.orMatch(f.Participant, excerpt, resolvers); !match {
		return false
	}

	if match := f.orMatch(f.Actor, excerpt, resolvers); !match {
		return false
	}

	if match := f.andMatch(f.Label, excerpt, resolvers); !match {
		return false
	}

	if match := f.andMatch(f.NoFilters, excerpt, resolvers); !match {
		return false
	}

	if match := f.andMatch(f.Title, excerpt, resolvers); !match {
		return false
	}

	return true
}

// Check if any of the filters provided match the bug
func (*Matcher) orMatch(filters []Filter, excerpt *BugExcerpt, resolvers entity.Resolvers) bool {
	if len(filters) == 0 {
		return true
	}

	match := false
	for _, f := range filters {
		match = match || f(excerpt, resolvers)
	}

	return match
}

// Check if all the filters provided match the bug
func (*Matcher) andMatch(filters []Filter, excerpt *BugExcerpt, resolvers entity.Resolvers) bool {
	if len(filters) == 0 {
		return true
	}

	match := true
	for _, f := range filters {
		match = match && f(excerpt, resolvers)
	}

	return match
}
