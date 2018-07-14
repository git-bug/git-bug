package bug

import (
	"bytes"
	"encoding/gob"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util"
)

// OperationPack represent an ordered set of operation to apply
// to a Bug. These operations are stored in a single Git commit.
//
// These commits will be linked together in a linear chain of commits
// inside Git to form the complete ordered chain of operation to
// apply to get the final state of the Bug
type OperationPack struct {
	Operations []Operation
}

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

func (opp *OperationPack) IsEmpty() bool {
	return len(opp.Operations) == 0
}

func (opp *OperationPack) IsValid() bool {
	return !opp.IsEmpty()
}

func (opp *OperationPack) Write(repo repository.Repo) (util.Hash, error) {
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
