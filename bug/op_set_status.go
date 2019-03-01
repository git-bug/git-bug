package bug

import (
	"encoding/json"

	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/util/git"
	"github.com/MichaelMure/git-bug/util/timestamp"
	"github.com/pkg/errors"
)

var _ Operation = &SetStatusOperation{}

// SetStatusOperation will change the status of a bug
type SetStatusOperation struct {
	OpBase
	Status Status
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
		UnixTime: timestamp.Timestamp(op.UnixTime),
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

// Workaround to avoid the inner OpBase.MarshalJSON overriding the outer op
// MarshalJSON
func (op *SetStatusOperation) MarshalJSON() ([]byte, error) {
	base, err := json.Marshal(op.OpBase)
	if err != nil {
		return nil, err
	}

	// revert back to a flat map to be able to add our own fields
	var data map[string]interface{}
	if err := json.Unmarshal(base, &data); err != nil {
		return nil, err
	}

	data["status"] = op.Status

	return json.Marshal(data)
}

// Workaround to avoid the inner OpBase.MarshalJSON overriding the outer op
// MarshalJSON
func (op *SetStatusOperation) UnmarshalJSON(data []byte) error {
	// Unmarshal OpBase and the op separately

	base := OpBase{}
	err := json.Unmarshal(data, &base)
	if err != nil {
		return err
	}

	aux := struct {
		Status Status `json:"status"`
	}{}

	err = json.Unmarshal(data, &aux)
	if err != nil {
		return err
	}

	op.OpBase = base
	op.Status = aux.Status

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
	UnixTime timestamp.Timestamp
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
