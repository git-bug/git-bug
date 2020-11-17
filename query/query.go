package query

import "github.com/MichaelMure/git-bug/bug"

// Query is the intermediary representation of a Bug's query. It is either
// produced by parsing a query string (ex: "status:open author:rene") or created
// manually. This query doesn't do anything by itself and need to be interpreted
// for the specific domain of application.
type Query struct {
	Search
	Filters
	OrderBy
	OrderDirection
}

// NewQuery return an identity query with the default sorting (creation-desc).
func NewQuery() *Query {
	return &Query{
		OrderBy:        OrderByCreation,
		OrderDirection: OrderDescending,
	}
}

type Search []string

// Filters is a collection of Filter that implement a complex filter
type Filters struct {
	Status      []bug.Status
	Author      []string
	Actor       []string
	Participant []string
	Label       []string
	Title       []string
	NoLabel     bool
}

type OrderBy int

const (
	_ OrderBy = iota
	OrderById
	OrderByCreation
	OrderByEdit
)

type OrderDirection int

const (
	_ OrderDirection = iota
	OrderAscending
	OrderDescending
)
