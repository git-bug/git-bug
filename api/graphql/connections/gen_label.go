// This file was automatically generated by genny.
// Any changes will be lost if this file is regenerated.
// see https://github.com/cheekybits/genny

package connections

import (
	"fmt"

	"github.com/git-bug/git-bug/api/graphql/models"
	"github.com/git-bug/git-bug/entities/common"
)

// BugLabelEdgeMaker define a function that take a bug.Label and an offset and
// create an Edge.
type LabelEdgeMaker func(value common.Label, offset int) Edge

// LabelConMaker define a function that create a models.LabelConnection
type LabelConMaker func(
	edges []*models.LabelEdge,
	nodes []common.Label,
	info *models.PageInfo,
	totalCount int) (*models.LabelConnection, error)

// LabelCon will paginate a source according to the input of a relay connection
func LabelCon(source []common.Label, edgeMaker LabelEdgeMaker, conMaker LabelConMaker, input models.ConnectionInput) (*models.LabelConnection, error) {
	var nodes []common.Label
	var edges []*models.LabelEdge
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

			e := edge.(models.LabelEdge)
			edges = append(edges, &e)
			cursors = append(cursors, edge.GetCursor())
			nodes = append(nodes, value)
		}
	} else {
		edges = make([]*models.LabelEdge, len(source))
		cursors = make([]string, len(source))
		nodes = source

		for i, value := range source {
			edge := edgeMaker(value, i+offset)
			e := edge.(models.LabelEdge)
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
