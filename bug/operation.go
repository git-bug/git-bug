package bug

import (
	"github.com/MichaelMure/git-bug/util/git"
	"time"
)

// OperationType is an identifier
type OperationType int

const (
	_ OperationType = iota
	CreateOp
	SetTitleOp
	AddCommentOp
	SetStatusOp
	LabelChangeOp
)

// Operation define the interface to fulfill for an edit operation of a Bug
type Operation interface {
	// OpType return the type of operation
	OpType() OperationType
	// Time return the time when the operation was added
	Time() time.Time
	// GetUnixTime return the unix timestamp when the operation was added
	GetUnixTime() int64
	// Apply the operation to a Snapshot to create the final state
	Apply(snapshot Snapshot) Snapshot
	// Files return the files needed by this operation
	Files() []git.Hash

	// TODO: data validation (ex: a title is a single line)
	// Validate() bool
}

// OpBase implement the common code for all operations
type OpBase struct {
	OperationType OperationType
	Author        Person
	UnixTime      int64
}

// NewOpBase is the constructor for an OpBase
func NewOpBase(opType OperationType, author Person) OpBase {
	return OpBase{
		OperationType: opType,
		Author:        author,
		UnixTime:      time.Now().Unix(),
	}
}

// OpType return the type of operation
func (op OpBase) OpType() OperationType {
	return op.OperationType
}

// Time return the time when the operation was added
func (op OpBase) Time() time.Time {
	return time.Unix(op.UnixTime, 0)
}

// GetUnixTime return the unix timestamp when the operation was added
func (op OpBase) GetUnixTime() int64 {
	return op.UnixTime
}

// Files return the files needed by this operation
func (op OpBase) Files() []git.Hash {
	return nil
}
