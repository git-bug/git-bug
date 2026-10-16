package cache

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/query"
)

func TestIdQuery(t *testing.T) {
	const id = "265e26f0123456789abcdef0123456789abcdef0123456789abcdef0123456789ab"
	excerpt := &BugExcerpt{
		id:     entity.Id(id),
		Status: common.OpenStatus,
		Labels: []common.Label{"bug"},
	}

	tests := []struct {
		query string
		match bool
	}{
		{query: "id:" + id, match: true},
		{query: "id:265e26f", match: true},
		{query: "id:265E26F", match: true},
		{query: "id:9ed1af4", match: false},
		{query: "id:e26f012", match: false},
		{query: "id:" + id + "0", match: false},
		{query: "id:9ed1af4 id:265e26f", match: true},
		{query: "id:9ed1af4 id:abcdef0", match: false},
		{query: "id:265e26f status:open label:bug", match: true},
		{query: "id:265e26f status:closed", match: false},
		{query: "id:265e26f label:other", match: false},
		{query: "status:open label:bug", match: true},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			q, err := query.Parse(tt.query)
			require.NoError(t, err)
			assert.Equal(t, tt.match, compileMatcher(q.Filters).Match(excerpt, nil))
		})
	}
}

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
