package board

import (
	"encoding/json"

	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
)

// OperationType is an operation type identifier
type OperationType dag.OperationType

const (
	_ dag.OperationType = iota
	CreateOp
	SetMetadataOp
	SetTitleOp
	SetDescriptionOp
	AddItemEntityOp
	AddItemDraftOp
	MoveItemOp
	RemoveItemOp

	// TODO: change columns?
)

// Operation define the interface to fulfill for an edit operation of a Board
type Operation = dag.OperationWithApply[*Snapshot]

// make sure that package external operations do conform to our interface
var _ Operation = &dag.SetMetadataOperation[*Snapshot]{}
var _ Operation = &dag.UnknownOperation[*Snapshot]{}

func operationUnmarshaler(raw json.RawMessage, resolvers entity.Resolvers) (dag.Operation, error) {
	var t struct {
		OperationType dag.OperationType `json:"type"`
	}

	if err := json.Unmarshal(raw, &t); err != nil {
		return nil, err
	}

	var op dag.Operation

	switch t.OperationType {
	case CreateOp:
		op = &CreateOperation{}
	case SetMetadataOp:
		op = &dag.SetMetadataOperation[*Snapshot]{}
	case SetTitleOp:
		op = &SetTitleOperation{}
	case SetDescriptionOp:
		op = &SetDescriptionOperation{}
	case AddItemDraftOp:
		op = &AddItemDraftOperation{}
	case AddItemEntityOp:
		op = &AddItemEntityOperation{}
	default:
		return dag.NewUnknownOp[*Snapshot](raw)
	}

	err := json.Unmarshal(raw, &op)
	if err != nil {
		return nil, err
	}

	return op, nil
}
