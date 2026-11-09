package config

import (
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
	"unicode/utf8"
)

// Every setting has one of the shapes below, implemented once here. A shape holds
// the state of its setting in the Snapshot, changed by the kinds of operations in
// kinds.go.
//
//   - a register holds one value, of a scalar type
//   - a map holds entries by key, each holding named fields, see entries
//   - a set is a map whose entries have no field, see set
//   - an ordered list is a map whose entries are sorted by position, see list

// scalar is the type of a value: a string, an integer or a boolean.
type scalar interface {
	~string | ~int64 | ~bool
}

// maxInteger bounds the integer values, so that every JSON implementation reads
// them exactly.
const maxInteger = 1<<53 - 1

// validateScalar checks what any value must satisfy, before the checks of its
// setting.
func validateScalar[V scalar](value V) error {
	v := reflect.ValueOf(value)
	if v.Kind() == reflect.Int64 && (v.Int() > maxInteger || v.Int() < -maxInteger) {
		return fmt.Errorf("integer %d out of range", v.Int())
	}
	return nil
}

// validateKey checks a key: non-empty, then with the check of its setting.
func validateKey[K ~string](key K, check func(K) error) error {
	if key == "" {
		return fmt.Errorf("empty key")
	}
	return check(key)
}

// validateLength checks that a string has at most limit characters, counted as
// Unicode code points.
func validateLength(s string, limit int) error {
	if n := utf8.RuneCountInString(s); n > limit {
		return fmt.Errorf("%d characters, more than %d", n, limit)
	}
	return nil
}

// Field is a field of a map entry. A setEntry operation lists the fields it
// changes: a Field there is absent, its zero value, to keep the field as it is,
// cleared for the field to read as its default, or set to a value. In the
// Snapshot, a Field is absent or set.
//
// A struct holding Fields tags each with omitzero: an absent Field is left out of
// the JSON, and a cleared one is null.
type Field[V scalar] struct {
	state fieldState
	value V
}

type fieldState uint8

const (
	fieldAbsent fieldState = iota
	fieldCleared
	fieldSet
)

// SetField returns a Field set to a value.
func SetField[V scalar](value V) Field[V] {
	return Field[V]{state: fieldSet, value: value}
}

// ClearField returns a cleared Field.
func ClearField[V scalar]() Field[V] {
	return Field[V]{state: fieldCleared}
}

// Get returns the value of a Field, false if it is not set.
func (f Field[V]) Get() (V, bool) {
	return f.value, f.state == fieldSet
}

// patch changes the Field of an entry as the same Field of a setEntry operation
// says.
func (f *Field[V]) patch(change Field[V]) {
	switch change.state {
	case fieldCleared:
		*f = Field[V]{}
	case fieldSet:
		*f = change
	}
}

// validate checks the value of a set Field, with the check of its setting.
func (f Field[V]) validate(check func(V) error) error {
	if f.state != fieldSet {
		return nil
	}
	if err := validateScalar(f.value); err != nil {
		return err
	}
	return check(f.value)
}

func (f Field[V]) MarshalJSON() ([]byte, error) {
	switch f.state {
	case fieldCleared:
		return []byte("null"), nil
	case fieldSet:
		return json.Marshal(f.value)
	default:
		return nil, fmt.Errorf("an absent field is not written, tag it omitzero")
	}
}

func (f *Field[V]) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*f = ClearField[V]()
		return nil
	}
	var value V
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	*f = SetField(value)
	return nil
}

// fields is the constraint of the fields of a map entry: a struct of Field, each
// tagged omitzero. In a setEntry operation, it holds the changes to the fields of
// the entry, applied by patch.
type fields[F any] interface {
	patch(change F) F
	Validate() error
}

// entries is the state of a map: the fields of each entry, by key. A key absent
// from it has no entry.
type entries[K ~string, F fields[F]] map[K]F

// has tells whether a key has an entry.
func (e entries[K, F]) has(key K) bool {
	_, ok := e[key]
	return ok
}

// keys returns the keys with an entry, sorted.
func (e entries[K, F]) keys() []K {
	return slices.Sorted(maps.Keys(e))
}

// noFields is the fields of the entries of a set: there are none.
type noFields struct{}

func (noFields) patch(noFields) noFields {
	return noFields{}
}

func (noFields) Validate() error {
	return nil
}

// set is the state of a set: a map whose entries have no field. Its operations are
// the ones of a map, setEntry adding a key and removeEntry removing it.
type set[K ~string] = entries[K, noFields]

// listFields is the constraint of the fields of the entries of an ordered list: on
// top of the others, a position, a Field[string] written "position" and checked
// with fracindex.Validate, returned by position.
type listFields[F any] interface {
	fields[F]
	position() (string, bool)
}

// list is the state of an ordered list: a map whose entries have a position, read
// with ordered. Its operations are the ones of a map, an entry moving when
// setEntry changes its position, to one from fracindex.Between.
type list[K ~string, F listFields[F]] = entries[K, F]

// ordered returns the keys of an ordered list in order: by position, compared byte
// by byte, then by key for the same position. The keys without a position come
// last, by key.
func ordered[K ~string, F listFields[F]](l list[K, F]) []K {
	keys := l.keys()
	slices.SortStableFunc(keys, func(a, b K) int {
		positionA, okA := l[a].position()
		positionB, okB := l[b].position()
		switch {
		case okA && okB:
			return strings.Compare(positionA, positionB)
		case okA:
			return -1
		case okB:
			return 1
		default:
			return 0
		}
	})
	return keys
}
