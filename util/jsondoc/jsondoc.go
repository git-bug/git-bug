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
	"encoding"
	"encoding/base64"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
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
// is null isn't absent. An integer has neither a fraction nor an exponent, and
// lies between -maxInt and maxInt. A byte slice is strict base64, see
// unmarshalBytes. An array or a map holding null is an error.
//
// When v points to a struct, a member whose field has the tag
// `jsondoc:"required"` is an error when absent or null, as null reads as absent,
// see values.md §3 in the spec. The members of an embedded struct are those of
// v, as json inlines them. Only the members of v itself are checked: a nested
// struct, or one with its own UnmarshalJSON method, checks its own with
// Unmarshal.
func Unmarshal(data []byte, v any) error {
	if err := json.Unmarshal(data, v, unmarshalers); err != nil {
		return err
	}
	return checkRequired(data, v)
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
// its Go value. It rejects a null element of an array, and decodes the integers,
// the byte slices and the maps, for their rules. It leaves any other value, and a value whose
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
	case jsontext.KindString:
		if e := reflect.ValueOf(v).Elem(); isBytes(e.Type()) && !hasMethod(v) {
			return unmarshalBytes(dec, e)
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

func isBytes(t reflect.Type) bool {
	return t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Uint8
}

// unmarshalBytes decodes a string into e, a byte slice, as base64 by the rules of
// values.md §10 in the spec: the standard alphabet, padded, the unused bits of the
// last character being 0, and nothing else, no line break in particular.
func unmarshalBytes(dec *jsontext.Decoder, e reflect.Value) error {
	value, err := dec.ReadValue()
	if err != nil {
		return err
	}
	text, err := jsontext.AppendUnquote(nil, value)
	if err != nil {
		return err
	}
	// Strict rejects unused bits that aren't 0, but skips line breaks
	if bytes.ContainsAny(text, "\r\n") {
		return semanticError(e.Type(), value, "not base64")
	}
	b, err := base64.StdEncoding.Strict().AppendDecode(nil, text)
	if err != nil {
		return semanticError(e.Type(), value, "not base64")
	}
	e.SetBytes(b)
	return nil
}

// checkRequired returns an error if data, decoded into v, lacks a member that v
// requires, or holds it as null, see Unmarshal.
func checkRequired(data []byte, v any) error {
	t := reflect.TypeOf(v).Elem()
	if t.Kind() != reflect.Struct || hasMethod(v) {
		return nil
	}
	required := requiredMembers(t, nil)
	if len(required) == 0 {
		return nil
	}
	// a member that is null decodes to nil, as an absent one
	var values map[string]*jsontext.Value
	if err := json.Unmarshal(data, &values); err != nil {
		return err
	}
	for _, name := range required {
		if values[name] == nil {
			return fmt.Errorf("missing member %q", name)
		}
	}
	return nil
}

// requiredMembers appends to names those of the required members of t, a
// struct, as json reads them: its exported fields with the tag
// `jsondoc:"required"`, and those of its embedded structs.
func requiredMembers(t reflect.Type, names []string) []string {
	for field := range t.Fields() {
		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		switch {
		case name == "-":
		case field.Anonymous && name == "" && field.Type.Kind() == reflect.Struct:
			names = requiredMembers(field.Type, names)
		case field.IsExported() && field.Tag.Get("jsondoc") == "required":
			if name == "" {
				name = field.Name
			}
			names = append(names, name)
		}
	}
	return names
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
