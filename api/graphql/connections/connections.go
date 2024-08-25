// Package connections implement a generic GraphQL relay connection
package connections

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/git-bug/git-bug/api/graphql/models"
)

const cursorPrefix = "cursor:"

// Edge define the contract for an edge in a relay connection
type Edge interface {
	GetCursor() string
}

// OffsetToCursor create the cursor string from an offset
func OffsetToCursor(offset int) string {
	str := fmt.Sprintf("%v%v", cursorPrefix, offset)
	return base64.StdEncoding.EncodeToString([]byte(str))
}

// CursorToOffset re-derives the offset from the cursor string.
func CursorToOffset(cursor string) (int, error) {
	str := ""
	b, err := base64.StdEncoding.DecodeString(cursor)
	if err == nil {
		str = string(b)
	}
	str = strings.Replace(str, cursorPrefix, "", -1)
	offset, err := strconv.Atoi(str)
	if err != nil {
		return 0, fmt.Errorf("Invalid cursor")
	}
	return offset, nil
}

// EdgeMaker defines a function that takes a NodeType and an offset and
// create an Edge.
type EdgeMaker[NodeType any] func(value NodeType, offset int) Edge

// ConMaker defines a function that creates a ConnectionType
type ConMaker[NodeType any, EdgeType Edge, ConType any] func(
	edges []*EdgeType,
	nodes []NodeType,
	info *models.PageInfo,
	totalCount int) (*ConType, error)

// Connection will paginate a source according to the input of a relay connection
func Connection[NodeType any, EdgeType Edge, ConType any](
	source []NodeType,
	edgeMaker EdgeMaker[NodeType],
	conMaker ConMaker[NodeType, EdgeType, ConType],
	input models.ConnectionInput) (*ConType, error) {

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
