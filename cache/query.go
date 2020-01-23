package cache

import (
	"fmt"
	"strings"
	"unicode"
)

type Query struct {
	Filters
	OrderBy
	OrderDirection
}

// Return an identity query with default sorting (creation-desc)
func NewQuery() *Query {
	return &Query{
		OrderBy:        OrderByCreation,
		OrderDirection: OrderDescending,
	}
}

// ParseQuery parse a query DSL
//
// Ex: "status:open author:descartes sort:edit-asc"
//
// Supported filter qualifiers and syntax are described in docs/queries.md
func ParseQuery(query string) (*Query, error) {
	fields := splitQuery(query)

	result := &Query{
		OrderBy:        OrderByCreation,
		OrderDirection: OrderDescending,
	}

	sortingDone := false

	for _, field := range fields {
		split := strings.Split(field, ":")
		if len(split) != 2 {
			return nil, fmt.Errorf("can't parse \"%s\"", field)
		}

		qualifierName := split[0]
		qualifierQuery := removeQuote(split[1])

		switch qualifierName {
		case "status", "state":
			f, err := StatusFilter(qualifierQuery)
			if err != nil {
				return nil, err
			}
			result.Status = append(result.Status, f)

		case "author":
			f := AuthorFilter(qualifierQuery)
			result.Author = append(result.Author, f)

		case "actor":
			f := ActorFilter(qualifierQuery)
			result.Actor = append(result.Actor, f)

		case "participant":
			f := ParticipantFilter(qualifierQuery)
			result.Participant = append(result.Participant, f)

		case "label":
			f := LabelFilter(qualifierQuery)
			result.Label = append(result.Label, f)

		case "title":
			f := TitleFilter(qualifierQuery)
			result.Title = append(result.Title, f)

		case "no":
			err := result.parseNoFilter(qualifierQuery)
			if err != nil {
				return nil, err
			}

		case "sort":
			if sortingDone {
				return nil, fmt.Errorf("multiple sorting")
			}

			err := result.parseSorting(qualifierQuery)
			if err != nil {
				return nil, err
			}

			sortingDone = true

		default:
			return nil, fmt.Errorf("unknown qualifier name %s", qualifierName)
		}
	}

	return result, nil
}

func splitQuery(query string) []string {
	lastQuote := rune(0)
	f := func(c rune) bool {
		switch {
		case c == lastQuote:
			lastQuote = rune(0)
			return false
		case lastQuote != rune(0):
			return false
		case unicode.In(c, unicode.Quotation_Mark):
			lastQuote = c
			return false
		default:
			return unicode.IsSpace(c)
		}
	}

	return strings.FieldsFunc(query, f)
}

func removeQuote(field string) string {
	if len(field) >= 2 {
		if field[0] == '"' && field[len(field)-1] == '"' {
			return field[1 : len(field)-1]
		}
	}
	return field
}

func (q *Query) parseNoFilter(query string) error {
	switch query {
	case "label":
		q.NoFilters = append(q.NoFilters, NoLabelFilter())
	default:
		return fmt.Errorf("unknown \"no\" filter %s", query)
	}

	return nil
}

func (q *Query) parseSorting(query string) error {
	switch query {
	// default ASC
	case "id-desc":
		q.OrderBy = OrderById
		q.OrderDirection = OrderDescending
	case "id", "id-asc":
		q.OrderBy = OrderById
		q.OrderDirection = OrderAscending

	// default DESC
	case "creation", "creation-desc":
		q.OrderBy = OrderByCreation
		q.OrderDirection = OrderDescending
	case "creation-asc":
		q.OrderBy = OrderByCreation
		q.OrderDirection = OrderAscending

	// default DESC
	case "edit", "edit-desc":
		q.OrderBy = OrderByEdit
		q.OrderDirection = OrderDescending
	case "edit-asc":
		q.OrderBy = OrderByEdit
		q.OrderDirection = OrderAscending

	default:
		return fmt.Errorf("unknown sorting %s", query)
	}

	return nil
}
