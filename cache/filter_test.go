package cache

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/git-bug/git-bug/entity"
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
			assert.Equal(t, tt.match, filter(t.Context(), excerpt, nil))
		})
	}
}

func TestIdentityFiltersDanglingId(t *testing.T) {
	alice := &IdentityExcerpt{id: entity.DeriveId([]byte("alice")), Name: "Alice"}
	dangling := entity.DeriveId([]byte("dangling"))
	resolvers := entity.Resolvers{&IdentityExcerpt{}: entity.MakeResolver(alice)}

	excerpt := &BugExcerpt{
		AuthorId:     dangling,
		Actors:       []entity.Id{dangling, alice.Id()},
		Participants: []entity.Id{dangling, alice.Id()},
	}

	tests := []struct {
		name   string
		filter Filter
		match  bool
	}{
		{name: "author", filter: AuthorFilter("alice"), match: false},
		{name: "actor", filter: ActorFilter("alice"), match: true},
		{name: "participant", filter: ParticipantFilter("alice"), match: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.match, tt.filter(t.Context(), excerpt, resolvers))
		})
	}
}
