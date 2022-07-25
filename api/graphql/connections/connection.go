package connections

import (
	"fmt"

	"github.com/MichaelMure/git-bug/api/graphql/models"
)

type Input struct {
	After  *string
	Before *string
	First  *int
	Last   *int
}

// Result is the result of a GraphQL connection pagination
type Result[NodeT any] struct {
	// A list of edges.
	Edges []*Edge[NodeT] `json:"edges"`
	Nodes []NodeT        `json:"nodes"`
	// Information to aid in pagination.
	PageInfo *models.PageInfo `json:"pageInfo"`
	// Identifies the total count of items in the connection.
	TotalCount int `json:"totalCount"`
}

// Edge hold a GraphQL connection edge
type Edge[NodeT any] struct {
	Cursor string `json:"cursor"`
	Node   NodeT  `json:"node"`
}

func newEdge[NodeT any](node NodeT, offset int) *Edge[NodeT] {
	return &Edge[NodeT]{Cursor: OffsetToCursor(offset), Node: node}
}

// Paginate will paginate a source according to the input of a relay connection
func Paginate[NodeT any](source []NodeT, input Input) (*Result[NodeT], error) {
	var nodes []NodeT
	var edges []*Edge[NodeT]
	var cursors []string
	var pageInfo = &models.PageInfo{}
	var totalCount = len(source)

	offset := 0

	if input.After != nil {
		for i, value := range source {
			edge := newEdge(value, offset)
			if edge.Cursor == *input.After {
				// remove all previous element including the "after" one
				source = source[i+1:]
				offset = i + 1
				pageInfo.HasPreviousPage = true
				break
			}
		}
	}

	if input.Before != nil {
		for i, value := range source {
			edge := newEdge(value, i+offset)
			if edge.Cursor == *input.Before {
				// remove all after element including the "before" one
				pageInfo.HasNextPage = true
				break
			}
			edges = append(edges, edge)
			cursors = append(cursors, edge.Cursor)
			nodes = append(nodes, value)
		}
	} else {
		edges = make([]*Edge[NodeT], len(source))
		cursors = make([]string, len(source))
		nodes = source

		for i, value := range source {
			edge := newEdge(value, i+offset)
			edges[i] = edge
			cursors[i] = edge.Cursor
		}
	}

	if input.First != nil {
		if *input.First < 0 {
			return nil, fmt.Errorf("first less than zero")
		}

		if len(edges) > *input.First {
			// Slice result to be of length first by removing edges from the end
			edges = edges[:*input.First]
			cursors = cursors[:*input.First]
			nodes = nodes[:*input.First]
			pageInfo.HasNextPage = true
		}
	}

	if input.Last != nil {
		if *input.Last < 0 {
			return nil, fmt.Errorf("last less than zero")
		}

		if len(edges) > *input.Last {
			// Slice result to be of length last by removing edges from the start
			edges = edges[len(edges)-*input.Last:]
			cursors = cursors[len(cursors)-*input.Last:]
			nodes = nodes[len(nodes)-*input.Last:]
			pageInfo.HasPreviousPage = true
		}
	}

	// Fill up pageInfo cursors
	if len(cursors) > 0 {
		pageInfo.StartCursor = cursors[0]
		pageInfo.EndCursor = cursors[len(cursors)-1]
	}

	return &Result[NodeT]{
		Edges:      edges,
		Nodes:      nodes,
		PageInfo:   pageInfo,
		TotalCount: totalCount,
	}, nil
}
