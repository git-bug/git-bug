package bug

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"

	"github.com/MichaelMure/git-bug/repository"
)

// 1: original format
// 2: no more legacy identities
// 3: Ids are generated from the create operation serialized data instead of from the first git commit
const formatVersion = 3

// OperationPack represent an ordered set of operation to apply
// to a Bug. These operations are stored in a single Git commit.
//
// These commits will be linked together in a linear chain of commits
// inside Git to form the complete ordered chain of operation to
// apply to get the final state of the Bug
type OperationPack struct {
	Operations []Operation

	// Private field so not serialized
	commitHash repository.Hash
}

func (opp *OperationPack) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Version    uint        `json:"version"`
		Operations []Operation `json:"ops"`
	}{
		Version:    formatVersion,
		Operations: opp.Operations,
	})
}

func (opp *OperationPack) UnmarshalJSON(data []byte) error {
	aux := struct {
		Version    uint              `json:"version"`
		Operations []json.RawMessage `json:"ops"`
	}{}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.Version < formatVersion {
		return fmt.Errorf("outdated repository format, please use https://github.com/MichaelMure/git-bug-migration to upgrade")
	}
	if aux.Version > formatVersion {
		return fmt.Errorf("your version of git-bug is too old for this repository (bug format %v), please upgrade to the latest version", aux.Version)
	}

	for _, raw := range aux.Operations {
		var t struct {
			OperationType OperationType `json:"type"`
		}

		if err := json.Unmarshal(raw, &t); err != nil {
			return err
		}

		// delegate to specialized unmarshal function
		op, err := opp.unmarshalOp(raw, t.OperationType)
		if err != nil {
			return err
		}

		opp.Operations = append(opp.Operations, op)
	}

	return nil
}

func (opp *OperationPack) unmarshalOp(raw []byte, _type OperationType) (Operation, error) {
	switch _type {
	case AddCommentOp:
		op := &AddCommentOperation{}
		err := json.Unmarshal(raw, &op)
		return op, err
	case CreateOp:
		op := &CreateOperation{}
		err := json.Unmarshal(raw, &op)
		return op, err
	case EditCommentOp:
		op := &EditCommentOperation{}
		err := json.Unmarshal(raw, &op)
		return op, err
	case LabelChangeOp:
		op := &LabelChangeOperation{}
		err := json.Unmarshal(raw, &op)
		return op, err
	case NoOpOp:
		op := &NoOpOperation{}
		err := json.Unmarshal(raw, &op)
		return op, err
	case SetMetadataOp:
		op := &SetMetadataOperation{}
		err := json.Unmarshal(raw, &op)
		return op, err
	case SetStatusOp:
		op := &SetStatusOperation{}
		err := json.Unmarshal(raw, &op)
		return op, err
	case SetTitleOp:
		op := &SetTitleOperation{}
		err := json.Unmarshal(raw, &op)
		return op, err
	default:
		return nil, fmt.Errorf("unknown operation type %v", _type)
	}
}

// Append a new operation to the pack
func (opp *OperationPack) Append(op Operation) {
	opp.Operations = append(opp.Operations, op)
}

// IsEmpty tell if the OperationPack is empty
func (opp *OperationPack) IsEmpty() bool {
	return len(opp.Operations) == 0
}

// IsValid tell if the OperationPack is considered valid
func (opp *OperationPack) Validate() error {
	if opp.IsEmpty() {
		return fmt.Errorf("empty")
	}

	for _, op := range opp.Operations {
		if err := op.Validate(); err != nil {
			return errors.Wrap(err, "op")
		}
	}

	return nil
}

// Write will serialize and store the OperationPack as a git blob and return
// its hash
func (opp *OperationPack) Write(repo repository.ClockedRepo) (repository.Hash, error) {
	// make sure we don't write invalid data
	err := opp.Validate()
	if err != nil {
		return "", errors.Wrap(err, "validation error")
	}

	// First, make sure that all the identities are properly Commit as well
	// TODO: this might be downgraded to "make sure it exist in git" but then, what make
	// sure no data is lost on identities ?
	for _, op := range opp.Operations {
		if op.base().Author.NeedCommit() {
			return "", fmt.Errorf("identity need commmit")
		}
	}

	data, err := json.Marshal(opp)
	if err != nil {
		return "", err
	}

	hash, err := repo.StoreData(data)
	if err != nil {
		return "", err
	}

	return hash, nil
}

// Make a deep copy
func (opp *OperationPack) Clone() OperationPack {

	clone := OperationPack{
		Operations: make([]Operation, len(opp.Operations)),
		commitHash: opp.commitHash,
	}

	for i, op := range opp.Operations {
		clone.Operations[i] = op
	}

	return clone
}
