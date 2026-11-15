// Package jsondoc reads JSON documents, the ops blob of an entity and the
// version blob of an identity, by the rules of values.md §2 to §4 in the spec.
// JSON libraries disagree on invalid UTF-8, repeated names, the case of names,
// large integers or deep nesting, so every client applies these rules to reach
// the same verdict on a document.
//
// A reader calls Check on the whole document, then Unmarshal for each part it
// decodes. A type with its own UnmarshalJSON method decodes its members with
// Unmarshal too, for the rules to apply within it.
package jsondoc

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
)

// maxDepth is how deep objects and arrays nest at most in a document, the
// outermost value counting as 1.
const maxDepth = 64

// maxInt is the largest integer, and -maxInt the smallest: every JSON
// implementation reads them exactly.
const maxInt = 1<<53 - 1

// Check returns an error if data is not a JSON document: valid UTF-8, without an
// escaped surrogate other than a pair, without a byte order mark, without an
// object holding two members of the same name, nesting at most maxDepth deep.
func Check(data []byte) error {
	// the decoder rejects invalid UTF-8, lone surrogates, a byte order mark and
	// repeated names by default
	dec := jsontext.NewDecoder(bytes.NewReader(data))
	for {
		if _, err := dec.ReadToken(); err != nil {
			return err
		}
		depth := dec.StackDepth()
		if depth > maxDepth {
			return fmt.Errorf("nested more than %d deep at %s", maxDepth, dec.StackPointer())
		}
		if depth == 0 {
			break
		}
	}
	if _, err := dec.ReadToken(); err != io.EOF {
		return errors.New("data after the document")
	}
	return nil
}

// Unmarshal decodes data, a part of a document that Check accepted, into v.
// Names match exactly, case included. A member that is null sets its Go value to
// zero, as an absent member leaves it in a new value: v is new, or a member that
// is null isn't absent. A value that is null in a map is the empty value of its
// type. An integer has neither a fraction nor an exponent, and lies between
// -maxInt and maxInt. An array holding null is an error.
func Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v, unmarshalers)
}

var unmarshalers = json.WithUnmarshalers(json.UnmarshalFromFunc(unmarshalValue))

var (
	legacyUnmarshaler = reflect.TypeFor[json.Unmarshaler]()
	unmarshalerFrom   = reflect.TypeFor[json.UnmarshalerFrom]()
)

// unmarshalValue decodes the integers and the slices, for their rules. It leaves
// any other value, and a value whose type has its own method, to the default
// decoding, by returning errors.ErrUnsupported.
func unmarshalValue(dec *jsontext.Decoder, v any) error {
	ptr := reflect.ValueOf(v)
	if ptr.Type().Implements(legacyUnmarshaler) || ptr.Type().Implements(unmarshalerFrom) {
		return errors.ErrUnsupported
	}
	switch kind := dec.PeekKind(); {
	case kind == '0' && isInteger(ptr.Elem().Kind()):
		return unmarshalInteger(dec, v)
	case kind == '[' && ptr.Elem().Kind() == reflect.Slice:
		return unmarshalSlice(dec, ptr.Elem())
	}
	return errors.ErrUnsupported
}

func isInteger(kind reflect.Kind) bool {
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return true
	}
	return false
}

// unmarshalInteger decodes a number into v, an integer.
func unmarshalInteger(dec *jsontext.Decoder, v any) error {
	value, err := dec.ReadValue()
	if err != nil {
		return err
	}
	// a JSON number is a valid ParseInt input unless it has a fraction or an
	// exponent; once in range it fits v, or the default decoding of v rejects it
	n, err := strconv.ParseInt(string(value), 10, 64)
	switch {
	case errors.Is(err, strconv.ErrSyntax):
		return semanticError(reflect.TypeOf(v).Elem(), value, "not an integer")
	case err != nil || n > maxInt || n < -maxInt:
		return semanticError(reflect.TypeOf(v).Elem(), value, "integer out of range")
	}
	// avoid doing reflection on our own
	return json.Unmarshal(value, v)
}

// semanticError reports that value, when known, can't decode into a Go value of
// type t by the rules. json fills in where in the document, and the error names
// t rather than the any of the unmarshaler.
func semanticError(t reflect.Type, value jsontext.Value, reason string) error {
	return &json.SemanticError{
		GoType:    t,
		JSONKind:  value.Kind(),
		JSONValue: value,
		Err:       errors.New(reason),
	}
}

// unmarshalSlice decodes an array into slice, each element as Unmarshal does.
func unmarshalSlice(dec *jsontext.Decoder, slice reflect.Value) error {
	if _, err := dec.ReadToken(); err != nil {
		return err
	}
	result := reflect.MakeSlice(slice.Type(), 0, 0)
	for dec.PeekKind() != ']' {
		if dec.PeekKind() == 'n' {
			return semanticError(slice.Type(), nil, "null in an array")
		}
		elem := reflect.New(slice.Type().Elem())
		// the options of dec carry unmarshalers, for the rules to apply to elem
		if err := json.UnmarshalDecode(dec, elem.Interface()); err != nil {
			return err
		}
		result = reflect.Append(result, elem.Elem())
	}
	if _, err := dec.ReadToken(); err != nil {
		return err
	}
	slice.Set(result)
	return nil
}
