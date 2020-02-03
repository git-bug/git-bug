package bug

import (
	"encoding/json"

	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/identity"
)

var _ Operation = &NoOpOperation{}

// NoOpOperation is an operation that does not change the bug state. It can
// however be used to store arbitrary metadata in the bug history, for example
// to support a bridge feature.
type NoOpOperation struct {
	OpBase
}

// Sign-post method for gqlgen
func (op *NoOpOperation) IsOperation() {}

func (op *NoOpOperation) base() *OpBase {
	return &op.OpBase
}

func (op *NoOpOperation) Id() entity.Id {
	return idOperation(op)
}

func (op *NoOpOperation) Apply(snapshot *Snapshot) {
	// Nothing to do
}

func (op *NoOpOperation) Validate() error {
	return opBaseValidate(op, NoOpOp)
}

// UnmarshalJSON is a two step JSON unmarshaling
// This workaround is necessary to avoid the inner OpBase.MarshalJSON
// overriding the outer op's MarshalJSON
func (op *NoOpOperation) UnmarshalJSON(data []byte) error {
	// Unmarshal OpBase and the op separately

	base := OpBase{}
	err := json.Unmarshal(data, &base)
	if err != nil {
		return err
	}

	aux := struct{}{}

	err = json.Unmarshal(data, &aux)
	if err != nil {
		return err
	}

	op.OpBase = base

	return nil
}

// Sign post method for gqlgen
func (op *NoOpOperation) IsAuthored() {}

func NewNoOpOp(author identity.Interface, unixTime int64) *NoOpOperation {
	return &NoOpOperation{
		OpBase: newOpBase(NoOpOp, author, unixTime),
	}
}

// Convenience function to apply the operation
func NoOp(b Interface, author identity.Interface, unixTime int64, metadata map[string]string) (*NoOpOperation, error) {
	op := NewNoOpOp(author, unixTime)

	for key, value := range metadata {
		op.SetMetadata(key, value)
	}

	if err := op.Validate(); err != nil {
		return nil, err
	}
	b.Append(op)
	return op, nil
}
