package config

import (
	"fmt"

	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/repository"
)

// The operations of the labels section, see labels.go.

// Operation types of the labels section
const (
	LabelsSetOp dag.OperationType = labelsSection*sectionSize + 1 + iota
	LabelsRemoveOp
	LabelsAllowOp
	LabelsDisallowOp
)

var labelsOperations = map[dag.OperationType]func() Operation{
	LabelsSetOp:      func() Operation { return &LabelsSetOperation{} },
	LabelsRemoveOp:   func() Operation { return &LabelsRemoveOperation{} },
	LabelsAllowOp:    func() Operation { return &LabelsAllowOperation{} },
	LabelsDisallowOp: func() Operation { return &LabelsDisallowOperation{} },
}

var _ Operation = &LabelsSetOperation{}

// LabelsSetOperation will define a label if it has no definition, then change the
// fields of its definition that it lists.
type LabelsSetOperation struct {
	setEntry[common.Label, LabelsDefinition]
}

func (op *LabelsSetOperation) Id() entity.Id {
	return dag.IdOperation(op, &op.OpBase)
}

func (op *LabelsSetOperation) Apply(snapshot *Snapshot) {
	op.apply(&snapshot.Labels.definitions)
}

func (op *LabelsSetOperation) Validate() error {
	return op.validate(op, LabelsSetOp, validateLabelName)
}

func NewLabelsSetOp(repo repository.RepoClock, author identity.Interface, unixTime int64, name common.Label, change LabelsDefinition) (*LabelsSetOperation, error) {
	kind, err := newSetEntry(repo, LabelsSetOp, author, unixTime, name, change)
	if err != nil {
		return nil, err
	}
	return &LabelsSetOperation{setEntry: kind}, nil
}

// LabelsSet is a convenience function to define a label, or change its definition
func LabelsSet(c Mutable, repo repository.RepoClock, author identity.Interface, unixTime int64, name common.Label, change LabelsDefinition) (*LabelsSetOperation, error) {
	op, err := NewLabelsSetOp(repo, author, unixTime, name, change)
	if err != nil {
		return nil, err
	}
	if err := op.Validate(); err != nil {
		return nil, err
	}
	c.Append(op)
	return op, nil
}

var _ Operation = &LabelsRemoveOperation{}

// LabelsRemoveOperation will remove the definition of a label, with all its fields.
// The label takes the color derived from its name, and whether it is allowed
// doesn't change.
type LabelsRemoveOperation struct {
	removeEntry[common.Label, LabelsDefinition]
}

func (op *LabelsRemoveOperation) Id() entity.Id {
	return dag.IdOperation(op, &op.OpBase)
}

func (op *LabelsRemoveOperation) Apply(snapshot *Snapshot) {
	op.apply(&snapshot.Labels.definitions)
}

func (op *LabelsRemoveOperation) Validate() error {
	return op.validate(op, LabelsRemoveOp, validateLabelName)
}

func NewLabelsRemoveOp(repo repository.RepoClock, author identity.Interface, unixTime int64, name common.Label) (*LabelsRemoveOperation, error) {
	kind, err := newRemoveEntry[common.Label, LabelsDefinition](repo, LabelsRemoveOp, author, unixTime, name)
	if err != nil {
		return nil, err
	}
	return &LabelsRemoveOperation{removeEntry: kind}, nil
}

// LabelsRemove is a convenience function to remove the definition of a label
func LabelsRemove(c Mutable, repo repository.RepoClock, author identity.Interface, unixTime int64, name common.Label) (*LabelsRemoveOperation, error) {
	op, err := NewLabelsRemoveOp(repo, author, unixTime, name)
	if err != nil {
		return nil, err
	}
	if err := op.Validate(); err != nil {
		return nil, err
	}
	c.Append(op)
	return op, nil
}

var _ Operation = &LabelsAllowOperation{}

// LabelsAllowOperation will allow a label. Once a label is allowed, only the allowed
// labels may be used.
type LabelsAllowOperation struct {
	setEntry[common.Label, noFields]
}

func (op *LabelsAllowOperation) Id() entity.Id {
	return dag.IdOperation(op, &op.OpBase)
}

func (op *LabelsAllowOperation) Apply(snapshot *Snapshot) {
	op.apply(&snapshot.Labels.allowed)
}

