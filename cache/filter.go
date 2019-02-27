package cache

import (
	"strings"

	"github.com/MichaelMure/git-bug/bug"
)

// Filter is a predicate that match a subset of bugs
type Filter func(repoCache *RepoCache, excerpt *BugExcerpt) bool

// StatusFilter return a Filter that match a bug status
func StatusFilter(query string) (Filter, error) {
	status, err := bug.StatusFromString(query)
	if err != nil {
		return nil, err
	}

	return func(repoCache *RepoCache, excerpt *BugExcerpt) bool {
		return excerpt.Status == status
	}, nil
}

// AuthorFilter return a Filter that match a bug author
func AuthorFilter(query string) Filter {
	return func(repoCache *RepoCache, excerpt *BugExcerpt) bool {
		query = strings.ToLower(query)

		// Normal identity
		if excerpt.AuthorId != "" {
			author, ok := repoCache.identitiesExcerpts[excerpt.AuthorId]
			if !ok {
				panic("missing identity in the cache")
			}

			return strings.Contains(strings.ToLower(author.Name), query) ||
				strings.Contains(strings.ToLower(author.Login), query)
		}

		// Legacy identity support
		return strings.Contains(strings.ToLower(excerpt.LegacyAuthor.Name), query) ||
			strings.Contains(strings.ToLower(excerpt.LegacyAuthor.Login), query)
	}
}

// LabelFilter return a Filter that match a label
func LabelFilter(label string) Filter {
	return func(repoCache *RepoCache, excerpt *BugExcerpt) bool {
		for _, l := range excerpt.Labels {
			if string(l) == label {
				return true
			}
		}
		return false
	}
}

// TitleFilter return a Filter that match a title
func TitleFilter(title string) Filter {
	return func(excerpt *BugExcerpt) bool {
		if strings.Contains(excerpt.Title, title) {
			return true
		}
		return false
	}

}

// NoLabelFilter return a Filter that match the absence of labels
func NoLabelFilter() Filter {
	return func(repoCache *RepoCache, excerpt *BugExcerpt) bool {
		return len(excerpt.Labels) == 0
	}
}

// Filters is a collection of Filter that implement a complex filter
type Filters struct {
	Status    []Filter
	Author    []Filter
	Label     []Filter
	Title     []Filter
	NoFilters []Filter
}

// Match check if a bug match the set of filters
func (f *Filters) Match(repoCache *RepoCache, excerpt *BugExcerpt) bool {
	if match := f.orMatch(f.Status, repoCache, excerpt); !match {
		return false
	}

	if match := f.orMatch(f.Author, repoCache, excerpt); !match {
		return false
	}

	if match := f.orMatch(f.Label, repoCache, excerpt); !match {
		return false
	}

	if match := f.andMatch(f.NoFilters, repoCache, excerpt); !match {
		return false
	}

	if match := f.andMatch(f.Title, excerpt); !match {
		return false
	}

	return true
}

// Check if any of the filters provided match the bug
func (*Filters) orMatch(filters []Filter, repoCache *RepoCache, excerpt *BugExcerpt) bool {
	if len(filters) == 0 {
		return true
	}

	match := false
	for _, f := range filters {
		match = match || f(repoCache, excerpt)
	}

	return match
}

// Check if all of the filters provided match the bug
func (*Filters) andMatch(filters []Filter, repoCache *RepoCache, excerpt *BugExcerpt) bool {
	if len(filters) == 0 {
		return true
	}

	match := true
	for _, f := range filters {
		match = match && f(repoCache, excerpt)
	}

	return match
}
