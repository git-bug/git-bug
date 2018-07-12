package operations

import (
	"github.com/MichaelMure/git-bug/bug"
)

type OperationType int

const (
	UNKNOW OperationType = iota
	CREATE
	SET_TITLE
	ADD_COMMENT
)

type Operation interface {
	OpType() OperationType
	Apply(snapshot bug.Snapshot) bug.Snapshot
}
