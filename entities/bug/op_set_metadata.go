package bug

import (
	"github.com/MichaelMure/git-bug/entities/identity"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/entity/dag"
)

func NewSetMetadataOp(author identity.Interface, unixTime int64, target entity.Id, newMetadata map[string]string) *dag.SetMetadataOperation[*Snapshot] {
	return dag.NewSetMetadataOp[*Snapshot](SetMetadataOp, author, unixTime, target, newMetadata)
}

// SetMetadata is a convenience function to add metadata on another operation
func SetMetadata(b Interface, author identity.Interface, unixTime int64, target entity.Id, newMetadata map[string]string) (*dag.SetMetadataOperation[*Snapshot], error) {
	op := NewSetMetadataOp(author, unixTime, target, newMetadata)
	if err := op.Validate(); err != nil {
		return nil, err
	}
	b.Append(op)
	return op, nil
}
