// Package jsondoc reads JSON documents, the ops blob of an entity and the
// version blob of an identity, by the rules of values.md §2 to §4 in the spec.
// JSON libraries disagree on invalid UTF-8, repeated names, the case of names
// or large integers, so every client applies these rules to reach the same
// verdict on a document.
//
// A reader decodes a document, and each part of it that it decodes further,
// with Unmarshal. A type with its own UnmarshalJSON method decodes its members
// with Unmarshal too, for the rules to apply within it.
package jsondoc

import (
	"bytes"
	"encoding"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strconv"
)

// maxInt is the largest integer, and -maxInt the smallest: every JSON
// implementation reads them exactly.
const maxInt = 1<<53 - 1

// Unmarshal decodes data, a JSON document or a part of one, into v. It rejects
// invalid UTF-8, an escaped surrogate other than a pair, a byte order mark, an
// object holding two members of the same name, and data after the document,
// even within what v keeps as raw JSON. Names match exactly, case included. A member that is null sets its Go value to
// zero, as an absent member leaves it in a new value: v is new, or a member that
// is null isn't absent. An integer has neither a fraction nor an exponent, and
// lies between -maxInt and maxInt. An array or a map holding null is an error.
func Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v, unmarshalers)
}

var unmarshalers = json.WithUnmarshalers(json.UnmarshalFromFunc(unmarshalValue))

var (
	legacyUnmarshaler = reflect.TypeFor[json.Unmarshaler]()
	unmarshalerFrom   = reflect.TypeFor[json.UnmarshalerFrom]()
)

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

// unmarshalValue is called for every value that Unmarshal decodes, v pointing to
// its Go value. It rejects a null element of an array, and decodes the integers
// and the maps, for their rules. It leaves any other value, and a value whose
// type has its own method, to the default decoding, by returning
// errors.ErrUnsupported.
func unmarshalValue(dec *jsontext.Decoder, v any) error {
	switch dec.PeekKind() {
	case jsontext.KindNull:
		// whatever the Go value, as an array of raw messages holds raw JSON
		if parent, _ := dec.StackIndex(dec.StackDepth()); parent == jsontext.KindBeginArray {
			return semanticError(reflect.TypeOf(v).Elem(), nil, "null in an array")
		}
	case jsontext.KindNumber:
		if e := reflect.ValueOf(v).Elem(); isInteger(e.Kind()) && !hasMethod(v) {
			return unmarshalInteger(dec, e)
		}
	case jsontext.KindBeginObject:
		if e := reflect.ValueOf(v).Elem(); isStringMap(e.Type()) && !hasMethod(v) {
			return unmarshalMap(dec, e)
		}
	}
	return errors.ErrUnsupported
}

// hasMethod tells if the type of v has its own method to decode JSON.
func hasMethod(v any) bool {
	t := reflect.TypeOf(v)
	// TODO: cleanup after fully moving to json/v2
	return t.Implements(legacyUnmarshaler) || t.Implements(unmarshalerFrom)
}

func isInteger(kind reflect.Kind) bool {
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return true
	}
	return false
}

// unmarshalInteger decodes a number into e, an integer.
func unmarshalInteger(dec *jsontext.Decoder, e reflect.Value) error {
	value, err := dec.ReadValue()
	if err != nil {
		return err
	}
	// a JSON number is a valid ParseInt input unless it has a fraction or an
	// exponent
	n, err := strconv.ParseInt(string(value), 10, 64)
	switch {
	case errors.Is(err, strconv.ErrSyntax):
		return semanticError(e.Type(), value, "not an integer")
	case err != nil || n > maxInt || n < -maxInt:
		return semanticError(e.Type(), value, "integer out of range")
	}
	// then it has to fit e
	switch {
	case e.CanInt() && !e.OverflowInt(n):
		e.SetInt(n)
	case e.CanUint() && n >= 0 && !e.OverflowUint(uint64(n)):
		e.SetUint(uint64(n))
	default:
		return semanticError(e.Type(), value, "integer out of range")
	}
	return nil
}

// Require returns an error unless data, an object, has each of names as a
// member that isn't null, as null reads as absent. A reader calls it for the
// members that a format requires and whose zero value is valid, such as an
// empty message: decoding can't tell them from absent ones. A required member
// whose zero value is invalid needs no call, as validating rejects it. See
// values.md §3 in the spec.
func Require(data []byte, names ...string) error {
	if len(names) == 0 {
		return nil
	}
	missing := slices.Clone(names)
	dec := jsontext.NewDecoder(bytes.NewReader(data))
	if _, err := dec.ReadToken(); err != nil {
		return err
	}
	for len(missing) > 0 && dec.PeekKind() == jsontext.KindString {
		token, err := dec.ReadToken()
		if err != nil {
			return err
		}
		// the token is valid until the next call to dec
		if name := token.String(); dec.PeekKind() != jsontext.KindNull {
			missing = slices.DeleteFunc(missing, func(required string) bool { return required == name })
		}
		if err := dec.SkipValue(); err != nil {
			return err
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing member %q", missing[0])
	}
	return nil
}

var textUnmarshaler = reflect.TypeFor[encoding.TextUnmarshaler]()

// isStringMap tells if t is a map whose keys are strings, read as they are.
func isStringMap(t reflect.Type) bool {
	return t.Kind() == reflect.Map &&
		t.Key().Kind() == reflect.String &&
		!reflect.PointerTo(t.Key()).Implements(textUnmarshaler)
}

// unmarshalMap decodes an object into m, a map whose keys are strings (see
// isStringMap), each value as Unmarshal does.
func unmarshalMap(dec *jsontext.Decoder, m reflect.Value) error {
	if _, err := dec.ReadToken(); err != nil {
		return err
	}
	result := reflect.MakeMap(m.Type())
	for dec.PeekKind() != jsontext.KindEndObject {
		token, err := dec.ReadToken()
		if err != nil {
			return err
		}
		// the token is valid until the next call to dec
		key := reflect.ValueOf(token.String()).Convert(m.Type().Key())
		if dec.PeekKind() == jsontext.KindNull {
			return semanticError(m.Type(), nil, "null in a map")
		}
		elem := reflect.New(m.Type().Elem())
		// the options of dec carry unmarshalers, for the rules to apply to elem
		if err := json.UnmarshalDecode(dec, elem.Interface()); err != nil {
			return err
		}
		result.SetMapIndex(key, elem.Elem())
	}
	if _, err := dec.ReadToken(); err != nil {
		return err
	}
	m.Set(result)
	return nil
}
