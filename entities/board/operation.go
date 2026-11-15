package board

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/util/fracindex"
	"github.com/git-bug/git-bug/util/text"
)

// The operations of a board are in op_*.go for the board itself, column_ops.go
// for its columns and item_ops.go for its items.
//
// Applying an operation never fails: any operation may come after a concurrent one
// that removed what it targets, and each defines what it does then, most often
// nothing. See board.md §3.4 in https://github.com/git-bug/spec.

// Operation types. Numbers never change once released.
const (
	_ dag.OperationType = iota
	CreateOp
	SetTitleOp
	SetDescriptionOp
	AddColumnOp
	EditColumnOp
	RemoveColumnOp
	AddItemDraftOp
	AddItemEntityOp
	MoveItemOp
	RemoveItemOp
	EditDraftOp
	ConvertDraftOp
	SetMetadataOp
)

// Operation define the interface to fulfill for an edit operation of a Board
type Operation = dag.OperationWithApply[*Snapshot]

// make sure that package external operations do conform to our interface
var _ Operation = &dag.SetMetadataOperation[*Snapshot]{}
var _ Operation = &dag.UnknownOperation[*Snapshot]{}

func operationUnmarshaler(_ context.Context, raw json.RawMessage, _ entity.Resolvers) (dag.Operation, error) {
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
	case SetTitleOp:
		op = &SetTitleOperation{}
	case SetDescriptionOp:
		op = &SetDescriptionOperation{}
	case AddColumnOp:
		op = &AddColumnOperation{}
	case EditColumnOp:
		op = &EditColumnOperation{}
	case RemoveColumnOp:
		op = &RemoveColumnOperation{}
	case AddItemDraftOp:
		op = &AddItemDraftOperation{}
	case AddItemEntityOp:
		op = &AddItemEntityOperation{}
	case MoveItemOp:
		op = &MoveItemOperation{}
	case RemoveItemOp:
		op = &RemoveItemOperation{}
	case EditDraftOp:
		op = &EditDraftOperation{}
	case ConvertDraftOp:
		op = &ConvertDraftOperation{}
	case SetMetadataOp:
		op = &dag.SetMetadataOperation[*Snapshot]{}
	default:
		return dag.NewUnknownOp[*Snapshot](raw)
	}

	err := json.Unmarshal(raw, &op)
	if err != nil {
		return nil, err
	}

	return op, nil
}

// rejectNull checks that none of these members of the JSON of an operation is
// null. A member that may be absent is never null instead: null is not a value,
// except where an operation gives it a meaning of its own.
func rejectNull(data []byte, members ...string) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	for _, member := range members {
		if string(raw[member]) == "null" {
			return fmt.Errorf("%s: null is not allowed", member)
		}
	}
	return nil
}

// validateTitle checks the title of a board or of a draft, or the name of a
// column, named what in the error: a single line, not empty.
func validateTitle(what string, title string) error {
	if text.Empty(title) {
		return fmt.Errorf("%s is empty", what)
	}
	if !text.SafeOneLine(title) {
		return fmt.Errorf("%s has unsafe characters", what)
	}
	return nil
}

// validatePosition checks the position of a column or of an item.
func validatePosition(position string) error {
	if err := fracindex.Validate(position); err != nil {
		return fmt.Errorf("position: %w", err)
	}
	return nil
}

// validateTarget checks the id of a column or of an item that an operation
// targets, named what in the error.
func validateTarget(what string, id entity.Id) error {
	if err := id.Validate(); err != nil {
		return fmt.Errorf("%s: %w", what, err)
	}
	return nil
}
