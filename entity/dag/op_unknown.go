package dag

import (
	"encoding/json"

	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/util/jsondoc"
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

// NewUnknownOp creates an UnknownOperation from raw JSON, a part of a document
// that jsondoc.Check accepted. The common OpBase fields (type, timestamp, nonce,
// metadata) are decoded from the raw JSON when they can be; entity-specific
// fields are left in RawJSON. An operation of an unknown type is kept verbatim
// whatever it holds, so only its type is decoded when the other fields can't be,
// see dag-entity.md §6.4 in the spec.
func NewUnknownOp[SnapT Snapshot](raw json.RawMessage) (*UnknownOperation[SnapT], error) {
	base, err := rawOpBase(raw)
	if err != nil {
		return nil, err
	}
	return &UnknownOperation[SnapT]{OpBase: base, RawJSON: raw}, nil
}

// rawOpBase decodes the common OpBase fields of the raw JSON of an operation kept
// verbatim, or only its type when the other fields can't be decoded.
func rawOpBase(raw json.RawMessage) (OpBase, error) {
	var base OpBase
	if err := jsondoc.Unmarshal(raw, &base); err != nil {
		var t struct {
			OperationType OperationType `json:"type"`
		}
		if err := jsondoc.Unmarshal(raw, &t); err != nil {
			return OpBase{}, err
		}
		base = OpBase{OperationType: t.OperationType}
	}
	// derived from the raw JSON on first use, as the one of an operation read
	base.id = entity.UnsetId
	return base, nil
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
