package query

import (
	"fmt"
	"strings"
	"unicode"
)

type token struct {
	qualifier string
	value     string
}

// TODO: this lexer implementation behave badly with unmatched quotes.
// A hand written one would be better instead of relying on strings.FieldsFunc()

// tokenize parse and break a input into tokens ready to be
// interpreted later by a parser to get the semantic.
func tokenize(query string) ([]token, error) {
	fields := splitQuery(query)

	var tokens []token
	for _, field := range fields {
		split := strings.Split(field, ":")
		if len(split) != 2 {
			return nil, fmt.Errorf("can't tokenize \"%s\"", field)
		}

		if len(split[0]) == 0 {
			return nil, fmt.Errorf("can't tokenize \"%s\": empty qualifier", field)
		}
		if len(split[1]) == 0 {
			return nil, fmt.Errorf("empty value for qualifier \"%s\"", split[0])
		}

		tokens = append(tokens, token{
			qualifier: split[0],
			value:     removeQuote(split[1]),
		})
	}
	return tokens, nil
}

func splitQuery(query string) []string {
	lastQuote := rune(0)
	f := func(c rune) bool {
		switch {
		case c == lastQuote:
			lastQuote = rune(0)
			return false
		case lastQuote != rune(0):
			return false
		case unicode.In(c, unicode.Quotation_Mark):
			lastQuote = c
			return false
		default:
			return unicode.IsSpace(c)
		}
	}

	return strings.FieldsFunc(query, f)
}

func removeQuote(field string) string {
	if len(field) >= 2 {
		if field[0] == '"' && field[len(field)-1] == '"' {
			return field[1 : len(field)-1]
		}
	}
	return field
}
