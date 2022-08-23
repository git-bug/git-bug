package query

import (
	"github.com/MichaelMure/git-bug/entities/common"
)

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

// StringPair is a key/value pair of strings
type StringPair struct {
	Key   string
	Value string
}

// Filters is a collection of Filter that implement a complex filter
type Filters struct {
	Status      []common.Status
	Author      []string
	Metadata    []StringPair
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
