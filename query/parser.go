package query

import (
	"fmt"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/query/ast"
)

// Parse parse a query DSL
//
// Ex: "status:open author:descartes sort:edit-asc"
//
// Supported filter qualifiers and syntax are described in docs/queries.md
func Parse(query string) (*ast.Query, error) {
	tokens, err := tokenize(query)
	if err != nil {
		return nil, err
	}

	q := &ast.Query{
		OrderBy:        ast.OrderByCreation,
		OrderDirection: ast.OrderDescending,
	}
	sortingDone := false

	for _, t := range tokens {
		switch t.qualifier {
		case "status", "state":
			status, err := bug.StatusFromString(t.value)
			if err != nil {
				return nil, err
			}
			q.Status = append(q.Status, status)
		case "author":
			q.Author = append(q.Author, t.value)
		case "actor":
			q.Actor = append(q.Actor, t.value)
		case "participant":
			q.Participant = append(q.Participant, t.value)
		case "label":
			q.Label = append(q.Label, t.value)
		case "title":
			q.Title = append(q.Title, t.value)
		case "no":
			switch t.value {
			case "label":
				q.NoLabel = true
			default:
				return nil, fmt.Errorf("unknown \"no\" filter \"%s\"", t.value)
			}
		case "sort":
			if sortingDone {
				return nil, fmt.Errorf("multiple sorting")
			}
			err = parseSorting(q, t.value)
			if err != nil {
				return nil, err
			}
			sortingDone = true

		default:
			return nil, fmt.Errorf("unknown qualifier \"%s\"", t.qualifier)
		}
	}
	return q, nil
}

func parseSorting(q *ast.Query, value string) error {
	switch value {
	// default ASC
	case "id-desc":
		q.OrderBy = ast.OrderById
		q.OrderDirection = ast.OrderDescending
	case "id", "id-asc":
		q.OrderBy = ast.OrderById
		q.OrderDirection = ast.OrderAscending

	// default DESC
	case "creation", "creation-desc":
		q.OrderBy = ast.OrderByCreation
		q.OrderDirection = ast.OrderDescending
	case "creation-asc":
		q.OrderBy = ast.OrderByCreation
		q.OrderDirection = ast.OrderAscending

	// default DESC
	case "edit", "edit-desc":
		q.OrderBy = ast.OrderByEdit
		q.OrderDirection = ast.OrderDescending
	case "edit-asc":
		q.OrderBy = ast.OrderByEdit
		q.OrderDirection = ast.OrderAscending

	default:
		return fmt.Errorf("unknown sorting %s", value)
	}

	return nil
}
