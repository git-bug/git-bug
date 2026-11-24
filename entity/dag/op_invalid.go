package dag

import (
	"encoding/json"

	"github.com/git-bug/git-bug/entity"
)

var _ Operation = &InvalidOperation[Snapshot]{}
var _ OperationDoesntChangeSnapshot = &InvalidOperation[Snapshot]{}

// InvalidOperation holds an operation of a known type that can't be decoded, or
// misses a required member, as there is no value of its type to hold it. It preserves the raw JSON verbatim so
// that the entity can be written back, and is invalid: it has no effect, see
// Operation.Invalid and dag-entity.md §6 in the spec. An operation that decodes
// but doesn't validate keeps its own type instead.
type InvalidOperation[SnapT Snapshot] struct {
	OpBase
	// RawJSON holds the original serialized bytes of the operation.
	RawJSON json.RawMessage
}

// NewInvalidOp creates an InvalidOperation from raw JSON, an element of the ops of
// a pack that unmarshallPack accepted, which can't be decoded for reason: it is
// flagged invalid from the start. The common OpBase fields are decoded when they can be, as for
// an UnknownOperation.
func NewInvalidOp[SnapT Snapshot](raw json.RawMessage, reason error) (*InvalidOperation[SnapT], error) {
	base, err := rawOpBase(raw)
	if err != nil {
		return nil, err
	}
	base.setInvalid(reason)
	return &InvalidOperation[SnapT]{OpBase: base, RawJSON: raw}, nil
}

func (op *InvalidOperation[SnapT]) Id() entity.Id {
	return IdOperation(op, &op.OpBase)
}

func (op *InvalidOperation[SnapT]) Apply(_ SnapT) {}

// Validate returns why the operation can't be decoded, see Operation.Invalid.
func (op *InvalidOperation[SnapT]) Validate() error { return op.Invalid() }

func (op *InvalidOperation[SnapT]) DoesntChangeSnapshot() {}

// MarshalJSON returns the original raw JSON verbatim.
func (op *InvalidOperation[SnapT]) MarshalJSON() ([]byte, error) {
	return op.RawJSON, nil
}