func (op *LabelsAllowOperation) Validate() error {
	return op.validate(op, LabelsAllowOp, validateLabelName)
}

func NewLabelsAllowOp(repo repository.RepoClock, author identity.Interface, unixTime int64, name common.Label) (*LabelsAllowOperation, error) {
	kind, err := newSetEntry(repo, LabelsAllowOp, author, unixTime, name, noFields{})
	if err != nil {
		return nil, err
	}
	return &LabelsAllowOperation{setEntry: kind}, nil
}

// LabelsAllow is a convenience function to allow a label
func LabelsAllow(c Mutable, repo repository.RepoClock, author identity.Interface, unixTime int64, name common.Label) (*LabelsAllowOperation, error) {
	op, err := NewLabelsAllowOp(repo, author, unixTime, name)
	if err != nil {
		return nil, err
	}
	if err := op.Validate(); err != nil {
		return nil, err
	}
	c.Append(op)
	return op, nil
}

var _ Operation = &LabelsDisallowOperation{}

// LabelsDisallowOperation will disallow a label. Disallowing the last allowed label
// makes any label allowed again. The definition of the label stays, for the bugs
// that have it.
type LabelsDisallowOperation struct {
	removeEntry[common.Label, noFields]
}

func (op *LabelsDisallowOperation) Id() entity.Id {
	return dag.IdOperation(op, &op.OpBase)
}

func (op *LabelsDisallowOperation) Apply(snapshot *Snapshot) {
	op.apply(&snapshot.Labels.allowed)
}

func (op *LabelsDisallowOperation) Validate() error {
	return op.validate(op, LabelsDisallowOp, validateLabelName)
}

func NewLabelsDisallowOp(repo repository.RepoClock, author identity.Interface, unixTime int64, name common.Label) (*LabelsDisallowOperation, error) {
	kind, err := newRemoveEntry[common.Label, noFields](repo, LabelsDisallowOp, author, unixTime, name)
	if err != nil {
		return nil, err
	}
	return &LabelsDisallowOperation{removeEntry: kind}, nil
}

// LabelsDisallow is a convenience function to disallow a label
func LabelsDisallow(c Mutable, repo repository.RepoClock, author identity.Interface, unixTime int64, name common.Label) (*LabelsDisallowOperation, error) {
	op, err := NewLabelsDisallowOp(repo, author, unixTime, name)
	if err != nil {
		return nil, err
	}
	if err := op.Validate(); err != nil {
		return nil, err
	}
	c.Append(op)
	return op, nil
}

// LabelsRestrict is a convenience function to restrict the labels to the defined
// ones, by allowing each of them. Committed together, these operations are written
// in one pack, and so are ordered together against concurrent changes.
func LabelsRestrict(c Mutable, repo repository.RepoClock, author identity.Interface, unixTime int64) ([]*LabelsAllowOperation, error) {
	names := c.Compile().Labels.Names()
	if len(names) == 0 {
		return nil, fmt.Errorf("no label is defined")
	}

	ops := make([]*LabelsAllowOperation, len(names))
	for i, name := range names {
		op, err := NewLabelsAllowOp(repo, author, unixTime, name)
		if err != nil {
			return nil, err
		}
		if err := op.Validate(); err != nil {
			return nil, err
		}
		ops[i] = op
	}
	for _, op := range ops {
		c.Append(op)
	}
	return ops, nil
}

// LabelsLiftRestriction is a convenience function to make any label allowed again, by
// disallowing every allowed label. Committed together, these operations are written
// in one pack, and so are ordered together against concurrent changes.
func LabelsLiftRestriction(c Mutable, repo repository.RepoClock, author identity.Interface, unixTime int64) ([]*LabelsDisallowOperation, error) {
	names := c.Compile().Labels.AllowedNames()
	if len(names) == 0 {
		return nil, fmt.Errorf("labels are not restricted")
	}

	ops := make([]*LabelsDisallowOperation, len(names))
	for i, name := range names {
		op, err := NewLabelsDisallowOp(repo, author, unixTime, name)
		if err != nil {
			return nil, err
		}
		if err := op.Validate(); err != nil {
			return nil, err
		}
		ops[i] = op
	}
	for _, op := range ops {
		c.Append(op)
	}
	return ops, nil
}
