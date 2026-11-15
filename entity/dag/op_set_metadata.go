package dag

import (
	"fmt"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/util/text"
)

var _ Operation = &SetMetadataOperation[Snapshot]{}
var _ OperationDoesntChangeSnapshot = &SetMetadataOperation[Snapshot]{}

type SetMetadataOperation[SnapT Snapshot] struct {
	OpBase
	Target      entity.Id         `json:"target" jsondoc:"required"`
	NewMetadata map[string]string `json:"new_metadata,omitempty"`
}

func NewSetMetadataOp[SnapT Snapshot](opType OperationType, author identity.Interface, unixTime int64, target entity.Id, newMetadata map[string]string) *SetMetadataOperation[SnapT] {
	return &SetMetadataOperation[SnapT]{
		OpBase:      NewOpBase(opType, author, unixTime),
		Target:      target,
		NewMetadata: newMetadata,
	}
}

func (op *SetMetadataOperation[SnapT]) Id() entity.Id {
	return IdOperation(op, &op.OpBase)
}

func (op *SetMetadataOperation[SnapT]) Apply(snapshot SnapT) {
	ops := snapshot.AllOperations()
	for i, target := range ops {
		if target.Id() == op.Target {
			// Apply the metadata in an immutable way: if a metadata already
			// exist, it's not possible to override it.
			// The target is shared with other snapshots, replace it in this one
			// by a copy carrying the metadata.
			ops[i] = withExtraMetadata(target, op.NewMetadata)
			return
		}
	}
}

func (op *SetMetadataOperation[SnapT]) Validate() error {
	if err := op.OpBase.Validate(op, op.OperationType); err != nil {
		return err
	}

	if err := op.Target.Validate(); err != nil {
		return fmt.Errorf("target invalid: %w", err)
	}

	if err := validateMetadata(op.NewMetadata); err != nil {
		return fmt.Errorf("new metadata: %w", err)
	}

	return nil
}

func (op *SetMetadataOperation[SnapT]) DoesntChangeSnapshot() {}

// validateMetadata checks the metadata of an operation, or the metadata that a
// SetMetadataOperation adds: each key is text and each value multi-line text, see
// dag-entity.md §6.1 in the spec.
func validateMetadata(metadata map[string]string) error {
	for key, val := range metadata {
		if !text.SafeOneLine(key) {
			return fmt.Errorf("key %q has unsafe characters", key)
		}
		if !text.Safe(val) {
			return fmt.Errorf("value of %q has unsafe characters", key)
		}
	}
	return nil
}
