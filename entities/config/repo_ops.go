package config

import (
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/repository"
)

// The operations of the repo section, see repo.go.

// Operation types of the repo section
const (
	RepoSetDescriptionOp dag.OperationType = repoSection*sectionSize + 1 + iota
)

var repoOperations = map[dag.OperationType]func() Operation{
	RepoSetDescriptionOp: func() Operation { return &RepoSetDescriptionOperation{} },
}

var _ Operation = &RepoSetDescriptionOperation{}

// RepoSetDescriptionOperation will change the description of the repository.
// An empty description clears it.
type RepoSetDescriptionOperation struct {
	setValue[string]
}

func (op *RepoSetDescriptionOperation) Id() entity.Id {
	return dag.IdOperation(op, &op.OpBase)
}

func (op *RepoSetDescriptionOperation) Apply(snapshot *Snapshot) {
	op.apply(&snapshot.Repo.description)
}

func (op *RepoSetDescriptionOperation) Validate() error {
	return op.validate(op, RepoSetDescriptionOp, validateRepoDescription)
}

func NewRepoSetDescriptionOp(repo repository.RepoClock, author identity.Interface, unixTime int64, description string) (*RepoSetDescriptionOperation, error) {
	kind, err := newSetValue(repo, RepoSetDescriptionOp, author, unixTime, description)
	if err != nil {
		return nil, err
	}
	return &RepoSetDescriptionOperation{setValue: kind}, nil
}

// RepoSetDescription is a convenience function to change the description of the repository
func RepoSetDescription(c Mutable, repo repository.RepoClock, author identity.Interface, unixTime int64, description string) (*RepoSetDescriptionOperation, error) {
	op, err := NewRepoSetDescriptionOp(repo, author, unixTime, description)
	if err != nil {
		return nil, err
	}
	if err := op.Validate(); err != nil {
		return nil, err
	}
	c.Append(op)
	return op, nil
}
