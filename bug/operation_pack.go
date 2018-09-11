package bug

import (
	"bytes"
	"encoding/gob"

	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/git"
)

// OperationPack represent an ordered set of operation to apply
// to a Bug. These operations are stored in a single Git commit.
//
// These commits will be linked together in a linear chain of commits
// inside Git to form the complete ordered chain of operation to
// apply to get the final state of the Bug
type OperationPack struct {
	Operations []Operation

	// Private field so not serialized by gob
	commitHash git.Hash
}

// ParseOperationPack will deserialize an OperationPack from raw bytes
func ParseOperationPack(data []byte) (*OperationPack, error) {
	reader := bytes.NewReader(data)
	decoder := gob.NewDecoder(reader)

	var opp OperationPack

	err := decoder.Decode(&opp)

	if err != nil {
		return nil, err
	}

	return &opp, nil
}

// Serialize will serialise an OperationPack into raw bytes
func (opp *OperationPack) Serialize() ([]byte, error) {
	var data bytes.Buffer

	encoder := gob.NewEncoder(&data)
	err := encoder.Encode(*opp)

	if err != nil {
		return nil, err
	}

	return data.Bytes(), nil
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
func (opp *OperationPack) IsValid() bool {
	return !opp.IsEmpty()
}

// Write will serialize and store the OperationPack as a git blob and return
// its hash
func (opp *OperationPack) Write(repo repository.Repo) (git.Hash, error) {
	data, err := opp.Serialize()

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
