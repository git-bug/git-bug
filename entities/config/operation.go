package config

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/repository"
	"github.com/git-bug/git-bug/util/lamport"
)

// The settings of the config are grouped in sections, a feature area each. A
// section owns its part of the Snapshot, with its rules, defaults and queries, in
// <section>.go, and its operations in <section>_ops.go. The Snapshot gathers the
// sections, and the shapes and kinds of operations they are built from are in
// shapes.go and kinds.go.
//
// The operation types are numbered by section, sectionSize per section: the
// operations of the config itself first, then those of each section from its
// number * sectionSize + 1. A section numbers its operations without touching the
// others, and the type of an operation tells its section, even for a type this
// client doesn't know. Numbers never change once released.
const sectionSize = 10000

// Numbers of the sections
const (
	repoSection   = 1
	labelsSection = 2
)

// Operation types of the config itself
const (
	_ dag.OperationType = iota
	CreateOp
)

// sectionOperations makes an empty operation of each type of a section, by section
// number, to decode an operation into.
var sectionOperations = map[int]map[dag.OperationType]func() Operation{
	repoSection:   repoOperations,
	labelsSection: labelsOperations,
}

// Operation define the interface to fulfill for an edit operation of a Config
type Operation = dag.OperationWithApply[*Snapshot]

// make sure that package external operations do conform to our interface
var _ Operation = &dag.UnknownOperation[*Snapshot]{}

// opBase is the common part of the operations of this package. On top of
// dag.OpBase, it holds the time of every Lamport clock of the repository when the
// operation was made, which places it relative to the operations of every other
// entity: the operation applies to those of a later time, see Snapshot.At.
type opBase struct {
	dag.OpBase
	Times map[string]lamport.Time `json:"times"`
}

// Validate checks the common part of the operation op, of type opType: the one of
// dag.OpBase, and the times, which every operation records.
func (base *opBase) Validate(op Operation, opType dag.OperationType) error {
	if err := base.OpBase.Validate(op, opType); err != nil {
		return err
	}
	if base.Times == nil {
		return fmt.Errorf("times not set")
	}
	return nil
}

// timed is an operation of this package, recording times, see opBase.
type timed interface {
	times() map[string]lamport.Time
}

func (base *opBase) times() map[string]lamport.Time {
	return base.Times
}

// opTimes returns the times recorded by an operation, see opBase. Those of an
// operation this client doesn't know are read from its JSON, and are none if they
// can't be read.
func opTimes(op dag.Operation) map[string]lamport.Time {
	switch op := op.(type) {
	case timed:
		return op.times()
	case *dag.UnknownOperation[*Snapshot]:
		var raw struct {
			Times map[string]lamport.Time `json:"times"`
		}
		_ = json.Unmarshal(op.RawJSON, &raw)
		return raw.Times
	default:
		return nil
	}
}

func newOpBase(repo repository.RepoClock, opType dag.OperationType, author identity.Interface, unixTime int64) (opBase, error) {
	clocks, err := repo.AllClocks()
	if err != nil {
		return opBase{}, err
	}

	times := make(map[string]lamport.Time, len(clocks))
	for name, clock := range clocks {
		time, err := clock.Time()
		if err != nil {
			return opBase{}, err
		}
		times[name] = time
	}

	return opBase{
		OpBase: dag.NewOpBase(opType, author, unixTime),
		Times:  times,
	}, nil
}

func operationUnmarshaler(_ context.Context, raw json.RawMessage, resolvers entity.Resolvers) (dag.Operation, error) {
	var t struct {
		OperationType dag.OperationType `json:"type"`
	}

	if err := json.Unmarshal(raw, &t); err != nil {
		return nil, err
	}

	var op dag.Operation

	switch t.OperationType {
	case CreateOp:
		op = &CreateOperation{}
	default:
		newOp, ok := sectionOperations[int(t.OperationType)/sectionSize][t.OperationType]
		if !ok {
			return dag.NewUnknownOp[*Snapshot](raw)
		}
		op = newOp()
	}

	err := json.Unmarshal(raw, &op)
	if err != nil {
		return nil, err
	}

	return op, nil
}

var _ Operation = &CreateOperation{}

// CreateOperation define the initial creation of a config. It sets nothing: every
// setting starts unset.
type CreateOperation struct {
	opBase
}

func (op *CreateOperation) Id() entity.Id {
	return dag.IdOperation(op, &op.OpBase)
}

func (op *CreateOperation) Apply(_ *Snapshot) {}

func (op *CreateOperation) Validate() error {
	return op.opBase.Validate(op, CreateOp)
}

func NewCreateOp(repo repository.RepoClock, author identity.Interface, unixTime int64) (*CreateOperation, error) {
	base, err := newOpBase(repo, CreateOp, author, unixTime)
	if err != nil {
		return nil, err
	}
	return &CreateOperation{opBase: base}, nil
}

// Create is a convenience function to create a config
func Create(repo repository.RepoClock, author identity.Interface, unixTime int64) (*Config, *CreateOperation, error) {
	c := NewConfig()
	op, err := NewCreateOp(repo, author, unixTime)
	if err != nil {
		return nil, nil, err
	}
	if err := op.Validate(); err != nil {
		return nil, op, err
	}
	c.Append(op)
	return c, op, nil
}
