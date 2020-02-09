package cache

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTitleFilter(t *testing.T) {
	tests := []struct {
		name  string
		title string
		query string
		match bool
	}{
		{name: "complete match", title: "hello world", query: "hello world", match: true},
		{name: "partial match", title: "hello world", query: "hello", match: true},
		{name: "no match", title: "hello world", query: "foo", match: false},
		{name: "cased title", title: "Hello World", query: "hello", match: true},
		{name: "cased query", title: "hello world", query: "Hello", match: true},

		// Those following tests should work eventually but are left for a future iteration.

		// {name: "cased accents", title: "ÑOÑO", query: "ñoño", match: true},
		// {name: "natural language matching", title: "Århus", query: "Aarhus", match: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := TitleFilter(tt.query)
			excerpt := &BugExcerpt{Title: tt.title}
			assert.Equal(t, tt.match, filter(excerpt, nil))
		})
	}
}
