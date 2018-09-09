package cache

import (
	"fmt"
	"strings"
)

type Query struct {
	Filters
	OrderBy
	OrderDirection
}

// ParseQuery parse a query DSL
//
// Ex: "status:open author:descartes sort:edit-asc"
//
// Supported filter fields are:
// - status:
// - author:
// - label:
// - no:
//
// Sorting is done with:
// - sort:
//
// Todo: write a complete doc
func ParseQuery(query string) (*Query, error) {
	fields := strings.Fields(query)

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

		switch split[0] {
		case "status":
			f, err := StatusFilter(split[1])
			if err != nil {
				return nil, err
			}
			result.Status = append(result.Status, f)

		case "author":
			f := AuthorFilter(split[1])
			result.Author = append(result.Author, f)

		case "label":
			f := LabelFilter(split[1])
			result.Label = append(result.Label, f)

		case "no":
			err := result.parseNoFilter(split[1])
			if err != nil {
				return nil, err
			}

		case "sort":
			if sortingDone {
				return nil, fmt.Errorf("multiple sorting")
			}

			err := result.parseSorting(split[1])
			if err != nil {
				return nil, err
			}

			sortingDone = true

		default:
			return nil, fmt.Errorf("unknow query field %s", split[0])
		}
	}

	return result, nil
}

func (q *Query) parseNoFilter(query string) error {
	switch query {
	case "label":
		q.NoFilters = append(q.NoFilters, NoLabelFilter())
	default:
		return fmt.Errorf("unknown \"no\" filter")
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
		return fmt.Errorf("unknow sorting %s", query)
	}

	return nil
}
