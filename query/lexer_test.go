package query

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTokenize(t *testing.T) {
	var tests = []struct {
		input  string
		tokens []token
	}{
		{"status:", nil},
		{":value", nil},

		{"status:open", []token{newTokenKV("status", "open")}},
		{"status:closed", []token{newTokenKV("status", "closed")}},

		{"author:rene", []token{newTokenKV("author", "rene")}},
		{`author:"René Descartes"`, []token{newTokenKV("author", "René Descartes")}},

		{
			`status:open status:closed author:rene author:"René Descartes"`,
			[]token{
				newTokenKV("status", "open"),
				newTokenKV("status", "closed"),
				newTokenKV("author", "rene"),
				newTokenKV("author", "René Descartes"),
			},
		},

		// quotes
		{`key:"value value"`, []token{newTokenKV("key", "value value")}},
		{`key:'value value'`, []token{newTokenKV("key", "value value")}},
		// unmatched quotes
		{`key:'value value`, nil},
		{`key:value value'`, nil},

		// full text search
		{"search", []token{newTokenSearch("search")}},
		{"search more terms", []token{
			newTokenSearch("search"),
			newTokenSearch("more"),
			newTokenSearch("terms"),
		}},
		{"search \"more terms\"", []token{
			newTokenSearch("search"),
			newTokenSearch("more terms"),
		}},
	}

	for _, tc := range tests {
		tokens, err := tokenize(tc.input)
		if tc.tokens == nil {
			assert.Error(t, err)
			assert.Nil(t, tokens)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, tc.tokens, tokens)
		}
	}
}
