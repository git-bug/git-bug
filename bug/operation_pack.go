package bug

import (
	"encoding/json"
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
	Operations []Operation `json:"ops"`
	hash       util.Hash
}

func Parse() (OperationPack, error) {
	// TODO
	return OperationPack{}, nil
}

func (opp *OperationPack) Serialize() ([]byte, error) {
	jsonBytes, err := json.Marshal(*opp)
	if err != nil {
		return nil, err
	}

	return jsonBytes, nil
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
