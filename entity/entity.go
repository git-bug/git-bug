package entity

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/pkg/errors"

	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/lamport"
)

const refsPattern = "refs/%s/%s"
const creationClockPattern = "%s-create"
const editClockPattern = "%s-edit"

type Operation interface {
	Id() Id
	// MarshalJSON() ([]byte, error)
	Validate() error
}

// Definition hold the details defining one specialization of an Entity.
type Definition struct {
	// the name of the entity (bug, pull-request, ...)
	typename string
	// the namespace in git (bugs, prs, ...)
	namespace string
	// a function decoding a JSON message into an Operation
	operationUnmarshaler func(raw json.RawMessage) (Operation, error)
	// the expected format version number
	formatVersion uint
}

type Entity struct {
	Definition

	ops     []Operation
	staging []Operation

	packClock  lamport.Clock
	lastCommit repository.Hash
}

func New(definition Definition) *Entity {
	return &Entity{
		Definition: definition,
		packClock:  lamport.NewMemClock(),
	}
}

func Read(def Definition, repo repository.ClockedRepo, id Id) (*Entity, error) {
	if err := id.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid id")
	}

	ref := fmt.Sprintf("refs/%s/%s", def.namespace, id.String())

	return read(def, repo, ref)
}

// read fetch from git and decode an Entity at an arbitrary git reference.
func read(def Definition, repo repository.ClockedRepo, ref string) (*Entity, error) {
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
	var packClock = lamport.NewMemClock()

	for i := len(DFSOrder) - 1; i >= 0; i-- {
		commit := DFSOrder[i]
		firstCommit := i == len(DFSOrder)-1

		// Verify DAG structure: single chronological root, so only the root
		// can have no parents
		if !firstCommit && len(commit.Parents) == 0 {
			return nil, fmt.Errorf("multiple root in the entity DAG")
		}

		opp, err := readOperationPack(def, repo, commit.TreeHash)
		if err != nil {
			return nil, err
		}

		// Check that the lamport clocks are set
		if firstCommit && opp.CreateTime <= 0 {
			return nil, fmt.Errorf("creation lamport time not set")
		}
		if opp.EditTime <= 0 {
			return nil, fmt.Errorf("edition lamport time not set")
		}
		if opp.PackTime <= 0 {
			return nil, fmt.Errorf("pack lamport time not set")
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

	// The clocks are fine, we witness them
	for _, opp := range oppMap {
		err = repo.Witness(fmt.Sprintf(creationClockPattern, def.namespace), opp.CreateTime)
		if err != nil {
			return nil, err
		}
		err = repo.Witness(fmt.Sprintf(editClockPattern, def.namespace), opp.EditTime)
		if err != nil {
			return nil, err
		}
		err = packClock.Witness(opp.PackTime)
		if err != nil {
			return nil, err
		}
	}

	// Now that we know that the topological order and clocks are fine, we order the operationPacks
	// based on the logical clocks, entirely ignoring the DAG topology

	oppSlice := make([]*operationPack, 0, len(oppMap))
	for _, pack := range oppMap {
		oppSlice = append(oppSlice, pack)
	}
	sort.Slice(oppSlice, func(i, j int) bool {
		// Primary ordering with the dedicated "pack" Lamport time that encode causality
		// within the entity
		if oppSlice[i].PackTime != oppSlice[j].PackTime {
			return oppSlice[i].PackTime < oppSlice[i].PackTime
		}
		// We have equal PackTime, which means we had a concurrent edition. We can't tell which exactly
		// came first. As a secondary arbitrary ordering, we can use the EditTime. It's unlikely to be
		// enough but it can give us an edge to approach what really happened.
		if oppSlice[i].EditTime != oppSlice[j].EditTime {
			return oppSlice[i].EditTime < oppSlice[j].EditTime
		}
		// Well, what now? We still need a total ordering, the most stable possible.
		// As a last resort, we can order based on a hash of the serialized Operations in the
		// operationPack. It doesn't carry much meaning but it's unbiased and hard to abuse.
		// This is a lexicographic ordering.
		return oppSlice[i].Id < oppSlice[j].Id
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
		lastCommit: rootHash,
	}, nil
}

// Id return the Entity identifier
func (e *Entity) Id() Id {
	// id is the id of the first operation
	return e.FirstOp().Id()
}

func (e *Entity) Validate() error {
	// non-empty
	if len(e.ops) == 0 && len(e.staging) == 0 {
		return fmt.Errorf("entity has no operations")
	}

	// check if each operations are valid
	for _, op := range e.ops {
		if err := op.Validate(); err != nil {
			return err
		}
	}

	// check if staging is valid if needed
	for _, op := range e.staging {
		if err := op.Validate(); err != nil {
			return err
		}
	}

	// Check that there is no colliding operation's ID
	ids := make(map[Id]struct{})
	for _, op := range e.Operations() {
		if _, ok := ids[op.Id()]; ok {
			return fmt.Errorf("id collision: %s", op.Id())
		}
		ids[op.Id()] = struct{}{}
	}

	return nil
}

// return the ordered operations
func (e *Entity) Operations() []Operation {
	return append(e.ops, e.staging...)
}

// Lookup for the very first operation of the Entity.
func (e *Entity) FirstOp() Operation {
	for _, op := range e.ops {
		return op
	}
	for _, op := range e.staging {
		return op
	}
	return nil
}

func (e *Entity) Append(op Operation) {
	e.staging = append(e.staging, op)
}

func (e *Entity) NeedCommit() bool {
	return len(e.staging) > 0
}

func (e *Entity) CommitAdNeeded(repo repository.ClockedRepo) error {
	if e.NeedCommit() {
		return e.Commit(repo)
	}
	return nil
}

// TODO: support commit signature
func (e *Entity) Commit(repo repository.ClockedRepo) error {
	if !e.NeedCommit() {
		return fmt.Errorf("can't commit an entity with no pending operation")
	}

	if err := e.Validate(); err != nil {
		return errors.Wrapf(err, "can't commit a %s with invalid data", e.Definition.typename)
	}

	// increment the various clocks for this new operationPack
	packTime, err := e.packClock.Increment()
	if err != nil {
		return err
	}
	editTime, err := repo.Increment(fmt.Sprintf(editClockPattern, e.namespace))
	if err != nil {
		return err
	}
	var creationTime lamport.Time
	if e.lastCommit == "" {
		creationTime, err = repo.Increment(fmt.Sprintf(creationClockPattern, e.namespace))
		if err != nil {
			return err
		}
	}

	opp := &operationPack{
		Operations: e.staging,
		CreateTime: creationTime,
		EditTime:   editTime,
		PackTime:   packTime,
	}

	treeHash, err := opp.write(e.Definition, repo)
	if err != nil {
		return err
	}

	// Write a Git commit referencing the tree, with the previous commit as parent
	var commitHash repository.Hash
	if e.lastCommit != "" {
		commitHash, err = repo.StoreCommitWithParent(treeHash, e.lastCommit)
	} else {
		commitHash, err = repo.StoreCommit(treeHash)
	}
	if err != nil {
		return err
	}

	e.lastCommit = commitHash
	e.ops = append(e.ops, e.staging...)
	e.staging = nil

	// Create or update the Git reference for this entity
	// When pushing later, the remote will ensure that this ref update
	// is fast-forward, that is no data has been overwritten.
	ref := fmt.Sprintf(refsPattern, e.namespace, e.Id().String())
	return repo.UpdateRef(ref, commitHash)
}
