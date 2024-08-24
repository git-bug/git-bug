package connections

import (
	"fmt"

	"github.com/cheekybits/genny/generic"

	"github.com/git-bug/git-bug/api/graphql/models"
)

// Name define the name of the connection
type Name generic.Type

// NodeType define the node type handled by this relay connection
type NodeType generic.Type

// EdgeType define the edge type handled by this relay connection
type EdgeType generic.Type

// ConnectionType define the connection type handled by this relay connection
type ConnectionType generic.Type

// NameEdgeMaker define a function that take a NodeType and an offset and
// create an Edge.
type NameEdgeMaker func(value NodeType, offset int) Edge

// NameConMaker define a function that create a ConnectionType
type NameConMaker func(
	edges []*EdgeType,
	nodes []NodeType,
	info *models.PageInfo,
	totalCount int) (*ConnectionType, error)

// NameCon will paginate a source according to the input of a relay connection
func NameCon(source []NodeType, edgeMaker NameEdgeMaker, conMaker NameConMaker, input models.ConnectionInput) (*ConnectionType, error) {
	var nodes []NodeType
	var edges []*EdgeType
	var cursors []string
	var pageInfo = &models.PageInfo{}
	var totalCount = len(source)

	emptyCon, _ := conMaker(edges, nodes, pageInfo, 0)

	offset := 0

	if input.After != nil {
		for i, value := range source {
			edge := edgeMaker(value, i)
			if edge.GetCursor() == *input.After {
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
			edge := edgeMaker(value, i+offset)

			if edge.GetCursor() == *input.Before {
				// remove all after element including the "before" one
				pageInfo.HasNextPage = true
				break
			}

			e := edge.(EdgeType)
			edges = append(edges, &e)
			cursors = append(cursors, edge.GetCursor())
			nodes = append(nodes, value)
		}
	} else {
		edges = make([]*EdgeType, len(source))
		cursors = make([]string, len(source))
		nodes = source

		for i, value := range source {
			edge := edgeMaker(value, i+offset)
			e := edge.(EdgeType)
			edges[i] = &e
			cursors[i] = edge.GetCursor()
		}
	}

	if input.First != nil {
		if *input.First < 0 {
			return emptyCon, fmt.Errorf("first less than zero")
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
			return emptyCon, fmt.Errorf("last less than zero")
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

	return conMaker(edges, nodes, pageInfo, totalCount)
}
