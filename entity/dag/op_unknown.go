package dag

import (
	"encoding/json"

	"github.com/git-bug/git-bug/entity"
)

var _ Operation = &UnknownOperation[Snapshot]{}
var _ OperationDoesntChangeSnapshot = &UnknownOperation[Snapshot]{}

// UnknownOperation is a placeholder for operations with an unrecognized type.
// It preserves the raw JSON verbatim so data is not lost when a client encounters
// an operation type it does not implement. Apply is a no-op.
type UnknownOperation[SnapT Snapshot] struct {
	OpBase
	// RawJSON holds the original serialized bytes of the operation.
	RawJSON json.RawMessage
}

// NewUnknownOp creates an UnknownOperation from raw JSON.
// The common OpBase fields (type, timestamp, nonce, metadata) are decoded from
// the raw JSON; entity-specific fields are left in RawJSON.
func NewUnknownOp[SnapT Snapshot](raw json.RawMessage) (*UnknownOperation[SnapT], error) {
	op := &UnknownOperation[SnapT]{
		RawJSON: raw,
	}
	if err := json.Unmarshal(raw, &op.OpBase); err != nil {
		return nil, err
	}
	// derived from the raw JSON on first use, as the one of an operation read
	op.id = entity.UnsetId
	return op, nil
}

func (op *UnknownOperation[SnapT]) Id() entity.Id {
	return IdOperation(op, &op.OpBase)
}

func (op *UnknownOperation[SnapT]) Apply(_ SnapT) {}

// Validate skips OpBase validation: the shape of unknown operations is not known.
func (op *UnknownOperation[SnapT]) Validate() error { return nil }

func (op *UnknownOperation[SnapT]) DoesntChangeSnapshot() {}

// MarshalJSON returns the original raw JSON verbatim.
func (op *UnknownOperation[SnapT]) MarshalJSON() ([]byte, error) {
	return op.RawJSON, nil
}
