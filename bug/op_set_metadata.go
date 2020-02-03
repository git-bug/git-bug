package bug

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/identity"
)

var _ Operation = &SetMetadataOperation{}

type SetMetadataOperation struct {
	OpBase
	Target      entity.Id         `json:"target"`
	NewMetadata map[string]string `json:"new_metadata"`
}

// Sign-post method for gqlgen
func (op *SetMetadataOperation) IsOperation() {}

func (op *SetMetadataOperation) base() *OpBase {
	return &op.OpBase
}

func (op *SetMetadataOperation) Id() entity.Id {
	return idOperation(op)
}

func (op *SetMetadataOperation) Apply(snapshot *Snapshot) {
	for _, target := range snapshot.Operations {
		if target.Id() == op.Target {
			base := target.base()

			if base.extraMetadata == nil {
				base.extraMetadata = make(map[string]string)
			}

			// Apply the metadata in an immutable way: if a metadata already
			// exist, it's not possible to override it.
			for key, val := range op.NewMetadata {
				if _, exist := base.extraMetadata[key]; !exist {
					base.extraMetadata[key] = val
				}
			}

			return
		}
	}
}

func (op *SetMetadataOperation) Validate() error {
	if err := opBaseValidate(op, SetMetadataOp); err != nil {
		return err
	}

	if err := op.Target.Validate(); err != nil {
		return errors.Wrap(err, "target invalid")
	}

	return nil
}

// UnmarshalJSON is a two step JSON unmarshaling
// This workaround is necessary to avoid the inner OpBase.MarshalJSON
// overriding the outer op's MarshalJSON
func (op *SetMetadataOperation) UnmarshalJSON(data []byte) error {
	// Unmarshal OpBase and the op separately

	base := OpBase{}
	err := json.Unmarshal(data, &base)
	if err != nil {
		return err
	}

	aux := struct {
		Target      entity.Id         `json:"target"`
		NewMetadata map[string]string `json:"new_metadata"`
	}{}

	err = json.Unmarshal(data, &aux)
	if err != nil {
		return err
	}

	op.OpBase = base
	op.Target = aux.Target
	op.NewMetadata = aux.NewMetadata

	return nil
}

// Sign post method for gqlgen
func (op *SetMetadataOperation) IsAuthored() {}

func NewSetMetadataOp(author identity.Interface, unixTime int64, target entity.Id, newMetadata map[string]string) *SetMetadataOperation {
	return &SetMetadataOperation{
		OpBase:      newOpBase(SetMetadataOp, author, unixTime),
		Target:      target,
		NewMetadata: newMetadata,
	}
}

// Convenience function to apply the operation
func SetMetadata(b Interface, author identity.Interface, unixTime int64, target entity.Id, newMetadata map[string]string) (*SetMetadataOperation, error) {
	SetMetadataOp := NewSetMetadataOp(author, unixTime, target, newMetadata)
	if err := SetMetadataOp.Validate(); err != nil {
		return nil, err
	}
	b.Append(SetMetadataOp)
	return SetMetadataOp, nil
}
