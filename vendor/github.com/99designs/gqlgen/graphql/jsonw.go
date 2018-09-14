package graphql

import (
	"io"
	"strconv"
)

var nullLit = []byte(`null`)
var trueLit = []byte(`true`)
var falseLit = []byte(`false`)
var openBrace = []byte(`{`)
var closeBrace = []byte(`}`)
var openBracket = []byte(`[`)
var closeBracket = []byte(`]`)
var colon = []byte(`:`)
var comma = []byte(`,`)

var Null = &lit{nullLit}
var True = &lit{trueLit}
var False = &lit{falseLit}

type Marshaler interface {
	MarshalGQL(w io.Writer)
}

type Unmarshaler interface {
	UnmarshalGQL(v interface{}) error
}

type OrderedMap struct {
	Keys   []string
	Values []Marshaler
}

type WriterFunc func(writer io.Writer)

func (f WriterFunc) MarshalGQL(w io.Writer) {
	f(w)
}

func NewOrderedMap(len int) *OrderedMap {
	return &OrderedMap{
		Keys:   make([]string, len),
		Values: make([]Marshaler, len),
	}
}

func (m *OrderedMap) Add(key string, value Marshaler) {
	m.Keys = append(m.Keys, key)
	m.Values = append(m.Values, value)
}

func (m *OrderedMap) MarshalGQL(writer io.Writer) {
	writer.Write(openBrace)
	for i, key := range m.Keys {
		if i != 0 {
			writer.Write(comma)
		}
		io.WriteString(writer, strconv.Quote(key))
		writer.Write(colon)
		m.Values[i].MarshalGQL(writer)
	}
	writer.Write(closeBrace)
}

type Array []Marshaler

func (a Array) MarshalGQL(writer io.Writer) {
	writer.Write(openBracket)
	for i, val := range a {
		if i != 0 {
			writer.Write(comma)
		}
		val.MarshalGQL(writer)
	}
	writer.Write(closeBracket)
}

type lit struct{ b []byte }

func (l lit) MarshalGQL(w io.Writer) {
	w.Write(l.b)
}
