package bug

import (
	"encoding/json"
	"fmt"

	"github.com/MichaelMure/git-bug/identity"
)

var _ Operation = &SetMetadataOperation{}

type SetMetadataOperation struct {
	OpBase
	Target      string
	NewMetadata map[string]string
}

func (op *SetMetadataOperation) base() *OpBase {
	return &op.OpBase
}

func (op *SetMetadataOperation) ID() string {
	return idOperation(op)
}

func (op *SetMetadataOperation) Apply(snapshot *Snapshot) {
	for _, target := range snapshot.Operations {
		if target.ID() == op.Target {
			base := target.base()

			if base.extraMetadata == nil {
				base.extraMetadata = make(map[string]string)
			}

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

	if !IDIsValid(op.Target) {
		return fmt.Errorf("target hash is invalid")
	}

	return nil
}

// Workaround to avoid the inner OpBase.MarshalJSON overriding the outer op
// MarshalJSON
func (op *SetMetadataOperation) MarshalJSON() ([]byte, error) {
	base, err := json.Marshal(op.OpBase)
	if err != nil {
		return nil, err
	}

	// revert back to a flat map to be able to add our own fields
	var data map[string]interface{}
	if err := json.Unmarshal(base, &data); err != nil {
		return nil, err
	}

	data["target"] = op.Target
	data["new_metadata"] = op.NewMetadata

	return json.Marshal(data)
}

// Workaround to avoid the inner OpBase.MarshalJSON overriding the outer op
// MarshalJSON
func (op *SetMetadataOperation) UnmarshalJSON(data []byte) error {
	// Unmarshal OpBase and the op separately

	base := OpBase{}
	err := json.Unmarshal(data, &base)
	if err != nil {
		return err
	}

	aux := struct {
		Target      string            `json:"target"`
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

func NewSetMetadataOp(author identity.Interface, unixTime int64, target string, newMetadata map[string]string) *SetMetadataOperation {
	return &SetMetadataOperation{
		OpBase:      newOpBase(SetMetadataOp, author, unixTime),
		Target:      target,
		NewMetadata: newMetadata,
	}
}

// Convenience function to apply the operation
func SetMetadata(b Interface, author identity.Interface, unixTime int64, target string, newMetadata map[string]string) (*SetMetadataOperation, error) {
	SetMetadataOp := NewSetMetadataOp(author, unixTime, target, newMetadata)
	if err := SetMetadataOp.Validate(); err != nil {
		return nil, err
	}
	b.Append(SetMetadataOp)
	return SetMetadataOp, nil
}
