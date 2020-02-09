//go:generate genny -in=connection_template.go -out=gen_lazy_bug.go gen "Name=LazyBug NodeType=entity.Id EdgeType=LazyBugEdge ConnectionType=models.BugConnection"
//go:generate genny -in=connection_template.go -out=gen_lazy_identity.go gen "Name=LazyIdentity NodeType=entity.Id EdgeType=LazyIdentityEdge ConnectionType=models.IdentityConnection"
//go:generate genny -in=connection_template.go -out=gen_identity.go gen "Name=Identity NodeType=models.IdentityWrapper EdgeType=models.IdentityEdge ConnectionType=models.IdentityConnection"
//go:generate genny -in=connection_template.go -out=gen_operation.go gen "Name=Operation NodeType=bug.Operation EdgeType=models.OperationEdge ConnectionType=models.OperationConnection"
//go:generate genny -in=connection_template.go -out=gen_comment.go gen "Name=Comment NodeType=bug.Comment EdgeType=models.CommentEdge ConnectionType=models.CommentConnection"
//go:generate genny -in=connection_template.go -out=gen_timeline.go gen "Name=TimelineItem NodeType=bug.TimelineItem EdgeType=models.TimelineItemEdge ConnectionType=models.TimelineItemConnection"
//go:generate genny -in=connection_template.go -out=gen_label.go gen "Name=Label NodeType=bug.Label EdgeType=models.LabelEdge ConnectionType=models.LabelConnection"

// Package connections implement a generic GraphQL relay connection
package connections

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
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
