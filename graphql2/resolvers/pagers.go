//go:generate genny -in=pagers_template.go -out=pager_bug.go gen "NodeType=bug.Snapshot EdgeType=BugEdge"
//go:generate genny -in=pagers_template.go -out=pager_operation.go gen "NodeType=bug.Operation EdgeType=OperationEdge"
//go:generate genny -in=pagers_template.go -out=pager_comment.go gen "NodeType=bug.Comment EdgeType=CommentEdge"

package resolvers

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

const cursorPrefix = "cursor:"

type Edge interface {
	GetCursor() string
}

// Creates the cursor string from an offset
func offsetToCursor(offset int) string {
	str := fmt.Sprintf("%v%v", cursorPrefix, offset)
	return base64.StdEncoding.EncodeToString([]byte(str))
}

// Re-derives the offset from the cursor string.
func cursorToOffset(cursor string) (int, error) {
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

func (e OperationEdge) GetCursor() string {
	return e.Cursor
}

func (e BugEdge) GetCursor() string {
	return e.Cursor
}

func (e CommentEdge) GetCursor() string {
	return e.Cursor
}
