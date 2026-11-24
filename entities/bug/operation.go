package bug

import (
	"context"
	"encoding/json"

	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/util/jsondoc"
)

const (
	_ dag.OperationType = iota
	CreateOp
	SetTitleOp
	AddCommentOp
	SetStatusOp
	LabelChangeOp
	EditCommentOp
	NoOpOp
	SetMetadataOp
)

// Operation define the interface to fulfill for an edit operation of a Bug
type Operation = dag.OperationWithApply[*Snapshot]

// make sure that package external operations do conform to our interface
var _ Operation = &dag.NoOpOperation[*Snapshot]{}
var _ Operation = &dag.SetMetadataOperation[*Snapshot]{}
var _ Operation = &dag.UnknownOperation[*Snapshot]{}
var _ Operation = &dag.InvalidOperation[*Snapshot]{}

func operationUnmarshaler(_ context.Context, raw json.RawMessage, resolvers entity.Resolvers) (dag.Operation, error) {
	var t struct {
		OperationType dag.OperationType `json:"type"`
	}

	if err := jsondoc.Unmarshal(raw, &t); err != nil {
		return nil, err
	}

	var op dag.Operation
	// the required members whose zero value is valid, which decoding can't tell
	// from absent ones, see jsondoc.Require
	var required []string

	switch t.OperationType {
	case AddCommentOp:
		op, required = &AddCommentOperation{}, []string{"message"}
	case CreateOp:
		op, required = &CreateOperation{}, []string{"message"}
	case EditCommentOp:
		op, required = &EditCommentOperation{}, []string{"message"}
	case LabelChangeOp:
		op = &LabelChangeOperation{}
	case NoOpOp:
		op = &dag.NoOpOperation[*Snapshot]{}
	case SetMetadataOp:
		op = &dag.SetMetadataOperation[*Snapshot]{}
	case SetStatusOp:
		op = &SetStatusOperation{}
	case SetTitleOp:
		op, required = &SetTitleOperation{}, []string{"was"}
	default:
		return dag.NewUnknownOp[*Snapshot](raw)
	}

	// an operation of a known type that can't be decoded or misses a required
	// member is invalid, see dag.InvalidOperation
	if err := jsondoc.Unmarshal(raw, op); err != nil {
		return dag.NewInvalidOp[*Snapshot](raw, err)
	}
	if err := jsondoc.Require(raw, required...); err != nil {
		return dag.NewInvalidOp[*Snapshot](raw, err)
	}

	return op, nil
}
