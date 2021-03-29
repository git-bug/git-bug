package query

import (
	"testing"

	"github.com/stretchr/testify/require"
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

		// sub-qualifier positive testing
		{`key:subkey:"value:value"`, []token{newTokenKVV("key", "subkey", "value:value")}},

		// sub-qualifier negative testing
		{`key:subkey:value:value`, nil},
		{`key:subkey:`, nil},
		{`key:subkey:"value`, nil},

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
		t.Run(tc.input, func(t *testing.T) {
			tokens, err := tokenize(tc.input)
			if tc.tokens == nil {
				require.Error(t, err)
				require.Nil(t, tokens)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.tokens, tokens)
			}
		})
	}
}
