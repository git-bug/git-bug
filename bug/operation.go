package bug

import (
	"github.com/MichaelMure/git-bug/util/git"
	"github.com/pkg/errors"

	"fmt"
	"time"
)

// OperationType is an operation type identifier
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
	// GetAuthor return the author of the operation
	GetAuthor() Person
	// GetFiles return the files needed by this operation
	GetFiles() []git.Hash
	// Apply the operation to a Snapshot to create the final state
	Apply(snapshot Snapshot) Snapshot
	// Validate check if the operation is valid (ex: a title is a single line)
	Validate() error
}

// OpBase implement the common code for all operations
type OpBase struct {
	OperationType OperationType `json:"type"`
	Author        Person        `json:"author"`
	UnixTime      int64         `json:"timestamp"`
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

// GetAuthor return the author of the operation
func (op OpBase) GetAuthor() Person {
	return op.Author
}

// GetFiles return the files needed by this operation
func (op OpBase) GetFiles() []git.Hash {
	return nil
}

// Validate check the OpBase for errors
func OpBaseValidate(op Operation, opType OperationType) error {
	if op.OpType() != opType {
		return fmt.Errorf("incorrect operation type (expected: %v, actual: %v)", opType, op.OpType())
	}

	if op.GetUnixTime() == 0 {
		return fmt.Errorf("time not set")
	}

	if err := op.GetAuthor().Validate(); err != nil {
		return errors.Wrap(err, "author")
	}

	for _, hash := range op.GetFiles() {
		if !hash.IsValid() {
			return fmt.Errorf("file with invalid hash %v", hash)
		}
	}

	return nil
}
