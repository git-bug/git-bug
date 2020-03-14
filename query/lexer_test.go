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
		{"gibberish", nil},
		{"status:", nil},
		{":value", nil},

		{"status:open", []token{{"status", "open"}}},
		{"status:closed", []token{{"status", "closed"}}},

		{"author:rene", []token{{"author", "rene"}}},
		{`author:"René Descartes"`, []token{{"author", "René Descartes"}}},

		{
			`status:open status:closed author:rene author:"René Descartes"`,
			[]token{
				{"status", "open"},
				{"status", "closed"},
				{"author", "rene"},
				{"author", "René Descartes"},
			},
		},
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
