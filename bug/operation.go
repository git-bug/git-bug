package bug

import (
	"github.com/MichaelMure/git-bug/util"
	"time"
)

type OperationType int

const (
	_ OperationType = iota
	CreateOp
	SetTitleOp
	AddCommentOp
	SetStatusOp
	LabelChangeOp
)

type Operation interface {
	OpType() OperationType
	Time() time.Time
	Apply(snapshot Snapshot) Snapshot
	Files() []util.Hash

	// TODO: data validation (ex: a title is a single line)
	// Validate() bool
}

type OpBase struct {
	OperationType OperationType
	Author        Person
	UnixTime      int64
}

func NewOpBase(opType OperationType, author Person) OpBase {
	return OpBase{
		OperationType: opType,
		Author:        author,
		UnixTime:      time.Now().Unix(),
	}
}

func (op OpBase) OpType() OperationType {
	return op.OperationType
}

func (op OpBase) Time() time.Time {
	return time.Unix(op.UnixTime, 0)
}
