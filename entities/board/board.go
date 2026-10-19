// Package board contains the board data model and low-level related functions
package board

import (
	"fmt"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/repository"
)

var _ dag.Mutable[*Snapshot, Operation] = &Board{}
var _ dag.Tracked[*Snapshot, Operation] = &Board{}
var _ entity.Interface = &Board{}

// 1: original format
const formatVersion = 1

const Typename = "board"
const Namespace = "boards"

var def = dag.Definition{
	Typename:             Typename,
	Namespace:            Namespace,
	OperationUnmarshaler: operationUnmarshaler,
	FormatVersion:        formatVersion,
}

// Mutable is what the functions creating operations (SetTitle, AddItemDraft, ...)
// need from a board, see dag.Mutable.
type Mutable = dag.Mutable[*Snapshot, Operation]

// Board holds the data of a project board.
type Board struct {
	*dag.Entity
}

// NewBoard create a new Board
func NewBoard() *Board {
	return wrapper(dag.New(def))
}

func wrapper(e *dag.Entity) *Board {
	return &Board{Entity: e}
}

func simpleResolvers(repo repository.Repo) entity.Resolvers {
	return entity.Resolvers{
		&identity.Identity{}: identity.NewSimpleResolver(repo),
	}
}

// Read will read a board from a repository
func Read(repo repository.Repo, id entity.Id) (*Board, error) {
	return ReadWithResolver(repo, simpleResolvers(repo), id)
}

// ReadWithResolver will read a board from its Id, with custom resolvers
func ReadWithResolver(repo repository.Repo, resolvers entity.Resolvers, id entity.Id) (*Board, error) {
	return dag.Read(def, wrapper, repo, resolvers, id)
}

// ReadAll read and parse all local boards
func ReadAll(repo repository.Repo) <-chan entity.StreamedEntity[*Board] {
	return dag.ReadAll(def, wrapper, repo, simpleResolvers(repo))
}

// ReadAllWithResolver read and parse all local boards
func ReadAllWithResolver(repo repository.Repo, resolvers entity.Resolvers) <-chan entity.StreamedEntity[*Board] {
	return dag.ReadAll(def, wrapper, repo, resolvers)
}

// ListLocalIds list all the available local board ids
func ListLocalIds(repo repository.Repo) ([]entity.Id, error) {
	return dag.ListLocalIds(def, repo)
}

// EnsureClocks makes sure that the clocks ordering boards are usable, and rebuilds
// them from the boards if not. See dag.EnsureClocks.
func EnsureClocks(repo repository.Repo) error {
	return dag.EnsureClocks(def, repo)
}

// Validate check if the Board data is valid
func (board *Board) Validate() error {
	if err := board.Entity.Validate(); err != nil {
		return err
	}

	// The very first Op should be a CreateOp
	firstOp := board.FirstOp()
	if firstOp == nil || firstOp.Type() != CreateOp {
		return fmt.Errorf("first operation should be a Create op")
	}

	// Check that there is no more CreateOp op
	for i, op := range board.Entity.Operations() {
		if i == 0 {
			continue
		}
		if op.Type() == CreateOp {
			return fmt.Errorf("only one Create op allowed")
		}
	}

	return nil
}

// Append add a new Operation to the Board
func (board *Board) Append(op Operation) {
	board.Entity.Append(op)
}

// CommitOperations writes the given operations, see dag.Entity.CommitOperations
func (board *Board) CommitOperations(repo repository.Repo, ops []Operation) error {
	converted := make([]dag.Operation, len(ops))
	for i, op := range ops {
		converted[i] = op
	}
	return board.Entity.CommitOperations(repo, converted)
}

// Operations return the ordered operations
func (board *Board) Operations() []Operation {
	source := board.Entity.Operations()
	result := make([]Operation, len(source))
	for i, op := range source {
		result[i] = op.(Operation)
	}
	return result
}

// Compile a board in an easily usable snapshot
func (board *Board) Compile() *Snapshot {
	snap := &Snapshot{
		id: board.Id(),
	}

	for _, op := range board.Operations() {
		op.Apply(snap)
		snap.Operations = append(snap.Operations, op)
	}

	return snap
}

// FirstOp lookup for the very first operation of the board.
// For a valid Board, this operation should be a CreateOp
func (board *Board) FirstOp() Operation {
	if fo := board.Entity.FirstOp(); fo != nil {
		return fo.(Operation)
	}
	return nil
}

// LastOp lookup for the very last operation of the board.
// For a valid Board, should never be nil
func (board *Board) LastOp() Operation {
	if lo := board.Entity.LastOp(); lo != nil {
		return lo.(Operation)
	}
	return nil
}
