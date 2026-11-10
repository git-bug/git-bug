package config

import (
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/repository"
)

// The operations of a setting are types of their own, each embedding the kind of
// operation it is, with the types of the setting, and adding no field: the JSON of
// an operation is the one of its kind.
//
//   - setValue replaces the value of a register
//   - setEntry adds the entry of a key if it is absent, then changes the fields it
//     lists
//   - removeEntry removes the entry of a key, with all its fields
//
// Each kind applies to the state of its setting in the Snapshot, of a shape from
// shapes.go: a value for setValue, entries for setEntry and removeEntry. As
// operations are applied in order, the last change wins, separately for the value
// of a register, whether each entry of a map is present, and each field of each
// entry.

// setValue is the kind of the operations replacing the value of a register.
type setValue[V scalar] struct {
	opBase
	Value V `json:"value,omitzero"`
}

func newSetValue[V scalar](repo repository.RepoClock, opType dag.OperationType, author identity.Interface, unixTime int64, value V) (setValue[V], error) {
	base, err := newOpBase(repo, opType, author, unixTime)
	if err != nil {
		return setValue[V]{}, err
	}
	return setValue[V]{opBase: base, Value: value}, nil
}

// validate checks the operation op embedding this kind, of type opType, and its
// value with checkValue.
func (s *setValue[V]) validate(op Operation, opType dag.OperationType, checkValue func(V) error) error {
	if err := s.opBase.Validate(op, opType); err != nil {
		return err
	}
	if err := validateScalar(s.Value); err != nil {
		return err
	}
	return checkValue(s.Value)
}

// apply replaces the value of the register.
func (s *setValue[V]) apply(register *V) {
	*register = s.Value
}

// setEntry is the kind of the operations adding the entry of a key if it is
// absent, then changing the fields they list.
type setEntry[K ~string, F fields[F]] struct {
	opBase
	Key    K `json:"key"`
	Fields F `json:"fields,omitzero"`
}

func newSetEntry[K ~string, F fields[F]](repo repository.RepoClock, opType dag.OperationType, author identity.Interface, unixTime int64, key K, change F) (setEntry[K, F], error) {
	base, err := newOpBase(repo, opType, author, unixTime)
	if err != nil {
		return setEntry[K, F]{}, err
	}
	return setEntry[K, F]{opBase: base, Key: key, Fields: change}, nil
}

// validate checks the operation op embedding this kind, of type opType, its key
// with checkKey, and its fields.
func (s *setEntry[K, F]) validate(op Operation, opType dag.OperationType, checkKey func(K) error) error {
	if err := s.opBase.Validate(op, opType); err != nil {
		return err
	}
	if err := validateKey(s.Key, checkKey); err != nil {
		return err
	}
	return s.Fields.Validate()
}

// apply adds the entry of the key if it is absent, then changes its fields.
func (s *setEntry[K, F]) apply(register *entries[K, F]) {
	if *register == nil {
		*register = make(entries[K, F])
	}
	(*register)[s.Key] = (*register)[s.Key].patch(s.Fields)
}

// removeEntry is the kind of the operations removing the entry of a key, with all
// its fields. F is the type of the fields of the entries.
type removeEntry[K ~string, F fields[F]] struct {
	opBase
	Key K `json:"key"`
}

func newRemoveEntry[K ~string, F fields[F]](repo repository.RepoClock, opType dag.OperationType, author identity.Interface, unixTime int64, key K) (removeEntry[K, F], error) {
	base, err := newOpBase(repo, opType, author, unixTime)
	if err != nil {
		return removeEntry[K, F]{}, err
	}
	return removeEntry[K, F]{opBase: base, Key: key}, nil
}

// validate checks the operation op embedding this kind, of type opType, and its
// key with checkKey.
func (s *removeEntry[K, F]) validate(op Operation, opType dag.OperationType, checkKey func(K) error) error {
	if err := s.opBase.Validate(op, opType); err != nil {
		return err
	}
	return validateKey(s.Key, checkKey)
}

// apply removes the entry of the key, with all its fields.
func (s *removeEntry[K, F]) apply(register *entries[K, F]) {
	delete(*register, s.Key)
}
