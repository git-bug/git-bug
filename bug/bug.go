// Package bug contains the bug data model and low-level related functions
package bug

import (
	"fmt"

	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/entity/dag"
	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/repository"
)

var _ Interface = &Bug{}
var _ entity.Interface = &Bug{}

// 1: original format
// 2: no more legacy identities
// 3: Ids are generated from the create operation serialized data instead of from the first git commit
// 4: with DAG entity framework
const formatVersion = 4

var def = dag.Definition{
	Typename:             "bug",
	Namespace:            "bugs",
	OperationUnmarshaler: operationUnmarshaller,
	FormatVersion:        formatVersion,
}

var ClockLoader = dag.ClockLoader(def)

// Bug holds the data of a bug thread, organized in a way close to
// how it will be persisted inside Git. This is the data structure
// used to merge two different version of the same Bug.
type Bug struct {
	*dag.Entity
}

// NewBug create a new Bug
func NewBug() *Bug {
	return &Bug{
		Entity: dag.New(def),
	}
}

// Read will read a bug from a repository
func Read(repo repository.ClockedRepo, id entity.Id) (*Bug, error) {
	return ReadWithResolver(repo, identity.NewSimpleResolver(repo), id)
}

// ReadWithResolver will read a bug from its Id, with a custom identity.Resolver
func ReadWithResolver(repo repository.ClockedRepo, identityResolver identity.Resolver, id entity.Id) (*Bug, error) {
	e, err := dag.Read(def, repo, identityResolver, id)
	if err != nil {
		return nil, err
	}
	return &Bug{Entity: e}, nil
}

type StreamedBug struct {
	Bug *Bug
	Err error
}

// ReadAll read and parse all local bugs
func ReadAll(repo repository.ClockedRepo) <-chan StreamedBug {
	return readAll(repo, identity.NewSimpleResolver(repo))
}

// ReadAllWithResolver read and parse all local bugs
func ReadAllWithResolver(repo repository.ClockedRepo, identityResolver identity.Resolver) <-chan StreamedBug {
	return readAll(repo, identityResolver)
}

// Read and parse all available bug with a given ref prefix
func readAll(repo repository.ClockedRepo, identityResolver identity.Resolver) <-chan StreamedBug {
	out := make(chan StreamedBug)

	go func() {
		defer close(out)

		for streamedEntity := range dag.ReadAll(def, repo, identityResolver) {
			if streamedEntity.Err != nil {
				out <- StreamedBug{
					Err: streamedEntity.Err,
				}
			} else {
				out <- StreamedBug{
					Bug: &Bug{Entity: streamedEntity.Entity},
				}
			}
		}
	}()

	return out
}

// ListLocalIds list all the available local bug ids
func ListLocalIds(repo repository.Repo) ([]entity.Id, error) {
	return dag.ListLocalIds(def, repo)
}

// Validate check if the Bug data is valid
func (bug *Bug) Validate() error {
	if err := bug.Entity.Validate(); err != nil {
		return err
	}

	// The very first Op should be a CreateOp
	firstOp := bug.FirstOp()
	if firstOp == nil || firstOp.Type() != CreateOp {
		return fmt.Errorf("first operation should be a Create op")
	}

	// Check that there is no more CreateOp op
	for i, op := range bug.Operations() {
		if i == 0 {
			continue
		}
		if op.Type() == CreateOp {
			return fmt.Errorf("only one Create op allowed")
		}
	}

	return nil
}

// Append add a new Operation to the Bug
func (bug *Bug) Append(op Operation) {
	bug.Entity.Append(op)
}

// Operations return the ordered operations
func (bug *Bug) Operations() []Operation {
	source := bug.Entity.Operations()
	result := make([]Operation, len(source))
	for i, op := range source {
		result[i] = op.(Operation)
	}
	return result
}

// Compile a bug in a easily usable snapshot
func (bug *Bug) Compile() *Snapshot {
	snap := &Snapshot{
		id:     bug.Id(),
		Status: OpenStatus,
	}

	for _, op := range bug.Operations() {
		op.Apply(snap)
		snap.Operations = append(snap.Operations, op)
	}

	return snap
}

// FirstOp lookup for the very first operation of the bug.
// For a valid Bug, this operation should be a CreateOp
func (bug *Bug) FirstOp() Operation {
	if fo := bug.Entity.FirstOp(); fo != nil {
		return fo.(Operation)
	}
	return nil
}

// LastOp lookup for the very last operation of the bug.
// For a valid Bug, should never be nil
func (bug *Bug) LastOp() Operation {
	if lo := bug.Entity.LastOp(); lo != nil {
		return lo.(Operation)
	}
	return nil
}
