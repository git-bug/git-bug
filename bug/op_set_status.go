package bug

import (
	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/util/git"
	"github.com/pkg/errors"
)

var _ Operation = &SetStatusOperation{}

// SetStatusOperation will change the status of a bug
type SetStatusOperation struct {
	OpBase
	Status Status `json:"status"`
}

func (op *SetStatusOperation) base() *OpBase {
	return &op.OpBase
}

func (op *SetStatusOperation) Hash() (git.Hash, error) {
	return hashOperation(op)
}

func (op *SetStatusOperation) Apply(snapshot *Snapshot) {
	snapshot.Status = op.Status

	hash, err := op.Hash()
	if err != nil {
		// Should never error unless a programming error happened
		// (covered in OpBase.Validate())
		panic(err)
	}

	item := &SetStatusTimelineItem{
		hash:     hash,
		Author:   op.Author,
		UnixTime: Timestamp(op.UnixTime),
		Status:   op.Status,
	}

	snapshot.Timeline = append(snapshot.Timeline, item)
}

func (op *SetStatusOperation) Validate() error {
	if err := opBaseValidate(op, SetStatusOp); err != nil {
		return err
	}

	if err := op.Status.Validate(); err != nil {
		return errors.Wrap(err, "status")
	}

	return nil
}

// Sign post method for gqlgen
func (op *SetStatusOperation) IsAuthored() {}

func NewSetStatusOp(author identity.Interface, unixTime int64, status Status) *SetStatusOperation {
	return &SetStatusOperation{
		OpBase: newOpBase(SetStatusOp, author, unixTime),
		Status: status,
	}
}

type SetStatusTimelineItem struct {
	hash     git.Hash
	Author   identity.Interface
	UnixTime Timestamp
	Status   Status
}

func (s SetStatusTimelineItem) Hash() git.Hash {
	return s.hash
}

// Convenience function to apply the operation
func Open(b Interface, author identity.Interface, unixTime int64) (*SetStatusOperation, error) {
	op := NewSetStatusOp(author, unixTime, OpenStatus)
	if err := op.Validate(); err != nil {
		return nil, err
	}
	b.Append(op)
	return op, nil
}

// Convenience function to apply the operation
func Close(b Interface, author identity.Interface, unixTime int64) (*SetStatusOperation, error) {
	op := NewSetStatusOp(author, unixTime, ClosedStatus)
	if err := op.Validate(); err != nil {
		return nil, err
	}
	b.Append(op)
	return op, nil
}
