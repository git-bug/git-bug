package ast

import "github.com/MichaelMure/git-bug/bug"

type Query struct {
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
