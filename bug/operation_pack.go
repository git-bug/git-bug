package bug

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/git"
	"github.com/pkg/errors"
)

const formatVersion = 1

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

// hold the different operation type to instantiate to parse JSON
var operations map[OperationType]reflect.Type

// Register will register a new type of Operation to be able to parse the corresponding JSON
func Register(t OperationType, op interface{}) {
	if operations == nil {
		operations = make(map[OperationType]reflect.Type)
	}
	operations[t] = reflect.TypeOf(op)
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

	if aux.Version != formatVersion {
		return fmt.Errorf("unknown format version %v", aux.Version)
	}

	for _, raw := range aux.Operations {
		var t struct {
			OperationType OperationType `json:"type"`
		}

		if err := json.Unmarshal(raw, &t); err != nil {
			return err
		}

		opType, ok := operations[t.OperationType]
		if !ok {
			return fmt.Errorf("unknown operation type %v", t.OperationType)
		}

		op := reflect.New(opType).Interface()

		if err := json.Unmarshal(raw, op); err != nil {
			return err
		}

		deref := reflect.ValueOf(op).Elem().Interface()

		opp.Operations = append(opp.Operations, deref.(Operation))
	}

	return nil
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
func (opp *OperationPack) Write(repo repository.Repo) (git.Hash, error) {
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
