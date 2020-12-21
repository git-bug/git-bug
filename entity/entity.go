package entity

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/pkg/errors"

	"github.com/MichaelMure/git-bug/repository"
)

type Operation interface {
	// Id() Id
	// MarshalJSON() ([]byte, error)
	Validate() error

	base() *OpBase
}

type OperationIterator struct {
}

type Definition struct {
	namespace            string
	operationUnmarshaler func(raw json.RawMessage) (Operation, error)
	formatVersion        uint
}

type Entity struct {
	Definition

	ops []Operation
}

func New(definition Definition) *Entity {
	return &Entity{
		Definition: definition,
	}
}

func Read(def Definition, repo repository.ClockedRepo, id Id) (*Entity, error) {
	if err := id.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid id")
	}

	ref := fmt.Sprintf("refs/%s/%s", def.namespace, id.String())

	rootHash, err := repo.ResolveRef(ref)
	if err != nil {
		return nil, err
	}

	// Perform a depth-first search to get a topological order of the DAG where we discover the
	// parents commit and go back in time up to the chronological root

	stack := make([]repository.Hash, 0, 32)
	visited := make(map[repository.Hash]struct{})
	DFSOrder := make([]repository.Commit, 0, 32)

	stack = append(stack, rootHash)

	for len(stack) > 0 {
		// pop
		hash := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if _, ok := visited[hash]; ok {
			continue
		}

		// mark as visited
		visited[hash] = struct{}{}

		commit, err := repo.ReadCommit(hash)
		if err != nil {
			return nil, err
		}

		DFSOrder = append(DFSOrder, commit)

		for _, parent := range commit.Parents {
			stack = append(stack, parent)
		}
	}

	// Now, we can reverse this topological order and read the commits in an order where
	// we are sure to have read all the chronological ancestors when we read a commit.

	// Next step is to:
	// 1) read the operationPacks
	// 2) make sure that the clocks causality respect the DAG topology.

	oppMap := make(map[repository.Hash]*operationPack)
	var opsCount int

	rootCommit := DFSOrder[len(DFSOrder)-1]
	rootOpp, err := readOperationPack(def, repo, rootCommit.TreeHash)
	if err != nil {
		return nil, err
	}
	oppMap[rootCommit.Hash] = rootOpp

	for i := len(DFSOrder) - 2; i >= 0; i-- {
		commit := DFSOrder[i]

		// Verify DAG structure: single chronological root
		if len(commit.Parents) == 0 {
			return nil, fmt.Errorf("multiple root in the entity DAG")
		}

		opp, err := readOperationPack(def, repo, commit.TreeHash)
		if err != nil {
			return nil, err
		}

		// make sure that the lamport clocks causality match the DAG topology
		for _, parentHash := range commit.Parents {
			parentPack, ok := oppMap[parentHash]
			if !ok {
				panic("DFS failed")
			}

			if parentPack.EditTime >= opp.EditTime {
				return nil, fmt.Errorf("lamport clock ordering doesn't match the DAG")
			}

			// to avoid an attack where clocks are pushed toward the uint64 rollover, make sure
			// that the clocks don't jump too far in the future
			if opp.EditTime-parentPack.EditTime > 10_000 {
				return nil, fmt.Errorf("lamport clock jumping too far in the future, likely an attack")
			}
		}

		oppMap[commit.Hash] = opp
		opsCount += len(opp.Operations)
	}

	// Now that we know that the topological order and clocks are fine, we order the operationPacks
	// based on the logical clocks, entirely ignoring the DAG topology

	oppSlice := make([]*operationPack, 0, len(oppMap))
	for _, pack := range oppMap {
		oppSlice = append(oppSlice, pack)
	}
	sort.Slice(oppSlice, func(i, j int) bool {
		// TODO: no secondary ordering?
		return oppSlice[i].EditTime < oppSlice[i].EditTime
	})

	// Now that we ordered the operationPacks, we have the order of the Operations

	ops := make([]Operation, 0, opsCount)
	for _, pack := range oppSlice {
		for _, operation := range pack.Operations {
			ops = append(ops, operation)
		}
	}

	return &Entity{
		Definition: def,
		ops:        ops,
	}, nil
}

func Remove() error {
	panic("")
}

func (e *Entity) Id() {

}

// return the ordered operations
func (e *Entity) Operations() []Operation {
	return e.ops
}

func (e *Entity) Commit() error {
	panic("")
}
