package dag

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/util/text"
)

var _ Operation = &SetMetadataOperation[Snapshot]{}
var _ OperationDoesntChangeSnapshot = &SetMetadataOperation[Snapshot]{}

type SetMetadataOperation[SnapT Snapshot] struct {
	OpBase
	Target      entity.Id         `json:"target"`
	NewMetadata map[string]string `json:"new_metadata"`
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
		return errors.Wrap(err, "target invalid")
	}

	for key, val := range op.NewMetadata {
		if !text.SafeOneLine(key) {
			return fmt.Errorf("metadata key is unsafe")
		}
		if !text.Safe(val) {
			return fmt.Errorf("metadata value is not fully printable")
		}
	}

	return nil
}

func (op *SetMetadataOperation[SnapT]) DoesntChangeSnapshot() {}
