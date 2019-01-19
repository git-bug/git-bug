package bug

import (
	"encoding/json"

	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/util/git"
)

var _ Operation = &SetMetadataOperation{}

type SetMetadataOperation struct {
	OpBase
	Target      git.Hash
	NewMetadata map[string]string
}

func (op *SetMetadataOperation) base() *OpBase {
	return &op.OpBase
}

func (op *SetMetadataOperation) Hash() (git.Hash, error) {
	return hashOperation(op)
}

func (op *SetMetadataOperation) Apply(snapshot *Snapshot) {
	for _, target := range snapshot.Operations {
		hash, err := target.Hash()
		if err != nil {
			// Should never error unless a programming error happened
			// (covered in OpBase.Validate())
			panic(err)
		}

		if hash == op.Target {
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
		Target      git.Hash          `json:"target"`
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

func NewSetMetadataOp(author identity.Interface, unixTime int64, target git.Hash, newMetadata map[string]string) *SetMetadataOperation {
	return &SetMetadataOperation{
		OpBase:      newOpBase(SetMetadataOp, author, unixTime),
		Target:      target,
		NewMetadata: newMetadata,
	}
}

// Convenience function to apply the operation
func SetMetadata(b Interface, author identity.Interface, unixTime int64, target git.Hash, newMetadata map[string]string) (*SetMetadataOperation, error) {
	SetMetadataOp := NewSetMetadataOp(author, unixTime, target, newMetadata)
	if err := SetMetadataOp.Validate(); err != nil {
		return nil, err
	}
	b.Append(SetMetadataOp)
	return SetMetadataOp, nil
}
